package operations

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tympanix/nexus-cli/internal/archive"
	"github.com/tympanix/nexus-cli/internal/config"
	"github.com/tympanix/nexus-cli/internal/nexusapi"
	"github.com/tympanix/nexus-cli/internal/output"
	"github.com/tympanix/nexus-cli/internal/progress"
	"github.com/tympanix/nexus-cli/internal/util"
)

func listAssets(repository, src string, config *config.Config, recursive bool) ([]nexusapi.Asset, error) {
	client := nexusapi.NewClient(config.NexusURL, config.Username, config.Password)
	return client.ListAssets(repository, src, recursive)
}

func filterAssetsByGlob(assets []nexusapi.Asset, basePath string, globPattern string) ([]nexusapi.Asset, error) {
	return util.FilterWithGlob(assets, globPattern, func(asset nexusapi.Asset) string {
		return getRelativePath(asset.Path, basePath)
	})
}

func downloadAsset(asset nexusapi.Asset, destDir string, basePath string, wg *sync.WaitGroup, errCh chan error, bar *progress.ProgressBarWithCount, tracker *output.TransferTracker, config *config.Config, opts *DownloadOptions) {
	defer wg.Done()
	// Use helper to get relative path, applying flatten logic if enabled
	resultPath := getRelativePath(asset.Path, "")
	if opts.Flatten && basePath != "" {
		resultPath = getRelativePath(asset.Path, basePath)
	}

	localPath := filepath.Join(destDir, resultPath)
	startTime := time.Now()

	// Check if file exists and validate checksum or skip based on file existence (skip this check if Force is enabled)
	shouldSkip := false

	if !opts.Force {
		if _, err := os.Stat(localPath); err == nil {
			if opts.SkipChecksum {
				// When checksum validation is skipped, only check if file exists and add to progress
				shouldSkip = true
				if bar != nil {
					bar.Add64(asset.FileSize)
				}
			} else if opts.checksumValidator != nil {
				// Use the new checksum.Validator for validation with progress tracking
				valid, err := opts.checksumValidator.ValidateWithProgress(localPath, asset.Checksum, bar)
				if err == nil && valid {
					shouldSkip = true
				}
			}
		}
	}

	if shouldSkip {
		relPath := getRelativePath(asset.Path, basePath)
		tracker.RecordFile(output.FileTransfer{
			Path:      relPath,
			Size:      asset.FileSize,
			Status:    output.TransferStatusSkipped,
			StartTime: startTime,
			EndTime:   time.Now(),
		})
		// Increment file count for skipped files
		if bar != nil {
			bar.IncrementFile()
		}
		return
	}

	// If dry-run is enabled, just log what would be downloaded (without creating directories)
	if opts.DryRun {
		relPath := getRelativePath(asset.Path, basePath)
		opts.Logger.VerbosePrintf("Would download: %s\n", relPath)
		tracker.RecordFile(output.FileTransfer{
			Path:      relPath,
			Size:      asset.FileSize,
			Status:    output.TransferStatusSuccess,
			StartTime: startTime,
			EndTime:   time.Now(),
		})
		if bar != nil {
			bar.Add64(asset.FileSize)
			bar.IncrementFile()
		}
		return
	}

	// Create directory structure for actual download
	os.MkdirAll(filepath.Dir(localPath), 0755)

	client := nexusapi.NewClient(config.NexusURL, config.Username, config.Password)
	f, err := os.Create(localPath)
	if err != nil {
		relPath := getRelativePath(asset.Path, basePath)
		tracker.RecordFile(output.FileTransfer{
			Path:      relPath,
			Size:      asset.FileSize,
			Status:    output.TransferStatusFailed,
			Error:     err,
			StartTime: startTime,
			EndTime:   time.Now(),
		})
		errCh <- err
		return
	}
	defer f.Close()

	// Use a tee reader to update progress bar while downloading
	writer := io.MultiWriter(f, bar)
	err = client.DownloadAsset(asset.DownloadURL, writer)
	endTime := time.Now()

	relPath := getRelativePath(asset.Path, basePath)

	if err != nil {
		tracker.RecordFile(output.FileTransfer{
			Path:      relPath,
			Size:      asset.FileSize,
			Status:    output.TransferStatusFailed,
			Error:     err,
			StartTime: startTime,
			EndTime:   endTime,
		})
		errCh <- err
	} else {
		tracker.RecordFile(output.FileTransfer{
			Path:      relPath,
			Size:      asset.FileSize,
			Status:    output.TransferStatusSuccess,
			StartTime: startTime,
			EndTime:   endTime,
		})
		// Only increment file count on successful download
		bar.IncrementFile()
	}
}

func downloadFolder(srcArg, destDir string, config *config.Config, opts *DownloadOptions) DownloadStatus {
	repository, src, ok := util.ParseRepositoryPath(srcArg)
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
	assets, err := listAssets(repository, src, config, opts.Recursive)
	if err != nil {
		opts.Logger.Println("Error listing assets:", err)
		return DownloadError
	}

	// Apply glob filtering if specified
	if opts.GlobPattern != "" {
		assets, err = filterAssetsByGlob(assets, src, opts.GlobPattern)
		if err != nil {
			opts.Logger.Println("Error filtering assets:", err)
			return DownloadError
		}
	}

	if len(assets) == 0 {
		opts.Logger.Printf("No assets found in folder '%s' in repository '%s'\n", src, repository)
		return DownloadNoAssetsFound
	}

	// Build a map of remote asset paths for delete-extra functionality
	remoteAssetPaths := make(map[string]bool)
	for _, asset := range assets {
		resultPath := getRelativePath(asset.Path, "")
		if opts.Flatten && src != "" {
			resultPath = getRelativePath(asset.Path, src)
		}
		remoteAssetPaths[filepath.Join(destDir, resultPath)] = true
	}

	// Calculate total bytes to download using fileSize from search API
	totalBytes := int64(0)
	for _, asset := range assets {
		totalBytes += asset.FileSize
	}

	target := repository
	if src != "" {
		target = path.Join(repository, src)
	}
	showProgress := util.IsATTY() && !opts.QuietMode && !opts.DryRun
	tracker := output.NewTransferTracker(output.TransferTypeDownload, target, opts.Logger, opts.QuietMode, opts.Logger.IsVerbose(), showProgress)
	tracker.PrintHeader(len(assets), totalBytes)

	bar := progress.NewProgressBarWithCount(totalBytes, "Processing files", len(assets), showProgress)

	var wg sync.WaitGroup
	errCh := make(chan error, len(assets))
	for _, asset := range assets {
		wg.Add(1)
		go func(asset nexusapi.Asset) {
			downloadAsset(asset, destDir, src, &wg, errCh, bar, tracker, config, opts)
		}(asset)
	}
	wg.Wait()
	close(errCh)

	nErrors := 0
	for err := range errCh {
		opts.Logger.Println("Error downloading asset:", err)
		nErrors++
	}

	bar.Finish()

	// Delete extra files if requested (but not in dry-run mode)
	var nDeleted int
	if opts.DeleteExtra && !opts.DryRun {
		nDeleted = deleteExtraFiles(destDir, remoteAssetPaths, opts)
	} else if opts.DeleteExtra && opts.DryRun {
		opts.Logger.Println("Dry-run mode: --delete flag ignored (no files would be deleted)")
	}

	if nDeleted > 0 {
		opts.Logger.VerbosePrintf("Deleted %d extra files\n", nDeleted)
	}

	tracker.PrintSummary()

	if nErrors == 0 {
		return DownloadSuccess
	}
	return DownloadError
}

// downloadFolderCompressed downloads and extracts a compressed archive
func downloadFolderCompressed(repository, src, destDir string, config *config.Config, opts *DownloadOptions) DownloadStatus {
	return downloadFolderCompressedWithArchiveName(repository, src, "", destDir, config, opts)
}

// downloadFolderCompressedWithArchiveName downloads and extracts a compressed archive with optional explicit name
func downloadFolderCompressedWithArchiveName(repository, src, explicitArchiveName, destDir string, config *config.Config, opts *DownloadOptions) DownloadStatus {
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
		opts.CompressionFormat = archive.DetectFromFilename(archiveName)
	}

	opts.Logger.VerbosePrintf("Looking for compressed archive: %s (format: %s)\n", archiveName, opts.CompressionFormat)

	// List assets to find the archive
	assets, err := listAssets(repository, src, config, opts.Recursive)
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

	// If dry-run is enabled, just report what would be downloaded
	if opts.DryRun {
		opts.Logger.Printf("Dry-run mode: Would download and extract archive '%s' from '%s' in repository '%s' to '%s'\n",
			archiveName, src, repository, destDir)
		return DownloadSuccess
	}

	showProgress := util.IsATTY() && !opts.QuietMode
	bar := progress.NewProgressBarWithCount(archiveAsset.FileSize, "Downloading archive", 1, showProgress)

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

func DownloadMain(src, dest string, config *config.Config, opts *DownloadOptions) {
	processedSrc, err := processKeyTemplateWrapper(src, opts.KeyFromFile)
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
