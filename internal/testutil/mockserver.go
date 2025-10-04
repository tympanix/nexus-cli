package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Asset represents a Nexus asset (local copy to avoid import cycle)
type Asset struct {
	DownloadURL    string          `json:"downloadUrl"`
	Path           string          `json:"path"`
	ID             string          `json:"id"`
	Repository     string          `json:"repository"`
	Format         string          `json:"format"`
	Checksum       Checksum        `json:"checksum"`
	ContentType    string          `json:"contentType"`
	LastModified   string          `json:"lastModified"`
	LastDownloaded string          `json:"lastDownloaded"`
	Uploader       string          `json:"uploader"`
	UploaderIP     string          `json:"uploaderIp"`
	FileSize       int64           `json:"fileSize"`
	BlobCreated    *string         `json:"blobCreated"`
	BlobStoreName  *string         `json:"blobStoreName"`
	Raw            json.RawMessage `json:"raw"`
}

// Checksum represents checksums for an asset
type Checksum struct {
	SHA1   string `json:"sha1"`
	SHA256 string `json:"sha256"`
	SHA512 string `json:"sha512"`
	MD5    string `json:"md5"`
}

// SearchResponse represents the response from the search API
type SearchResponse struct {
	Items             []Asset `json:"items"`
	ContinuationToken string  `json:"continuationToken"`
}

// ConvertAsset converts any asset type to testutil.Asset using JSON marshaling
// This allows us to work with assets from any package without importing them
func ConvertAsset(asset interface{}) Asset {
	// Marshal to JSON and unmarshal back to our type
	data, _ := json.Marshal(asset)
	var result Asset
	json.Unmarshal(data, &result)
	return result
}

// ConvertAssets converts a slice of assets to testutil.Asset
func ConvertAssets(assets interface{}) []Asset {
	data, _ := json.Marshal(assets)
	var result []Asset
	json.Unmarshal(data, &result)
	return result
}

// ConvertSearchResponse converts any search response type to testutil.SearchResponse
func ConvertSearchResponse(response interface{}) SearchResponse {
	data, _ := json.Marshal(response)
	var result SearchResponse
	json.Unmarshal(data, &result)
	return result
}

// Handler interface defines a handler for specific Nexus API operations
// Single Responsibility: each handler handles one type of operation
type Handler interface {
	Handle(w http.ResponseWriter, r *http.Request, t *testing.T) bool
}

// MockNexusServer provides a mock implementation of Nexus server for testing
// Open/Closed Principle: extend functionality by adding new handlers without modifying core
type MockNexusServer struct {
	handlers []Handler
	server   *httptest.Server
	t        *testing.T
}

// NewMockNexusServer creates a new mock Nexus server
func NewMockNexusServer(t *testing.T) *MockNexusServer {
	mock := &MockNexusServer{
		handlers: []Handler{},
		t:        t,
	}
	
	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, handler := range mock.handlers {
			if handler.Handle(w, r, t) {
				return
			}
		}
		http.NotFound(w, r)
	}))
	
	return mock
}

// AddHandler adds a handler to the mock server
// Allows composing different behaviors
func (m *MockNexusServer) AddHandler(handler Handler) *MockNexusServer {
	m.handlers = append(m.handlers, handler)
	return m
}

// URL returns the mock server URL
func (m *MockNexusServer) URL() string {
	return m.server.URL
}

// Close closes the mock server
func (m *MockNexusServer) Close() {
	m.server.Close()
}

// AssetListHandler handles asset listing requests
type AssetListHandler struct {
	Assets            []Asset
	ContinuationToken string
	ValidateAuth      bool
	ExpectedRepo      string
	ExpectedQuery     string
	ValidateFunc      func(r *http.Request, t *testing.T)
}

// Handle implements Handler interface
func (h *AssetListHandler) Handle(w http.ResponseWriter, r *http.Request, t *testing.T) bool {
	if !strings.Contains(r.URL.Path, "/service/rest/v1/search/assets") {
		return false
	}

	if h.ValidateAuth {
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("Expected basic auth to be set")
		}
		if username == "" || password == "" {
			t.Error("Expected non-empty credentials")
		}
	}

	if h.ExpectedRepo != "" {
		repo := r.URL.Query().Get("repository")
		if repo != h.ExpectedRepo {
			t.Errorf("Expected repository '%s', got '%s'", h.ExpectedRepo, repo)
		}
	}

	if h.ExpectedQuery != "" {
		query := r.URL.Query().Get("q")
		if query != h.ExpectedQuery {
			t.Errorf("Expected query '%s', got '%s'", h.ExpectedQuery, query)
		}
	}

	if h.ValidateFunc != nil {
		h.ValidateFunc(r, t)
	}

	response := SearchResponse{
		Items:             h.Assets,
		ContinuationToken: h.ContinuationToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	return true
}

// UploadHandler handles component upload requests
type UploadHandler struct {
	ValidateAuth     bool
	ExpectedRepo     string
	OnUpload         func(r *http.Request, t *testing.T)
	StatusCode       int
}

// Handle implements Handler interface
func (h *UploadHandler) Handle(w http.ResponseWriter, r *http.Request, t *testing.T) bool {
	if r.Method != "POST" || !strings.Contains(r.URL.Path, "/service/rest/v1/components") {
		return false
	}

	if h.ValidateAuth {
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("Expected basic auth to be set")
		}
		if username == "" || password == "" {
			t.Error("Expected non-empty credentials")
		}
	}

	if h.ExpectedRepo != "" {
		repo := r.URL.Query().Get("repository")
		if repo != h.ExpectedRepo {
			t.Errorf("Expected repository '%s', got '%s'", h.ExpectedRepo, repo)
		}
	}

	if h.OnUpload != nil {
		h.OnUpload(r, t)
	}

	statusCode := h.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusNoContent
	}
	w.WriteHeader(statusCode)
	return true
}

// DownloadHandler handles file download requests
type DownloadHandler struct {
	PathPrefix   string
	Content      []byte
	ValidateAuth bool
	ContentType  string
	StatusCode   int
}

// Handle implements Handler interface
func (h *DownloadHandler) Handle(w http.ResponseWriter, r *http.Request, t *testing.T) bool {
	if !strings.Contains(r.URL.Path, h.PathPrefix) {
		return false
	}

	if h.ValidateAuth {
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("Expected basic auth to be set")
		}
		if username == "" || password == "" {
			t.Error("Expected non-empty credentials")
		}
	}

	contentType := h.ContentType
	if contentType == "" {
		contentType = "text/plain"
	}
	w.Header().Set("Content-Type", contentType)

	statusCode := h.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	w.WriteHeader(statusCode)
	
	if h.Content != nil {
		w.Write(h.Content)
	}
	return true
}

// PaginatedAssetListHandler handles paginated asset listing requests
type PaginatedAssetListHandler struct {
	Pages        []SearchResponse
	currentPage  int
	ValidateAuth bool
}

// Handle implements Handler interface
func (h *PaginatedAssetListHandler) Handle(w http.ResponseWriter, r *http.Request, t *testing.T) bool {
	if !strings.Contains(r.URL.Path, "/service/rest/v1/search/assets") {
		return false
	}

	if h.ValidateAuth {
		username, password, ok := r.BasicAuth()
		if !ok {
			t.Error("Expected basic auth to be set")
		}
		if username == "" || password == "" {
			t.Error("Expected non-empty credentials")
		}
	}

	// Check continuation token to determine which page to return
	continuationToken := r.URL.Query().Get("continuationToken")
	pageIndex := 0
	
	if continuationToken != "" && h.currentPage < len(h.Pages) {
		pageIndex = h.currentPage
		h.currentPage++
	} else if continuationToken == "" {
		pageIndex = 0
		h.currentPage = 1
	}

	if pageIndex >= len(h.Pages) {
		pageIndex = len(h.Pages) - 1
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.Pages[pageIndex])
	return true
}

// CaptureHandler captures request data for validation
type CaptureHandler struct {
	MatchFunc      func(r *http.Request) bool
	CapturedBody   *string
	CapturedHeader *http.Header
	ResponseStatus int
	ResponseBody   []byte
}

// Handle implements Handler interface
func (h *CaptureHandler) Handle(w http.ResponseWriter, r *http.Request, t *testing.T) bool {
	if h.MatchFunc != nil && !h.MatchFunc(r) {
		return false
	}

	if h.CapturedBody != nil {
		body, _ := io.ReadAll(r.Body)
		*h.CapturedBody = string(body)
	}

	if h.CapturedHeader != nil {
		*h.CapturedHeader = r.Header.Clone()
	}

	statusCode := h.ResponseStatus
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	w.WriteHeader(statusCode)

	if h.ResponseBody != nil {
		w.Write(h.ResponseBody)
	}
	return true
}

// DynamicHandler allows handlers with dynamic content
type DynamicHandler struct {
	MatchFunc      func(r *http.Request) bool
	HandleFunc     func(w http.ResponseWriter, r *http.Request, t *testing.T)
}

// Handle implements Handler interface
func (h *DynamicHandler) Handle(w http.ResponseWriter, r *http.Request, t *testing.T) bool {
	if h.MatchFunc != nil && !h.MatchFunc(r) {
		return false
	}

	if h.HandleFunc != nil {
		h.HandleFunc(w, r, t)
	}
	return true
}
