package nexus

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

// mockNexusServer provides a high-level mock Nexus server for testing in the nexus package
type mockNexusServer struct {
	*httptest.Server
	mu sync.RWMutex

	// Assets stores the assets that will be returned by ListAssets
	Assets map[string][]nexusapi.Asset
	// AssetContent stores the content of assets by their download URL path
	AssetContent map[string][]byte
	// ContinuationTokens maps request IDs to continuation tokens for pagination
	ContinuationTokens map[string]string

	// Captured data from requests
	UploadedFiles     []uploadedFile
	UploadedArchives  []uploadedArchive
	RequestCount      int
	LastUploadRepo    string
	LastListRepo      string
	LastListPath      string
}

// uploadedFile represents a file that was uploaded to the mock server
type uploadedFile struct {
	Filename   string
	Content    []byte
	Repository string
}

// uploadedArchive represents an archive that was uploaded to the mock server
type uploadedArchive struct {
	Filename   string
	Content    []byte
	Repository string
}

// newMockNexusServer creates a new mock Nexus server
func newMockNexusServer() *mockNexusServer {
	mock := &mockNexusServer{
		Assets:             make(map[string][]nexusapi.Asset),
		AssetContent:       make(map[string][]byte),
		ContinuationTokens: make(map[string]string),
		UploadedFiles:      make([]uploadedFile, 0),
		UploadedArchives:   make([]uploadedArchive, 0),
	}

	mock.Server = httptest.NewServer(http.HandlerFunc(mock.handler))
	return mock
}

// handler is the main HTTP handler for the mock server
func (m *mockNexusServer) handler(w http.ResponseWriter, r *http.Request) {
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
func (m *mockNexusServer) handleUpload(w http.ResponseWriter, r *http.Request) {
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
				m.UploadedArchives = append(m.UploadedArchives, uploadedArchive{
					Filename:   header.Filename,
					Content:    content,
					Repository: repository,
				})
			} else {
				// Regular file upload
				m.UploadedFiles = append(m.UploadedFiles, uploadedFile{
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
func (m *mockNexusServer) handleListAssets(w http.ResponseWriter, r *http.Request) {
	repository := r.URL.Query().Get("repository")
	query := r.URL.Query().Get("q")

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

	m.mu.RLock()
	assets := m.Assets[key]
	nextToken := m.ContinuationTokens[key]
	m.mu.RUnlock()

	response := nexusapi.SearchResponse{
		Items:             assets,
		ContinuationToken: nextToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDownloadAsset handles asset download requests
func (m *mockNexusServer) handleDownloadAsset(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	// Try with the full URL path
	content, exists := m.AssetContent[r.URL.Path]
	m.mu.RUnlock()

	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

// addAssetWithQuery adds an asset to the mock server using a custom query string
func (m *mockNexusServer) addAssetWithQuery(repository, query string, asset nexusapi.Asset) {
	key := repository + ":" + query
	m.mu.Lock()
	m.Assets[key] = append(m.Assets[key], asset)
	m.mu.Unlock()
}

// setAssetContent sets the content that will be returned when downloading an asset
// The downloadPath should be the URL path (e.g., "/repository/test-repo/file.txt")
func (m *mockNexusServer) setAssetContent(downloadPath string, content []byte) {
	m.mu.Lock()
	m.AssetContent[downloadPath] = content
	m.mu.Unlock()
}
