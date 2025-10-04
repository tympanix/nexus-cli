package nexus

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tympanix/nexus-cli/internal/testutil"
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
	receivedArchive := false
	receivedArchiveName := ""
	
	mockServer := testutil.NewMockNexusServer(t)
	defer mockServer.Close()

	mockServer.AddHandler(&testutil.UploadHandler{
		OnUpload: func(r *http.Request, t *testing.T) {
			// Parse multipart form
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Errorf("Failed to parse form: %v", err)
				return
			}

			// Check if archive was uploaded
			if file, header, err := r.FormFile("raw.asset1"); err == nil {
				receivedArchive = true
				receivedArchiveName = header.Filename
				file.Close()
			}
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
		Compress:  true,
	}

	// Upload compressed
	err = uploadFiles(testDir, "test-repo", "test-folder", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	if !receivedArchive {
		t.Error("Archive was not uploaded")
	}

	expectedName := "test-repo-test-folder.tar.gz"
	if receivedArchiveName != expectedName {
		t.Errorf("Expected archive name %q, got %q", expectedName, receivedArchiveName)
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

	if err := CreateTarGz(srcDir, archiveFile); err != nil {
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
	mockServer := testutil.NewMockNexusServer(t)
	defer mockServer.Close()

	mockServer.AddHandler(&testutil.AssetListHandler{
		Assets: []testutil.Asset{
			{
				DownloadURL: mockServer.URL() + "/repository/test-repo/test-folder/" + archiveName,
				Path:        "/test-folder/" + archiveName,
				ID:          "test-id",
				Repository:  "test-repo",
				FileSize:    int64(len(archiveContent)),
				Checksum: testutil.Checksum{
					SHA1: "abc123",
				},
			},
		},
	})

	mockServer.AddHandler(&testutil.DownloadHandler{
		PathPrefix:  "/repository/test-repo",
		Content:     archiveContent,
		ContentType: "application/gzip",
	})

	config := &Config{
		NexusURL: mockServer.URL(),
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
		Logger:            NewLogger(io.Discard),
		QuietMode:         true,
		Compress:          true,
	}

	// Download and extract
	success := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if !success {
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

	var uploadedArchiveContent []byte
	archiveName := "test-repo-test-folder.tar.gz"

	// Create mock server
	mockServer := testutil.NewMockNexusServer(t)
	defer mockServer.Close()

	// Add upload handler that captures the archive
	mockServer.AddHandler(&testutil.UploadHandler{
		OnUpload: func(r *http.Request, t *testing.T) {
			// Parse multipart form and capture the archive
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Errorf("Failed to parse form: %v", err)
				return
			}

			if file, _, err := r.FormFile("raw.asset1"); err == nil {
				uploadedArchiveContent, _ = io.ReadAll(file)
				file.Close()
			}
		},
	})

	// Add dynamic handler for asset listing that uses captured content
	mockServer.AddHandler(&testutil.DynamicHandler{
		MatchFunc: func(r *http.Request) bool {
			return strings.Contains(r.URL.Path, "/service/rest/v1/search/assets")
		},
		HandleFunc: func(w http.ResponseWriter, r *http.Request, t *testing.T) {
			assets := testutil.SearchResponse{
				Items: []testutil.Asset{
					{
						DownloadURL: mockServer.URL() + "/repository/test-repo/test-folder/" + archiveName,
						Path:        "/test-folder/" + archiveName,
						ID:          "test-id",
						Repository:  "test-repo",
						FileSize:    int64(len(uploadedArchiveContent)),
						Checksum: testutil.Checksum{
							SHA1: "abc123",
						},
					},
				},
				ContinuationToken: "",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(assets)
		},
	})

	// Add download handler that serves the uploaded archive
	mockServer.AddHandler(&testutil.DynamicHandler{
		MatchFunc: func(r *http.Request) bool {
			return strings.Contains(r.URL.Path, "/repository/test-repo")
		},
		HandleFunc: func(w http.ResponseWriter, r *http.Request, t *testing.T) {
			w.Header().Set("Content-Type", "application/gzip")
			w.WriteHeader(http.StatusOK)
			w.Write(uploadedArchiveContent)
		},
	})

	config := &Config{
		NexusURL: mockServer.URL(),
		Username: "test",
		Password: "test",
	}

	// Upload compressed
	uploadOpts := &UploadOptions{
		Logger:    NewLogger(io.Discard),
		QuietMode: true,
		Compress:  true,
	}

	err = uploadFiles(srcDir, "test-repo", "test-folder", config, uploadOpts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	if len(uploadedArchiveContent) == 0 {
		t.Fatal("Archive was not captured during upload")
	}

	// Download and extract
	destDir, err := os.MkdirTemp("", "test-roundtrip-dest-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	downloadOpts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            NewLogger(io.Discard),
		QuietMode:         true,
		Compress:          true,
	}

	success := downloadFolder("test-repo/test-folder", destDir, config, downloadOpts)
	if !success {
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
