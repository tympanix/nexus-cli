package nexusapi

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"mime/multipart"

	"github.com/tympanix/nexus-cli/internal/testutil"
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
	// Create mock server with asset list handler
	mockServer := testutil.NewMockNexusServer(t)
	defer mockServer.Close()

	mockServer.AddHandler(&testutil.AssetListHandler{
		Assets: []testutil.Asset{
			{
				ID:       "asset1",
				Path:     "/test-path/file1.txt",
				FileSize: 100,
			},
			{
				ID:       "asset2",
				Path:     "/test-path/file2.txt",
				FileSize: 200,
			},
		},
		ValidateAuth:  true,
		ExpectedRepo:  "test-repo",
		ExpectedQuery: "/test-path/*",
	})

	client := NewClient(mockServer.URL(), "testuser", "testpass")
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
	mockServer := testutil.NewMockNexusServer(t)
	defer mockServer.Close()

	mockServer.AddHandler(&testutil.PaginatedAssetListHandler{
		Pages: []testutil.SearchResponse{
			{
				Items: []testutil.Asset{
					{ID: "asset1", Path: "/path/file1.txt"},
				},
				ContinuationToken: "token123",
			},
			{
				Items: []testutil.Asset{
					{ID: "asset2", Path: "/path/file2.txt"},
				},
				ContinuationToken: "",
			},
		},
		ValidateAuth: true,
	})

	client := NewClient(mockServer.URL(), "user", "pass")
	assets, err := client.ListAssets("repo", "path")

	if err != nil {
		t.Fatalf("ListAssets failed: %v", err)
	}

	if len(assets) != 2 {
		t.Errorf("Expected 2 assets, got %d", len(assets))
	}
}

// TestUploadComponent tests uploading a component
func TestUploadComponent(t *testing.T) {
	receivedBody := ""
	receivedContentType := ""

	mockServer := testutil.NewMockNexusServer(t)
	defer mockServer.Close()

	mockServer.AddHandler(&testutil.UploadHandler{
		ValidateAuth: true,
		ExpectedRepo: "test-repo",
		OnUpload: func(r *http.Request, t *testing.T) {
			receivedContentType = r.Header.Get("Content-Type")
			body, _ := io.ReadAll(r.Body)
			receivedBody = string(body)
		},
	})

	client := NewClient(mockServer.URL(), "testuser", "testpass")
	body := strings.NewReader("test content")
	err := client.UploadComponent("test-repo", body, "multipart/form-data")

	if err != nil {
		t.Fatalf("UploadComponent failed: %v", err)
	}

	if receivedBody != "test content" {
		t.Errorf("Expected body 'test content', got '%s'", receivedBody)
	}

	if receivedContentType != "multipart/form-data" {
		t.Errorf("Expected content type 'multipart/form-data', got '%s'", receivedContentType)
	}
}

// TestUploadComponentError tests upload error handling
func TestUploadComponentError(t *testing.T) {
	mockServer := testutil.NewMockNexusServer(t)
	defer mockServer.Close()

	mockServer.AddHandler(&testutil.UploadHandler{
		StatusCode: http.StatusBadRequest,
	})

	client := NewClient(mockServer.URL(), "user", "pass")
	body := strings.NewReader("test")
	err := client.UploadComponent("repo", body, "text/plain")

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "400") {
		t.Errorf("Expected error to contain status code 400, got: %v", err)
	}
}

// TestDownloadAsset tests downloading an asset
func TestDownloadAsset(t *testing.T) {
	testContent := "downloaded content"

	mockServer := testutil.NewMockNexusServer(t)
	defer mockServer.Close()

	mockServer.AddHandler(&testutil.DownloadHandler{
		PathPrefix:   "/test-asset",
		Content:      []byte(testContent),
		ValidateAuth: true,
	})

	client := NewClient(mockServer.URL(), "testuser", "testpass")

	var buf strings.Builder
	err := client.DownloadAsset(mockServer.URL()+"/test-asset", &buf)

	if err != nil {
		t.Fatalf("DownloadAsset failed: %v", err)
	}

	if buf.String() != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, buf.String())
	}
}

// TestDownloadAssetError tests download error handling
func TestDownloadAssetError(t *testing.T) {
	mockServer := testutil.NewMockNexusServer(t)
	defer mockServer.Close()

	mockServer.AddHandler(&testutil.DownloadHandler{
		PathPrefix: "/missing",
		StatusCode: http.StatusNotFound,
	})

	client := NewClient(mockServer.URL(), "user", "pass")

	var buf strings.Builder
	err := client.DownloadAsset(mockServer.URL()+"/missing", &buf)

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
	
	err = BuildRawUploadForm(writer, files, "test-subdir", nil)
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

