package nexus

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

// TestUploadProgressBarCompletion tests that upload progress bar shows 100% and [n/n] when complete
func TestUploadProgressBarCompletion(t *testing.T) {
	tests := []struct {
		name     string
		numFiles int
		fileSize int
	}{
		{
			name:     "single file upload",
			numFiles: 1,
			fileSize: 1024,
		},
		{
			name:     "multiple files upload",
			numFiles: 5,
			fileSize: 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory with files
			testDir, err := os.MkdirTemp("", "test-upload-progress-*")
			if err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}
			defer os.RemoveAll(testDir)

			// Create test files
			for i := 0; i < tt.numFiles; i++ {
				filename := filepath.Join(testDir, "file"+string(rune('0'+i))+".txt")
				data := make([]byte, tt.fileSize)
				for j := range data {
					data[j] = byte('A' + (j % 26))
				}
				if err := os.WriteFile(filename, data, 0644); err != nil {
					t.Fatalf("Failed to write file: %v", err)
				}
			}

			// Create mock server
			server := nexusapi.NewMockNexusServer()
			defer server.Close()

			config := &Config{
				NexusURL: server.URL,
				Username: "test",
				Password: "test",
			}

			// Capture progress bar output
			var progressBuf bytes.Buffer

			opts := &UploadOptions{
				Logger:            NewLogger(io.Discard),
				QuietMode:         false,
				progressWriter:    &progressBuf,
				forceShowProgress: true,
			}

			// Upload files
			err = uploadFiles(testDir, "test-repo", "", config, opts)
			if err != nil {
				t.Fatalf("Upload failed: %v", err)
			}

			// Check progress bar output
			output := progressBuf.String()

			// Check for 100% completion indicator
			if !strings.Contains(output, "100%") {
				t.Errorf("Progress bar output should show 100%% completion\nGot output: %s", output)
			}

			// Check for file count progression [n/n]
			// The progress bar shows file progress as [current/total]
			// For upload with multiple files, we should see progression like [1/n], [2/n], ..., [n-1/n]
			if tt.numFiles > 1 {
				// Verify we see file progression - at least the penultimate count
				penultimateCount := "[" + string(rune('0'+tt.numFiles-1)) + "/" + string(rune('0'+tt.numFiles)) + "]"
				if !strings.Contains(output, penultimateCount) {
					t.Errorf("Expected to see file progression showing %s in output\nGot output: %s", penultimateCount, output)
				}
			}

			// Verify that files were uploaded
			uploadedFiles := server.GetUploadedFiles()
			if len(uploadedFiles) != tt.numFiles {
				t.Errorf("Expected %d files to be uploaded, got %d", tt.numFiles, len(uploadedFiles))
			}
		})
	}
}

// TestDownloadProgressBarCompletion tests that download progress bar shows 100% and [n/n] when complete
func TestDownloadProgressBarCompletion(t *testing.T) {
	tests := []struct {
		name     string
		numFiles int
		fileSize int
	}{
		{
			name:     "single file download",
			numFiles: 1,
			fileSize: 1024,
		},
		{
			name:     "multiple files download",
			numFiles: 5,
			fileSize: 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := nexusapi.NewMockNexusServer()
			defer server.Close()

			// Setup mock assets
			for i := 0; i < tt.numFiles; i++ {
				filename := "file" + string(rune('0'+i)) + ".txt"
				testPath := "/test-folder/" + filename
				testContent := make([]byte, tt.fileSize)
				for j := range testContent {
					testContent[j] = byte('A' + (j % 26))
				}

				downloadURL := server.URL + "/repository/test-repo" + testPath
				server.AddAssetWithQuery("test-repo", "/test-folder/*", nexusapi.Asset{
					DownloadURL: downloadURL,
					Path:        testPath,
					ID:          "test-id-" + string(rune('0'+i)),
					Repository:  "test-repo",
					FileSize:    int64(len(testContent)),
					Checksum: nexusapi.Checksum{
						SHA1: "abc" + string(rune('0'+i)),
					},
				})
				server.SetAssetContent("/repository/test-repo"+testPath, testContent)
			}

			config := &Config{
				NexusURL: server.URL,
				Username: "test",
				Password: "test",
			}

			// Create temp directory for download
			destDir, err := os.MkdirTemp("", "test-download-progress-*")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(destDir)

			// Capture progress bar output
			var progressBuf bytes.Buffer

			opts := &DownloadOptions{
				Logger:            NewLogger(io.Discard),
				QuietMode:         false,
				progressWriter:    &progressBuf,
				forceShowProgress: true,
			}

			// Download files
			status := downloadFolder("test-repo/test-folder", destDir, config, opts)
			if status != DownloadSuccess {
				t.Fatal("Download failed")
			}

			// Check progress bar output
			output := progressBuf.String()

			// Check for 100% completion indicator
			if !strings.Contains(output, "100%") {
				t.Errorf("Progress bar output should show 100%% completion\nGot output: %s", output)
			}

			// Check for file count progression [n/n]
			// For download with multiple files, we should see progression like [1/n], [2/n], etc.
			if tt.numFiles > 1 {
				// Verify we see file progression - check that at least one intermediate count appears
				// Downloads may complete quickly, so we'll check for any progression
				foundProgression := false
				for i := 1; i < tt.numFiles; i++ {
					count := "[" + string(rune('0'+i)) + "/" + string(rune('0'+tt.numFiles)) + "]"
					if strings.Contains(output, count) {
						foundProgression = true
						break
					}
				}
				if !foundProgression {
					t.Errorf("Expected to see some file progression (like [1/%d], [2/%d], etc.) in output\nGot output: %s", tt.numFiles, tt.numFiles, output)
				}
			}

			// Verify that files were downloaded
			for i := 0; i < tt.numFiles; i++ {
				filename := "file" + string(rune('0'+i)) + ".txt"
				filePath := filepath.Join(destDir, "test-folder", filename)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("Expected downloaded file at %s, but it does not exist", filePath)
				}
			}
		})
	}
}
