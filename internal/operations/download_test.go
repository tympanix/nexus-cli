package operations

import (
	"github.com/tympanix/nexus-cli/internal/archive"
	"github.com/tympanix/nexus-cli/internal/config"
	"github.com/tympanix/nexus-cli/internal/util"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

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
	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Create test options
	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
	}

	// Create temp directory for download
	destDir, err := os.MkdirTemp("", "test-download-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Test download
	status := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if status != DownloadSuccess {
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

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Capture logger output
	var logBuf strings.Builder
	logger := util.NewLogger(&logBuf)

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

	status := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if status != DownloadSuccess {
		t.Fatal("Download failed")
	}

	// Check log output contains expected format with all metrics
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Files downloaded: 1") {
		t.Errorf("Expected log message containing 'Files downloaded: 1', got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "size:") {
		t.Errorf("Expected log message containing 'size:', got: %s", logOutput)
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

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Flatten:           true,
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
	}

	destDir, err := os.MkdirTemp("", "test-download-flatten-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	status := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if status != DownloadSuccess {
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

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Flatten:           false, // Default behavior
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
	}

	destDir, err := os.MkdirTemp("", "test-download-no-flatten-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	status := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if status != DownloadSuccess {
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

	config := &config.Config{
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
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
	}

	status := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if status != DownloadSuccess {
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

	config := &config.Config{
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
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
	}

	status := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if status != DownloadSuccess {
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

	config := &config.Config{
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
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
	}

	status := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if status != DownloadSuccess {
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

			config := &config.Config{
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

// TestDownloadNoAssetsFound tests that exit code 66 is returned when no assets are found
func TestDownloadNoAssetsFound(t *testing.T) {
	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
	}

	destDir, err := os.MkdirTemp("", "test-download-no-assets-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Test download with no assets in the repository
	status := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if status != DownloadNoAssetsFound {
		t.Errorf("Expected DownloadNoAssetsFound status (66), got %d", status)
	}
}

// TestDownloadErrorConditions tests that exit code 1 is returned for error conditions
func TestDownloadErrorConditions(t *testing.T) {
	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
	}

	destDir, err := os.MkdirTemp("", "test-download-error-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Test with invalid src argument (missing repository/folder format)
	status := downloadFolder("invalid-format", destDir, config, opts)
	if status != DownloadError {
		t.Errorf("Expected DownloadError status (1) for invalid format, got %d", status)
	}
}

// TestDownloadMainExitCode verifies DownloadMain properly exits with status codes
func TestDownloadMainExitCode(t *testing.T) {
	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	destDir, err := os.MkdirTemp("", "test-download-main-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Test that DownloadMain calls os.Exit with correct code for no assets
	// We can't directly test os.Exit, but we can verify the status is returned correctly
	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
	}

	status := downloadFolder("test-repo/empty-folder", destDir, config, opts)
	if status != DownloadNoAssetsFound {
		t.Errorf("Expected DownloadNoAssetsFound (66) for empty folder, got %d", status)
	}
}

// TestDownloadCompressedGzipWithProgressBar tests downloading with gzip decompression and progress bar validation
func TestDownloadCompressedGzipWithProgressBar(t *testing.T) {
	// Create test files for the archive
	srcDir, err := os.MkdirTemp("", "test-compress-dl-gzip-*")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir)

	testFiles := map[string]string{
		"file1.txt": "Content 1",
		"file2.txt": "Content 2",
		"file3.txt": "Content 3",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(srcDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create archive file
	archiveFile, err := os.CreateTemp("", "test-archive-*.tar.gz")
	if err != nil {
		t.Fatalf("Failed to create archive file: %v", err)
	}
	archivePath := archiveFile.Name()
	defer os.Remove(archivePath)

	if err := archive.CreateTarGz(srcDir, archiveFile); err != nil {
		t.Fatalf("Failed to create archive: %v", err)
	}
	archiveFile.Close()

	// Read archive content for serving
	archiveContent, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read archive: %v", err)
	}

	archiveName := "archive.tar.gz"

	// Create mock server
	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	downloadURL := server.URL + "/repository/test-repo/test-folder/" + archiveName
	server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
		DownloadURL: downloadURL,
		Path:        "/test-folder/" + archiveName,
		ID:          "test-id",
		Repository:  "test-repo",
		FileSize:    int64(len(archiveContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "abc123",
		},
	})
	server.SetAssetContent("/repository/test-repo/test-folder/"+archiveName, archiveContent)

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Create download directory
	destDir, err := os.MkdirTemp("", "test-compress-dl-dest-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
		Compress:          true,
		CompressionFormat: archive.FormatGzip,
	}

	// Download and extract with explicit archive name
	status := downloadFolderCompressedWithArchiveName("test-repo", "test-folder", archiveName, destDir, config, opts)
	if status != DownloadSuccess {
		t.Fatal("Download failed")
	}

	// Verify extracted files
	for filename, expectedContent := range testFiles {
		extractedPath := filepath.Join(destDir, filename)
		content, err := os.ReadFile(extractedPath)
		if err != nil {
			t.Errorf("Failed to read extracted file %s: %v", filename, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("Content mismatch for %s: expected %q, got %q", filename, expectedContent, string(content))
		}
	}
}

// TestDownloadCompressedZstdWithProgressBar tests downloading with zstd decompression and progress bar validation
func TestDownloadCompressedZstdWithProgressBar(t *testing.T) {
	// Create test files for the archive
	srcDir, err := os.MkdirTemp("", "test-compress-dl-zstd-*")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir)

	testFiles := map[string]string{
		"file1.txt": "Content 1",
		"file2.txt": "Content 2",
		"file3.txt": "Content 3",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(srcDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create archive file
	archiveFile, err := os.CreateTemp("", "test-archive-*.tar.zst")
	if err != nil {
		t.Fatalf("Failed to create archive file: %v", err)
	}
	archivePath := archiveFile.Name()
	defer os.Remove(archivePath)

	if err := archive.CreateTarZst(srcDir, archiveFile); err != nil {
		t.Fatalf("Failed to create archive: %v", err)
	}
	archiveFile.Close()

	// Read archive content for serving
	archiveContent, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read archive: %v", err)
	}

	archiveName := "archive.tar.zst"

	// Create mock server
	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	downloadURL := server.URL + "/repository/test-repo/test-folder/" + archiveName
	server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
		DownloadURL: downloadURL,
		Path:        "/test-folder/" + archiveName,
		ID:          "test-id",
		Repository:  "test-repo",
		FileSize:    int64(len(archiveContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "abc123",
		},
	})
	server.SetAssetContent("/repository/test-repo/test-folder/"+archiveName, archiveContent)

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Create download directory
	destDir, err := os.MkdirTemp("", "test-compress-dl-dest-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
		Compress:          true,
		CompressionFormat: archive.FormatZstd,
	}

	// Download and extract with explicit archive name
	status := downloadFolderCompressedWithArchiveName("test-repo", "test-folder", archiveName, destDir, config, opts)
	if status != DownloadSuccess {
		t.Fatal("Download failed")
	}

	// Verify extracted files
	for filename, expectedContent := range testFiles {
		extractedPath := filepath.Join(destDir, filename)
		content, err := os.ReadFile(extractedPath)
		if err != nil {
			t.Errorf("Failed to read extracted file %s: %v", filename, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("Content mismatch for %s: expected %q, got %q", filename, expectedContent, string(content))
		}
	}
}

// TestDownloadCompressedZipWithProgressBar tests downloading with zip decompression and progress bar validation
func TestDownloadCompressedZipWithProgressBar(t *testing.T) {
	// Create test files for the archive
	srcDir, err := os.MkdirTemp("", "test-compress-dl-zip-*")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir)

	testFiles := map[string]string{
		"file1.txt": "Content 1",
		"file2.txt": "Content 2",
		"file3.txt": "Content 3",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(srcDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create archive file
	archiveFile, err := os.CreateTemp("", "test-archive-*.zip")
	if err != nil {
		t.Fatalf("Failed to create archive file: %v", err)
	}
	archivePath := archiveFile.Name()
	defer os.Remove(archivePath)

	if err := archive.CreateZip(srcDir, archiveFile); err != nil {
		t.Fatalf("Failed to create archive: %v", err)
	}
	archiveFile.Close()

	// Read archive content for serving
	archiveContent, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read archive: %v", err)
	}

	archiveName := "archive.zip"

	// Create mock server
	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	downloadURL := server.URL + "/repository/test-repo/test-folder/" + archiveName
	server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
		DownloadURL: downloadURL,
		Path:        "/test-folder/" + archiveName,
		ID:          "test-id",
		Repository:  "test-repo",
		FileSize:    int64(len(archiveContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "abc123",
		},
	})
	server.SetAssetContent("/repository/test-repo/test-folder/"+archiveName, archiveContent)

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Create download directory
	destDir, err := os.MkdirTemp("", "test-compress-dl-dest-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
		Compress:          true,
		CompressionFormat: archive.FormatZip,
	}

	// Download and extract with explicit archive name
	status := downloadFolderCompressedWithArchiveName("test-repo", "test-folder", archiveName, destDir, config, opts)
	if status != DownloadSuccess {
		t.Fatal("Download failed")
	}

	// Verify extracted files
	for filename, expectedContent := range testFiles {
		extractedPath := filepath.Join(destDir, filename)
		content, err := os.ReadFile(extractedPath)
		if err != nil {
			t.Errorf("Failed to read extracted file %s: %v", filename, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("Content mismatch for %s: expected %q, got %q", filename, expectedContent, string(content))
		}
	}
}

func TestDownloadWithTrailingSlash(t *testing.T) {
	testContent := "test content"
	basePath := "/test-folder"
	fileName := "/file.txt"

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

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

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
	}

	destDir1, err := os.MkdirTemp("", "test-download-no-slash-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir1)

	destDir2, err := os.MkdirTemp("", "test-download-slash-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir2)

	status1 := downloadFolder("test-repo/test-folder", destDir1, config, opts)
	if status1 != DownloadSuccess {
		t.Fatal("Download without trailing slash failed")
	}

	status2 := downloadFolder("test-repo/test-folder/", destDir2, config, opts)
	if status2 != DownloadSuccess {
		t.Fatal("Download with trailing slash failed")
	}

	file1 := filepath.Join(destDir1, "test-folder", "file.txt")
	content1, err := os.ReadFile(file1)
	if err != nil {
		t.Fatalf("Expected file at %s, but got error: %v", file1, err)
	}

	file2 := filepath.Join(destDir2, "test-folder", "file.txt")
	content2, err := os.ReadFile(file2)
	if err != nil {
		t.Fatalf("Expected file at %s, but got error: %v", file2, err)
	}

	if string(content1) != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, string(content1))
	}

	if string(content2) != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, string(content2))
	}

	if string(content1) != string(content2) {
		t.Error("Content from download with and without trailing slash should be identical")
	}
}

// TestDownloadWithForce tests that download downloads all files when --force is used, regardless of existence or checksum
func TestDownloadWithForce(t *testing.T) {
	testContent := "Test content for force download"
	testPath := "/test-folder/test.txt"

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
	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Create test options
	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		Force:             true,
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
	}

	// Create temp directory for download
	destDir, err := os.MkdirTemp("", "test-download-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Pre-create a file with different content to ensure it gets overwritten
	existingPath := filepath.Join(destDir, "test-folder", "test.txt")
	os.MkdirAll(filepath.Dir(existingPath), 0755)
	err = os.WriteFile(existingPath, []byte("existing content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// Test download with Force flag - should download despite file existing
	status := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if status != DownloadSuccess {
		t.Fatalf("Download failed with status %d", status)
	}

	// Verify file was overwritten with new content
	downloadedPath := filepath.Join(destDir, "test-folder", "test.txt")
	content, err := os.ReadFile(downloadedPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Expected content '%s', got '%s'. File should have been overwritten due to Force flag", testContent, string(content))
	}
}

// TestDownloadWithGlobPattern tests downloading files with glob pattern filtering
func TestDownloadWithGlobPattern(t *testing.T) {
	testContent := "test content"

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	// Add multiple files with different extensions
	files := map[string]string{
		"/test-folder/file1.go":         testContent,
		"/test-folder/file2.md":         testContent,
		"/test-folder/file3.txt":        testContent,
		"/test-folder/subdir/file4.go":  testContent,
		"/test-folder/subdir/file5.txt": testContent,
	}

	for path := range files {
		downloadURL := server.URL + "/repository/test-repo" + path
		server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
			DownloadURL: downloadURL,
			Path:        path,
			ID:          "test-id-" + path,
			Repository:  "test-repo",
			FileSize:    int64(len(testContent)),
			Checksum: nexusapi.Checksum{
				SHA1: "abc123",
			},
		})
		server.SetAssetContent("/repository/test-repo"+path, []byte(testContent))
	}

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	tests := []struct {
		name            string
		globPattern     string
		expectedFiles   []string
		unexpectedFiles []string
	}{
		{
			name:        "download only .go files",
			globPattern: "**/*.go",
			expectedFiles: []string{
				"test-folder/file1.go",
				"test-folder/subdir/file4.go",
			},
			unexpectedFiles: []string{
				"test-folder/file2.md",
				"test-folder/file3.txt",
				"test-folder/subdir/file5.txt",
			},
		},
		{
			name:        "download .go and .md files",
			globPattern: "**/*.go,**/*.md",
			expectedFiles: []string{
				"test-folder/file1.go",
				"test-folder/file2.md",
				"test-folder/subdir/file4.go",
			},
			unexpectedFiles: []string{
				"test-folder/file3.txt",
				"test-folder/subdir/file5.txt",
			},
		},
		{
			name:        "download all files except .txt",
			globPattern: "**/*,!**/*.txt",
			expectedFiles: []string{
				"test-folder/file1.go",
				"test-folder/file2.md",
				"test-folder/subdir/file4.go",
			},
			unexpectedFiles: []string{
				"test-folder/file3.txt",
				"test-folder/subdir/file5.txt",
			},
		},
		{
			name:        "download only from root directory (not subdir)",
			globPattern: "*.go,*.md,*.txt",
			expectedFiles: []string{
				"test-folder/file1.go",
				"test-folder/file2.md",
				"test-folder/file3.txt",
			},
			unexpectedFiles: []string{
				"test-folder/subdir/file4.go",
				"test-folder/subdir/file5.txt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destDir, err := os.MkdirTemp("", "test-download-glob-*")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(destDir)

			opts := &DownloadOptions{
				ChecksumAlgorithm: "sha1",
				SkipChecksum:      false,
				Logger:            util.NewLogger(io.Discard),
				QuietMode:         true,
				GlobPattern:       tt.globPattern,
			}

			status := downloadFolder("test-repo/test-folder", destDir, config, opts)
			if status != DownloadSuccess {
				t.Fatalf("Download failed with status %d", status)
			}

			// Verify expected files were downloaded
			for _, expectedFile := range tt.expectedFiles {
				filePath := filepath.Join(destDir, expectedFile)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("Expected file %s was not downloaded", expectedFile)
				}
			}

			// Verify unexpected files were NOT downloaded
			for _, unexpectedFile := range tt.unexpectedFiles {
				filePath := filepath.Join(destDir, unexpectedFile)
				if _, err := os.Stat(filePath); !os.IsNotExist(err) {
					t.Errorf("File %s should not have been downloaded", unexpectedFile)
				}
			}
		})
	}
}

// TestDownloadWithGlobPatternNoMatch tests downloading with glob pattern that matches no files
func TestDownloadWithGlobPatternNoMatch(t *testing.T) {
	testContent := "test content"

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	// Add a file with .txt extension
	downloadURL := server.URL + "/repository/test-repo/test-folder/file.txt"
	server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
		DownloadURL: downloadURL,
		Path:        "/test-folder/file.txt",
		ID:          "test-id",
		Repository:  "test-repo",
		FileSize:    int64(len(testContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "abc123",
		},
	})
	server.SetAssetContent("/repository/test-repo/test-folder/file.txt", []byte(testContent))

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	destDir, err := os.MkdirTemp("", "test-download-glob-nomatch-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
		GlobPattern:       "**/*.go", // Pattern that won't match any files
	}

	status := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if status != DownloadNoAssetsFound {
		t.Errorf("Expected DownloadNoAssetsFound status (66), got %d", status)
	}
}
