package nexus

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tympanix/nexus-cli/internal/testutil"
)

// TestUploadSingleFile tests uploading a single file to the Nexus API
func TestUploadSingleFile(t *testing.T) {
	// Create test directory and file in a real temp directory
	testDir, err := os.MkdirTemp("", "test-upload-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	testContent := "Hello, Nexus!"

	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Track upload request
	var uploadedContent string
	var uploadedFilename string
	receivedRepository := ""

	// Create mock Nexus server
	mockServer := testutil.NewMockNexusServer(t)
	defer mockServer.Close()

	mockServer.AddHandler(&testutil.UploadHandler{
		ExpectedRepo: "test-repo",
		OnUpload: func(r *http.Request, t *testing.T) {
			receivedRepository = r.URL.Query().Get("repository")

			// Parse multipart form
			err := r.ParseMultipartForm(32 << 20) // 32 MB
			if err != nil {
				t.Errorf("Failed to parse multipart form: %v", err)
				return
			}

			// Check for uploaded file
			file, header, err := r.FormFile("raw.asset1")
			if err != nil {
				t.Errorf("Failed to get uploaded file: %v", err)
				return
			}
			defer file.Close()

			uploadedFilename = header.Filename

			// Read file content
			content, err := io.ReadAll(file)
			if err != nil {
				t.Errorf("Failed to read file content: %v", err)
				return
			}
			uploadedContent = string(content)
		},
	})

	// Create test config
	config := &Config{
		NexusURL: mockServer.URL(),
		Username: "test",
		Password: "test",
	}

	// Create test options
	opts := &UploadOptions{
		Logger:    NewLogger(io.Discard),
		QuietMode: true,
	}

	// Test upload
	err = uploadFiles(testDir, "test-repo", "", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Validate uploaded content
	if uploadedContent != testContent {
		t.Errorf("Expected uploaded content '%s', got '%s'", testContent, uploadedContent)
	}

	if uploadedFilename != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got '%s'", uploadedFilename)
	}

	if receivedRepository != "test-repo" {
		t.Errorf("Expected repository 'test-repo', got '%s'", receivedRepository)
	}
}

// TestDownloadSingleFile tests downloading a directory with a single file
func TestDownloadSingleFile(t *testing.T) {
	testContent := "Downloaded content from Nexus"
	testPath := "/test-folder/downloaded.txt"

	// Create mock Nexus server
	mockServer := testutil.NewMockNexusServer(t)
	defer mockServer.Close()

	// Add asset list handler
	mockServer.AddHandler(&testutil.AssetListHandler{
		Assets: []testutil.Asset{
			{
				DownloadURL: mockServer.URL() + "/repository/test-repo" + testPath,
				Path:        testPath,
				ID:          "test-id",
				Repository:  "test-repo",
				FileSize:    int64(len(testContent)),
				Checksum: testutil.Checksum{
					SHA1: "abc123",
				},
			},
		},
	})

	// Add download handler
	mockServer.AddHandler(&testutil.DownloadHandler{
		PathPrefix:  "/repository/test-repo",
		Content:     []byte(testContent),
		ContentType: "text/plain",
	})

	// Create test config
	config := &Config{
		NexusURL: mockServer.URL(),
		Username: "test",
		Password: "test",
	}

	// Create test options
	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            NewLogger(io.Discard),
		QuietMode:         true,
	}

	// Create temp directory for download
	destDir, err := os.MkdirTemp("", "test-download-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Test download
	success := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if !success {
		t.Fatal("Download failed")
	}

	// Validate downloaded content
	downloadedFile := filepath.Join(destDir, testPath)
	content, err := os.ReadFile(downloadedFile)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected downloaded content '%s', got '%s'", testContent, string(content))
	}
}

// TestURLConstruction tests that URLs are properly constructed with special characters
func TestURLConstruction(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		src        string
		wantRepo   string
		wantQuery  string
	}{
		{
			name:       "simple repository and path",
			repository: "test-repo",
			src:        "test-folder",
			wantRepo:   "test-repo",
			wantQuery:  "/test-folder/*",
		},
		{
			name:       "repository with special chars",
			repository: "test repo",
			src:        "folder",
			wantRepo:   "test repo",
			wantQuery:  "/folder/*",
		},
		{
			name:       "path with special chars",
			repository: "repo",
			src:        "path with spaces",
			wantRepo:   "repo",
			wantQuery:  "/path with spaces/*",
		},
		{
			name:       "path with percent encoding",
			repository: "repo",
			src:        "path%20test",
			wantRepo:   "repo",
			wantQuery:  "/path%20test/*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receivedRepo := ""
			receivedQuery := ""

			mockServer := testutil.NewMockNexusServer(t)
			defer mockServer.Close()

			mockServer.AddHandler(&testutil.AssetListHandler{
				Assets: []testutil.Asset{},
				ValidateFunc: func(r *http.Request, t *testing.T) {
					receivedRepo = r.URL.Query().Get("repository")
					receivedQuery = r.URL.Query().Get("q")
				},
			})

			config := &Config{
				NexusURL: mockServer.URL(),
				Username: "test",
				Password: "test",
			}

			_, err := listAssets(tt.repository, tt.src, config)
			if err != nil {
				t.Fatalf("listAssets failed: %v", err)
			}

			if receivedRepo != tt.wantRepo {
				t.Errorf("Expected repository '%s', got '%s'", tt.wantRepo, receivedRepo)
			}

			if receivedQuery != tt.wantQuery {
				t.Errorf("Expected query '%s', got '%s'", tt.wantQuery, receivedQuery)
			}
		})
	}
}

// TestUploadLogging tests that upload logging is simplified
func TestUploadLogging(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-upload-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mockServer := testutil.NewMockNexusServer(t)
	defer mockServer.Close()

	mockServer.AddHandler(&testutil.UploadHandler{})

	config := &Config{
		NexusURL: mockServer.URL(),
		Username: "test",
		Password: "test",
	}

	// Capture logger output
	var logBuf strings.Builder
	logger := NewLogger(&logBuf)

	opts := &UploadOptions{
		Logger:    logger,
		QuietMode: true,
	}

	err = uploadFiles(testDir, "test-repo", "", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Check log output contains expected message
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Uploaded 1 files from") {
		t.Errorf("Expected log message containing 'Uploaded 1 files from', got: %s", logOutput)
	}
}

// TestDownloadLogging tests that download logging is simplified
func TestDownloadLogging(t *testing.T) {
	testContent := "test content"
	testPath := "/test-folder/test.txt"

	mockServer := testutil.NewMockNexusServer(t)
	defer mockServer.Close()

	mockServer.AddHandler(&testutil.AssetListHandler{
		Assets: []testutil.Asset{
			{
				DownloadURL: mockServer.URL() + "/repository/test-repo" + testPath,
				Path:        testPath,
				ID:          "test-id",
				Repository:  "test-repo",
				FileSize:    int64(len(testContent)),
				Checksum: testutil.Checksum{
					SHA1: "abc123",
				},
			},
		},
	})

	mockServer.AddHandler(&testutil.DownloadHandler{
		PathPrefix:  "/repository/test-repo",
		Content:     []byte(testContent),
		ContentType: "text/plain",
	})

	config := &Config{
		NexusURL: mockServer.URL(),
		Username: "test",
		Password: "test",
	}

	// Capture logger output
	var logBuf strings.Builder
	logger := NewLogger(&logBuf)

	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            logger,
		QuietMode:         true,
	}

	destDir, err := os.MkdirTemp("", "test-download-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	success := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if !success {
		t.Fatal("Download failed")
	}

	// Check log output contains expected format with all metrics
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Downloaded 1/1 files") {
		t.Errorf("Expected log message containing 'Downloaded 1/1 files', got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "(skipped: 0, failed: 0)") {
		t.Errorf("Expected log message containing '(skipped: 0, failed: 0)', got: %s", logOutput)
	}
}

// TestDownloadFlatten tests that download with flatten option removes base path
func TestDownloadFlatten(t *testing.T) {
	testContent := "test content"
	basePath := "/test-folder"
	subPath := "/subdir"
	fileName := "/file.txt"

	mockServer := testutil.NewMockNexusServer(t)
	defer mockServer.Close()

	// Test two files:
	// 1. /test-folder/file.txt -> should become file.txt
	// 2. /test-folder/subdir/file.txt -> should become subdir/file.txt
	mockServer.AddHandler(&testutil.AssetListHandler{
		Assets: []testutil.Asset{
			{
				DownloadURL: mockServer.URL() + "/repository/test-repo" + basePath + fileName,
				Path:        basePath + fileName,
				ID:          "test-id-1",
				Repository:  "test-repo",
				FileSize:    int64(len(testContent)),
				Checksum: testutil.Checksum{
					SHA1: "abc123",
				},
			},
			{
				DownloadURL: mockServer.URL() + "/repository/test-repo" + basePath + subPath + fileName,
				Path:        basePath + subPath + fileName,
				ID:          "test-id-2",
				Repository:  "test-repo",
				FileSize:    int64(len(testContent)),
				Checksum: testutil.Checksum{
					SHA1: "def456",
				},
			},
		},
	})

	mockServer.AddHandler(&testutil.DownloadHandler{
		PathPrefix:  "/repository/test-repo",
		Content:     []byte(testContent),
		ContentType: "text/plain",
	})

	config := &Config{
		NexusURL: mockServer.URL(),
		Username: "test",
		Password: "test",
	}

	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Flatten:           true,
		Logger:            NewLogger(io.Discard),
		QuietMode:         true,
	}

	destDir, err := os.MkdirTemp("", "test-download-flatten-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	success := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if !success {
		t.Fatal("Download failed")
	}

	// Check that file.txt exists directly in destDir (not in test-folder/)
	file1 := filepath.Join(destDir, "file.txt")
	if _, err := os.Stat(file1); os.IsNotExist(err) {
		t.Errorf("Expected file at %s, but it does not exist", file1)
	}

	// Check that subdir/file.txt exists (subdirectory preserved)
	file2 := filepath.Join(destDir, "subdir", "file.txt")
	if _, err := os.Stat(file2); os.IsNotExist(err) {
		t.Errorf("Expected file at %s, but it does not exist", file2)
	}

	// Verify that test-folder directory was NOT created
	oldPath := filepath.Join(destDir, "test-folder")
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("Expected test-folder directory to NOT exist at %s, but it does", oldPath)
	}
}

// TestDownloadNoFlatten tests that download without flatten preserves full path
func TestDownloadNoFlatten(t *testing.T) {
	testContent := "test content"
	testPath := "/test-folder/file.txt"

	mockServer := testutil.NewMockNexusServer(t)
	defer mockServer.Close()

	mockServer.AddHandler(&testutil.AssetListHandler{
		Assets: []testutil.Asset{
			{
				DownloadURL: mockServer.URL() + "/repository/test-repo" + testPath,
				Path:        testPath,
				ID:          "test-id",
				Repository:  "test-repo",
				FileSize:    int64(len(testContent)),
				Checksum: testutil.Checksum{
					SHA1: "abc123",
				},
			},
		},
	})

	mockServer.AddHandler(&testutil.DownloadHandler{
		PathPrefix:  "/repository/test-repo",
		Content:     []byte(testContent),
		ContentType: "text/plain",
	})

	config := &Config{
		NexusURL: mockServer.URL(),
		Username: "test",
		Password: "test",
	}

	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Flatten:           false, // Default behavior
		Logger:            NewLogger(io.Discard),
		QuietMode:         true,
	}

	destDir, err := os.MkdirTemp("", "test-download-no-flatten-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	success := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if !success {
		t.Fatal("Download failed")
	}

	// Check that full path is preserved (test-folder/file.txt)
	expectedFile := filepath.Join(destDir, "test-folder", "file.txt")
	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Expected file at %s, but got error: %v", expectedFile, err)
	}

	if string(content) != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, string(content))
	}
}

// TestUploadURLConstruction tests that upload URLs are properly constructed
func TestUploadURLConstruction(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		wantRepo   string
	}{
		{
			name:       "simple repository",
			repository: "test-repo",
			wantRepo:   "test-repo",
		},
		{
			name:       "repository with special chars",
			repository: "test repo",
			wantRepo:   "test repo",
		},
		{
			name:       "repository with percent encoding",
			repository: "test%20repo",
			wantRepo:   "test%20repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir, err := os.MkdirTemp("", "test-upload-*")
			if err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}
			defer os.RemoveAll(testDir)

			testFile := filepath.Join(testDir, "test.txt")
			err = os.WriteFile(testFile, []byte("test"), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			receivedRepo := ""

			mockServer := testutil.NewMockNexusServer(t)
			defer mockServer.Close()

			mockServer.AddHandler(&testutil.UploadHandler{
				OnUpload: func(r *http.Request, t *testing.T) {
					receivedRepo = r.URL.Query().Get("repository")
				},
			})

			config := &Config{
				NexusURL: mockServer.URL(),
				Username: "test",
				Password: "test",
			}

			opts := &UploadOptions{
				Logger:    NewLogger(io.Discard),
				QuietMode: true,
			}

			err = uploadFiles(testDir, tt.repository, "", config, opts)
			if err != nil {
				t.Fatalf("Upload failed: %v", err)
			}

			if receivedRepo != tt.wantRepo {
				t.Errorf("Expected repository '%s', got '%s'", tt.wantRepo, receivedRepo)
			}
		})
	}
}
