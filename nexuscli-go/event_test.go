package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestEventChannelUpload tests the unified event channel for upload operations
func TestEventChannelUpload(t *testing.T) {
	// Create test directory and file
	testDir, err := os.MkdirTemp("", "test-event-upload-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create mock Nexus server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
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

	// Test successful upload - should send EventSuccess
	err = uploadFiles(testDir, "test-repo", "", config, opts)
	if err != nil {
		t.Errorf("Upload should succeed but got error: %v", err)
	}
}

// TestEventChannelDownload tests the unified event channel for download operations
func TestEventChannelDownload(t *testing.T) {
	testContent := "test download content"
	testPath := "/test-folder/test.txt"

	var serverURL string

	// Create mock Nexus server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle asset listing request
		if r.URL.Path == "/service/rest/v1/search/assets" {
			assets := searchResponse{
				Items: []Asset{
					{
						DownloadURL: serverURL + "/repository/test-repo" + testPath,
						Path:        testPath,
						ID:          "test-id",
						Repository:  "test-repo",
						FileSize:    int64(len(testContent)),
						Checksum: Checksum{
							SHA1: "abc123",
						},
					},
				},
				ContinuationToken: "",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(assets)
			return
		}

		// Handle file download request
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(testContent))
	}))
	defer server.Close()
	serverURL = server.URL

	config := &Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            NewLogger(io.Discard),
		QuietMode:         true,
	}

	destDir, err := os.MkdirTemp("", "test-event-download-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// First download - should succeed
	success := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if !success {
		t.Fatal("First download should succeed")
	}

	// Second download - should skip (EventSkip sent)
	success = downloadFolder("test-repo/test-folder", destDir, config, opts)
	if !success {
		t.Fatal("Second download should succeed (files skipped)")
	}
}

// TestEventChannelDownloadError tests error handling in the event channel
func TestEventChannelDownloadError(t *testing.T) {
	testPath := "/test-folder/test.txt"

	var serverURL string

	// Create mock Nexus server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle asset listing request
		if r.URL.Path == "/service/rest/v1/search/assets" {
			assets := searchResponse{
				Items: []Asset{
					{
						DownloadURL: serverURL + "/repository/test-repo" + testPath,
						Path:        testPath,
						ID:          "test-id",
						Repository:  "test-repo",
						FileSize:    100,
						Checksum: Checksum{
							SHA1: "abc123",
						},
					},
				},
				ContinuationToken: "",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(assets)
			return
		}

		// Return error for download
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	serverURL = server.URL

	config := &Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            NewLogger(io.Discard),
		QuietMode:         true,
	}

	destDir, err := os.MkdirTemp("", "test-event-error-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)

	// Download should fail with error event
	success := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if success {
		t.Fatal("Download should fail when server returns error")
	}
}
