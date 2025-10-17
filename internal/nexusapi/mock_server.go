package nexusapi

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/bmatcuk/doublestar/v4"
)

// MockNexusServer provides a high-level mock Nexus server for testing
type MockNexusServer struct {
	*httptest.Server
	mu sync.RWMutex

	// Assets stores the assets by repository and path
	// Key format: "repository:path"
	Assets map[string]Asset
	// AssetContent stores the content of assets by their download URL
	AssetContent map[string][]byte
	// ContinuationTokens maps pagination keys to continuation tokens
	ContinuationTokens map[string]string
	// Repositories stores the repositories that will be returned by ListRepositories
	Repositories []Repository

	// Captured data from requests
	UploadedFiles  []UploadedFile
	RequestCount   int
	LastUploadRepo string
	LastListRepo   string
	LastListPath   string

	// Error configuration
	RepositoryNotFoundList map[string]bool
}

// UploadedFile represents a file that was uploaded to the mock server
type UploadedFile struct {
	Filename   string
	Content    []byte
	Repository string
}

// NewMockNexusServer creates a new mock Nexus server
func NewMockNexusServer() *MockNexusServer {
	mock := &MockNexusServer{
		Assets:                 make(map[string]Asset),
		AssetContent:           make(map[string][]byte),
		ContinuationTokens:     make(map[string]string),
		UploadedFiles:          make([]UploadedFile, 0),
		RepositoryNotFoundList: make(map[string]bool),
		Repositories:           make([]Repository, 0),
	}

	mock.Server = httptest.NewServer(http.HandlerFunc(mock.handler))
	return mock
}

// handler is the main HTTP handler for the mock server
func (m *MockNexusServer) handler(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	m.RequestCount++
	m.mu.Unlock()

	// Handle upload requests
	if r.Method == "POST" && strings.Contains(r.URL.Path, "/service/rest/v1/components") {
		m.handleUpload(w, r)
		return
	}

	// Handle repository listing requests
	if r.Method == "GET" && strings.Contains(r.URL.Path, "/service/rest/v1/repositories") {
		m.handleListRepositories(w, r)
		return
	}

	// Handle asset listing requests
	if r.Method == "GET" && strings.Contains(r.URL.Path, "/service/rest/v1/search/assets") {
		m.handleListAssets(w, r)
		return
	}

	// Handle asset download requests
	if r.Method == "GET" && strings.Contains(r.URL.Path, "/repository/") {
		m.handleDownloadAsset(w, r)
		return
	}

	http.NotFound(w, r)
}

// handleUpload handles file upload requests
func (m *MockNexusServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	repository := r.URL.Query().Get("repository")
	m.mu.Lock()
	m.LastUploadRepo = repository
	notFound := m.RepositoryNotFoundList[repository]
	m.mu.Unlock()

	// Simulate repository not found error
	if notFound {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"Repository '` + repository + `' not found"}`))
		return
	}

	// Parse multipart form (ignore errors for non-multipart content)
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		// If it's not a multipart form, just return success
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Capture uploaded files
	for key := range r.MultipartForm.File {
		if strings.HasPrefix(key, "raw.asset") || strings.HasPrefix(key, "apt.asset") || strings.HasPrefix(key, "yum.asset") {
			file, header, err := r.FormFile(key)
			if err != nil {
				continue
			}
			content, err := io.ReadAll(file)
			file.Close()
			if err != nil {
				continue
			}

			m.mu.Lock()
			m.UploadedFiles = append(m.UploadedFiles, UploadedFile{
				Filename:   header.Filename,
				Content:    content,
				Repository: repository,
			})
			m.mu.Unlock()
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleListRepositories handles repository listing requests
func (m *MockNexusServer) handleListRepositories(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	repos := m.Repositories
	m.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(repos)
}

// handleListAssets handles asset listing requests
func (m *MockNexusServer) handleListAssets(w http.ResponseWriter, r *http.Request) {
	repository := r.URL.Query().Get("repository")
	query := r.URL.Query().Get("q")
	name := r.URL.Query().Get("name")
	continuationToken := r.URL.Query().Get("continuationToken")

	m.mu.Lock()
	m.LastListRepo = repository
	// Extract path from query (format: /path/*)
	if len(query) > 2 && strings.HasPrefix(query, "/") && strings.HasSuffix(query, "/*") {
		m.LastListPath = query[1 : len(query)-2]
	}
	m.mu.Unlock()

	// Filter assets based on repository and query parameters
	m.mu.RLock()
	var filteredAssets []Asset

	// Collect keys first to ensure consistent ordering
	var keys []string
	for key := range m.Assets {
		keys = append(keys, key)
	}

	// Sort keys for consistent ordering
	// This ensures tests get predictable results
	for i := 0; i < len(keys)-1; i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	for _, key := range keys {
		asset := m.Assets[key]
		// Check if asset belongs to the requested repository
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 || parts[0] != repository {
			continue
		}

		assetPath := parts[1]

		// Apply filtering based on query parameters
		// Both "q" (keyword search) and "name" parameters support glob patterns
		matched := true

		if name != "" {
			// "name" parameter supports glob patterns
			matched = matchGlobPattern(name, assetPath)
		} else if query != "" {
			// "q" parameter supports glob patterns
			matched = matchGlobPattern(query, assetPath)
		}

		if matched {
			filteredAssets = append(filteredAssets, asset)
		}
	}

	// Handle pagination
	pageKey := repository
	if name != "" {
		pageKey = repository + ":name=" + name
	} else if query != "" {
		pageKey = repository + ":" + query
	}

	// Check for continuation token
	nextToken := m.ContinuationTokens[pageKey]
	var responseAssets []Asset

	if continuationToken != "" {
		// This is a request for page 2 or later
		// Split assets in half for testing (simplified pagination)
		if len(filteredAssets) > 1 {
			responseAssets = filteredAssets[len(filteredAssets)/2:]
		} else {
			responseAssets = []Asset{}
		}
		nextToken = "" // No more pages after this
	} else {
		// First page
		if nextToken != "" && len(filteredAssets) > 1 {
			// If continuation token is set, return first half
			responseAssets = filteredAssets[:len(filteredAssets)/2]
		} else {
			// Return all assets
			responseAssets = filteredAssets
			nextToken = "" // Clear token if we're returning everything
		}
	}
	m.mu.RUnlock()

	response := SearchResponse{
		Items:             responseAssets,
		ContinuationToken: nextToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDownloadAsset handles asset download requests
func (m *MockNexusServer) handleDownloadAsset(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	// Try with the full URL path first
	content, exists := m.AssetContent[r.URL.Path]
	if !exists {
		// Try with the full server URL + path
		for key, val := range m.AssetContent {
			if strings.HasSuffix(key, r.URL.Path) {
				content = val
				exists = true
				break
			}
		}
	}
	m.mu.RUnlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

// matchGlobPattern checks if a path matches a glob pattern.
// Both "q" (keyword search) and "name" parameters support glob patterns.
// In Nexus API, a single "*" matches any characters including path separators.
// Uses the doublestar library for proper glob matching with special handling for Nexus patterns.
func matchGlobPattern(pattern, path string) bool {
	// Normalize paths to ensure they start with /
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// Handle special cases for backward compatibility
	if pattern == "//*" || pattern == "/*" {
		// Match all files in repository
		return true
	}

	// For patterns ending with /*, also match the exact path without /*
	// This maintains backward compatibility with tests
	if strings.HasSuffix(pattern, "/*") {
		patternWithoutSlashStar := strings.TrimSuffix(pattern, "/*")
		if path == patternWithoutSlashStar {
			return true
		}
	}

	// Convert Nexus-style patterns to doublestar patterns
	// In Nexus API, /path/* matches all files under /path/ including subdirectories
	// We need to convert this to /path/** for doublestar
	convertedPattern := pattern
	if strings.HasSuffix(pattern, "/*") {
		// Replace trailing /* with /** to match all subdirectories
		convertedPattern = strings.TrimSuffix(pattern, "/*") + "/**"
	} else if strings.Contains(pattern, "*") && !strings.Contains(pattern, "**") {
		// For patterns with * that aren't already **, treat * as matching any characters
		// including path separators by converting to doublestar-compatible pattern
		// Replace single * with ** only when it's meant to match across directories
		// This is a simplified approach - for exact Nexus behavior, we might need prefix matching

		// If the pattern ends with *, it should match as a prefix
		if strings.HasSuffix(pattern, "*") && !strings.HasSuffix(pattern, "/**") {
			// Use prefix matching for patterns like /docs/2025-*
			prefix := strings.TrimSuffix(pattern, "*")
			return strings.HasPrefix(path, prefix)
		}
	}

	// Use doublestar for glob matching
	matched, err := doublestar.Match(convertedPattern, path)
	if err != nil {
		// If pattern is invalid, fall back to exact match
		return pattern == path
	}
	return matched
}

// computeChecksums computes all supported checksums for the given content
func computeChecksums(content []byte) Checksum {
	sha1Hash := sha1.Sum(content)
	sha256Hash := sha256.Sum256(content)
	sha512Hash := sha512.Sum512(content)
	md5Hash := md5.Sum(content)

	return Checksum{
		SHA1:   hex.EncodeToString(sha1Hash[:]),
		SHA256: hex.EncodeToString(sha256Hash[:]),
		SHA512: hex.EncodeToString(sha512Hash[:]),
		MD5:    hex.EncodeToString(md5Hash[:]),
	}
}

// AddAsset adds an asset to the mock server's asset list by path
// The asset will be stored and retrievable via queries that match its path
// If content is provided, it will be automatically set for downloading
// Default values are automatically filled for common fields:
// - DownloadURL: server.URL + "/repository/" + repository + normalizedPath
// - Path: normalizedPath (if not set)
// - Repository: repository (if not set)
// - FileSize: len(content) (if content provided and FileSize is 0)
// - ID: generated from repository:normalizedPath (if not set)
// - Checksum: computed from content (if content provided and checksums not set)
// - Format: "raw" (if not set)
// - ContentType: "application/octet-stream" (if not set)
// Any explicitly set fields in the asset parameter take precedence over defaults
func (m *MockNexusServer) AddAsset(repository, path string, asset Asset, content []byte) {
	// Normalize path to ensure it starts with /
	normalizedPath := path
	if !strings.HasPrefix(normalizedPath, "/") {
		normalizedPath = "/" + normalizedPath
	}

	// Fill in default values if not explicitly set
	if asset.Path == "" {
		asset.Path = normalizedPath
	}
	if asset.Repository == "" {
		asset.Repository = repository
	}
	if asset.DownloadURL == "" {
		asset.DownloadURL = m.Server.URL + "/repository/" + repository + normalizedPath
	}
	if asset.ID == "" {
		// Generate a default ID from repository and path
		asset.ID = repository + ":" + normalizedPath
	}
	if asset.Format == "" {
		asset.Format = "raw"
	}
	if asset.ContentType == "" {
		asset.ContentType = "application/octet-stream"
	}

	// If content is provided, compute defaults from it
	if content != nil {
		if asset.FileSize == 0 {
			asset.FileSize = int64(len(content))
		}
		// Compute checksums if not explicitly set
		if asset.Checksum.SHA1 == "" || asset.Checksum.SHA256 == "" || asset.Checksum.SHA512 == "" || asset.Checksum.MD5 == "" {
			checksums := computeChecksums(content)
			if asset.Checksum.SHA1 == "" {
				asset.Checksum.SHA1 = checksums.SHA1
			}
			if asset.Checksum.SHA256 == "" {
				asset.Checksum.SHA256 = checksums.SHA256
			}
			if asset.Checksum.SHA512 == "" {
				asset.Checksum.SHA512 = checksums.SHA512
			}
			if asset.Checksum.MD5 == "" {
				asset.Checksum.MD5 = checksums.MD5
			}
		}
	}

	key := repository + ":" + normalizedPath
	m.mu.Lock()
	m.Assets[key] = asset
	// If content is provided, set it for downloading
	if content != nil {
		m.AssetContent[asset.DownloadURL] = content
		// Also set content using the path format for backward compatibility
		m.AssetContent["/repository/"+repository+normalizedPath] = content
	}
	m.mu.Unlock()
}

// AddRepository adds a repository to the mock server's repository list
func (m *MockNexusServer) AddRepository(repo Repository) {
	m.mu.Lock()
	m.Repositories = append(m.Repositories, repo)
	m.mu.Unlock()
}

// SetAssetContent sets the content that will be returned when downloading an asset
func (m *MockNexusServer) SetAssetContent(downloadURL string, content []byte) {
	m.mu.Lock()
	m.AssetContent[downloadURL] = content
	m.mu.Unlock()
}

// SetContinuationToken sets a continuation token for pagination testing
func (m *MockNexusServer) SetContinuationToken(repository, query, token string) {
	key := repository + ":" + query
	m.mu.Lock()
	m.ContinuationTokens[key] = token
	m.mu.Unlock()
}

// AddAssetForPage adds assets to a specific page for pagination testing
// This is a helper method that configures pagination behavior
func (m *MockNexusServer) AddAssetForPage(repository, query string, asset Asset, page int) {
	// Add to main storage regardless of page
	path := asset.Path
	m.AddAsset(repository, path, asset, nil)
}

// Reset clears all stored data in the mock server
func (m *MockNexusServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Assets = make(map[string]Asset)
	m.AssetContent = make(map[string][]byte)
	m.ContinuationTokens = make(map[string]string)
	m.UploadedFiles = make([]UploadedFile, 0)
	m.RepositoryNotFoundList = make(map[string]bool)
	m.RequestCount = 0
	m.LastUploadRepo = ""
	m.LastListRepo = ""
	m.LastListPath = ""
}

// GetUploadedFiles returns the list of uploaded files
func (m *MockNexusServer) GetUploadedFiles() []UploadedFile {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]UploadedFile{}, m.UploadedFiles...)
}

// GetRequestCount returns the number of requests received
func (m *MockNexusServer) GetRequestCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.RequestCount
}

// SetRepositoryNotFound marks a repository as not found for error testing
func (m *MockNexusServer) SetRepositoryNotFound(repository string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RepositoryNotFoundList[repository] = true
}
