package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// checksumAlgorithm holds the selected checksum algorithm for validation
var checksumAlgorithm = "sha1"

// setChecksumAlgorithm sets the checksum algorithm for validation
// Returns an error if the algorithm is not supported
func setChecksumAlgorithm(algorithm string) error {
	alg := strings.ToLower(algorithm)
	switch alg {
	case "sha1", "sha256", "sha512", "md5":
		checksumAlgorithm = alg
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

func listAssets(repository, src string) ([]Asset, error) {
	var assets []Asset
	continuationToken := ""
	for {
		params := fmt.Sprintf("?repository=%s&format=raw&direction=asc&sort=name&q=/%s/*", repository, src)
		if continuationToken != "" {
			params += "&continuationToken=" + continuationToken
		}
		url := fmt.Sprintf("%s/service/rest/v1/search/assets%s", nexusURL, params)
		req, _ := http.NewRequest("GET", url, nil)
		req.SetBasicAuth(username, password)
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

func downloadAssetUnified(asset Asset, destDir string, wg *sync.WaitGroup, errCh chan error, bar *progressbar.ProgressBar) {
	defer wg.Done()
	path := strings.TrimLeft(asset.Path, "/")
	localPath := filepath.Join(destDir, path)
	os.MkdirAll(filepath.Dir(localPath), 0755)

	// Check if file exists and validate checksum
	expectedChecksum := getExpectedChecksum(asset.Checksum)
	if expectedChecksum != "" {
		if _, err := os.Stat(localPath); err == nil {
			actualChecksum, err := computeChecksum(localPath, checksumAlgorithm)
			if err == nil && strings.EqualFold(actualChecksum, expectedChecksum) {
				fmt.Printf("Skipped (%s match): %s\n", strings.ToUpper(checksumAlgorithm), localPath)
				return
			}
		}
	}

	resp, err := http.NewRequest("GET", asset.DownloadURL, nil)
	if err != nil {
		errCh <- err
		return
	}
	resp.SetBasicAuth(username, password)
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
	var reader io.Reader = res.Body
	if bar != nil {
		reader = io.TeeReader(res.Body, bar)
	}
	_, err = io.Copy(f, reader)
	if err != nil {
		errCh <- err
	}
}

func downloadFolder(srcArg, destDir string) bool {
	parts := strings.SplitN(srcArg, "/", 2)
	if len(parts) != 2 {
		fmt.Println("Error: The src argument must be in the form 'repository/folder' or 'repository/folder/subfolder'.")
		return false
	}
	repository, src := parts[0], parts[1]
	assets, err := listAssets(repository, src)
	if err != nil {
		fmt.Println("Error listing assets:", err)
		return false
	}
	if len(assets) == 0 {
		fmt.Printf("No assets found in folder '%s' in repository '%s'\n", src, repository)
		return true
	}
	// Calculate total bytes to download using fileSize from search API
	totalBytes := int64(0)
	for _, asset := range assets {
		totalBytes += asset.FileSize
	}
	bar := progressbar.DefaultBytes(totalBytes, "Downloading bytes")

	var wg sync.WaitGroup
	errCh := make(chan error, len(assets))
	for _, asset := range assets {
		wg.Add(1)
		go func(asset Asset) {
			downloadAssetUnified(asset, destDir, &wg, errCh, bar)
		}(asset)
	}
	wg.Wait()
	close(errCh)
	nErrors := 0
	for err := range errCh {
		fmt.Println("Error downloading asset:", err)
		nErrors++
	}
	if nErrors == 0 {
		fmt.Printf("Downloaded %d files from '%s' in repository '%s' to '%s'\n", len(assets), src, repository, destDir)
	} else {
		fmt.Printf("Downloaded %d of %d files from '%s' in repository '%s' to '%s'. %d failed.\n", len(assets)-nErrors, len(assets), src, repository, destDir, nErrors)
	}
	return nErrors == 0
}

// getExpectedChecksum returns the expected checksum value for the selected algorithm
func getExpectedChecksum(checksum Checksum) string {
	switch strings.ToLower(checksumAlgorithm) {
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

func downloadMain(src, dest string) {
	success := downloadFolder(src, dest)
	if !success {
		os.Exit(1)
	}
}
