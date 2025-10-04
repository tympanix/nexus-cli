package nexus

import (
	"bytes"
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
		repository      string
		subdir          string
		compressionType string
		expected        string
	}{
		{
			repository:      "my-repo",
			subdir:          "",
			compressionType: "gzip",
			expected:        "my-repo.tar.gz",
		},
		{
			repository:      "my-repo",
			subdir:          "folder",
			compressionType: "gzip",
			expected:        "my-repo-folder.tar.gz",
		},
		{
			repository:      "my-repo",
			subdir:          "path/to/folder",
			compressionType: "gzip",
			expected:        "my-repo-path-to-folder.tar.gz",
		},
		{
			repository:      "my repo with spaces",
			subdir:          "folder with spaces",
			compressionType: "gzip",
			expected:        "my_repo_with_spaces-folder_with_spaces.tar.gz",
		},
		{
			repository:      "my-repo",
			subdir:          "",
			compressionType: "zstd",
			expected:        "my-repo.tar.zst",
		},
		{
			repository:      "my-repo",
			subdir:          "folder",
			compressionType: "zstd",
			expected:        "my-repo-folder.tar.zst",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := GenerateArchiveName(tt.repository, tt.subdir, tt.compressionType)
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

	// Create a test file
	testFile := filepath.Join(srcDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compress to buffer
	var buf bytes.Buffer
	if err := CreateTarGz(srcDir, &buf); err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}

	// Extract to destination
	destDir, err := os.MkdirTemp("", "test-compress-progress-dest-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Extract archive
	if err := ExtractTarGz(&buf, destDir); err != nil {
		t.Fatalf("Failed to extract: %v", err)
	}

	// Verify the extracted file exists and has correct content
	extractedPath := filepath.Join(destDir, "test.txt")
	content, err := os.ReadFile(extractedPath)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("Content mismatch: expected 'test content', got %q", string(content))
	}
}


func TestCreateTarZstd(t *testing.T) {
	// Create a temporary directory with test files
	testDir, err := os.MkdirTemp("", "test-compress-zstd-*")
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

	// Create tar.zst archive
	var buf bytes.Buffer
	err = CreateTarZstd(testDir, &buf)
	if err != nil {
		t.Fatalf("Failed to create tar.zst archive: %v", err)
	}

	// Verify that we got some compressed data
	if buf.Len() == 0 {
		t.Fatal("Archive buffer is empty")
	}

	t.Logf("Created tar.zst archive of %d bytes", buf.Len())
}

func TestExtractTarZstd(t *testing.T) {
	// Create a temporary directory with test files
	srcDir, err := os.MkdirTemp("", "test-compress-zstd-src-*")
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

	// Create archive
	var buf bytes.Buffer
	err = CreateTarZstd(srcDir, &buf)
	if err != nil {
		t.Fatalf("Failed to create archive: %v", err)
	}

	// Extract to destination
	destDir, err := os.MkdirTemp("", "test-compress-zstd-dest-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	err = ExtractTarZstd(&buf, destDir)
	if err != nil {
		t.Fatalf("Failed to extract archive: %v", err)
	}

	// Verify all files exist and have correct content
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

func TestRoundTripCompressionZstd(t *testing.T) {
	// Create a temporary directory with test files
	srcDir, err := os.MkdirTemp("", "test-roundtrip-zstd-src-*")
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
	err = CreateTarZstd(srcDir, &buf)
	if err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}

	// Decompress
	destDir, err := os.MkdirTemp("", "test-roundtrip-zstd-dest-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	err = ExtractTarZstd(&buf, destDir)
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
