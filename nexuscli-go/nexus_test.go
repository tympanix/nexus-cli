package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if !strings.Contains(r.URL.Path, "/service/rest/v1/components") {
			t.Errorf("Unexpected URL path: %s", r.URL.Path)
		}

		receivedRepository = r.URL.Query().Get("repository")

		// Parse multipart form
		err := r.ParseMultipartForm(32 << 20) // 32 MB
		if err != nil {
			t.Errorf("Failed to parse multipart form: %v", err)
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		// Check for uploaded file
		file, header, err := r.FormFile("raw.asset1")
		if err != nil {
			t.Errorf("Failed to get uploaded file: %v", err)
			http.Error(w, "No file uploaded", http.StatusBadRequest)
			return
		}
		defer file.Close()

		uploadedFilename = header.Filename

		// Read file content
		content, err := io.ReadAll(file)
		if err != nil {
			t.Errorf("Failed to read file content: %v", err)
			http.Error(w, "Failed to read file", http.StatusInternalServerError)
			return
		}
		uploadedContent = string(content)

		// Return success
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	// Create test config
	config := &Config{
		NexusURL: server.URL,
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

	var serverURL string

	// Create mock Nexus server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle asset listing request
		if strings.Contains(r.URL.Path, "/service/rest/v1/search/assets") {
			// Return mock asset list
			assets := searchResponse{
				Items: []Asset{
					{
						DownloadURL: serverURL + "/repository/test-repo" + testPath,
						Path:        testPath,
						ID:          "test-id",
						Repository:  "test-repo",
						FileSize:    int64(len(testContent)),
						Checksum: Checksum{
							SHA1: "abc123",
						},
					},
				},
				ContinuationToken: "",
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(assets)
			return
		}

		// Handle file download request
		if strings.Contains(r.URL.Path, "/repository/test-repo") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(testContent))
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()
	serverURL = server.URL

	// Create test config
	config := &Config{
		NexusURL: server.URL,
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

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedRepo = r.URL.Query().Get("repository")
				receivedQuery = r.URL.Query().Get("q")

				assets := searchResponse{
					Items:             []Asset{},
					ContinuationToken: "",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(assets)
			}))
			defer server.Close()

			config := &Config{
				NexusURL: server.URL,
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

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	config := &Config{
		NexusURL: server.URL,
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

	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/service/rest/v1/search/assets") {
			assets := searchResponse{
				Items: []Asset{
					{
						DownloadURL: serverURL + "/repository/test-repo" + testPath,
						Path:        testPath,
						ID:          "test-id",
						Repository:  "test-repo",
						FileSize:    int64(len(testContent)),
						Checksum: Checksum{
							SHA1: "abc123",
						},
					},
				},
				ContinuationToken: "",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(assets)
			return
		}

		if strings.Contains(r.URL.Path, "/repository/test-repo") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(testContent))
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()
	serverURL = server.URL

	config := &Config{
		NexusURL: server.URL,
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

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedRepo = r.URL.Query().Get("repository")
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			config := &Config{
				NexusURL: server.URL,
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

// TestDownloadStripFolders tests the strip-folders functionality
func TestDownloadStripFolders(t *testing.T) {
	testContent1 := "Content 1"
	testContent2 := "Content 2"
	testPath1 := "/test-folder/subfolder/file1.txt"
	testPath2 := "/test-folder/other/file2.txt"

	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/service/rest/v1/search/assets") {
			assets := searchResponse{
				Items: []Asset{
					{
						DownloadURL: serverURL + "/repository/test-repo" + testPath1,
						Path:        testPath1,
						ID:          "test-id-1",
						Repository:  "test-repo",
						FileSize:    int64(len(testContent1)),
						Checksum: Checksum{
							SHA1: "abc123",
						},
					},
					{
						DownloadURL: serverURL + "/repository/test-repo" + testPath2,
						Path:        testPath2,
						ID:          "test-id-2",
						Repository:  "test-repo",
						FileSize:    int64(len(testContent2)),
						Checksum: Checksum{
							SHA1: "def456",
						},
					},
				},
				ContinuationToken: "",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(assets)
			return
		}

		if strings.Contains(r.URL.Path, testPath1) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(testContent1))
			return
		}

		if strings.Contains(r.URL.Path, testPath2) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(testContent2))
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()
	serverURL = server.URL

	config := &Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	tests := []struct {
		name         string
		stripFolders bool
		expectedPath1 string
		expectedPath2 string
	}{
		{
			name:         "with strip folders enabled",
			stripFolders: true,
			expectedPath1: "file1.txt",
			expectedPath2: "file2.txt",
		},
		{
			name:         "with strip folders disabled",
			stripFolders: false,
			expectedPath1: testPath1,
			expectedPath2: testPath2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &DownloadOptions{
				ChecksumAlgorithm: "sha1",
				SkipChecksum:      false,
				StripFolders:      tt.stripFolders,
				Logger:            NewLogger(io.Discard),
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

			// Validate file1 was downloaded to expected path
			downloadedFile1 := filepath.Join(destDir, tt.expectedPath1)
			content1, err := os.ReadFile(downloadedFile1)
			if err != nil {
				t.Fatalf("Failed to read downloaded file1 at %s: %v", downloadedFile1, err)
			}
			if string(content1) != testContent1 {
				t.Errorf("Expected file1 content '%s', got '%s'", testContent1, string(content1))
			}

			// Validate file2 was downloaded to expected path
			downloadedFile2 := filepath.Join(destDir, tt.expectedPath2)
			content2, err := os.ReadFile(downloadedFile2)
			if err != nil {
				t.Fatalf("Failed to read downloaded file2 at %s: %v", downloadedFile2, err)
			}
			if string(content2) != testContent2 {
				t.Errorf("Expected file2 content '%s', got '%s'", testContent2, string(content2))
			}
		})
	}
}

