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
	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Create test options
	opts := &UploadOptions{
		Logger:    util.NewLogger(io.Discard),
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

// TestUploadWithChecksumValidation tests that upload skips files with matching checksums
func TestUploadWithChecksumValidation(t *testing.T) {
	testContent := "test content for checksum validation"

	testDir, err := os.MkdirTemp("", "test-upload-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	// Add an existing asset with matching checksum (SHA1 of testContent)
	// Query pattern is "//*" when basePath is empty
	server.AddAssetWithQuery("test-repo", "//*", nexusapi.Asset{
		Path:       "/test.txt",
		ID:         "test-id",
		Repository: "test-repo",
		FileSize:   int64(len(testContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "d38a2973b20670764496e490a7f638302eb96602",
		},
	})

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	var logBuf strings.Builder
	logger := util.NewLogger(&logBuf)

	opts := &UploadOptions{
		Logger:    logger,
		QuietMode: true,
	}

	// Set checksum algorithm
	err = opts.SetChecksumAlgorithm("sha1")
	if err != nil {
		t.Fatalf("Failed to set checksum algorithm: %v", err)
	}

	err = uploadFiles(testDir, "test-repo", "", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Check that no files were uploaded (all were skipped)
	uploadedFiles := server.GetUploadedFiles()
	if len(uploadedFiles) != 0 {
		t.Errorf("Expected 0 files to be uploaded (all skipped), got %d", len(uploadedFiles))
	}

	// Check log output contains expected message about all files skipped
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "All 1 files already exist with matching checksums") {
		t.Errorf("Expected log message about all files skipped, got: %s", logOutput)
	}
}

// TestUploadWithChecksumMismatch tests that upload uploads files when checksums don't match
func TestUploadWithChecksumMismatch(t *testing.T) {
	testContent := "test content for checksum validation"

	testDir, err := os.MkdirTemp("", "test-upload-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	// Add an existing asset with different checksum
	server.AddAssetWithQuery("test-repo", "//*", nexusapi.Asset{
		Path:       "/test.txt",
		ID:         "test-id",
		Repository: "test-repo",
		FileSize:   int64(len(testContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "wrongchecksum",
		},
	})

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	var logBuf strings.Builder
	logger := util.NewLogger(&logBuf)

	opts := &UploadOptions{
		Logger:    logger,
		QuietMode: true,
	}

	// Set checksum algorithm
	err = opts.SetChecksumAlgorithm("sha1")
	if err != nil {
		t.Fatalf("Failed to set checksum algorithm: %v", err)
	}

	err = uploadFiles(testDir, "test-repo", "", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Check that the file was uploaded (checksum didn't match)
	uploadedFiles := server.GetUploadedFiles()
	if len(uploadedFiles) != 1 {
		t.Errorf("Expected 1 file to be uploaded, got %d", len(uploadedFiles))
	}

	// Check log output
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Uploaded 1 files from") {
		t.Errorf("Expected log message about 1 file uploaded, got: %s", logOutput)
	}
}

// TestUploadWithSkipChecksum tests that upload skips files based on existence when --skip-checksum is used
func TestUploadWithSkipChecksum(t *testing.T) {
	testContent := "test content"

	testDir, err := os.MkdirTemp("", "test-upload-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	// Add an existing asset (checksum doesn't matter when skip-checksum is enabled)
	server.AddAssetWithQuery("test-repo", "//*", nexusapi.Asset{
		Path:       "/test.txt",
		ID:         "test-id",
		Repository: "test-repo",
		FileSize:   int64(len(testContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "anychecksum",
		},
	})

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	var logBuf strings.Builder
	logger := util.NewLogger(&logBuf)

	opts := &UploadOptions{
		Logger:       logger,
		QuietMode:    true,
		SkipChecksum: true,
	}

	err = uploadFiles(testDir, "test-repo", "", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Check that no files were uploaded (all were skipped based on existence)
	uploadedFiles := server.GetUploadedFiles()
	if len(uploadedFiles) != 0 {
		t.Errorf("Expected 0 files to be uploaded (all skipped), got %d", len(uploadedFiles))
	}

	// Check log output contains expected message about all files skipped
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "All 1 files already exist with matching checksums") {
		t.Errorf("Expected log message about all files skipped, got: %s", logOutput)
	}
}

// TestUploadWithForce tests that upload uploads all files when --force is used, regardless of existence or checksum
func TestUploadWithForce(t *testing.T) {
	testContent := "test content"

	testDir, err := os.MkdirTemp("", "test-upload-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	// Add an existing asset with matching checksum
	server.AddAssetWithQuery("test-repo", "//*", nexusapi.Asset{
		Path:       "/test.txt",
		ID:         "test-id",
		Repository: "test-repo",
		FileSize:   int64(len(testContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "d38a2973b20670764496e490a7f638302eb96602",
		},
	})

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	var logBuf strings.Builder
	logger := util.NewLogger(&logBuf)

	opts := &UploadOptions{
		Logger:    logger,
		QuietMode: true,
		Force:     true,
	}

	// Set checksum algorithm (even with matching checksum, file should be uploaded when Force is true)
	err = opts.SetChecksumAlgorithm("sha1")
	if err != nil {
		t.Fatalf("Failed to set checksum algorithm: %v", err)
	}

	err = uploadFiles(testDir, "test-repo", "", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Check that the file was uploaded despite having matching checksum
	uploadedFiles := server.GetUploadedFiles()
	if len(uploadedFiles) != 1 {
		t.Errorf("Expected 1 file to be uploaded (force flag set), got %d", len(uploadedFiles))
	}

	// Check log output
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Uploaded 1 files from") {
		t.Errorf("Expected log message about 1 file uploaded, got: %s", logOutput)
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

			config := &config.Config{
				NexusURL: server.URL,
				Username: "test",
				Password: "test",
			}

			opts := &UploadOptions{
				Logger:    util.NewLogger(io.Discard),
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

// TestUploadToNonExistentRepository tests uploading to a repository that doesn't exist
func TestUploadToNonExistentRepository(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-upload-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	// Mark the repository as not found
	server.SetRepositoryNotFound("non-existent-repo")

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	opts := &UploadOptions{
		Logger:    util.NewLogger(io.Discard),
		QuietMode: true,
	}

	err = uploadFiles(testDir, "non-existent-repo", "", config, opts)
	if err == nil {
		t.Fatal("Expected error when uploading to non-existent repository, got nil")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error message to contain 'not found', got: %v", err)
	}

	if !strings.Contains(err.Error(), "non-existent-repo") {
		t.Errorf("Expected error message to contain repository name, got: %v", err)
	}
}

// TestUploadCompressedGzipWithProgressBar tests uploading with gzip compression and progress bar validation
func TestUploadCompressedGzipWithProgressBar(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-upload-gzip-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFiles := map[string]string{
		"file1.txt": "Test content 1",
		"file2.txt": "Test content 2",
		"file3.txt": "Test content 3",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	opts := &UploadOptions{
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
		Compress:          true,
		CompressionFormat: archive.FormatGzip,
	}

	err = uploadFilesWithArchiveName(testDir, "test-repo", "", "archive.tar.gz", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	uploadedFiles := server.GetUploadedFiles()
	if len(uploadedFiles) == 0 {
		t.Fatal("Archive was not uploaded")
	}

	if uploadedFiles[0].Filename != "archive.tar.gz" {
		t.Errorf("Expected archive filename 'archive.tar.gz', got '%s'", uploadedFiles[0].Filename)
	}
}

// TestUploadCompressedZstdWithProgressBar tests uploading with zstd compression and progress bar validation
func TestUploadCompressedZstdWithProgressBar(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-upload-zstd-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFiles := map[string]string{
		"file1.txt": "Test content 1",
		"file2.txt": "Test content 2",
		"file3.txt": "Test content 3",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	opts := &UploadOptions{
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
		Compress:          true,
		CompressionFormat: archive.FormatZstd,
	}

	err = uploadFilesWithArchiveName(testDir, "test-repo", "", "archive.tar.zst", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	uploadedFiles := server.GetUploadedFiles()
	if len(uploadedFiles) == 0 {
		t.Fatal("Archive was not uploaded")
	}

	if uploadedFiles[0].Filename != "archive.tar.zst" {
		t.Errorf("Expected archive filename 'archive.tar.zst', got '%s'", uploadedFiles[0].Filename)
	}
}

// TestUploadCompressedZipWithProgressBar tests uploading with zip compression and progress bar validation
func TestUploadCompressedZipWithProgressBar(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-upload-zip-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFiles := map[string]string{
		"file1.txt": "Test content 1",
		"file2.txt": "Test content 2",
		"file3.txt": "Test content 3",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	opts := &UploadOptions{
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
		Compress:          true,
		CompressionFormat: archive.FormatZip,
	}

	err = uploadFilesWithArchiveName(testDir, "test-repo", "", "archive.zip", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	uploadedFiles := server.GetUploadedFiles()
	if len(uploadedFiles) == 0 {
		t.Fatal("Archive was not uploaded")
	}

	if uploadedFiles[0].Filename != "archive.zip" {
		t.Errorf("Expected archive filename 'archive.zip', got '%s'", uploadedFiles[0].Filename)
	}
}
