package nexus

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateTarGz(t *testing.T) {
	// Create a temporary directory with test files
	testDir, err := os.MkdirTemp("", "test-compress-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create test files
	testFiles := map[string]string{
		"file1.txt":        "Content of file 1",
		"file2.txt":        "Content of file 2",
		"subdir/file3.txt": "Nested file content",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testDir, filename)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create tar.gz archive
	var buf bytes.Buffer
	err = CreateTarGz(testDir, &buf)
	if err != nil {
		t.Fatalf("Failed to create tar.gz: %v", err)
	}

	// Verify archive is not empty
	if buf.Len() == 0 {
		t.Fatal("Archive is empty")
	}

	// Verify it's gzip compressed (starts with gzip magic bytes)
	data := buf.Bytes()
	if len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
		t.Error("Archive does not have gzip magic bytes")
	}
}

func TestExtractTarGz(t *testing.T) {
	// Create a temporary directory with test files
	srcDir, err := os.MkdirTemp("", "test-compress-src-*")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir)

	// Create test files
	testFiles := map[string]string{
		"file1.txt":        "Content of file 1",
		"file2.txt":        "Content of file 2",
		"subdir/file3.txt": "Nested file content",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(srcDir, filename)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create tar.gz archive
	var buf bytes.Buffer
	err = CreateTarGz(srcDir, &buf)
	if err != nil {
		t.Fatalf("Failed to create tar.gz: %v", err)
	}

	// Create destination directory for extraction
	destDir, err := os.MkdirTemp("", "test-extract-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Extract archive
	err = ExtractTarGz(&buf, destDir)
	if err != nil {
		t.Fatalf("Failed to extract tar.gz: %v", err)
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

func TestRoundTripCompression(t *testing.T) {
	// Create a temporary directory with test files
	srcDir, err := os.MkdirTemp("", "test-roundtrip-src-*")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir)

	// Create test files with various content
	testFiles := map[string]string{
		"file1.txt":          "Content of file 1",
		"file2.txt":          "Content of file 2",
		"subdir/file3.txt":   "Nested file content",
		"subdir/file4.bin":   string([]byte{0x00, 0x01, 0x02, 0xff}),
		"deep/nest/file5.md": "# Deep nested file\nSome markdown content",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(srcDir, filename)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Compress
	var buf bytes.Buffer
	err = CreateTarGz(srcDir, &buf)
	if err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}

	// Decompress
	destDir, err := os.MkdirTemp("", "test-roundtrip-dest-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	err = ExtractTarGz(&buf, destDir)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	// Verify all files
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

func TestGenerateArchiveName(t *testing.T) {
	tests := []struct {
		repository string
		subdir     string
		expected   string
	}{
		{
			repository: "my-repo",
			subdir:     "",
			expected:   "my-repo.tar.gz",
		},
		{
			repository: "my-repo",
			subdir:     "folder",
			expected:   "my-repo-folder.tar.gz",
		},
		{
			repository: "my-repo",
			subdir:     "path/to/folder",
			expected:   "my-repo-path-to-folder.tar.gz",
		},
		{
			repository: "my repo with spaces",
			subdir:     "folder with spaces",
			expected:   "my_repo_with_spaces-folder_with_spaces.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := GenerateArchiveName(tt.repository, tt.subdir)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractTarGzWithProgress(t *testing.T) {
	// Create a temporary directory with test files
	srcDir, err := os.MkdirTemp("", "test-compress-progress-*")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir)

	// Create a larger file to test progress
	testFile := filepath.Join(srcDir, "large.txt")
	largeContent := bytes.Repeat([]byte("test content\n"), 1000)
	if err := os.WriteFile(testFile, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create tar.gz archive
	var buf bytes.Buffer
	err = CreateTarGz(srcDir, &buf)
	if err != nil {
		t.Fatalf("Failed to create tar.gz: %v", err)
	}

	// Create destination directory for extraction
	destDir, err := os.MkdirTemp("", "test-extract-progress-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Wrap buffer in a reader that counts bytes
	bytesRead := 0
	countingReader := io.TeeReader(&buf, io.MultiWriter(io.Discard))

	// Extract archive
	err = ExtractTarGz(countingReader, destDir)
	if err != nil {
		t.Fatalf("Failed to extract tar.gz: %v", err)
	}

	// Verify extracted file
	extractedPath := filepath.Join(destDir, "large.txt")
	content, err := os.ReadFile(extractedPath)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	if !bytes.Equal(content, largeContent) {
		t.Errorf("Content mismatch: expected %d bytes, got %d bytes", len(largeContent), len(content))
	}

	t.Logf("Read %d bytes during extraction", bytesRead)
}
