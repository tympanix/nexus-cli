package nexus

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

// UploadOptions holds options for upload operations
type UploadOptions struct {
	ChecksumAlgorithm string
	SkipChecksum      bool
	Logger            Logger
	QuietMode         bool
	Compress          bool              // Enable compression (tar.gz, tar.zst, or zip)
	CompressionFormat CompressionFormat // Compression format to use (gzip, zstd, or zip)
	GlobPattern       string            // Optional glob pattern(s) to filter files (comma-separated, supports negation with !)
	KeyFromFile       string            // Path to file to compute hash from for {key} template
	checksumValidator ChecksumValidator // Internal validator instance
}

// SetChecksumAlgorithm validates and sets the checksum algorithm
// Returns an error if the algorithm is not supported
func (opts *UploadOptions) SetChecksumAlgorithm(algorithm string) error {
	validator, err := NewChecksumValidator(algorithm)
	if err != nil {
		return err
	}
	opts.ChecksumAlgorithm = validator.Algorithm()
	opts.checksumValidator = validator
	return nil
}

func collectFiles(src string) ([]string, error) {
	return collectFilesWithGlob(src, "")
}

func collectFilesWithGlob(src string, globPattern string) ([]string, error) {
	var files []string

	// Parse glob patterns - split by comma and separate positive from negative patterns
	var positivePatterns []string
	var negativePatterns []string

	if globPattern != "" {
		patterns := strings.Split(globPattern, ",")
		for _, pattern := range patterns {
			pattern = strings.TrimSpace(pattern)
			if pattern == "" {
				continue
			}
			if strings.HasPrefix(pattern, "!") {
				// Negative pattern - remove the ! prefix
				negativePatterns = append(negativePatterns, strings.TrimPrefix(pattern, "!"))
			} else {
				// Positive pattern
				positivePatterns = append(positivePatterns, pattern)
			}
		}
	}

	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}
			// Normalize to forward slashes for consistent matching
			relPath = filepath.ToSlash(relPath)

			// If we have glob patterns, filter files
			if len(positivePatterns) > 0 || len(negativePatterns) > 0 {
				// Check positive patterns first (at least one must match if any exist)
				matchesPositive := len(positivePatterns) == 0 // If no positive patterns, default to true
				for _, pattern := range positivePatterns {
					matched, err := doublestar.Match(pattern, relPath)
					if err != nil {
						return fmt.Errorf("invalid glob pattern '%s': %w", pattern, err)
					}
					if matched {
						matchesPositive = true
						break
					}
				}

				// If no positive match, skip this file
				if !matchesPositive {
					return nil
				}

				// Check negative patterns (none should match)
				for _, pattern := range negativePatterns {
					matched, err := doublestar.Match(pattern, relPath)
					if err != nil {
						return fmt.Errorf("invalid glob pattern '%s': %w", pattern, err)
					}
					if matched {
						// File matches a negative pattern, exclude it
						return nil
					}
				}
			}

			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func uploadFiles(src, repository, subdir string, config *Config, opts *UploadOptions) error {
	// If compression is enabled, use compressed upload
	if opts.Compress {
		return uploadFilesCompressed(src, repository, subdir, config, opts)
	}

	// Original uncompressed upload logic
	filePaths, err := collectFilesWithGlob(src, opts.GlobPattern)
	if err != nil {
		return err
	}

	// Build a map of remote assets if checksum validation is enabled or skip-checksum is enabled
	var remoteAssets map[string]nexusapi.Asset
	if opts.SkipChecksum || opts.checksumValidator != nil {
		basePath := subdir
		if basePath == "" {
			basePath = ""
		}
		assets, err := listAssets(repository, basePath, config)
		if err != nil {
			opts.Logger.VerbosePrintf("Could not list existing assets (will upload all files): %v\n", err)
			remoteAssets = make(map[string]nexusapi.Asset)
		} else {
			remoteAssets = make(map[string]nexusapi.Asset)
			for _, asset := range assets {
				path := strings.TrimLeft(asset.Path, "/")
				if basePath != "" {
					normalizedBasePath := strings.TrimLeft(basePath, "/")
					if strings.HasPrefix(path, normalizedBasePath+"/") {
						path = strings.TrimPrefix(path, normalizedBasePath+"/")
					}
				}
				remoteAssets[path] = asset
			}
		}
	}

	// Filter files based on checksum validation
	var filesToUpload []string
	var filesToUploadSizes []int64
	var skippedCount int
	totalBytes := int64(0)

	for _, filePath := range filePaths {
		relPath, _ := filepath.Rel(src, filePath)
		relPath = filepath.ToSlash(relPath)
		info, err := os.Stat(filePath)
		if err != nil {
			return err
		}

		shouldSkip := false
		skipReason := ""

		// Check if file exists remotely and validate checksum
		if remoteAssets != nil {
			if asset, exists := remoteAssets[relPath]; exists {
				if opts.SkipChecksum {
					shouldSkip = true
					skipReason = "Skipped (file exists): %s\n"
				} else if opts.checksumValidator != nil {
					valid, err := opts.checksumValidator.Validate(filePath, asset.Checksum)
					if err == nil && valid {
						shouldSkip = true
						skipReason = fmt.Sprintf("Skipped (%s match): %%s\n", strings.ToUpper(opts.ChecksumAlgorithm))
					}
				}
			}
		}

		if shouldSkip {
			opts.Logger.VerbosePrintf(skipReason, filePath)
			skippedCount++
		} else {
			filesToUpload = append(filesToUpload, filePath)
			filesToUploadSizes = append(filesToUploadSizes, info.Size())
			totalBytes += info.Size()
		}
	}

	if len(filesToUpload) == 0 {
		opts.Logger.Printf("All %d files already exist with matching checksums\n", len(filePaths))
		return nil
	}

	bar := newProgressBar(totalBytes, "Uploading files", 0, len(filePaths), opts.QuietMode)

	// Prepare file upload information
	files := make([]nexusapi.FileUpload, len(filesToUpload))
	for i, filePath := range filesToUpload {
		relPath, _ := filepath.Rel(src, filePath)
		relPath = filepath.ToSlash(relPath)
		files[i] = nexusapi.FileUpload{
			FilePath:     filePath,
			RelativePath: relPath,
		}
	}

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	// Write multipart form in a goroutine
	errChan := make(chan error, 1)
	go func() {
		defer pw.Close()
		// Callback to update progress bar description when each file completes
		onFileComplete := func(idx, total int) {
			bar.Describe(fmt.Sprintf("[cyan][%d/%d][reset] Uploading files", idx+1, total))
		}
		err := nexusapi.BuildRawUploadForm(writer, files, subdir, bar, nil, onFileComplete)
		writer.Close()
		errChan <- err
	}()

	client := nexusapi.NewClient(config.NexusURL, config.Username, config.Password)
	contentType := nexusapi.GetFormDataContentType(writer)

	err = client.UploadComponent(repository, pr, contentType)
	if goroutineErr := <-errChan; goroutineErr != nil {
		return goroutineErr
	}
	if err != nil {
		opts.Logger.Printf("Failed to upload files: %v\n", err)
		return err
	}
	// Update description one final time to ensure correct count is shown
	bar.Describe(fmt.Sprintf("[cyan][%d/%d][reset] Uploading files", len(filesToUpload), len(filesToUpload)))
	bar.Finish()
	if isatty() && !opts.QuietMode {
		fmt.Println()
	}
	if skippedCount > 0 {
		opts.Logger.Printf("Uploaded %d files from %s (skipped: %d)\n", len(filesToUpload), src, skippedCount)
	} else {
		opts.Logger.Printf("Uploaded %d files from %s\n", len(filesToUpload), src)
	}
	return nil
}

// uploadFilesCompressed creates a tar.gz archive and uploads it as a single file
func uploadFilesCompressed(src, repository, subdir string, config *Config, opts *UploadOptions) error {
	return uploadFilesCompressedWithArchiveName(src, repository, subdir, "", config, opts)
}

// uploadFilesCompressedWithArchiveName creates a compressed archive and uploads it as a single file with optional explicit name
func uploadFilesCompressedWithArchiveName(src, repository, subdir, explicitArchiveName string, config *Config, opts *UploadOptions) error {
	filePaths, err := collectFilesWithGlob(src, opts.GlobPattern)
	if err != nil {
		return err
	}

	if len(filePaths) == 0 {
		return fmt.Errorf("no files to upload in %s", src)
	}

	// Require explicit archive name
	if explicitArchiveName == "" {
		ext := opts.CompressionFormat.Extension()
		return fmt.Errorf("when using --compress, you must specify the %s filename in the destination path (e.g., repo/path/archive%s)", ext, ext)
	}

	archiveName := explicitArchiveName
	opts.Logger.VerbosePrintf("Creating compressed archive: %s (format: %s)\n", archiveName, opts.CompressionFormat)

	// Calculate total uncompressed size for progress bar
	totalBytes := int64(0)
	for _, filePath := range filePaths {
		info, err := os.Stat(filePath)
		if err != nil {
			return err
		}
		totalBytes += info.Size()
	}

	// Create progress bar using uncompressed size as approximation
	bar := newProgressBar(totalBytes, "Uploading compressed archive", 0, 1, opts.QuietMode)

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	// Create archive and upload in a goroutine
	errChan := make(chan error, 1)
	go func() {
		defer pw.Close()

		// Create form file for the archive
		part, err := writer.CreateFormFile("raw.asset1", archiveName)
		if err != nil {
			errChan <- err
			return
		}

		// Wrap part with capping writer and progress bar
		// Use io.MultiWriter to send bytes to both the form part and progress bar
		cappedBar := newCappingWriter(bar, totalBytes)
		progressWriter := io.MultiWriter(part, cappedBar)

		// Create compressed archive with progress tracking
		if err := opts.CompressionFormat.CreateArchiveWithGlob(src, progressWriter, opts.GlobPattern); err != nil {
			errChan <- fmt.Errorf("failed to create archive: %w", err)
			return
		}

		// Set the filename field - archive goes to subdir if specified
		if subdir != "" {
			_ = writer.WriteField("raw.asset1.filename", archiveName)
			_ = writer.WriteField("raw.directory", subdir)
		} else {
			_ = writer.WriteField("raw.asset1.filename", archiveName)
		}

		writer.Close()
		errChan <- nil
	}()

	client := nexusapi.NewClient(config.NexusURL, config.Username, config.Password)
	contentType := nexusapi.GetFormDataContentType(writer)

	err = client.UploadComponent(repository, pr, contentType)
	if goroutineErr := <-errChan; goroutineErr != nil {
		return goroutineErr
	}
	if err != nil {
		opts.Logger.Printf("Failed to upload archive: %v\n", err)
		return err
	}
	bar.Finish()
	if isatty() && !opts.QuietMode {
		fmt.Println()
	}
	opts.Logger.Printf("Uploaded compressed archive containing %d files from %s\n", len(filePaths), src)
	return nil
}

func UploadMain(src, dest string, config *Config, opts *UploadOptions) {
	processedDest, err := processKeyTemplate(dest, opts.KeyFromFile)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	if opts.KeyFromFile != "" {
		opts.Logger.Printf("Using key template: %s -> %s\n", dest, processedDest)
	}

	repository := processedDest
	subdir := ""
	explicitArchiveName := ""

	if strings.Contains(processedDest, "/") {
		parts := strings.SplitN(processedDest, "/", 2)
		repository = parts[0]
		subdir = parts[1]

		// If compress is enabled and dest ends with .tar.gz or .tar.zst or .zip, treat it as explicit archive name
		if opts.Compress && (strings.HasSuffix(subdir, ".tar.gz") || strings.HasSuffix(subdir, ".tar.zst") || strings.HasSuffix(subdir, ".zip")) {
			// Extract the archive name from the path
			lastSlash := strings.LastIndex(subdir, "/")
			if lastSlash >= 0 {
				explicitArchiveName = subdir[lastSlash+1:]
				subdir = subdir[:lastSlash]
			} else {
				// The entire subdir is the archive name
				explicitArchiveName = subdir
				subdir = ""
			}
			// Detect compression format from filename if not explicitly set
			if explicitArchiveName != "" && opts.CompressionFormat == "" {
				opts.CompressionFormat = DetectCompressionFromFilename(explicitArchiveName)
			}
		}
	} else if opts.Compress && (strings.HasSuffix(processedDest, ".tar.gz") || strings.HasSuffix(processedDest, ".tar.zst") || strings.HasSuffix(processedDest, ".zip")) {
		// Repository name ends with .tar.gz or .tar.zst or .zip, treat it as explicit archive name
		explicitArchiveName = processedDest
	} else if opts.Compress && (strings.HasSuffix(dest, ".tar.gz") || strings.HasSuffix(dest, ".tar.zst") || strings.HasSuffix(dest, ".zip")) {
		// Repository name ends with .tar.gz or .tar.zst or .zip, treat it as explicit archive name
		explicitArchiveName = dest
		repository = ""
		subdir = ""
		// Detect compression format from filename if not explicitly set
		if opts.CompressionFormat == "" {
			opts.CompressionFormat = DetectCompressionFromFilename(explicitArchiveName)
		}
	}

	// Default compression format if not set
	if opts.Compress && opts.CompressionFormat == "" {
		opts.CompressionFormat = CompressionGzip
	}

	err = uploadFilesWithArchiveName(src, repository, subdir, explicitArchiveName, config, opts)
	if err != nil {
		fmt.Println("Upload error:", err)
		os.Exit(1)
	}
}

func uploadFilesWithArchiveName(src, repository, subdir, explicitArchiveName string, config *Config, opts *UploadOptions) error {
	// If compression is enabled, use compressed upload
	if opts.Compress {
		return uploadFilesCompressedWithArchiveName(src, repository, subdir, explicitArchiveName, config, opts)
	}

	return uploadFiles(src, repository, subdir, config, opts)
}
