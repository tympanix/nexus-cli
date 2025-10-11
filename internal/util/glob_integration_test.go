package util

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGlobIntegrationWithFilesystem tests the glob implementation with actual filesystem
func TestGlobIntegrationWithFilesystem(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "glob-integration-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFiles := []string{
		"main.go",
		"main_test.go",
		"README.md",
		"pkg/util/helper.go",
		"pkg/util/helper_test.go",
		"vendor/lib.go",
		"data.txt",
	}

	for _, relPath := range testFiles {
		fullPath := filepath.Join(tmpDir, filepath.FromSlash(relPath))
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", relPath, err)
		}
	}

	tests := []struct {
		name        string
		globPattern string
		wantPaths   []string
	}{
		{
			name:        "all .go files",
			globPattern: "**/*.go",
			wantPaths:   []string{"main.go", "main_test.go", "pkg/util/helper.go", "pkg/util/helper_test.go", "vendor/lib.go"},
		},
		{
			name:        ".go files excluding tests",
			globPattern: "**/*.go,!**/*_test.go",
			wantPaths:   []string{"main.go", "pkg/util/helper.go", "vendor/lib.go"},
		},
		{
			name:        ".go files excluding vendor",
			globPattern: "**/*.go,!vendor/**",
			wantPaths:   []string{"main.go", "main_test.go", "pkg/util/helper.go", "pkg/util/helper_test.go"},
		},
		{
			name:        "all files except .txt",
			globPattern: "**/*,!**/*.txt",
			wantPaths:   []string{"main.go", "main_test.go", "README.md", "pkg/util/helper.go", "pkg/util/helper_test.go", "vendor/lib.go"},
		},
		{
			name:        "root level files only",
			globPattern: "*",
			wantPaths:   []string{"main.go", "main_test.go", "README.md", "data.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var allFiles []string
			err := filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() {
					allFiles = append(allFiles, path)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("Failed to walk directory: %v", err)
			}

			filtered, err := FilterWithGlob(allFiles, tt.globPattern, func(path string) string {
				relPath, _ := filepath.Rel(tmpDir, path)
				return filepath.ToSlash(relPath)
			})

			if err != nil {
				t.Errorf("FilterWithGlob() error = %v", err)
				return
			}

			gotPaths := make([]string, len(filtered))
			for i, path := range filtered {
				relPath, _ := filepath.Rel(tmpDir, path)
				gotPaths[i] = filepath.ToSlash(relPath)
			}

			if len(gotPaths) != len(tt.wantPaths) {
				t.Errorf("FilterWithGlob() got %d files, want %d files", len(gotPaths), len(tt.wantPaths))
				t.Errorf("Got: %v", gotPaths)
				t.Errorf("Want: %v", tt.wantPaths)
				return
			}

			wantMap := make(map[string]bool)
			for _, p := range tt.wantPaths {
				wantMap[p] = true
			}

			for _, path := range gotPaths {
				if !wantMap[path] {
					t.Errorf("FilterWithGlob() unexpected file: %s", path)
				}
			}
		})
	}
}

// TestGlobIntegrationWithStructs tests the glob implementation with custom structs (simulating nexusapi.Asset)
func TestGlobIntegrationWithStructs(t *testing.T) {
	type MockAsset struct {
		Path string
		ID   string
	}

	assets := []MockAsset{
		{Path: "releases/v1.0.0/app.tar.gz", ID: "1"},
		{Path: "releases/v1.0.0/app.tar.gz.sha256", ID: "2"},
		{Path: "releases/v1.0.1/app.tar.gz", ID: "3"},
		{Path: "releases/v1.0.1/app.tar.gz.sha256", ID: "4"},
		{Path: "releases/dev/snapshot.tar.gz", ID: "5"},
		{Path: "docs/README.md", ID: "6"},
		{Path: "docs/guide.pdf", ID: "7"},
	}

	tests := []struct {
		name        string
		globPattern string
		wantIDs     []string
	}{
		{
			name:        "all tar.gz archives",
			globPattern: "**/*.tar.gz",
			wantIDs:     []string{"1", "3", "5"},
		},
		{
			name:        "all files except checksums",
			globPattern: "**/*,!**/*.sha256",
			wantIDs:     []string{"1", "3", "5", "6", "7"},
		},
		{
			name:        "only stable releases",
			globPattern: "releases/v*/**",
			wantIDs:     []string{"1", "2", "3", "4"},
		},
		{
			name:        "docs only",
			globPattern: "docs/**",
			wantIDs:     []string{"6", "7"},
		},
		{
			name:        "all releases (stable and dev)",
			globPattern: "releases/**",
			wantIDs:     []string{"1", "2", "3", "4", "5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered, err := FilterWithGlob(assets, tt.globPattern, func(asset MockAsset) string {
				return filepath.ToSlash(asset.Path)
			})

			if err != nil {
				t.Errorf("FilterWithGlob() error = %v", err)
				return
			}

			gotIDs := make([]string, len(filtered))
			for i, asset := range filtered {
				gotIDs[i] = asset.ID
			}

			if len(gotIDs) != len(tt.wantIDs) {
				t.Errorf("FilterWithGlob() got %d assets, want %d assets", len(gotIDs), len(tt.wantIDs))
				t.Errorf("Got IDs: %v", gotIDs)
				t.Errorf("Want IDs: %v", tt.wantIDs)
				return
			}

			wantMap := make(map[string]bool)
			for _, id := range tt.wantIDs {
				wantMap[id] = true
			}

			for _, id := range gotIDs {
				if !wantMap[id] {
					t.Errorf("FilterWithGlob() unexpected asset ID: %s", id)
				}
			}
		})
	}
}
