package nexusapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// MockNexusServer provides a high-level mock Nexus server for testing
type MockNexusServer struct {
	*httptest.Server
	mu sync.RWMutex

	// Assets stores the assets that will be returned by ListAssets
	Assets map[string][]Asset
	// AssetContent stores the content of assets by their download URL
	AssetContent map[string][]byte
	// ContinuationTokens maps request IDs to continuation tokens for pagination
	ContinuationTokens map[string]string

	// Captured data from requests
	UploadedFiles     []UploadedFile
	UploadedArchives  []UploadedArchive
	RequestCount      int
	LastUploadRepo    string
	LastListRepo      string
	LastListPath      string
}

// UploadedFile represents a file that was uploaded to the mock server
type UploadedFile struct {
	Filename   string
	Content    []byte
	Repository string
}

// UploadedArchive represents an archive that was uploaded to the mock server
type UploadedArchive struct {
	Filename   string
	Content    []byte
	Repository string
}

// NewMockNexusServer creates a new mock Nexus server
func NewMockNexusServer() *MockNexusServer {
	mock := &MockNexusServer{
		Assets:             make(map[string][]Asset),
		AssetContent:       make(map[string][]byte),
		ContinuationTokens: make(map[string]string),
		UploadedFiles:      make([]UploadedFile, 0),
		UploadedArchives:   make([]UploadedArchive, 0),
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
	m.mu.Unlock()

	// Parse multipart form (ignore errors for non-multipart content)
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		// If it's not a multipart form, just return success
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Capture uploaded files
	for key := range r.MultipartForm.File {
		if strings.HasPrefix(key, "raw.asset") {
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
			if strings.HasSuffix(header.Filename, ".tar.gz") {
				// This is an archive upload
				m.UploadedArchives = append(m.UploadedArchives, UploadedArchive{
					Filename:   header.Filename,
					Content:    content,
					Repository: repository,
				})
			} else {
				// Regular file upload
				m.UploadedFiles = append(m.UploadedFiles, UploadedFile{
					Filename:   header.Filename,
					Content:    content,
					Repository: repository,
				})
			}
			m.mu.Unlock()
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleListAssets handles asset listing requests
func (m *MockNexusServer) handleListAssets(w http.ResponseWriter, r *http.Request) {
	repository := r.URL.Query().Get("repository")
	query := r.URL.Query().Get("q")
	continuationToken := r.URL.Query().Get("continuationToken")

	m.mu.Lock()
	m.LastListRepo = repository
	// Extract path from query (format: /path/*)
	if len(query) > 2 && strings.HasPrefix(query, "/") && strings.HasSuffix(query, "/*") {
		m.LastListPath = query[1 : len(query)-2]
	}
	m.mu.Unlock()

	// Build the key for looking up assets
	key := repository
	if query != "" {
		key = repository + ":" + query
	}

	// Support for pagination: if a continuation token is provided,
	// check if there's a second page
	pageKey := key
	if continuationToken != "" {
		pageKey = key + ":page2"
	}

	m.mu.RLock()
	assets := m.Assets[pageKey]
	nextToken := ""
	if continuationToken == "" {
		// Check if there's a continuation token for this request
		nextToken = m.ContinuationTokens[key]
	}
	m.mu.RUnlock()

	response := SearchResponse{
		Items:             assets,
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

// AddAsset adds an asset to the mock server's asset list
func (m *MockNexusServer) AddAsset(repository, path string, asset Asset) {
	key := repository + "://" + path + "/*"
	m.mu.Lock()
	m.Assets[key] = append(m.Assets[key], asset)
	m.mu.Unlock()
}

// AddAssetWithQuery adds an asset to the mock server using a custom query string
func (m *MockNexusServer) AddAssetWithQuery(repository, query string, asset Asset) {
	key := repository + ":" + query
	m.mu.Lock()
	m.Assets[key] = append(m.Assets[key], asset)
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

// AddAssetForPage adds an asset to a specific page (for pagination testing)
func (m *MockNexusServer) AddAssetForPage(repository, query string, asset Asset, page int) {
	key := repository + ":" + query
	if page > 1 {
		key = key + ":page2"
	}
	m.mu.Lock()
	m.Assets[key] = append(m.Assets[key], asset)
	m.mu.Unlock()
}

// Reset clears all stored data in the mock server
func (m *MockNexusServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Assets = make(map[string][]Asset)
	m.AssetContent = make(map[string][]byte)
	m.ContinuationTokens = make(map[string]string)
	m.UploadedFiles = make([]UploadedFile, 0)
	m.UploadedArchives = make([]UploadedArchive, 0)
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

// GetUploadedArchives returns the list of uploaded archives
func (m *MockNexusServer) GetUploadedArchives() []UploadedArchive {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]UploadedArchive{}, m.UploadedArchives...)
}

// GetRequestCount returns the number of requests received
func (m *MockNexusServer) GetRequestCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.RequestCount
}
