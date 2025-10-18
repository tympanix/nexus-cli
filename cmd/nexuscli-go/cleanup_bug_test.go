package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

// Test case to reproduce the bug where downloaded files are deleted
// when asset paths from Nexus have a leading slash
func TestDepsSyncCleanupWithLeadingSlashPaths(t *testing.T) {
	mockServer := nexusapi.NewMockNexusServer()
	defer mockServer.Close()

	testFileContent := []byte("test file content for sync")
	testChecksum := "0505007cc25ef733fb754c26db7dd8c38c5cf8f75f571f60a66548212c25b2fa"

	// Add asset with path that includes leading slash (mimicking real Nexus behavior)
	// Note: We pass the path to AddAsset with leading slash, and DON'T set asset.Path
	// so that the mock server will use the normalized path with leading slash
	mockServer.AddAsset("libs", "/docs/example-1.0.0.txt", nexusapi.Asset{
		// Path is intentionally NOT set, so mock server will use normalized path with leading slash
		FileSize: int64(len(testFileContent)),
		Checksum: nexusapi.Checksum{
			SHA256: testChecksum,
		},
		DownloadURL: mockServer.URL + "/repository/libs/docs/example-1.0.0.txt",
	}, testFileContent)

	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	depsIniContent := `[defaults]
repository = libs
checksum = sha256
output_dir = ./local

[example_txt]
path = docs/example-${version}.txt
version = 1.0.0
`
	if err := os.WriteFile("deps.ini", []byte(depsIniContent), 0644); err != nil {
		t.Fatal(err)
	}

	// First run deps lock to create lock file with real asset paths
	rootCmd := buildRootCommand()
	rootCmd.SetArgs([]string{"deps", "lock", "--url", mockServer.URL})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deps lock failed: %v", err)
	}

	// Verify lock file was created
	if _, err := os.Stat("deps-lock.ini"); os.IsNotExist(err) {
		t.Fatal("deps-lock.ini was not created")
	}

	// Now run deps sync with cleanup enabled (default behavior)
	rootCmd = buildRootCommand()
	rootCmd.SetArgs([]string{"deps", "sync", "--url", mockServer.URL})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deps sync failed: %v", err)
	}

	// The downloaded file should still exist after sync with cleanup
	downloadedFile := filepath.Join("local", "docs", "example-1.0.0.txt")
	if _, err := os.Stat(downloadedFile); os.IsNotExist(err) {
		t.Error("downloaded file should exist but was deleted by cleanup")
	}

	// Verify file content
	content, err := os.ReadFile(downloadedFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(testFileContent) {
		t.Errorf("file content mismatch: expected %s, got %s", testFileContent, content)
	}
}
