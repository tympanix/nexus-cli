package operations

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tympanix/nexus-cli/internal/config"
	"github.com/tympanix/nexus-cli/internal/nexusapi"
	"github.com/tympanix/nexus-cli/internal/util"
)

// TestUploadDryRun tests that upload dry-run mode doesn't actually upload files
func TestUploadDryRun(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-upload-dryrun-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create test files
	testFiles := map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Setup mock server
	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Capture logger output
	var logBuf strings.Builder
	logger := util.NewLogger(&logBuf)

	opts := &UploadOptions{
		Logger:    logger,
		QuietMode: true,
		DryRun:    true,
	}

	// Upload files with dry-run
	err = uploadFiles(testDir, "test-repo", "", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Verify that the server received no upload requests
	if len(server.UploadedFiles) > 0 {
		t.Errorf("Expected no uploads in dry-run mode, but got %d uploads", len(server.UploadedFiles))
	}

	// Verify log output contains dry-run message
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Files uploaded:") {
		t.Errorf("Expected log to contain 'Files uploaded:', got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "size:") {
		t.Errorf("Expected log to contain 'size:', got: %s", logOutput)
	}
}

// TestDownloadDryRun tests that download dry-run mode doesn't actually download files
func TestDownloadDryRun(t *testing.T) {
	testContent := "test content"
	testPath1 := "/test-folder/test1.txt"
	testPath2 := "/test-folder/test2.txt"

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	server.AddAsset("test-repo", testPath1, nexusapi.Asset{}, []byte(testContent))
	server.AddAsset("test-repo", testPath2, nexusapi.Asset{}, []byte(testContent))

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
		DryRun:            true,
		Recursive:         true,
	}

	destDir, err := os.MkdirTemp("", "test-download-dryrun-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	status := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if status != DownloadSuccess {
		t.Fatal("Download failed")
	}

	// Verify that no files were actually downloaded
	files, err := os.ReadDir(destDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}
	if len(files) > 0 {
		t.Errorf("Expected no files to be downloaded in dry-run mode, but found %d files", len(files))
	}

	// Verify log output contains dry-run message
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Files downloaded:") {
		t.Errorf("Expected log to contain 'Files downloaded:', got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "size:") {
		t.Errorf("Expected log to contain 'size:', got: %s", logOutput)
	}
}

// TestUploadCompressedDryRun tests that upload dry-run mode works with compressed uploads
func TestUploadCompressedDryRun(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-upload-compressed-dryrun-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create test files
	testFiles := map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Setup mock server
	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Capture logger output
	var logBuf strings.Builder
	logger := util.NewLogger(&logBuf)

	opts := &UploadOptions{
		Logger:    logger,
		QuietMode: true,
		DryRun:    true,
		Compress:  true,
	}

	// Upload files with dry-run and compression
	err = uploadFilesCompressedWithArchiveName(testDir, "test-repo", "", "archive.tar.gz", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Verify that the server received no upload requests
	if len(server.UploadedFiles) > 0 {
		t.Errorf("Expected no uploads in dry-run mode, but got %d uploads", len(server.UploadedFiles))
	}

	// Verify log output contains dry-run message
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Dry-run mode") {
		t.Errorf("Expected log to contain 'Dry-run mode', got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "Would upload compressed archive containing 2 files") {
		t.Errorf("Expected log to contain 'Would upload compressed archive containing 2 files', got: %s", logOutput)
	}
}
