package nexusapi

import (
	"testing"
)

func TestMockNexusServer(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	if server.URL == "" {
		t.Error("Expected server URL to be set")
	}

	if server.Server == nil {
		t.Fatal("Expected httptest.Server to be initialized")
	}
}

func TestMockNexusServerAddAsset(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	asset := Asset{
		ID:       "test-asset-1",
		Path:     "/test-path/file.txt",
		FileSize: 100,
	}

	server.AddAssetWithQuery("test-repo", "/test-path/*", asset)

	// Verify asset was added
	server.mu.RLock()
	assets := server.Assets["test-repo:/test-path/*"]
	server.mu.RUnlock()

	if len(assets) != 1 {
		t.Fatalf("Expected 1 asset, got %d", len(assets))
	}

	if assets[0].ID != "test-asset-1" {
		t.Errorf("Expected asset ID 'test-asset-1', got '%s'", assets[0].ID)
	}
}

func TestMockNexusServerReset(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	// Add some data
	asset := Asset{ID: "test", Path: "/path", FileSize: 100}
	server.AddAssetWithQuery("repo", "/path/*", asset)
	server.SetAssetContent("url", []byte("content"))

	server.mu.Lock()
	server.UploadedFiles = append(server.UploadedFiles, UploadedFile{
		Filename: "test.txt",
		Content:  []byte("test"),
	})
	server.RequestCount = 5
	server.mu.Unlock()

	// Reset
	server.Reset()

	// Verify everything is cleared
	if len(server.GetUploadedFiles()) != 0 {
		t.Error("Expected uploaded files to be cleared")
	}
	if server.GetRequestCount() != 0 {
		t.Error("Expected request count to be reset to 0")
	}

	server.mu.RLock()
	if len(server.Assets) != 0 {
		t.Error("Expected assets to be cleared")
	}
	if len(server.AssetContent) != 0 {
		t.Error("Expected asset content to be cleared")
	}
	server.mu.RUnlock()
}
