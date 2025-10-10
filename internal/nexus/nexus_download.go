package nexus

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

// DownloadOptions holds options for download operations
type DownloadOptions struct {
	ChecksumAlgorithm    string
	SkipChecksum         bool
	Logger               Logger
	QuietMode            bool
	Flatten              bool
	DeleteExtra          bool
	Compress             bool              // Enable decompression (tar.gz, tar.zst, or zip)
	CompressionFormat    CompressionFormat // Compression format to use (gzip, zstd, or zip)
	CompressionFormatStr string            // String representation of compression format from flag
	ChecksumAlgorithmStr string            // String representation of checksum algorithm from flag
	KeyFromFile          string            // Path to file to compute hash from for {key} template
	checksumValidator    ChecksumValidator // Internal validator instance
}

// SetChecksumAlgorithm validates and sets the checksum algorithm
// Returns an error if the algorithm is not supported
func (opts *DownloadOptions) SetChecksumAlgorithm(algorithm string) error {
	validator, err := NewChecksumValidator(algorithm)
	if err != nil {
		return err
	}
	opts.ChecksumAlgorithm = validator.Algorithm()
	opts.checksumValidator = validator
	return nil
}

// SetCompressionFormat validates and sets the compression format
// Returns an error if the format is not supported
func (opts *DownloadOptions) SetCompressionFormat(format string) error {
	parsed, err := ParseCompressionFormat(format)
	if err != nil {
		return err
	}
	opts.CompressionFormat = parsed
	return nil
}

// Validate validates and configures all options
// Returns an error if any option is invalid
func (opts *DownloadOptions) Validate() error {
	if opts.CompressionFormatStr != "" {
		if err := opts.SetCompressionFormat(opts.CompressionFormatStr); err != nil {
			return err
		}
	}
	if opts.ChecksumAlgorithmStr != "" {
		if err := opts.SetChecksumAlgorithm(opts.ChecksumAlgorithmStr); err != nil {
			return err
		}
	}
	return nil
}

func listAssets(repository, src string, config *Config) ([]nexusapi.Asset, error) {
	client := nexusapi.NewClient(config.NexusURL, config.Username, config.Password)
	return client.ListAssets(repository, src)
}

func downloadAsset(asset nexusapi.Asset, destDir string, basePath string, wg *sync.WaitGroup, errCh chan error, bar *progressBarWithCount, skipCh chan bool, config *Config, opts *DownloadOptions) {
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
		} else if opts.checksumValidator != nil {
			// Use the new ChecksumValidator for validation
			valid, err := opts.checksumValidator.Validate(localPath, asset.Checksum)
			if err == nil && valid {
				shouldSkip = true
				skipReason = fmt.Sprintf("Skipped (%s match): %%s\n", strings.ToUpper(opts.ChecksumAlgorithm))
			}
		}
	}

	if shouldSkip {
		opts.Logger.VerbosePrintf(skipReason, localPath)
		// Advance progress bar by file size for skipped files
		if bar != nil {
			bar.Add64(asset.FileSize)
			bar.incrementFile()
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
	} else {
		// Only increment file count on successful download
		bar.incrementFile()
	}
}

// DownloadStatus represents the result of a download operation
type DownloadStatus int

const (
	DownloadSuccess       DownloadStatus = 0
	DownloadError         DownloadStatus = 1
	DownloadNoAssetsFound DownloadStatus = 66
)

func downloadFolder(srcArg, destDir string, config *Config, opts *DownloadOptions) DownloadStatus {
	repository, src, ok := ParseRepositoryPath(srcArg)
	if !ok {
		opts.Logger.Println("Error: The src argument must be in the form 'repository/folder' or 'repository/folder/subfolder'.")
		return DownloadError
	}

	// Check if src ends with .tar.gz, .tar.zst, or .zip for explicit archive name
	explicitArchiveName := ""
	if opts.Compress && (strings.HasSuffix(src, ".tar.gz") || strings.HasSuffix(src, ".tar.zst") || strings.HasSuffix(src, ".zip")) {
		// Extract the archive name from the path
		lastSlash := strings.LastIndex(src, "/")
		if lastSlash >= 0 {
			explicitArchiveName = src[lastSlash+1:]
			src = src[:lastSlash]
		} else {
			// The entire src is the archive name
			explicitArchiveName = src
			src = ""
		}
	}

	// If compression is enabled, look for a compressed archive
	if opts.Compress {
		return downloadFolderCompressedWithArchiveName(repository, src, explicitArchiveName, destDir, config, opts)
	}

	// Original uncompressed download logic
	assets, err := listAssets(repository, src, config)
	if err != nil {
		opts.Logger.Println("Error listing assets:", err)
		return DownloadError
	}
	if len(assets) == 0 {
		opts.Logger.Printf("No assets found in folder '%s' in repository '%s'\n", src, repository)
		return DownloadNoAssetsFound
	}

	// Build a map of remote asset paths for delete-extra functionality
	remoteAssetPaths := make(map[string]bool)
	for _, asset := range assets {
		path := strings.TrimLeft(asset.Path, "/")

		// If flatten is enabled, strip the base path from the asset path
		if opts.Flatten && src != "" {
			normalizedBasePath := "/" + strings.TrimLeft(src, "/")
			assetPath := "/" + path

			if strings.HasPrefix(assetPath, normalizedBasePath+"/") {
				path = strings.TrimPrefix(assetPath, normalizedBasePath+"/")
			}
		}

		remoteAssetPaths[filepath.Join(destDir, path)] = true
	}

	// Calculate total bytes to download using fileSize from search API
	totalBytes := int64(0)
	for _, asset := range assets {
		totalBytes += asset.FileSize
	}

	baseBar := newProgressBar(totalBytes, "Downloading files", 0, len(assets), opts.QuietMode)
	var current int32
	bar := &progressBarWithCount{
		bar:          baseBar,
		current:      &current,
		total:        len(assets),
		description:  "Downloading files",
		showProgress: isatty() && !opts.QuietMode,
	}

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

	// Delete extra files if requested
	var nDeleted int
	if opts.DeleteExtra {
		nDeleted = deleteExtraFiles(destDir, remoteAssetPaths, opts)
	}

	if nDeleted > 0 {
		opts.Logger.Printf("Downloaded %d/%d files from '%s' in repository '%s' to '%s' (skipped: %d, deleted: %d, failed: %d)\n",
			nDownloaded, len(assets), src, repository, destDir, nSkipped, nDeleted, nErrors)
	} else {
		opts.Logger.Printf("Downloaded %d/%d files from '%s' in repository '%s' to '%s' (skipped: %d, failed: %d)\n",
			nDownloaded, len(assets), src, repository, destDir, nSkipped, nErrors)
	}
	if nErrors == 0 {
		return DownloadSuccess
	}
	return DownloadError
}

// downloadFolderCompressed downloads and extracts a compressed archive
func downloadFolderCompressed(repository, src, destDir string, config *Config, opts *DownloadOptions) DownloadStatus {
	return downloadFolderCompressedWithArchiveName(repository, src, "", destDir, config, opts)
}

// downloadFolderCompressedWithArchiveName downloads and extracts a compressed archive with optional explicit name
func downloadFolderCompressedWithArchiveName(repository, src, explicitArchiveName, destDir string, config *Config, opts *DownloadOptions) DownloadStatus {
	// Require explicit archive name
	if explicitArchiveName == "" {
		ext := opts.CompressionFormat.Extension()
		if opts.CompressionFormat == "" {
			ext = ".tar.gz"
		}
		opts.Logger.Printf("Error: when using --compress, you must specify the %s filename in the source path (e.g., repo/path/archive%s)\n", ext, ext)
		return DownloadError
	}

	archiveName := explicitArchiveName

	// Detect compression format from filename if not explicitly set
	if opts.CompressionFormat == "" {
		opts.CompressionFormat = DetectCompressionFromFilename(archiveName)
	}

	opts.Logger.VerbosePrintf("Looking for compressed archive: %s (format: %s)\n", archiveName, opts.CompressionFormat)

	// List assets to find the archive
	assets, err := listAssets(repository, src, config)
	if err != nil {
		opts.Logger.Println("Error listing assets:", err)
		return DownloadError
	}

	// Find the archive file
	var archiveAsset *nexusapi.Asset
	for _, asset := range assets {
		if strings.HasSuffix(asset.Path, archiveName) {
			archiveAsset = &asset
			break
		}
	}

	if archiveAsset == nil {
		opts.Logger.Printf("Archive '%s' not found in '%s' in repository '%s'\n", archiveName, src, repository)
		opts.Logger.VerbosePrintln("Available assets:")
		for _, asset := range assets {
			opts.Logger.VerbosePrintf("  - %s\n", asset.Path)
		}
		// If we got the asset list successfully but the specific archive wasn't found,
		// this is still a "no assets found" scenario (for the specific archive requested)
		if len(assets) == 0 {
			return DownloadNoAssetsFound
		}
		return DownloadError
	}

	bar := newProgressBar(archiveAsset.FileSize, "Downloading archive", 1, 1, opts.QuietMode)

	// Download and extract archive
	client := nexusapi.NewClient(config.NexusURL, config.Username, config.Password)

	// Create a pipe for streaming decompression
	pr, pw := io.Pipe()
	errChan := make(chan error, 1)

	// Extract in a goroutine
	go func() {
		if err := opts.CompressionFormat.ExtractArchive(pr, destDir); err != nil {
			errChan <- fmt.Errorf("failed to extract archive: %w", err)
		} else {
			errChan <- nil
		}
	}()

	// Download with progress tracking
	progressWriter := io.MultiWriter(pw, bar)
	err = client.DownloadAsset(archiveAsset.DownloadURL, progressWriter)
	pw.Close()

	if err != nil {
		opts.Logger.Printf("Failed to download archive: %v\n", err)
		return DownloadError
	}

	// Wait for extraction to complete
	if extractErr := <-errChan; extractErr != nil {
		opts.Logger.Printf("Failed to extract archive: %v\n", extractErr)
		return DownloadError
	}

	bar.Finish()
	if isatty() && !opts.QuietMode {
		fmt.Println()
	}
	opts.Logger.Printf("Downloaded and extracted archive '%s' from '%s' in repository '%s' to '%s'\n",
		archiveName, src, repository, destDir)
	return DownloadSuccess
}

// deleteExtraFiles removes local files that are not present in the remote asset map
func deleteExtraFiles(destDir string, remoteAssetPaths map[string]bool, opts *DownloadOptions) int {
	nDeleted := 0

	// Walk through all files in the destination directory
	err := filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if this file exists in remote assets
		if !remoteAssetPaths[path] {
			opts.Logger.VerbosePrintf("Deleting extra file: %s\n", path)
			if err := os.Remove(path); err != nil {
				opts.Logger.Printf("Failed to delete file %s: %v\n", path, err)
			} else {
				nDeleted++
			}
		}

		return nil
	})

	if err != nil {
		opts.Logger.Printf("Error walking directory: %v\n", err)
	}

	// Clean up empty directories
	cleanupEmptyDirectories(destDir, opts)

	return nDeleted
}

// cleanupEmptyDirectories removes empty directories from the destination
func cleanupEmptyDirectories(destDir string, opts *DownloadOptions) {
	// Walk in reverse order to remove nested empty directories first
	filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root destination directory itself
		if path == destDir {
			return nil
		}

		if info.IsDir() {
			// Check if directory is empty
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil
			}

			if len(entries) == 0 {
				opts.Logger.VerbosePrintf("Removing empty directory: %s\n", path)
				os.Remove(path)
			}
		}

		return nil
	})
}

func DownloadMain(src, dest string, config *Config, opts *DownloadOptions) {
	processedSrc, err := processKeyTemplate(src, opts.KeyFromFile)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	if opts.KeyFromFile != "" {
		opts.Logger.Printf("Using key template: %s -> %s\n", src, processedSrc)
	}

	status := downloadFolder(processedSrc, dest, config, opts)
	if status != DownloadSuccess {
		os.Exit(int(status))
	}
}
