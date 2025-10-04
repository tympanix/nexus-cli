package nexus

import (
	"fmt"
	"io"
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
	"github.com/tympanix/nexus-cli/internal/nexusapi"
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

func listAssets(repository, src string, config *Config) ([]nexusapi.Asset, error) {
	client := nexusapi.NewClient(config.NexusURL, config.Username, config.Password)
	return client.ListAssets(repository, src)
}

func downloadAsset(asset nexusapi.Asset, destDir string, basePath string, wg *sync.WaitGroup, errCh chan error, bar *progressbar.ProgressBar, skipCh chan bool, config *Config, opts *DownloadOptions) {
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

	client := nexusapi.NewClient(config.NexusURL, config.Username, config.Password)
	f, err := os.Create(localPath)
	if err != nil {
		errCh <- err
		return
	}
	defer f.Close()
	
	// Use a tee reader to update progress bar while downloading
	writer := io.MultiWriter(f, bar)
	err = client.DownloadAsset(asset.DownloadURL, writer)
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

	bar := newProgressBar(totalBytes, "Downloading bytes", opts.QuietMode)

	var wg sync.WaitGroup
	errCh := make(chan error, len(assets))
	skipCh := make(chan bool, len(assets))
	for _, asset := range assets {
		wg.Add(1)
		go func(asset nexusapi.Asset) {
			downloadAsset(asset, destDir, src, &wg, errCh, bar, skipCh, config, opts)
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
func getExpectedChecksum(checksum nexusapi.Checksum, opts *DownloadOptions) string {
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
