package operations

import (
	"bytes"
	"crypto/rand"
	"github.com/tympanix/nexus-cli/internal/archive"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestLargeFileCompressionZstd tests compression/decompression with a 10MB file
// This test reproduces the bug described in the issue where large files
// would cause "premature end" errors due to improper Close() handling
func TestLargeFileCompressionZstd(t *testing.T) {
	// Create a temporary directory with a large test file
	srcDir, err := os.MkdirTemp("", "test-large-zst-*")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir)

	// Create a 10MB file with random data (simulating /dev/urandom)
	largeFile := filepath.Join(srcDir, "large.bin")
	f, err := os.Create(largeFile)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// Write 10MB of random data
	randomData := make([]byte, 10*1024*1024)
	if _, err := rand.Read(randomData); err != nil {
		f.Close()
		t.Fatalf("Failed to generate random data: %v", err)
	}
	if _, err := f.Write(randomData); err != nil {
		f.Close()
		t.Fatalf("Failed to write random data: %v", err)
	}
	f.Close()

	// Verify file size
	info, err := os.Stat(largeFile)
	if err != nil {
		t.Fatalf("Failed to stat large file: %v", err)
	}
	if info.Size() != 10*1024*1024 {
		t.Fatalf("Large file has wrong size: expected 10485760, got %d", info.Size())
	}

	// Create tar.zst archive
	var buf bytes.Buffer
	if err := archive.CreateTarZst(srcDir, &buf); err != nil {
		t.Fatalf("Failed to create tar.zst: %v", err)
	}

	// Verify archive is not empty
	if buf.Len() == 0 {
		t.Fatal("Archive is empty")
	}

	// Verify it's zstd compressed (starts with zstd magic bytes 0x28 0xB5 0x2F 0xFD)
	data := buf.Bytes()
	if len(data) < 4 {
		t.Fatal("Archive too small to contain magic bytes")
	}
	if data[0] != 0x28 || data[1] != 0xB5 || data[2] != 0x2F || data[3] != 0xFD {
		t.Errorf("Invalid zstd magic bytes: got %x %x %x %x", data[0], data[1], data[2], data[3])
	}

	t.Logf("Compressed 10MB file to %d bytes (%.2f%% compression)", buf.Len(), float64(buf.Len())/float64(10*1024*1024)*100)

	// Extract to a new directory
	destDir, err := os.MkdirTemp("", "test-large-zst-dest-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Extract archive
	if err := archive.ExtractTarZst(&buf, destDir); err != nil {
		t.Fatalf("Failed to extract tar.zst: %v", err)
	}

	// Verify extracted file
	extractedPath := filepath.Join(destDir, "large.bin")
	extractedData, err := os.ReadFile(extractedPath)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	// Verify size matches
	if len(extractedData) != len(randomData) {
		t.Fatalf("Size mismatch: expected %d bytes, got %d bytes", len(randomData), len(extractedData))
	}

	// Verify content matches
	if !bytes.Equal(extractedData, randomData) {
		t.Fatal("Content mismatch: extracted data does not match original")
	}
}

// TestLargeFileCompressionGzip tests compression/decompression with a 10MB file using gzip
func TestLargeFileCompressionGzip(t *testing.T) {
	// Create a temporary directory with a large test file
	srcDir, err := os.MkdirTemp("", "test-large-gz-*")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir)

	// Create a 10MB file with random data
	largeFile := filepath.Join(srcDir, "large.bin")
	f, err := os.Create(largeFile)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// Write 10MB of random data
	randomData := make([]byte, 10*1024*1024)
	if _, err := rand.Read(randomData); err != nil {
		f.Close()
		t.Fatalf("Failed to generate random data: %v", err)
	}
	if _, err := f.Write(randomData); err != nil {
		f.Close()
		t.Fatalf("Failed to write random data: %v", err)
	}
	f.Close()

	// Create tar.gz archive
	var buf bytes.Buffer
	if err := archive.CreateTarGz(srcDir, &buf); err != nil {
		t.Fatalf("Failed to create tar.gz: %v", err)
	}

	// Verify archive is not empty
	if buf.Len() == 0 {
		t.Fatal("Archive is empty")
	}

	t.Logf("Compressed 10MB file to %d bytes (%.2f%% compression)", buf.Len(), float64(buf.Len())/float64(10*1024*1024)*100)

	// Extract to a new directory
	destDir, err := os.MkdirTemp("", "test-large-gz-dest-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Extract archive
	if err := archive.ExtractTarGz(&buf, destDir); err != nil {
		t.Fatalf("Failed to extract tar.gz: %v", err)
	}

	// Verify extracted file
	extractedPath := filepath.Join(destDir, "large.bin")
	extractedData, err := os.ReadFile(extractedPath)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	// Verify size matches
	if len(extractedData) != len(randomData) {
		t.Fatalf("Size mismatch: expected %d bytes, got %d bytes", len(randomData), len(extractedData))
	}

	// Verify content matches
	if !bytes.Equal(extractedData, randomData) {
		t.Fatal("Content mismatch: extracted data does not match original")
	}
}

// TestStreamingCompressionZstd tests that compression works correctly when writing to a pipe
// This simulates the real-world scenario where archives are uploaded while being created
func TestStreamingCompressionZstd(t *testing.T) {
	// Create a temporary directory with test files
	srcDir, err := os.MkdirTemp("", "test-stream-zst-*")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir)

	// Create a 5MB file
	largeFile := filepath.Join(srcDir, "file.bin")
	randomData := make([]byte, 5*1024*1024)
	if _, err := rand.Read(randomData); err != nil {
		t.Fatalf("Failed to generate random data: %v", err)
	}
	if err := os.WriteFile(largeFile, randomData, 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Simulate streaming compression via pipe (like in upload)
	pr, pw := io.Pipe()
	errChan := make(chan error, 1)

	go func() {
		defer pw.Close()
		err := archive.CreateTarZst(srcDir, pw)
		errChan <- err
	}()

	// Read the compressed data
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, pr); err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}

	// Check for compression error
	if err := <-errChan; err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	// Verify archive can be extracted
	destDir, err := os.MkdirTemp("", "test-stream-zst-dest-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	if err := archive.ExtractTarZst(&buf, destDir); err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	// Verify extracted file
	extractedData, err := os.ReadFile(filepath.Join(destDir, "file.bin"))
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	if !bytes.Equal(extractedData, randomData) {
		t.Fatal("Content mismatch after streaming compression")
	}
}
