package archive

import (
	"archive/tar"
	"archive/zip"
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

func TestCreateTarZst(t *testing.T) {
	// Create a temporary directory with test files
	testDir, err := os.MkdirTemp("", "test-compress-zst-*")
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
	err = CreateTarZst(testDir, &buf)
	if err != nil {
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
}

func TestExtractTarZst(t *testing.T) {
	// Create a temporary directory with test files
	srcDir, err := os.MkdirTemp("", "test-compress-zst-src-*")
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

	// Create tar.zst archive
	var buf bytes.Buffer
	err = CreateTarZst(srcDir, &buf)
	if err != nil {
		t.Fatalf("Failed to create tar.zst: %v", err)
	}

	// Extract to a new directory
	destDir, err := os.MkdirTemp("", "test-extract-zst-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Extract archive
	err = ExtractTarZst(&buf, destDir)
	if err != nil {
		t.Fatalf("Failed to extract tar.zst: %v", err)
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

func TestRoundTripCompressionZst(t *testing.T) {
	// Create a temporary directory with test files
	srcDir, err := os.MkdirTemp("", "test-roundtrip-zst-src-*")
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

	// Create tar.zst archive
	var buf bytes.Buffer
	if err := CreateTarZst(srcDir, &buf); err != nil {
		t.Fatalf("Failed to create tar.zst: %v", err)
	}

	// Extract to a new directory
	destDir, err := os.MkdirTemp("", "test-roundtrip-zst-dest-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	if err := ExtractTarZst(&buf, destDir); err != nil {
		t.Fatalf("Failed to extract tar.zst: %v", err)
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

func TestCreateZip(t *testing.T) {
	testFiles := map[string]string{
		"file1.txt":           "content1",
		"dir1/file2.txt":      "content2",
		"dir1/dir2/file3.txt": "content3",
	}

	srcDir, err := os.MkdirTemp("", "test-src-*")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir)

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

	var buf bytes.Buffer
	err = CreateZip(srcDir, &buf)
	if err != nil {
		t.Fatalf("Failed to create zip: %v", err)
	}

	if buf.Len() == 0 {
		t.Error("Created zip archive is empty")
	}
}

func TestExtractZip(t *testing.T) {
	testFiles := map[string]string{
		"file1.txt":           "content1",
		"dir1/file2.txt":      "content2",
		"dir1/dir2/file3.txt": "content3",
	}

	srcDir, err := os.MkdirTemp("", "test-src-*")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir)

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

	var buf bytes.Buffer
	err = CreateZip(srcDir, &buf)
	if err != nil {
		t.Fatalf("Failed to create zip: %v", err)
	}

	destDir, err := os.MkdirTemp("", "test-extract-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	err = ExtractZip(&buf, destDir)
	if err != nil {
		t.Fatalf("Failed to extract zip: %v", err)
	}

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

func TestRoundTripCompressionZip(t *testing.T) {
	testFiles := map[string]string{
		"file1.txt":           "content1",
		"dir1/file2.txt":      "content2",
		"dir1/dir2/file3.txt": "content3",
		"special chars.txt":   "special content!@#$%",
	}

	srcDir, err := os.MkdirTemp("", "test-roundtrip-zip-src-*")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(srcDir)

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

	var buf bytes.Buffer
	if err := CreateZip(srcDir, &buf); err != nil {
		t.Fatalf("Failed to create zip: %v", err)
	}

	destDir, err := os.MkdirTemp("", "test-roundtrip-zip-dest-*")
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	if err := ExtractZip(&buf, destDir); err != nil {
		t.Fatalf("Failed to extract zip: %v", err)
	}

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

func TestCreateTarArchiveHelper(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-helper-*")
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

	var buf bytes.Buffer
	if err := createTarArchive(testDir, &buf, ""); err != nil {
		t.Fatalf("createTarArchive failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("Archive is empty")
	}
}

func TestAddFileToTarHelper(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-add-tar-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	content := "test content"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var buf bytes.Buffer
	tw := newTestTarWriter(&buf)

	if err := addFileToTar(tw, testDir, testFile); err != nil {
		t.Fatalf("addFileToTar failed: %v", err)
	}

	tw.Close()

	if buf.Len() == 0 {
		t.Fatal("No data written to tar")
	}
}

func TestAddFileToZipHelper(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-add-zip-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	content := "test content"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var buf bytes.Buffer
	zw := newTestZipWriter(&buf)

	if err := addFileToZip(zw, testDir, testFile); err != nil {
		t.Fatalf("addFileToZip failed: %v", err)
	}

	zw.Close()

	if buf.Len() == 0 {
		t.Fatal("No data written to zip")
	}
}

func newTestTarWriter(w io.Writer) *tar.Writer {
	return tar.NewWriter(w)
}

func newTestZipWriter(w io.Writer) *zip.Writer {
	return zip.NewWriter(w)
}
