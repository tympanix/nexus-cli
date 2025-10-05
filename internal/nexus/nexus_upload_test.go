package nexus

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

// TestUploadSingleFile tests uploading a single file to the Nexus API
func TestUploadSingleFile(t *testing.T) {
	// Create test directory and file in a real temp directory
	testDir, err := os.MkdirTemp("", "test-upload-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	testContent := "Hello, Nexus!"

	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create mock Nexus server
	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	// Create test config
	config := &Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Create test options
	opts := &UploadOptions{
		Logger:    NewLogger(io.Discard),
		QuietMode: true,
	}

	// Test upload
	err = uploadFiles(testDir, "test-repo", "", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Validate uploaded content
	uploadedFiles := server.GetUploadedFiles()
	receivedRepository := server.LastUploadRepo

	if len(uploadedFiles) != 1 {
		t.Fatalf("Expected 1 uploaded file, got %d", len(uploadedFiles))
	}

	if string(uploadedFiles[0].Content) != testContent {
		t.Errorf("Expected uploaded content '%s', got '%s'", testContent, string(uploadedFiles[0].Content))
	}

	if uploadedFiles[0].Filename != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got '%s'", uploadedFiles[0].Filename)
	}

	if receivedRepository != "test-repo" {
		t.Errorf("Expected repository 'test-repo', got '%s'", receivedRepository)
	}
}

// TestUploadLogging tests that upload logging is simplified
func TestUploadLogging(t *testing.T) {
	testDir, err := os.MkdirTemp("", "test-upload-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	config := &Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	// Capture logger output
	var logBuf strings.Builder
	logger := NewLogger(&logBuf)

	opts := &UploadOptions{
		Logger:    logger,
		QuietMode: true,
	}

	err = uploadFiles(testDir, "test-repo", "", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Check log output contains expected message
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Uploaded 1 files from") {
		t.Errorf("Expected log message containing 'Uploaded 1 files from', got: %s", logOutput)
	}
}

// TestUploadURLConstruction tests that upload URLs are properly constructed
func TestUploadURLConstruction(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		wantRepo   string
	}{
		{
			name:       "simple repository",
			repository: "test-repo",
			wantRepo:   "test-repo",
		},
		{
			name:       "repository with special chars",
			repository: "test repo",
			wantRepo:   "test repo",
		},
		{
			name:       "repository with percent encoding",
			repository: "test%20repo",
			wantRepo:   "test%20repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir, err := os.MkdirTemp("", "test-upload-*")
			if err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}
			defer os.RemoveAll(testDir)

			testFile := filepath.Join(testDir, "test.txt")
			err = os.WriteFile(testFile, []byte("test"), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			server := nexusapi.NewMockNexusServer()
			defer server.Close()

			config := &Config{
				NexusURL: server.URL,
				Username: "test",
				Password: "test",
			}

			opts := &UploadOptions{
				Logger:    NewLogger(io.Discard),
				QuietMode: true,
			}

			err = uploadFiles(testDir, tt.repository, "", config, opts)
			if err != nil {
				t.Fatalf("Upload failed: %v", err)
			}

			if server.LastUploadRepo != tt.wantRepo {
				t.Errorf("Expected repository '%s', got '%s'", tt.wantRepo, server.LastUploadRepo)
			}
		})
	}
}
