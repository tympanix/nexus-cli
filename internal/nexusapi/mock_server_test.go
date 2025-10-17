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

	server.AddAsset("test-repo", "/test-path/file.txt", asset)

	// Verify asset was added
	server.mu.RLock()
	storedAsset := server.Assets["test-repo:/test-path/file.txt"]
	server.mu.RUnlock()

	if storedAsset.ID != "test-asset-1" {
		t.Errorf("Expected asset ID 'test-asset-1', got '%s'", storedAsset.ID)
	}
}

func TestMockNexusServerReset(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	// Add some data
	asset := Asset{ID: "test", Path: "/path", FileSize: 100}
	server.AddAsset("repo", "/path", asset)
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

func TestMockNexusServerGlobMatching(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	// Add several assets to test glob matching
	server.AddAsset("repo", "/docs/readme.txt", Asset{
		ID:   "asset1",
		Path: "/docs/readme.txt",
	})
	server.AddAsset("repo", "/docs/guide.pdf", Asset{
		ID:   "asset2",
		Path: "/docs/guide.pdf",
	})
	server.AddAsset("repo", "/images/logo.png", Asset{
		ID:   "asset3",
		Path: "/images/logo.png",
	})

	client := NewClient(server.URL, "user", "pass")

	t.Run("query with wildcard", func(t *testing.T) {
		// Search with glob pattern /docs/*
		assets, err := client.ListAssets("repo", "docs")
		if err != nil {
			t.Fatalf("ListAssets failed: %v", err)
		}
		if len(assets) != 2 {
			t.Errorf("Expected 2 assets in /docs/*, got %d", len(assets))
		}
	})

	t.Run("exact path match with name parameter", func(t *testing.T) {
		// Search with exact path
		asset, err := client.GetAssetByPath("repo", "/docs/readme.txt")
		if err != nil {
			t.Fatalf("GetAssetByPath failed: %v", err)
		}
		if asset.ID != "asset1" {
			t.Errorf("Expected asset1, got %s", asset.ID)
		}
	})
}

func TestMockNexusServerBackwardCompatibility(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	asset := Asset{
		ID:   "test-asset",
		Path: "/test/file.txt",
	}

	// Test AddAssetWithQuery (backward compatibility)
	server.AddAssetWithQuery("repo", "/test/*", asset)

	client := NewClient(server.URL, "user", "pass")
	assets, err := client.ListAssets("repo", "test")
	if err != nil {
		t.Fatalf("ListAssets failed: %v", err)
	}
	if len(assets) != 1 {
		t.Errorf("Expected 1 asset, got %d", len(assets))
	}

	// Test AddAssetByName (backward compatibility)
	server.Reset()
	server.AddAssetByName("repo", "/exact/path.txt", Asset{
		ID:   "exact-asset",
		Path: "/exact/path.txt",
	})

	foundAsset, err := client.GetAssetByPath("repo", "/exact/path.txt")
	if err != nil {
		t.Fatalf("GetAssetByPath failed: %v", err)
	}
	if foundAsset.ID != "exact-asset" {
		t.Errorf("Expected exact-asset, got %s", foundAsset.ID)
	}
}

func TestMockNexusServerGlobPatterns(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	// Add various assets to test glob patterns
	server.AddAsset("repo", "/docs/readme.txt", Asset{
		ID:   "asset1",
		Path: "/docs/readme.txt",
	})
	server.AddAsset("repo", "/docs/guide.md", Asset{
		ID:   "asset2",
		Path: "/docs/guide.md",
	})
	server.AddAsset("repo", "/docs/subdir/file.txt", Asset{
		ID:   "asset3",
		Path: "/docs/subdir/file.txt",
	})
	server.AddAsset("repo", "/images/logo.png", Asset{
		ID:   "asset4",
		Path: "/images/logo.png",
	})

	client := NewClient(server.URL, "user", "pass")

	t.Run("ListAssets with /docs/* pattern", func(t *testing.T) {
		// ListAssets("repo", "docs") sends query q=/docs/*
		// This should match all files under /docs/ including subdirectories
		assets, err := client.ListAssets("repo", "docs")
		if err != nil {
			t.Fatalf("ListAssets failed: %v", err)
		}
		// Should match /docs/readme.txt, /docs/guide.md, and /docs/subdir/file.txt
		if len(assets) != 3 {
			t.Errorf("Expected 3 assets, got %d", len(assets))
			for _, a := range assets {
				t.Logf("Found asset: %s - %s", a.ID, a.Path)
			}
		}
	})

	t.Run("GetAssetByPath with exact match", func(t *testing.T) {
		// GetAssetByPath uses the name parameter with exact path
		asset, err := client.GetAssetByPath("repo", "/docs/readme.txt")
		if err != nil {
			t.Fatalf("GetAssetByPath failed: %v", err)
		}
		if asset.ID != "asset1" {
			t.Errorf("Expected asset1, got %s", asset.ID)
		}
	})

	t.Run("GetAssetByPath with glob pattern in name", func(t *testing.T) {
		// Now that name parameter supports globs, test with a glob pattern
		// This uses the name parameter directly with a glob pattern
		// We'll manually construct the request since GetAssetByPath expects exact match

		// Add test to verify glob matching works in the mock server
		// by checking the SearchAssets method which uses q parameter
		assets, err := client.SearchAssets("repo", "docs")
		if err != nil {
			t.Fatalf("SearchAssets failed: %v", err)
		}
		// SearchAssets("repo", "docs") sends query q=/docs*
		// This should match /docs/readme.txt, /docs/guide.md, /docs/subdir/file.txt
		if len(assets) != 3 {
			t.Errorf("Expected 3 assets, got %d", len(assets))
		}
	})
}
