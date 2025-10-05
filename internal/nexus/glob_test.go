package nexus

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

func TestCollectFilesWithGlob(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-glob-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	files := []string{
		"file1.txt",
		"file2.go",
		"file3.js",
		"subdir/file4.txt",
		"subdir/file5.go",
		"subdir/nested/file6.txt",
		"subdir/nested/file7.go",
	}

	for _, file := range files {
		fullPath := filepath.Join(testDir, file)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	tests := []struct {
		name           string
		globPattern    string
		expectedCount  int
		expectedInList []string
		expectError    bool
	}{
		{
			name:           "no pattern - all files",
			globPattern:    "",
			expectedCount:  7,
			expectedInList: []string{"file1.txt", "file2.go", "file3.js", "subdir/file4.txt"},
			expectError:    false,
		},
		{
			name:           "match all .txt files",
			globPattern:    "*.txt",
			expectedCount:  1,
			expectedInList: []string{"file1.txt"},
			expectError:    false,
		},
		{
			name:           "match all .go files anywhere",
			globPattern:    "**/*.go",
			expectedCount:  3,
			expectedInList: []string{"file2.go", "subdir/file5.go", "subdir/nested/file7.go"},
			expectError:    false,
		},
		{
			name:           "match all .txt files anywhere",
			globPattern:    "**/*.txt",
			expectedCount:  3,
			expectedInList: []string{"file1.txt", "subdir/file4.txt", "subdir/nested/file6.txt"},
			expectError:    false,
		},
		{
			name:           "match files in subdir only",
			globPattern:    "subdir/*",
			expectedCount:  2,
			expectedInList: []string{"subdir/file4.txt", "subdir/file5.go"},
			expectError:    false,
		},
		{
			name:           "match all files in subdir and nested",
			globPattern:    "subdir/**",
			expectedCount:  4,
			expectedInList: []string{"subdir/file4.txt", "subdir/file5.go", "subdir/nested/file6.txt"},
			expectError:    false,
		},
		{
			name:           "match .js files",
			globPattern:    "*.js",
			expectedCount:  1,
			expectedInList: []string{"file3.js"},
			expectError:    false,
		},
		{
			name:           "no matches",
			globPattern:    "*.py",
			expectedCount:  0,
			expectedInList: []string{},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collected, err := collectFilesWithGlob(testDir, tt.globPattern)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(collected) != tt.expectedCount {
				t.Errorf("Expected %d files, got %d", tt.expectedCount, len(collected))
			}

			collectedMap := make(map[string]bool)
			for _, path := range collected {
				relPath, _ := filepath.Rel(testDir, path)
				relPath = filepath.ToSlash(relPath)
				collectedMap[relPath] = true
			}

			for _, expected := range tt.expectedInList {
				if !collectedMap[expected] {
					t.Errorf("Expected file '%s' not found in collected files", expected)
				}
			}
		})
	}
}

func TestUploadWithGlob(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-upload-glob-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	files := []string{
		"file1.txt",
		"file2.go",
		"file3.js",
		"subdir/file4.txt",
	}

	for _, file := range files {
		fullPath := filepath.Join(testDir, file)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	config := &Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	opts := &UploadOptions{
		Logger:      NewLogger(io.Discard),
		QuietMode:   true,
		GlobPattern: "**/*.txt",
	}

	err = uploadFiles(testDir, "test-repo", "", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	uploadedFiles := server.GetUploadedFiles()

	if len(uploadedFiles) != 2 {
		t.Logf("Uploaded files:")
		for i, uf := range uploadedFiles {
			t.Logf("  [%d] Filename: %s", i, uf.Filename)
		}
		t.Errorf("Expected 2 uploaded files (matching **/*.txt), got %d", len(uploadedFiles))
	}
}

func TestCompressedUploadWithGlob(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-compress-glob-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	files := []string{
		"file1.txt",
		"file2.go",
		"file3.js",
		"subdir/file4.txt",
	}

	for _, file := range files {
		fullPath := filepath.Join(testDir, file)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	config := &Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	opts := &UploadOptions{
		Logger:            NewLogger(io.Discard),
		QuietMode:         true,
		Compress:          true,
		CompressionFormat: CompressionGzip,
		GlobPattern:       "**/*.go",
	}

	err = uploadFilesCompressedWithArchiveName(testDir, "test-repo", "", "archive.tar.gz", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	uploadedArchives := server.GetUploadedArchives()

	if len(uploadedArchives) != 1 {
		t.Fatalf("Expected 1 uploaded archive, got %d", len(uploadedArchives))
	}

	if uploadedArchives[0].Filename != "archive.tar.gz" {
		t.Errorf("Expected archive name 'archive.tar.gz', got '%s'", uploadedArchives[0].Filename)
	}
}
