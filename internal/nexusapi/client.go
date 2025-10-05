package nexusapi

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// Client represents a Nexus API client
type Client struct {
	BaseURL    string
	Username   string
	Password   string
	HTTPClient *http.Client
}

// NewClient creates a new Nexus API client
func NewClient(baseURL, username, password string) *Client {
	return &Client{
		BaseURL:    baseURL,
		Username:   username,
		Password:   password,
		HTTPClient: http.DefaultClient,
	}
}

// Checksum represents checksums for an asset
type Checksum struct {
	SHA1   string `json:"sha1"`
	SHA256 string `json:"sha256"`
	SHA512 string `json:"sha512"`
	MD5    string `json:"md5"`
}

// Asset represents a Nexus asset
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

// SearchResponse represents the response from the search API
type SearchResponse struct {
	Items             []Asset `json:"items"`
	ContinuationToken string  `json:"continuationToken"`
}

// ListAssets lists all assets in a repository path
func (c *Client) ListAssets(repository, path string) ([]Asset, error) {
	var assets []Asset
	continuationToken := ""
	for {
		baseURL, err := url.Parse(c.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("invalid Nexus URL: %w", err)
		}
		baseURL.Path = "/service/rest/v1/search/assets"
		query := baseURL.Query()
		query.Set("repository", repository)
		query.Set("format", "raw")
		query.Set("direction", "asc")
		query.Set("sort", "name")
		query.Set("q", fmt.Sprintf("/%s/*", path))
		if continuationToken != "" {
			query.Set("continuationToken", continuationToken)
		}
		baseURL.RawQuery = query.Encode()

		req, _ := http.NewRequest("GET", baseURL.String(), nil)
		req.SetBasicAuth(c.Username, c.Password)
		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("Failed to list assets: %d", resp.StatusCode)
		}
		var sr SearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
			return nil, err
		}
		assets = append(assets, sr.Items...)
		if sr.ContinuationToken == "" {
			break
		}
		continuationToken = sr.ContinuationToken
	}
	return assets, nil
}

// UploadComponent uploads a component to a Nexus repository
func (c *Client) UploadComponent(repository string, body io.Reader, contentType string) error {
	baseURL, err := url.Parse(c.BaseURL)
	if err != nil {
		return fmt.Errorf("invalid Nexus URL: %w", err)
	}
	baseURL.Path = "/service/rest/v1/components"
	query := baseURL.Query()
	query.Set("repository", repository)
	baseURL.RawQuery = query.Encode()

	req, err := http.NewRequest("POST", baseURL.String(), body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Content-Type", contentType)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 204 {
		return nil
	}
	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(respBody))
}

// DownloadAsset downloads an asset from a Nexus repository
func (c *Client) DownloadAsset(downloadURL string, writer io.Writer) error {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.Username, c.Password)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to download asset: %d", resp.StatusCode)
	}
	_, err = io.Copy(writer, resp.Body)
	return err
}

// GetFormDataContentType returns the content type for a multipart form writer
func GetFormDataContentType(writer *multipart.Writer) string {
	return writer.FormDataContentType()
}

// FileUpload represents a file to be uploaded
type FileUpload struct {
	FilePath     string // Absolute path to the file
	RelativePath string // Relative path to use in Nexus (with forward slashes)
}

// FileProcessCallback is called before processing each file during upload
// idx is the 0-based index of the file being processed, total is the total number of files
type FileProcessCallback func(idx, total int)

// BuildRawUploadForm builds a multipart form for uploading files to a Nexus RAW repository
// It writes the form data to the provided writer and returns any error encountered
// If onFileStart is provided, it will be called before processing each file with the index and total count
// If onFileComplete is provided, it will be called after processing each file with the index and total count
func BuildRawUploadForm(writer *multipart.Writer, files []FileUpload, subdir string, progressWriter io.Writer, onFileStart, onFileComplete FileProcessCallback) error {
	for idx, file := range files {
		// Notify callback that we're starting to process this file
		if onFileStart != nil {
			onFileStart(idx, len(files))
		}

		f, err := os.Open(file.FilePath)
		if err != nil {
			return err
		}
		defer f.Close()

		// Create form file with Nexus RAW format: raw.asset1, raw.asset2, etc.
		part, err := writer.CreateFormFile(fmt.Sprintf("raw.asset%d", idx+1), filepath.Base(file.FilePath))
		if err != nil {
			return err
		}

		// Copy file content to form, optionally through progress writer
		var reader io.Reader = f
		if progressWriter != nil {
			reader = io.TeeReader(f, progressWriter)
		}
		if _, err := io.Copy(part, reader); err != nil {
			return err
		}

		// Add filename field with relative path
		_ = writer.WriteField(fmt.Sprintf("raw.asset%d.filename", idx+1), file.RelativePath)

		// Notify callback that we've completed processing this file
		if onFileComplete != nil {
			onFileComplete(idx, len(files))
		}
	}

	// Add directory field if subdirectory is specified
	if subdir != "" {
		_ = writer.WriteField("raw.directory", subdir)
	}

	return nil
}
