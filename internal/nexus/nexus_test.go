package nexus

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tympanix/nexus-cli/internal/nexusapi"
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

	// Create mock Nexus server
	server := nexusapi.NewMockNexusServer()
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
	uploadedFiles := server.GetUploadedFiles()
	receivedRepository := server.LastUploadRepo

	if len(uploadedFiles) != 1 {
		t.Fatalf("Expected 1 uploaded file, got %d", len(uploadedFiles))
	}

	if string(uploadedFiles[0].Content) != testContent {
		t.Errorf("Expected uploaded content '%s', got '%s'", testContent, string(uploadedFiles[0].Content))
	}

	if uploadedFiles[0].Filename != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got '%s'", uploadedFiles[0].Filename)
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
	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	// Setup mock data
	downloadURL := server.URL + "/repository/test-repo" + testPath
	server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
		DownloadURL: downloadURL,
		Path:        testPath,
		ID:          "test-id",
		Repository:  "test-repo",
		FileSize:    int64(len(testContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "abc123",
		},
	})
	server.SetAssetContent("/repository/test-repo"+testPath, []byte(testContent))

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
			server := nexusapi.NewMockNexusServer()
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

			if server.LastListRepo != tt.wantRepo {
				t.Errorf("Expected repository '%s', got '%s'", tt.wantRepo, server.LastListRepo)
			}

			expectedPath := tt.src
			if server.LastListPath != expectedPath {
				t.Errorf("Expected path '%s', got '%s'", expectedPath, server.LastListPath)
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

	server := nexusapi.NewMockNexusServer()
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

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	downloadURL := server.URL + "/repository/test-repo" + testPath
	server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
		DownloadURL: downloadURL,
		Path:        testPath,
		ID:          "test-id",
		Repository:  "test-repo",
		FileSize:    int64(len(testContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "abc123",
		},
	})
	server.SetAssetContent("/repository/test-repo"+testPath, []byte(testContent))

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

// TestDownloadFlatten tests that download with flatten option removes base path
func TestDownloadFlatten(t *testing.T) {
	testContent := "test content"
	basePath := "/test-folder"
	subPath := "/subdir"
	fileName := "/file.txt"

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	// Test two files:
	// 1. /test-folder/file.txt -> should become file.txt
	// 2. /test-folder/subdir/file.txt -> should become subdir/file.txt
	downloadURL1 := server.URL + "/repository/test-repo" + basePath + fileName
	downloadURL2 := server.URL + "/repository/test-repo" + basePath + subPath + fileName

	server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
		DownloadURL: downloadURL1,
		Path:        basePath + fileName,
		ID:          "test-id-1",
		Repository:  "test-repo",
		FileSize:    int64(len(testContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "abc123",
		},
	})
	server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
		DownloadURL: downloadURL2,
		Path:        basePath + subPath + fileName,
		ID:          "test-id-2",
		Repository:  "test-repo",
		FileSize:    int64(len(testContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "def456",
		},
	})

	server.SetAssetContent("/repository/test-repo"+basePath+fileName, []byte(testContent))
	server.SetAssetContent("/repository/test-repo"+basePath+subPath+fileName, []byte(testContent))

	config := &Config{
		NexusURL: server.URL,
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

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	downloadURL := server.URL + "/repository/test-repo" + testPath
	server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
		DownloadURL: downloadURL,
		Path:        testPath,
		ID:          "test-id",
		Repository:  "test-repo",
		FileSize:    int64(len(testContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "abc123",
		},
	})
	server.SetAssetContent("/repository/test-repo"+testPath, []byte(testContent))

	config := &Config{
		NexusURL: server.URL,
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

			server := nexusapi.NewMockNexusServer()
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

			if server.LastUploadRepo != tt.wantRepo {
				t.Errorf("Expected repository '%s', got '%s'", tt.wantRepo, server.LastUploadRepo)
			}
		})
	}
}

// TestDownloadDeleteExtra tests that download with delete-extra removes local files not in Nexus
func TestDownloadDeleteExtra(t *testing.T) {
	testContent := "test content"
	basePath := "/test-folder"
	fileName := "/file.txt"

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	// Nexus only has one file: /test-folder/file.txt
	downloadURL := server.URL + "/repository/test-repo" + basePath + fileName
	server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
		DownloadURL: downloadURL,
		Path:        basePath + fileName,
		ID:          "test-id-1",
		Repository:  "test-repo",
		FileSize:    int64(len(testContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "abc123",
		},
	})
	server.SetAssetContent("/repository/test-repo"+basePath+fileName, []byte(testContent))

	config := &Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Create destination directory with extra files that should be deleted
	destDir, err := os.MkdirTemp("", "test-download-delete-extra-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Create the directory structure
	testFolderPath := filepath.Join(destDir, "test-folder")
	if err := os.MkdirAll(testFolderPath, 0755); err != nil {
		t.Fatalf("Failed to create test-folder directory: %v", err)
	}

	// Create extra files that should be deleted
	extraFile1 := filepath.Join(testFolderPath, "extra-file1.txt")
	if err := os.WriteFile(extraFile1, []byte("extra content 1"), 0644); err != nil {
		t.Fatalf("Failed to create extra file 1: %v", err)
	}

	extraFile2 := filepath.Join(testFolderPath, "extra-file2.txt")
	if err := os.WriteFile(extraFile2, []byte("extra content 2"), 0644); err != nil {
		t.Fatalf("Failed to create extra file 2: %v", err)
	}

	// Verify extra files exist before download
	if _, err := os.Stat(extraFile1); os.IsNotExist(err) {
		t.Fatalf("Extra file 1 should exist before download")
	}
	if _, err := os.Stat(extraFile2); os.IsNotExist(err) {
		t.Fatalf("Extra file 2 should exist before download")
	}

	// Download with delete-extra enabled
	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Flatten:           false,
		DeleteExtra:       true,
		Logger:            NewLogger(io.Discard),
		QuietMode:         true,
	}

	success := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if !success {
		t.Fatal("Download failed")
	}

	// Check that the downloaded file exists
	expectedFile := filepath.Join(testFolderPath, "file.txt")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Expected file at %s, but it does not exist", expectedFile)
	}

	// Check that extra files were deleted
	if _, err := os.Stat(extraFile1); !os.IsNotExist(err) {
		t.Errorf("Extra file 1 should have been deleted at %s", extraFile1)
	}
	if _, err := os.Stat(extraFile2); !os.IsNotExist(err) {
		t.Errorf("Extra file 2 should have been deleted at %s", extraFile2)
	}
}

// TestDownloadNoDeleteExtra tests that download without delete-extra preserves local files
func TestDownloadNoDeleteExtra(t *testing.T) {
	testContent := "test content"
	basePath := "/test-folder"
	fileName := "/file.txt"

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	// Nexus only has one file: /test-folder/file.txt
	downloadURL := server.URL + "/repository/test-repo" + basePath + fileName
	server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
		DownloadURL: downloadURL,
		Path:        basePath + fileName,
		ID:          "test-id-1",
		Repository:  "test-repo",
		FileSize:    int64(len(testContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "abc123",
		},
	})
	server.SetAssetContent("/repository/test-repo"+basePath+fileName, []byte(testContent))

	config := &Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Create destination directory with extra files that should NOT be deleted
	destDir, err := os.MkdirTemp("", "test-download-no-delete-extra-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Create the directory structure
	testFolderPath := filepath.Join(destDir, "test-folder")
	if err := os.MkdirAll(testFolderPath, 0755); err != nil {
		t.Fatalf("Failed to create test-folder directory: %v", err)
	}

	// Create extra files that should be preserved
	extraFile1 := filepath.Join(testFolderPath, "extra-file1.txt")
	if err := os.WriteFile(extraFile1, []byte("extra content 1"), 0644); err != nil {
		t.Fatalf("Failed to create extra file 1: %v", err)
	}

	// Download with delete-extra disabled
	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Flatten:           false,
		DeleteExtra:       false,
		Logger:            NewLogger(io.Discard),
		QuietMode:         true,
	}

	success := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if !success {
		t.Fatal("Download failed")
	}

	// Check that the downloaded file exists
	expectedFile := filepath.Join(testFolderPath, "file.txt")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Expected file at %s, but it does not exist", expectedFile)
	}

	// Check that extra files were NOT deleted
	if _, err := os.Stat(extraFile1); os.IsNotExist(err) {
		t.Errorf("Extra file 1 should have been preserved at %s", extraFile1)
	}
}

// TestDownloadDeleteExtraWithFlatten tests that delete-extra works correctly with flatten option
func TestDownloadDeleteExtraWithFlatten(t *testing.T) {
	testContent := "test content"
	basePath := "/test-folder"
	fileName := "/file.txt"

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	// Nexus only has one file: /test-folder/file.txt
	downloadURL := server.URL + "/repository/test-repo" + basePath + fileName
	server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
		DownloadURL: downloadURL,
		Path:        basePath + fileName,
		ID:          "test-id-1",
		Repository:  "test-repo",
		FileSize:    int64(len(testContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "abc123",
		},
	})
	server.SetAssetContent("/repository/test-repo"+basePath+fileName, []byte(testContent))

	config := &Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Create destination directory with extra files that should be deleted
	destDir, err := os.MkdirTemp("", "test-download-delete-extra-flatten-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Create extra files that should be deleted (in flattened location)
	extraFile1 := filepath.Join(destDir, "extra-file1.txt")
	if err := os.WriteFile(extraFile1, []byte("extra content 1"), 0644); err != nil {
		t.Fatalf("Failed to create extra file 1: %v", err)
	}

	extraFile2 := filepath.Join(destDir, "extra-file2.txt")
	if err := os.WriteFile(extraFile2, []byte("extra content 2"), 0644); err != nil {
		t.Fatalf("Failed to create extra file 2: %v", err)
	}

	// Download with both delete-extra and flatten enabled
	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Flatten:           true,
		DeleteExtra:       true,
		Logger:            NewLogger(io.Discard),
		QuietMode:         true,
	}

	success := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if !success {
		t.Fatal("Download failed")
	}

	// Check that the downloaded file exists (in flattened location)
	expectedFile := filepath.Join(destDir, "file.txt")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("Expected file at %s, but it does not exist", expectedFile)
	}

	// Check that extra files were deleted
	if _, err := os.Stat(extraFile1); !os.IsNotExist(err) {
		t.Errorf("Extra file 1 should have been deleted at %s", extraFile1)
	}
	if _, err := os.Stat(extraFile2); !os.IsNotExist(err) {
		t.Errorf("Extra file 2 should have been deleted at %s", extraFile2)
	}
}
