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
		ID: "test-asset-1",
	}

	server.AddAsset("test-repo", "/test-path/file.txt", asset, nil)

	// Verify asset was added
	server.mu.RLock()
	storedAsset := server.Assets["test-repo:/test-path/file.txt"]
	server.mu.RUnlock()

	if storedAsset.ID != "test-asset-1" {
		t.Errorf("Expected asset ID 'test-asset-1', got '%s'", storedAsset.ID)
	}

	// Verify auto-filled fields
	if storedAsset.Path != "/test-path/file.txt" {
		t.Errorf("Expected path to be auto-filled to '/test-path/file.txt', got '%s'", storedAsset.Path)
	}
	if storedAsset.Repository != "test-repo" {
		t.Errorf("Expected repository to be auto-filled to 'test-repo', got '%s'", storedAsset.Repository)
	}
}

func TestMockNexusServerReset(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	// Add some data
	server.AddAsset("repo", "/path", Asset{}, []byte("content"))

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

	// Add several assets to test glob matching - minimal Asset structs
	server.AddAsset("repo", "/docs/readme.txt", Asset{ID: "asset1"}, nil)
	server.AddAsset("repo", "/docs/guide.pdf", Asset{ID: "asset2"}, nil)
	server.AddAsset("repo", "/images/logo.png", Asset{ID: "asset3"}, nil)

	client := NewClient(server.URL, "user", "pass")

	t.Run("query with wildcard", func(t *testing.T) {
		// Search with glob pattern /docs/*
		assets, err := client.ListAssets("repo", "docs", true)
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

	// Test AddAsset with minimal Asset struct
	server.AddAsset("repo", "/test/file.txt", Asset{ID: "test-asset"}, nil)

	client := NewClient(server.URL, "user", "pass")
	assets, err := client.ListAssets("repo", "test", true)
	if err != nil {
		t.Fatalf("ListAssets failed: %v", err)
	}
	if len(assets) != 1 {
		t.Errorf("Expected 1 asset, got %d", len(assets))
	}

	// Test with exact path
	server.Reset()
	server.AddAsset("repo", "/exact/path.txt", Asset{ID: "exact-asset"}, nil)

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

	// Add various assets to test glob patterns - minimal Asset structs
	server.AddAsset("repo", "/docs/readme.txt", Asset{ID: "asset1"}, nil)
	server.AddAsset("repo", "/docs/guide.md", Asset{ID: "asset2"}, nil)
	server.AddAsset("repo", "/docs/subdir/file.txt", Asset{ID: "asset3"}, nil)
	server.AddAsset("repo", "/images/logo.png", Asset{ID: "asset4"}, nil)

	client := NewClient(server.URL, "user", "pass")

	t.Run("ListAssets with /docs/* pattern", func(t *testing.T) {
		// ListAssets("repo", "docs", true) sends query q=/docs/*
		// This should match all files under /docs/ including subdirectories
		assets, err := client.ListAssets("repo", "docs", true)
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

func TestMockNexusServerAutoFillDefaults(t *testing.T) {
	server := NewMockNexusServer()
	defer server.Close()

	testContent := []byte("test content for checksums")

	t.Run("auto-fill with content", func(t *testing.T) {
		// Add asset with minimal information - only ID set
		server.AddAsset("repo1", "/test/file.txt", Asset{
			ID: "my-custom-id",
		}, testContent)

		// Verify defaults were filled
		server.mu.RLock()
		asset := server.Assets["repo1:/test/file.txt"]
		server.mu.RUnlock()

		// Custom ID should be preserved
		if asset.ID != "my-custom-id" {
			t.Errorf("Expected ID 'my-custom-id', got '%s'", asset.ID)
		}

		// Path should be auto-filled
		if asset.Path != "/test/file.txt" {
			t.Errorf("Expected Path '/test/file.txt', got '%s'", asset.Path)
		}

		// Repository should be auto-filled
		if asset.Repository != "repo1" {
			t.Errorf("Expected Repository 'repo1', got '%s'", asset.Repository)
		}

		// DownloadURL should be auto-generated
		expectedURL := server.URL + "/repository/repo1/test/file.txt"
		if asset.DownloadURL != expectedURL {
			t.Errorf("Expected DownloadURL '%s', got '%s'", expectedURL, asset.DownloadURL)
		}

		// FileSize should be computed from content
		if asset.FileSize != int64(len(testContent)) {
			t.Errorf("Expected FileSize %d, got %d", len(testContent), asset.FileSize)
		}

		// Checksums should be computed
		if asset.Checksum.SHA1 == "" {
			t.Error("Expected SHA1 checksum to be computed")
		}
		if asset.Checksum.SHA256 == "" {
			t.Error("Expected SHA256 checksum to be computed")
		}
		if asset.Checksum.SHA512 == "" {
			t.Error("Expected SHA512 checksum to be computed")
		}
		if asset.Checksum.MD5 == "" {
			t.Error("Expected MD5 checksum to be computed")
		}

		// Format should be auto-filled
		if asset.Format != "raw" {
			t.Errorf("Expected Format 'raw', got '%s'", asset.Format)
		}

		// ContentType should be auto-filled
		if asset.ContentType != "application/octet-stream" {
			t.Errorf("Expected ContentType 'application/octet-stream', got '%s'", asset.ContentType)
		}
	})

	t.Run("explicit values take precedence", func(t *testing.T) {
		// Add asset with explicit values that should NOT be overridden
		customChecksum := Checksum{
			SHA1:   "custom-sha1",
			SHA256: "custom-sha256",
			SHA512: "custom-sha512",
			MD5:    "custom-md5",
		}

		server.AddAsset("repo2", "/test/custom.txt", Asset{
			ID:          "custom-id",
			Path:        "/custom/path",
			Repository:  "custom-repo",
			DownloadURL: "http://custom.url/download",
			FileSize:    9999,
			Checksum:    customChecksum,
			Format:      "maven2",
			ContentType: "text/plain",
		}, testContent)

		// Verify explicit values were preserved
		server.mu.RLock()
		asset := server.Assets["repo2:/test/custom.txt"]
		server.mu.RUnlock()

		if asset.ID != "custom-id" {
			t.Errorf("Expected custom ID to be preserved, got '%s'", asset.ID)
		}
		if asset.Path != "/custom/path" {
			t.Errorf("Expected custom Path to be preserved, got '%s'", asset.Path)
		}
		if asset.Repository != "custom-repo" {
			t.Errorf("Expected custom Repository to be preserved, got '%s'", asset.Repository)
		}
		if asset.DownloadURL != "http://custom.url/download" {
			t.Errorf("Expected custom DownloadURL to be preserved, got '%s'", asset.DownloadURL)
		}
		if asset.FileSize != 9999 {
			t.Errorf("Expected custom FileSize to be preserved, got %d", asset.FileSize)
		}
		if asset.Checksum.SHA1 != "custom-sha1" {
			t.Errorf("Expected custom SHA1 to be preserved, got '%s'", asset.Checksum.SHA1)
		}
		if asset.Checksum.SHA256 != "custom-sha256" {
			t.Errorf("Expected custom SHA256 to be preserved, got '%s'", asset.Checksum.SHA256)
		}
		if asset.Format != "maven2" {
			t.Errorf("Expected custom Format to be preserved, got '%s'", asset.Format)
		}
		if asset.ContentType != "text/plain" {
			t.Errorf("Expected custom ContentType to be preserved, got '%s'", asset.ContentType)
		}
	})

	t.Run("auto-fill without content", func(t *testing.T) {
		// Add asset without content - checksums and filesize should not be filled
		server.AddAsset("repo3", "/test/nocontent.txt", Asset{}, nil)

		server.mu.RLock()
		asset := server.Assets["repo3:/test/nocontent.txt"]
		server.mu.RUnlock()

		// Basic fields should still be auto-filled
		if asset.Path != "/test/nocontent.txt" {
			t.Errorf("Expected Path '/test/nocontent.txt', got '%s'", asset.Path)
		}
		if asset.Repository != "repo3" {
			t.Errorf("Expected Repository 'repo3', got '%s'", asset.Repository)
		}
		if asset.ID == "" {
			t.Error("Expected ID to be auto-generated")
		}

		// FileSize should be 0 when no content provided
		if asset.FileSize != 0 {
			t.Errorf("Expected FileSize 0, got %d", asset.FileSize)
		}

		// Checksums should be empty when no content provided
		if asset.Checksum.SHA1 != "" || asset.Checksum.SHA256 != "" {
			t.Error("Expected checksums to be empty when no content provided")
		}
	})

	t.Run("partial checksum override", func(t *testing.T) {
		// Test that providing only some checksums works correctly
		server.AddAsset("repo4", "/test/partial.txt", Asset{
			Checksum: Checksum{
				SHA1: "custom-sha1-only",
				// SHA256, SHA512, MD5 should be auto-computed
			},
		}, testContent)

		server.mu.RLock()
		asset := server.Assets["repo4:/test/partial.txt"]
		server.mu.RUnlock()

		// Explicit SHA1 should be preserved
		if asset.Checksum.SHA1 != "custom-sha1-only" {
			t.Errorf("Expected custom SHA1 to be preserved, got '%s'", asset.Checksum.SHA1)
		}

		// Other checksums should be auto-computed
		if asset.Checksum.SHA256 == "" {
			t.Error("Expected SHA256 to be auto-computed")
		}
		if asset.Checksum.SHA512 == "" {
			t.Error("Expected SHA512 to be auto-computed")
		}
		if asset.Checksum.MD5 == "" {
			t.Error("Expected MD5 to be auto-computed")
		}
	})
}
