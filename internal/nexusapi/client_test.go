package nexusapi

import (
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestNewClient tests creating a new Nexus API client
func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:8081", "admin", "secret")

	if client.BaseURL != "http://localhost:8081" {
		t.Errorf("Expected BaseURL 'http://localhost:8081', got '%s'", client.BaseURL)
	}
	if client.Username != "admin" {
		t.Errorf("Expected Username 'admin', got '%s'", client.Username)
	}
	if client.Password != "secret" {
		t.Errorf("Expected Password 'secret', got '%s'", client.Password)
	}
	if client.HTTPClient == nil {
		t.Error("Expected HTTPClient to be initialized")
	}
}

// TestListAssets tests listing assets from Nexus
func TestListAssets(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	// Setup mock data
	server.AddAssetWithQuery("test-repo", "/test-path/*", Asset{
		ID:       "asset1",
		Path:     "/test-path/file1.txt",
		FileSize: 100,
	})
	server.AddAssetWithQuery("test-repo", "/test-path/*", Asset{
		ID:       "asset2",
		Path:     "/test-path/file2.txt",
		FileSize: 200,
	})

	client := NewClient(server.URL, "testuser", "testpass")
	assets, err := client.ListAssets("test-repo", "test-path")

	if err != nil {
		t.Fatalf("ListAssets failed: %v", err)
	}

	if len(assets) != 2 {
		t.Errorf("Expected 2 assets, got %d", len(assets))
	}

	if assets[0].ID != "asset1" {
		t.Errorf("Expected asset ID 'asset1', got '%s'", assets[0].ID)
	}
}

// TestListAssetsWithPagination tests listing assets with continuation tokens
func TestListAssetsWithPagination(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	// Setup first page
	server.AddAssetForPage("repo", "/path/*", Asset{ID: "asset1", Path: "/path/file1.txt"}, 1)
	server.SetContinuationToken("repo", "/path/*", "token123")

	// Setup second page
	server.AddAssetForPage("repo", "/path/*", Asset{ID: "asset2", Path: "/path/file2.txt"}, 2)

	client := NewClient(server.URL, "user", "pass")
	assets, err := client.ListAssets("repo", "path")

	if err != nil {
		t.Fatalf("ListAssets failed: %v", err)
	}

	if len(assets) != 2 {
		t.Errorf("Expected 2 assets, got %d", len(assets))
	}

	if server.GetRequestCount() < 2 {
		t.Errorf("Expected at least 2 API calls, got %d", server.GetRequestCount())
	}
}

// TestUploadComponent tests uploading a component
func TestUploadComponent(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	client := NewClient(server.URL, "testuser", "testpass")
	body := strings.NewReader("test content")
	err := client.UploadComponent("test-repo", body, "multipart/form-data")

	if err != nil {
		t.Fatalf("UploadComponent failed: %v", err)
	}

	if server.LastUploadRepo != "test-repo" {
		t.Errorf("Expected repository 'test-repo', got '%s'", server.LastUploadRepo)
	}
}

// TestUploadComponentError tests upload error handling
func TestUploadComponentError(t *testing.T) {
	// For error testing, we use a raw httptest server and close it immediately
	// to simulate connection errors. This is a valid use case for httptest.NewServer
	// rather than the mock server, as we need to test connection failure behavior.
	server := httptest.NewServer(nil)
	server.Config.Handler = nil
	// Close the server immediately to simulate connection error
	server.Close()

	client := NewClient(server.URL, "user", "pass")
	body := strings.NewReader("test")
	err := client.UploadComponent("repo", body, "text/plain")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
}

// TestDownloadAsset tests downloading an asset
func TestDownloadAsset(t *testing.T) {
	testContent := "downloaded content"

	server := NewMockNexusServer()
	defer server.Close()

	downloadURL := server.URL + "/repository/test-repo/test-asset"
	server.SetAssetContent(downloadURL, []byte(testContent))

	client := NewClient(server.URL, "testuser", "testpass")

	var buf strings.Builder
	err := client.DownloadAsset(downloadURL, &buf)

	if err != nil {
		t.Fatalf("DownloadAsset failed: %v", err)
	}

	if buf.String() != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, buf.String())
	}
}

// TestDownloadAssetError tests download error handling
func TestDownloadAssetError(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	client := NewClient(server.URL, "user", "pass")

	var buf strings.Builder
	err := client.DownloadAsset(server.URL+"/repository/missing", &buf)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("Expected error to contain status code 404, got: %v", err)
	}
}

// TestBuildRawUploadForm tests building multipart form for RAW repository upload
func TestBuildRawUploadForm(t *testing.T) {
	// Create test files
	tempDir := t.TempDir()
	file1Path := tempDir + "/file1.txt"
	file2Path := tempDir + "/subdir/file2.txt"

	// Create subdirectory
	err := os.MkdirAll(tempDir+"/subdir", 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create test files
	err = os.WriteFile(file1Path, []byte("content1"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	err = os.WriteFile(file2Path, []byte("content2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Prepare file uploads
	files := []FileUpload{
		{FilePath: file1Path, RelativePath: "file1.txt"},
		{FilePath: file2Path, RelativePath: "subdir/file2.txt"},
	}

	// Build form
	var buf strings.Builder
	writer := multipart.NewWriter(&buf)

	err = BuildRawUploadForm(writer, files, "test-subdir", nil, nil, nil)
	if err != nil {
		t.Fatalf("BuildRawUploadForm failed: %v", err)
	}
	writer.Close()

	// Parse the form
	formData := buf.String()

	// Verify form contains expected fields
	if !strings.Contains(formData, "raw.asset1") {
		t.Error("Expected form to contain 'raw.asset1'")
	}
	if !strings.Contains(formData, "raw.asset2") {
		t.Error("Expected form to contain 'raw.asset2'")
	}
	if !strings.Contains(formData, "raw.asset1.filename") {
		t.Error("Expected form to contain 'raw.asset1.filename'")
	}
	if !strings.Contains(formData, "raw.asset2.filename") {
		t.Error("Expected form to contain 'raw.asset2.filename'")
	}
	if !strings.Contains(formData, "file1.txt") {
		t.Error("Expected form to contain 'file1.txt'")
	}
	if !strings.Contains(formData, "subdir/file2.txt") {
		t.Error("Expected form to contain 'subdir/file2.txt'")
	}
	if !strings.Contains(formData, "raw.directory") {
		t.Error("Expected form to contain 'raw.directory'")
	}
	if !strings.Contains(formData, "test-subdir") {
		t.Error("Expected form to contain 'test-subdir'")
	}
	if !strings.Contains(formData, "content1") {
		t.Error("Expected form to contain 'content1'")
	}
	if !strings.Contains(formData, "content2") {
		t.Error("Expected form to contain 'content2'")
	}
}
