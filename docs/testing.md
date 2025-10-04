# Mock Nexus Server Testing

This document describes the high-level mock Nexus server infrastructure for unit tests.

## Overview

The codebase includes reusable mock Nexus servers that simulate the Nexus REST API for testing purposes. These mock servers eliminate the need for inline HTTP handlers in tests, making tests more readable and maintainable.

## Mock Server Implementations

### `internal/nexusapi/mock_nexus_test.go`

A comprehensive mock Nexus server for testing the `nexusapi` package.

**Features:**
- Asset listing with pagination support
- File and archive uploads
- Asset downloads
- Request tracking and data capture
- Thread-safe operations

**Usage Example:**
```go
func TestExample(t *testing.T) {
    server := NewMockNexusServer()
    defer server.Close()

    // Setup mock data
    server.AddAssetWithQuery("test-repo", "/path/*", Asset{
        ID: "asset1",
        Path: "/path/file.txt",
        FileSize: 100,
    })

    // Test your code
    client := NewClient(server.URL, "user", "pass")
    assets, err := client.ListAssets("test-repo", "path")
    
    // Validate
    if err != nil {
        t.Fatal(err)
    }
    if len(assets) != 1 {
        t.Errorf("Expected 1 asset, got %d", len(assets))
    }
}
```

### `internal/nexus/mock_server_test.go`

A similar mock server adapted for the `nexus` package tests.

**Usage Example:**
```go
func TestUpload(t *testing.T) {
    server := newMockNexusServer()
    defer server.Close()

    config := &Config{
        NexusURL: server.URL,
        Username: "test",
        Password: "test",
    }

    // Test upload
    err := uploadFiles(testDir, "test-repo", "", config, opts)
    if err != nil {
        t.Fatal(err)
    }

    // Validate captured data
    server.mu.RLock()
    uploadedFiles := server.UploadedFiles
    server.mu.RUnlock()

    if len(uploadedFiles) != 1 {
        t.Errorf("Expected 1 file, got %d", len(uploadedFiles))
    }
}
```

## API Reference

### MockNexusServer Methods

#### Setup Methods
- `AddAssetWithQuery(repository, query string, asset Asset)` - Add an asset to the mock response
- `AddAssetForPage(repository, query string, asset Asset, page int)` - Add an asset for a specific pagination page
- `SetAssetContent(downloadURL string, content []byte)` - Set content for asset downloads
- `SetContinuationToken(repository, query, token string)` - Configure pagination tokens
- `Reset()` - Clear all mock data

#### Accessors
- `GetUploadedFiles() []UploadedFile` - Get all uploaded files
- `GetUploadedArchives() []UploadedArchive` - Get all uploaded archives
- `GetRequestCount() int` - Get total number of requests

#### Public Fields
- `LastUploadRepo string` - Last repository used for upload
- `LastListRepo string` - Last repository queried
- `LastListPath string` - Last path queried

## Benefits

1. **Readability**: Tests focus on behavior rather than HTTP mechanics
2. **Maintainability**: Mock behavior centralized in one place
3. **Reusability**: Same mock server used across multiple tests
4. **Thread-safety**: Proper mutex locking for concurrent access
5. **Debugging**: Easy access to captured request data

## Migration Pattern

**Before:**
```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // 40+ lines of inline request handling
    if r.URL.Path != "/expected" {
        t.Error("wrong path")
    }
    // Manual JSON encoding
    json.NewEncoder(w).Encode(response)
}))
```

**After:**
```go
server := NewMockNexusServer()
defer server.Close()
server.AddAssetWithQuery("repo", "/path/*", asset)
// Test code uses server.URL
```

## Testing

All mock servers are tested in their respective `*_test.go` files to ensure they correctly simulate Nexus behavior.
