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
	server.AddAsset("test-repo", "/test-path/file1.txt", Asset{
		ID:       "asset1",
		Path:     "/test-path/file1.txt",
		FileSize: 100,
	}, nil)
	server.AddAsset("test-repo", "/test-path/file2.txt", Asset{
		ID:       "asset2",
		Path:     "/test-path/file2.txt",
		FileSize: 200,
	}, nil)

	client := NewClient(server.URL, "testuser", "testpass")
	assets, err := client.ListAssets("test-repo", "test-path", true)

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
	assets, err := client.ListAssets("repo", "path", true)

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

// TestUploadComponentRepositoryNotFound tests uploading to a non-existent repository
func TestUploadComponentRepositoryNotFound(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	// Mark the repository as not found
	server.SetRepositoryNotFound("non-existent-repo")

	client := NewClient(server.URL, "testuser", "testpass")
	body := strings.NewReader("test content")
	err := client.UploadComponent("non-existent-repo", body, "multipart/form-data")

	if err == nil {
		t.Fatal("Expected error for non-existent repository, got nil")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("Expected error to contain status code 404, got: %v", err)
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected error message to mention 'not found', got: %v", err)
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

// TestBuildAptUploadForm tests building multipart form for APT (Debian) package upload
func TestBuildAptUploadForm(t *testing.T) {
	// Create a test .deb file
	tempDir := t.TempDir()
	debFilePath := tempDir + "/test-package_1.0.0_amd64.deb"

	// Create test .deb file with some content
	debContent := []byte("fake deb file content")
	err := os.WriteFile(debFilePath, debContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test deb file: %v", err)
	}

	// Build form
	var buf strings.Builder
	writer := multipart.NewWriter(&buf)

	err = BuildAptUploadForm(writer, debFilePath, nil)
	if err != nil {
		t.Fatalf("BuildAptUploadForm failed: %v", err)
	}
	writer.Close()

	// Parse the form
	formData := buf.String()

	// Verify form contains expected fields
	if !strings.Contains(formData, "apt.asset") {
		t.Error("Expected form to contain 'apt.asset'")
	}
	if !strings.Contains(formData, "test-package_1.0.0_amd64.deb") {
		t.Error("Expected form to contain the deb filename")
	}
	if !strings.Contains(formData, "fake deb file content") {
		t.Error("Expected form to contain the deb file content")
	}
}

// TestBuildAptUploadFormFileNotFound tests error handling when deb file doesn't exist
func TestBuildAptUploadFormFileNotFound(t *testing.T) {
	var buf strings.Builder
	writer := multipart.NewWriter(&buf)

	err := BuildAptUploadForm(writer, "/non/existent/file.deb", nil)
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}
}

// TestBuildYumUploadForm tests building multipart form for YUM (RPM) package upload
func TestBuildYumUploadForm(t *testing.T) {
	// Create a test .rpm file
	tempDir := t.TempDir()
	rpmFilePath := tempDir + "/test-package-1.0.0-1.x86_64.rpm"

	// Create test .rpm file with some content
	rpmContent := []byte("fake rpm file content")
	err := os.WriteFile(rpmFilePath, rpmContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test rpm file: %v", err)
	}

	// Build form
	var buf strings.Builder
	writer := multipart.NewWriter(&buf)

	err = BuildYumUploadForm(writer, rpmFilePath, nil)
	if err != nil {
		t.Fatalf("BuildYumUploadForm failed: %v", err)
	}
	writer.Close()

	// Parse the form
	formData := buf.String()

	// Verify form contains expected fields
	if !strings.Contains(formData, "yum.asset") {
		t.Error("Expected form to contain 'yum.asset'")
	}
	if !strings.Contains(formData, "test-package-1.0.0-1.x86_64.rpm") {
		t.Error("Expected form to contain the rpm filename")
	}
	if !strings.Contains(formData, "fake rpm file content") {
		t.Error("Expected form to contain the rpm file content")
	}
}

// TestBuildYumUploadFormFileNotFound tests error handling when rpm file doesn't exist
func TestBuildYumUploadFormFileNotFound(t *testing.T) {
	var buf strings.Builder
	writer := multipart.NewWriter(&buf)

	err := BuildYumUploadForm(writer, "/non/existent/file.rpm", nil)
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}
}

// TestGetAssetByPath tests getting a single asset by path
func TestGetAssetByPath(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	testAsset := Asset{
		ID:       "asset1",
		Path:     "test3/file1.out",
		FileSize: 100,
		Checksum: Checksum{
			SHA256: "abc123",
		},
	}

	// Test with path without leading slash - should be prefixed with /
	server.AddAsset("builds", "/test3/file1.out", testAsset, nil)

	client := NewClient(server.URL, "testuser", "testpass")
	asset, err := client.GetAssetByPath("builds", "test3/file1.out")

	if err != nil {
		t.Fatalf("GetAssetByPath failed: %v", err)
	}

	if asset.ID != "asset1" {
		t.Errorf("Expected asset ID 'asset1', got '%s'", asset.ID)
	}

	if asset.Path != "test3/file1.out" {
		t.Errorf("Expected path 'test3/file1.out', got '%s'", asset.Path)
	}
}

// TestGetAssetByPathWithLeadingSlash tests getting asset when path already has leading slash
func TestGetAssetByPathWithLeadingSlash(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	testAsset := Asset{
		ID:       "asset2",
		Path:     "/docs/readme.txt",
		FileSize: 200,
	}

	// Mock server should expect the path with leading slash
	server.AddAsset("repo", "/docs/readme.txt", testAsset, nil)

	client := NewClient(server.URL, "testuser", "testpass")
	// Pass path with leading slash - should not create double slashes
	asset, err := client.GetAssetByPath("repo", "/docs/readme.txt")

	if err != nil {
		t.Fatalf("GetAssetByPath failed: %v", err)
	}

	if asset.ID != "asset2" {
		t.Errorf("Expected asset ID 'asset2', got '%s'", asset.ID)
	}
}

// TestListAssetsWithLeadingSlash tests listing assets when path has leading slash
func TestListAssetsWithLeadingSlash(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	// Setup mock data with leading slash in query
	server.AddAsset("test-repo", "/docs/file1.txt", Asset{
		ID:       "asset1",
		Path:     "/docs/file1.txt",
		FileSize: 100,
	}, nil)

	client := NewClient(server.URL, "testuser", "testpass")
	// Pass path with leading slash - should not create double slashes
	assets, err := client.ListAssets("test-repo", "/docs", true)

	if err != nil {
		t.Fatalf("ListAssets failed: %v", err)
	}

	if len(assets) != 1 {
		t.Errorf("Expected 1 asset, got %d", len(assets))
	}

	if assets[0].ID != "asset1" {
		t.Errorf("Expected asset ID 'asset1', got '%s'", assets[0].ID)
	}
}

// TestSearchAssetsWithLeadingSlash tests searching assets when path has leading slash
func TestSearchAssetsWithLeadingSlash(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	// Setup mock data with leading slash in query
	server.AddAsset("test-repo", "/libs/example.jar", Asset{
		ID:       "asset1",
		Path:     "/libs/example.jar",
		FileSize: 150,
	}, nil)

	client := NewClient(server.URL, "testuser", "testpass")
	// Pass path with leading slash - should not create double slashes
	assets, err := client.SearchAssets("test-repo", "/libs")

	if err != nil {
		t.Fatalf("SearchAssets failed: %v", err)
	}

	if len(assets) != 1 {
		t.Errorf("Expected 1 asset, got %d", len(assets))
	}

	if assets[0].ID != "asset1" {
		t.Errorf("Expected asset ID 'asset1', got '%s'", assets[0].ID)
	}
}

// TestSearchAssetsWithoutLeadingSlash tests searching assets when path lacks leading slash
func TestSearchAssetsWithoutLeadingSlash(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	// Setup mock data - expect path to be prefixed with /
	server.AddAsset("test-repo", "/libs/example2.jar", Asset{
		ID:       "asset2",
		Path:     "/libs/example2.jar",
		FileSize: 250,
	}, nil)

	client := NewClient(server.URL, "testuser", "testpass")
	// Pass path without leading slash - should be prefixed with /
	assets, err := client.SearchAssets("test-repo", "libs")

	if err != nil {
		t.Fatalf("SearchAssets failed: %v", err)
	}

	if len(assets) != 1 {
		t.Errorf("Expected 1 asset, got %d", len(assets))
	}

	if assets[0].ID != "asset2" {
		t.Errorf("Expected asset ID 'asset2', got '%s'", assets[0].ID)
	}
}

// TestSearchAssetsForCompletionWithLeadingSlash tests autocompletion search with leading slash
func TestSearchAssetsForCompletionWithLeadingSlash(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	// Setup mock data with leading slash in query
	server.AddAsset("test-repo", "/build/output.bin", Asset{
		ID:   "asset1",
		Path: "build/output.bin",
	}, nil)

	client := NewClient(server.URL, "testuser", "testpass")
	// Pass path with leading slash - should not create double slashes
	_, err := client.SearchAssetsForCompletion("test-repo", "/build")

	if err != nil {
		t.Fatalf("SearchAssetsForCompletion failed: %v", err)
	}

	// Test passes if no error occurred - the function normalizes paths correctly
}

// TestSearchAssetsForCompletionWithoutLeadingSlash tests autocompletion search without leading slash
func TestSearchAssetsForCompletionWithoutLeadingSlash(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	// Setup mock data - expect path to be prefixed with /
	server.AddAsset("test-repo", "/build/output.bin", Asset{
		ID:   "asset1",
		Path: "build/output.bin",
	}, nil)

	client := NewClient(server.URL, "testuser", "testpass")
	// Pass path without leading slash - should be prefixed with /
	_, err := client.SearchAssetsForCompletion("test-repo", "build")

	if err != nil {
		t.Fatalf("SearchAssetsForCompletion failed: %v", err)
	}

	// Test passes if no error occurred - the function normalizes paths correctly
}
