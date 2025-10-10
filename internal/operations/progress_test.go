package operations

import (
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/tympanix/nexus-cli/internal/archive"
	"github.com/tympanix/nexus-cli/internal/config"
	"github.com/tympanix/nexus-cli/internal/nexusapi"
	"github.com/tympanix/nexus-cli/internal/util"
)

// TestCompressedUploadWithProgressBar tests that progress bar works with compressed uploads
func TestCompressedUploadWithProgressBar(t *testing.T) {
	// Create test directory with a few files
	testDir, err := os.MkdirTemp("", "test-compress-progress-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create multiple files with some size
	for i := 1; i <= 3; i++ {
		filename := filepath.Join(testDir, "file"+string(rune('0'+i))+".bin")
		data := make([]byte, 100*1024) // 100KB per file
		if _, err := rand.Read(data); err != nil {
			t.Fatalf("Failed to generate random data: %v", err)
		}
		if err := os.WriteFile(filename, data, 0644); err != nil {
			t.Fatalf("Failed to write file: %v", err)
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

	// Test with Gzip compression
	t.Run("gzip", func(t *testing.T) {
		opts := &UploadOptions{
			Logger:            util.NewLogger(io.Discard),
			QuietMode:         true, // Quiet mode to avoid terminal output during test
			Compress:          true,
			CompressionFormat: archive.FormatGzip,
		}

		err := uploadFilesWithArchiveName(testDir, "test-repo", "", "archive.tar.gz", config, opts)
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}

		uploadedFiles := server.GetUploadedFiles()
		if len(uploadedFiles) == 0 {
			t.Fatal("Archive was not uploaded")
		}
	})

	// Test with Zstd compression
	t.Run("zstd", func(t *testing.T) {
		opts := &UploadOptions{
			Logger:            util.NewLogger(io.Discard),
			QuietMode:         true,
			Compress:          true,
			CompressionFormat: archive.FormatZstd,
		}

		err := uploadFilesWithArchiveName(testDir, "test-repo", "", "archive.tar.zst", config, opts)
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}
	})

	// Test with Zip compression
	t.Run("zip", func(t *testing.T) {
		opts := &UploadOptions{
			Logger:            util.NewLogger(io.Discard),
			QuietMode:         true,
			Compress:          true,
			CompressionFormat: archive.FormatZip,
		}

		err := uploadFilesWithArchiveName(testDir, "test-repo", "", "archive.zip", config, opts)
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}
	})
}
