package nexus

import (
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

// TestCompressedUploadPipeClosed tests that compressed upload doesn't cause "io: read/write on closed pipe" error
// This test reproduces the bug introduced in PR #72
func TestCompressedUploadPipeClosed(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-pipe-bug-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create test files with larger size to stress the pipe
	for i := 1; i <= 10; i++ {
		filename := filepath.Join(testDir, "file"+string(rune('0'+i))+".txt")
		data := make([]byte, 1024*1024) // 1MB per file = 10MB total
		if _, err := rand.Read(data); err != nil {
			t.Fatalf("Failed to generate random data: %v", err)
		}
		if err := os.WriteFile(filename, data, 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}
	}

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	config := &Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Test gzip compression with large files
	t.Run("gzip_large", func(t *testing.T) {
		opts := &UploadOptions{
			Logger:            NewLogger(io.Discard),
			QuietMode:         false, // Use false to enable progress bar (this triggers the bug)
			Compress:          true,
			CompressionFormat: CompressionGzip,
		}

		err := uploadFilesWithArchiveName(testDir, "test-repo", "", "test.tar.gz", config, opts)
		if err != nil {
			t.Fatalf("Upload failed with error: %v", err)
		}
	})

	// Test zstd compression with large files
	t.Run("zstd_large", func(t *testing.T) {
		opts := &UploadOptions{
			Logger:            NewLogger(io.Discard),
			QuietMode:         false, // Use false to enable progress bar (this triggers the bug)
			Compress:          true,
			CompressionFormat: CompressionZstd,
		}

		err := uploadFilesWithArchiveName(testDir, "test-repo", "", "test.tar.zst", config, opts)
		if err != nil {
			t.Fatalf("Upload failed with error: %v", err)
		}
	})

	// Test zip compression with large files
	t.Run("zip_large", func(t *testing.T) {
		opts := &UploadOptions{
			Logger:            NewLogger(io.Discard),
			QuietMode:         false, // Use false to enable progress bar (this triggers the bug)
			Compress:          true,
			CompressionFormat: CompressionZip,
		}

		err := uploadFilesWithArchiveName(testDir, "test-repo", "", "test.zip", config, opts)
		if err != nil {
			t.Fatalf("Upload failed with error: %v", err)
		}
	})
}
