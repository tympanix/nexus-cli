package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	
	// Track upload request
	var uploadedContent string
	var uploadedFilename string
	receivedRepository := ""
	
	// Create mock Nexus server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		
		if !strings.Contains(r.URL.Path, "/service/rest/v1/components") {
			t.Errorf("Unexpected URL path: %s", r.URL.Path)
		}
		
		receivedRepository = r.URL.Query().Get("repository")
		
		// Parse multipart form
		err := r.ParseMultipartForm(32 << 20) // 32 MB
		if err != nil {
			t.Errorf("Failed to parse multipart form: %v", err)
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}
		
		// Check for uploaded file
		file, header, err := r.FormFile("raw.asset1")
		if err != nil {
			t.Errorf("Failed to get uploaded file: %v", err)
			http.Error(w, "No file uploaded", http.StatusBadRequest)
			return
		}
		defer file.Close()
		
		uploadedFilename = header.Filename
		
		// Read file content
		content, err := io.ReadAll(file)
		if err != nil {
			t.Errorf("Failed to read file content: %v", err)
			http.Error(w, "Failed to read file", http.StatusInternalServerError)
			return
		}
		uploadedContent = string(content)
		
		// Return success
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	
	// Create test config
	config := &Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}
	
	// Create test options
	opts := &UploadOptions{
		QuietMode: true,
	}
	
	// Test upload
	err = uploadFiles(testDir, "test-repo", "", config, opts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	
	// Validate uploaded content
	if uploadedContent != testContent {
		t.Errorf("Expected uploaded content '%s', got '%s'", testContent, uploadedContent)
	}
	
	if uploadedFilename != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got '%s'", uploadedFilename)
	}
	
	if receivedRepository != "test-repo" {
		t.Errorf("Expected repository 'test-repo', got '%s'", receivedRepository)
	}
}

// TestDownloadSingleFile tests downloading a directory with a single file
func TestDownloadSingleFile(t *testing.T) {
	testContent := "Downloaded content from Nexus"
	testPath := "/test-folder/downloaded.txt"
	
	var serverURL string
	
	// Create mock Nexus server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle asset listing request
		if strings.Contains(r.URL.Path, "/service/rest/v1/search/assets") {
			// Return mock asset list
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
		if strings.Contains(r.URL.Path, "/repository/test-repo") {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(testContent))
			return
		}
		
		http.NotFound(w, r)
	}))
	defer server.Close()
	serverURL = server.URL
	
	// Create test config
	config := &Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}
	
	// Create test options
	opts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		QuietMode:         true,
	}
	
	// Create temp directory for download
	destDir, err := os.MkdirTemp("", "test-download-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(destDir)
	
	// Test download
	success := downloadFolder("test-repo/test-folder", destDir, config, opts)
	if !success {
		t.Fatal("Download failed")
	}
	
	// Validate downloaded content
	downloadedFile := filepath.Join(destDir, testPath)
	content, err := os.ReadFile(downloadedFile)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	
	if string(content) != testContent {
		t.Errorf("Expected downloaded content '%s', got '%s'", testContent, string(content))
	}
}
