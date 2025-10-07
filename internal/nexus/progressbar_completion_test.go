package nexus

import (
	"bytes"
	"fmt"
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

			// Check progress bar output - only validate the final (finished) state
			output := progressBuf.String()

			// Get the last non-empty line (the final state after Finish())
			lines := strings.Split(strings.TrimSpace(output), "\n")
			lastLine := ""
			for i := len(lines) - 1; i >= 0; i-- {
				if strings.TrimSpace(lines[i]) != "" {
					lastLine = lines[i]
					break
				}
			}

			// Check for 100% completion in the final state
			if !strings.Contains(lastLine, "100%") {
				t.Errorf("Final progress bar state should show 100%% completion\nLast line: %s", lastLine)
			}

			// Check for final file count [n/n] in the final state
			expectedCount := fmt.Sprintf("[%d/%d]", tt.numFiles, tt.numFiles)
			if !strings.Contains(lastLine, expectedCount) {
				t.Errorf("Final progress bar state should show %s\nLast line: %s", expectedCount, lastLine)
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

			// Check progress bar output - only validate the final (finished) state
			output := progressBuf.String()

			// Get the last non-empty line (the final state after Finish())
			lines := strings.Split(strings.TrimSpace(output), "\n")
			lastLine := ""
			for i := len(lines) - 1; i >= 0; i-- {
				if strings.TrimSpace(lines[i]) != "" {
					lastLine = lines[i]
					break
				}
			}

			// Check for 100% completion in the final state
			if !strings.Contains(lastLine, "100%") {
				t.Errorf("Final progress bar state should show 100%% completion\nLast line: %s", lastLine)
			}

			// Check for final file count [n/n] in the final state
			expectedCount := fmt.Sprintf("[%d/%d]", tt.numFiles, tt.numFiles)
			if !strings.Contains(lastLine, expectedCount) {
				t.Errorf("Final progress bar state should show %s\nLast line: %s", expectedCount, lastLine)
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
