package nexus

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"

	"github.com/schollz/progressbar/v3"
)

// DownloadOptions holds options for download operations
type DownloadOptions struct {
	ChecksumAlgorithm string
	SkipChecksum      bool
	Logger            Logger
	QuietMode         bool
	Flatten           bool
}

// SetChecksumAlgorithm validates and sets the checksum algorithm
// Returns an error if the algorithm is not supported
func (opts *DownloadOptions) SetChecksumAlgorithm(algorithm string) error {
	alg := strings.ToLower(algorithm)
	switch alg {
	case "sha1", "sha256", "sha512", "md5":
		opts.ChecksumAlgorithm = alg
		return nil
	default:
		return fmt.Errorf("unsupported checksum algorithm '%s': must be one of: sha1, sha256, sha512, md5", algorithm)
	}
}

type Checksum struct {
	SHA1   string `json:"sha1"`
	SHA256 string `json:"sha256"`
	SHA512 string `json:"sha512"`
	MD5    string `json:"md5"`
}

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

type searchResponse struct {
	Items             []Asset `json:"items"`
	ContinuationToken string  `json:"continuationToken"`
}

func listAssets(repository, src string, config *Config) ([]Asset, error) {
	var assets []Asset
	continuationToken := ""
	for {
		baseURL, err := url.Parse(config.NexusURL)
		if err != nil {
			return nil, fmt.Errorf("invalid Nexus URL: %w", err)
		}
		baseURL.Path = "/service/rest/v1/search/assets"
		query := baseURL.Query()
		query.Set("repository", repository)
		query.Set("format", "raw")
		query.Set("direction", "asc")
		query.Set("sort", "name")
		query.Set("q", fmt.Sprintf("/%s/*", src))
		if continuationToken != "" {
			query.Set("continuationToken", continuationToken)
		}
		baseURL.RawQuery = query.Encode()

		req, _ := http.NewRequest("GET", baseURL.String(), nil)
		req.SetBasicAuth(config.Username, config.Password)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("Failed to list assets: %d", resp.StatusCode)
		}
		var sr searchResponse
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

func downloadAssetUnified(asset Asset, destDir string, basePath string, wg *sync.WaitGroup, errCh chan error, bar *progressbar.ProgressBar, skipCh chan bool, config *Config, opts *DownloadOptions) {
	defer wg.Done()
	path := strings.TrimLeft(asset.Path, "/")
	
	// If flatten is enabled, strip the base path from the asset path
	if opts.Flatten && basePath != "" {
		// Normalize basePath to ensure it has a leading slash for comparison
		normalizedBasePath := "/" + strings.TrimLeft(basePath, "/")
		assetPath := "/" + path
		
		// If the asset path starts with the base path, remove it
		if strings.HasPrefix(assetPath, normalizedBasePath+"/") {
			path = strings.TrimPrefix(assetPath, normalizedBasePath+"/")
		}
	}
	
	localPath := filepath.Join(destDir, path)
	os.MkdirAll(filepath.Dir(localPath), 0755)

	// Check if file exists and validate checksum or skip based on file existence
	shouldSkip := false
	skipReason := ""

	if _, err := os.Stat(localPath); err == nil {
		if opts.SkipChecksum {
			// When checksum validation is skipped, only check if file exists
			shouldSkip = true
			skipReason = "Skipped (file exists): %s\n"
		} else {
			// Normal checksum validation
			expectedChecksum := getExpectedChecksum(asset.Checksum, opts)
			if expectedChecksum != "" {
				actualChecksum, err := computeChecksum(localPath, opts.ChecksumAlgorithm)
				if err == nil && strings.EqualFold(actualChecksum, expectedChecksum) {
					shouldSkip = true
					skipReason = fmt.Sprintf("Skipped (%s match): %%s\n", strings.ToUpper(opts.ChecksumAlgorithm))
				}
			}
		}
	}

	if shouldSkip {
		opts.Logger.Printf(skipReason, localPath)
		// Advance progress bar by file size for skipped files
		if bar != nil {
			bar.Add64(asset.FileSize)
		}
		// Signal that this file was skipped
		if skipCh != nil {
			skipCh <- true
		}
		return
	}

	resp, err := http.NewRequest("GET", asset.DownloadURL, nil)
	if err != nil {
		errCh <- err
		return
	}
	resp.SetBasicAuth(config.Username, config.Password)
	res, err := http.DefaultClient.Do(resp)
	if err != nil {
		errCh <- err
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		errCh <- fmt.Errorf("Failed to download %s: %d", asset.Path, res.StatusCode)
		return
	}
	f, err := os.Create(localPath)
	if err != nil {
		errCh <- err
		return
	}
	defer f.Close()
	reader := io.TeeReader(res.Body, bar)
	_, err = io.Copy(f, reader)
	if err != nil {
		errCh <- err
	}
}

func downloadFolder(srcArg, destDir string, config *Config, opts *DownloadOptions) bool {
	parts := strings.SplitN(srcArg, "/", 2)
	if len(parts) != 2 {
		opts.Logger.Println("Error: The src argument must be in the form 'repository/folder' or 'repository/folder/subfolder'.")
		return false
	}
	repository, src := parts[0], parts[1]
	assets, err := listAssets(repository, src, config)
	if err != nil {
		opts.Logger.Println("Error listing assets:", err)
		return false
	}
	if len(assets) == 0 {
		opts.Logger.Printf("No assets found in folder '%s' in repository '%s'\n", src, repository)
		return false
	}
	// Calculate total bytes to download using fileSize from search API
	totalBytes := int64(0)
	for _, asset := range assets {
		totalBytes += asset.FileSize
	}

	// Create progress bar - write to io.Discard when disabled
	showProgress := isatty() && !opts.QuietMode
	progressWriter := io.Writer(os.Stdout)
	if !showProgress {
		progressWriter = io.Discard
	}
	bar := progressbar.NewOptions64(totalBytes,
		progressbar.OptionSetWriter(progressWriter),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetDescription("Downloading bytes"),
		progressbar.OptionFullWidth(),
	)

	var wg sync.WaitGroup
	errCh := make(chan error, len(assets))
	skipCh := make(chan bool, len(assets))
	for _, asset := range assets {
		wg.Add(1)
		go func(asset Asset) {
			downloadAssetUnified(asset, destDir, src, &wg, errCh, bar, skipCh, config, opts)
		}(asset)
	}
	wg.Wait()
	close(errCh)
	close(skipCh)
	nErrors := 0
	for err := range errCh {
		opts.Logger.Println("Error downloading asset:", err)
		nErrors++
	}
	nSkipped := 0
	for range skipCh {
		nSkipped++
	}
	nDownloaded := len(assets) - nErrors - nSkipped
	bar.Finish()
	opts.Logger.Printf("Downloaded %d/%d files from '%s' in repository '%s' to '%s' (skipped: %d, failed: %d)\n", 
		nDownloaded, len(assets), src, repository, destDir, nSkipped, nErrors)
	return nErrors == 0
}

// getExpectedChecksum returns the expected checksum value for the selected algorithm
func getExpectedChecksum(checksum Checksum, opts *DownloadOptions) string {
	switch strings.ToLower(opts.ChecksumAlgorithm) {
	case "sha1":
		return checksum.SHA1
	case "sha256":
		return checksum.SHA256
	case "sha512":
		return checksum.SHA512
	case "md5":
		return checksum.MD5
	default:
		return checksum.SHA1 // Default to SHA1
	}
}

// computeChecksum computes the checksum of a file using the specified algorithm
func computeChecksum(path string, algorithm string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var h hash.Hash
	switch strings.ToLower(algorithm) {
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	case "md5":
		h = md5.New()
	default:
		h = sha1.New() // Default to SHA1
	}

	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// computeSHA1 computes the SHA1 checksum of a file at the given path.
// Deprecated: Use computeChecksum instead
func computeSHA1(path string) (string, error) {
	return computeChecksum(path, "sha1")
}

// NewSHA1 returns a new hash.Hash computing the SHA1 checksum.
func NewSHA1() hash.Hash {
	return sha1.New()
}

func DownloadMain(src, dest string, config *Config, opts *DownloadOptions) {
	success := downloadFolder(src, dest, config, opts)
	if !success {
		os.Exit(66)
	}
}
