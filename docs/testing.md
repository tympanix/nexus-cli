# Mock Nexus Server Testing

This document describes the high-level mock Nexus server infrastructure for unit tests.

## Overview

The codebase includes a reusable mock Nexus server that simulates the Nexus REST API for testing purposes. This mock server eliminates the need for inline HTTP handlers in tests, making tests more readable and maintainable.

## Mock Server Implementation

### `internal/nexusapi/mock_server.go`

A comprehensive mock Nexus server that can be used across all test packages.

**Features:**
- Asset listing with pagination support
- File uploads (including archives)
- Asset downloads
- Request tracking and data capture
- Thread-safe operations

**Usage Example:**
```go
func TestExample(t *testing.T) {
    server := nexusapi.NewMockNexusServer()
    defer server.Close()

    // Setup mock data
    server.AddAssetWithQuery("test-repo", "/path/*", nexusapi.Asset{
        ID: "asset1",
        Path: "/path/file.txt",
        FileSize: 100,
    })

    // Test your code
    client := nexusapi.NewClient(server.URL, "user", "pass")
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

## Using the Mock Server in Different Packages

### In nexusapi package tests

Since the mock server is defined in the `nexusapi` package, tests in this package can use it directly:

```go
func TestSomething(t *testing.T) {
    server := NewMockNexusServer()
    defer server.Close()
    // ... test code
}
```

### In other package tests (e.g., nexus)

Other packages can import and use the mock server from `nexusapi`:

```go
import "github.com/tympanix/nexus-cli/internal/nexusapi"

func TestSomething(t *testing.T) {
    server := nexusapi.NewMockNexusServer()
    defer server.Close()
    // ... test code
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
- `GetUploadedFiles() []UploadedFile` - Get all uploaded files (including archives)
- `GetRequestCount() int` - Get total number of requests

#### Public Fields
- `LastUploadRepo string` - Last repository used for upload
- `LastListRepo string` - Last repository queried
- `LastListPath string` - Last path queried

## Benefits

1. **Readability**: Tests focus on behavior rather than HTTP mechanics
2. **Maintainability**: Mock behavior centralized in one place
3. **Reusability**: Same mock server used across multiple packages
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
server := nexusapi.NewMockNexusServer()
defer server.Close()
server.AddAssetWithQuery("repo", "/path/*", asset)
// Test code uses server.URL
```

## Testing

The mock server is tested in `internal/nexusapi/mock_server_test.go` to ensure it correctly simulates Nexus behavior.
