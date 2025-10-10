package operations

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tympanix/nexus-cli/internal/archive"
	"github.com/tympanix/nexus-cli/internal/config"
	"github.com/tympanix/nexus-cli/internal/nexusapi"
	"github.com/tympanix/nexus-cli/internal/util"
)

// TestCompressedUpload tests uploading files as a compressed archive
func TestCompressedUpload(t *testing.T) {
	// Create test files
	testDir, err := os.MkdirTemp("", "test-compress-upload-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFiles := map[string]string{
		"file1.txt":        "Content 1",
		"file2.txt":        "Content 2",
		"subdir/file3.txt": "Content 3",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testDir, filename)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create mock server
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

	// Upload compressed with explicit archive name
	err = uploadFilesWithArchiveName(testDir, "test-repo", "test-folder", "test-repo-test-folder.tar.gz", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	uploadedFiles := server.GetUploadedFiles()

	if len(uploadedFiles) == 0 {
		t.Fatal("Archive was not uploaded")
	}

	expectedName := "test-repo-test-folder.tar.gz"
	if uploadedFiles[0].Filename != expectedName {
		t.Errorf("Expected archive name %q, got %q", expectedName, uploadedFiles[0].Filename)
	}
}

// TestCompressedUploadWithExplicitName tests uploading with an explicit archive name
func TestCompressedUploadWithExplicitName(t *testing.T) {
	// Create test files
	testDir, err := os.MkdirTemp("", "test-compress-upload-explicit-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFiles := map[string]string{
		"file1.txt": "Content 1",
		"file2.txt": "Content 2",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create mock server
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

	// Upload with explicit archive name via uploadFilesWithArchiveName
	err = uploadFilesWithArchiveName(testDir, "test-repo", "test-folder", "custom-archive.tar.gz", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	uploadedFiles := server.GetUploadedFiles()

	if len(uploadedFiles) == 0 {
		t.Fatal("No archive was uploaded")
	}

	expectedName := "custom-archive.tar.gz"
	if uploadedFiles[0].Filename != expectedName {
		t.Errorf("Expected archive name %q, got %q", expectedName, uploadedFiles[0].Filename)
	}
}

// TestCompressedUploadWithoutExplicitName tests that upload fails when compress is used without explicit archive name
func TestCompressedUploadWithoutExplicitName(t *testing.T) {
	// Create test files
	testDir, err := os.MkdirTemp("", "test-compress-upload-noname-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFiles := map[string]string{
		"file1.txt": "Content 1",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	config := &config.Config{
		NexusURL: "http://localhost:8081",
		Username: "test",
		Password: "test",
	}

	opts := &UploadOptions{
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
		Compress:          true,
		CompressionFormat: archive.FormatGzip,
	}

	// Upload without explicit archive name should fail
	err = uploadFilesWithArchiveName(testDir, "test-repo", "test-folder", "", config, opts)
	if err == nil {
		t.Fatal("Expected error when uploading with compress but no explicit archive name")
	}

	expectedErrorMsg := "when using --compress, you must specify the"
	if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Expected error message to contain %q, got %q", expectedErrorMsg, err.Error())
	}
}

// TestCompressedDownload tests downloading and extracting a compressed archive
func TestCompressedDownload(t *testing.T) {
	// Create test files for the archive
	srcDir, err := os.MkdirTemp("", "test-compress-dl-src-*")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir)

	testFiles := map[string]string{
		"file1.txt":        "Content 1",
		"file2.txt":        "Content 2",
		"subdir/file3.txt": "Content 3",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(srcDir, filename)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
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

	archiveName := "test-repo-test-folder.tar.gz"

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

// TestCompressedDownloadWithExplicitName tests downloading with an explicit archive name
func TestCompressedDownloadWithExplicitName(t *testing.T) {
	// Create test files for the archive
	srcDir, err := os.MkdirTemp("", "test-compress-dl-explicit-*")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir)

	testFiles := map[string]string{
		"file1.txt": "Content 1",
		"file2.txt": "Content 2",
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

	customArchiveName := "my-custom-name.tar.gz"

	// Create mock server
	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	downloadURL := server.URL + "/repository/test-repo/test-folder/" + customArchiveName
	server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
		DownloadURL: downloadURL,
		Path:        "/test-folder/" + customArchiveName,
		ID:          "test-id",
		Repository:  "test-repo",
		FileSize:    int64(len(archiveContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "abc123",
		},
	})
	server.SetAssetContent("/repository/test-repo/test-folder/"+customArchiveName, archiveContent)

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

	// Download with explicit archive name via downloadFolderCompressedWithArchiveName
	status := downloadFolderCompressedWithArchiveName("test-repo", "test-folder", customArchiveName, destDir, config, opts)
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

// TestCompressedDownloadWithoutExplicitName tests that download fails when compress is used without explicit archive name
func TestCompressedDownloadWithoutExplicitName(t *testing.T) {
	config := &config.Config{
		NexusURL: "http://localhost:8081",
		Username: "test",
		Password: "test",
	}

	// Create download directory
	destDir, err := os.MkdirTemp("", "test-compress-dl-noname-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Capture logger output
	var logBuf strings.Builder
	logger := util.NewLogger(&logBuf)

	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            logger,
		QuietMode:         true,
		Compress:          true,
		CompressionFormat: archive.FormatGzip,
	}

	// Download without explicit archive name should fail (return false)
	status := downloadFolderCompressedWithArchiveName("test-repo", "test-folder", "", destDir, config, opts)
	if status == DownloadSuccess {
		t.Fatal("Expected download to fail when using compress without explicit archive name")
	}

	// Verify error message is printed
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "when using --compress, you must specify the .tar.gz filename in the source path") {
		t.Errorf("Expected error message about missing archive filename, got: %s", logOutput)
	}
}

// TestCompressedDownloadWithoutExplicitNameDifferentFormats tests error messages for different compression formats
func TestCompressedDownloadWithoutExplicitNameDifferentFormats(t *testing.T) {
	config := &config.Config{
		NexusURL: "http://localhost:8081",
		Username: "test",
		Password: "test",
	}

	destDir, err := os.MkdirTemp("", "test-compress-dl-formats-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	testCases := []struct {
		format          archive.Format
		expectedMessage string
	}{
		{archive.FormatGzip, ".tar.gz"},
		{archive.FormatZstd, ".tar.zst"},
		{archive.FormatZip, ".zip"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.format), func(t *testing.T) {
			var logBuf strings.Builder
			logger := util.NewLogger(&logBuf)

			opts := &DownloadOptions{
				ChecksumAlgorithm: "sha1",
				SkipChecksum:      false,
				Logger:            logger,
				QuietMode:         true,
				Compress:          true,
				CompressionFormat: tc.format,
			}

			status := downloadFolderCompressedWithArchiveName("test-repo", "test-folder", "", destDir, config, opts)
			if status == DownloadSuccess {
				t.Fatal("Expected download to fail when using compress without explicit archive name")
			}

			logOutput := logBuf.String()
			if !strings.Contains(logOutput, tc.expectedMessage) {
				t.Errorf("Expected error message to contain %s, got: %s", tc.expectedMessage, logOutput)
			}
		})
	}
}

// TestCompressedRoundTrip tests the full upload-download cycle with compression
func TestCompressedRoundTrip(t *testing.T) {
	// This test simulates uploading files as compressed archive,
	// then downloading and extracting them

	// Create test files
	srcDir, err := os.MkdirTemp("", "test-roundtrip-src-*")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir)

	testFiles := map[string]string{
		"file1.txt":          "Content 1",
		"file2.txt":          "Content 2",
		"subdir/file3.txt":   "Content 3",
		"deep/nest/file4.md": "# Markdown content",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(srcDir, filename)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	archiveName := "test-repo-test-folder.tar.gz"

	// Create mock server
	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Upload compressed
	uploadOpts := &UploadOptions{
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
		Compress:          true,
		CompressionFormat: archive.FormatGzip,
	}

	// Upload compressed with explicit archive name
	err = uploadFilesWithArchiveName(srcDir, "test-repo", "test-folder", archiveName, config, uploadOpts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	uploadedFiles := server.GetUploadedFiles()
	if len(uploadedFiles) == 0 {
		t.Fatal("Archive was not captured during upload")
	}

	uploadedArchiveContent := uploadedFiles[0].Content

	// Setup mock server to serve the uploaded archive for download
	downloadURL := server.URL + "/repository/test-repo/test-folder/" + archiveName
	server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
		DownloadURL: downloadURL,
		Path:        "/test-folder/" + archiveName,
		ID:          "test-id",
		Repository:  "test-repo",
		FileSize:    int64(len(uploadedArchiveContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "abc123",
		},
	})
	server.SetAssetContent("/repository/test-repo/test-folder/"+archiveName, uploadedArchiveContent)

	// Download and extract
	destDir, err := os.MkdirTemp("", "test-roundtrip-dest-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	downloadOpts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
		Compress:          true,
		CompressionFormat: archive.FormatGzip,
	}

	status := downloadFolderCompressedWithArchiveName("test-repo", "test-folder", archiveName, destDir, config, downloadOpts)
	if status != DownloadSuccess {
		t.Fatal("Download failed")
	}

	// Verify all extracted files match original content
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

// TestCompressedRoundTripZstd tests the full upload-download cycle with zstd compression
func TestCompressedRoundTripZstd(t *testing.T) {
	// Create test files
	srcDir, err := os.MkdirTemp("", "test-zstd-roundtrip-src-*")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir)

	testFiles := map[string]string{
		"file1.txt":        "Content 1",
		"file2.txt":        "Content 2",
		"subdir/file3.txt": "Content 3",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(srcDir, filename)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	archiveName := "test-repo-test-folder.tar.zst"

	// Create mock server
	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	config := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Upload compressed with zstd
	uploadOpts := &UploadOptions{
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
		Compress:          true,
		CompressionFormat: archive.FormatZstd,
	}

	// Upload compressed with explicit archive name
	err = uploadFilesWithArchiveName(srcDir, "test-repo", "test-folder", archiveName, config, uploadOpts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	uploadedFiles := server.GetUploadedFiles()
	if len(uploadedFiles) == 0 {
		t.Fatal("Archive was not captured during upload")
	}

	uploadedArchiveContent := uploadedFiles[0].Content

	// Verify it's zstd compressed (starts with zstd magic bytes)
	if len(uploadedArchiveContent) < 4 {
		t.Fatal("Archive too small")
	}
	if uploadedArchiveContent[0] != 0x28 || uploadedArchiveContent[1] != 0xB5 || uploadedArchiveContent[2] != 0x2F || uploadedArchiveContent[3] != 0xFD {
		t.Errorf("Invalid zstd magic bytes: got %x %x %x %x", uploadedArchiveContent[0], uploadedArchiveContent[1], uploadedArchiveContent[2], uploadedArchiveContent[3])
	}

	// Setup mock server to serve the uploaded archive for download
	downloadURL := server.URL + "/repository/test-repo/test-folder/" + archiveName
	server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
		DownloadURL: downloadURL,
		Path:        "/test-folder/" + archiveName,
		FileSize:    int64(len(uploadedArchiveContent)),
		Checksum: nexusapi.Checksum{
			SHA1: "dummy",
		},
	})
	server.SetAssetContent("/repository/test-repo/test-folder/"+archiveName, uploadedArchiveContent)

	// Download and extract
	destDir, err := os.MkdirTemp("", "test-zstd-roundtrip-dest-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	downloadOpts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            util.NewLogger(io.Discard),
		QuietMode:         true,
		Compress:          true,
		CompressionFormat: archive.FormatZstd,
	}

	status := downloadFolderCompressedWithArchiveName("test-repo", "test-folder", archiveName, destDir, config, downloadOpts)
	if status != DownloadSuccess {
		t.Fatal("Download failed")
	}

	// Verify all extracted files match original content
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
