package operations

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/tympanix/nexus-cli/internal/archive"
	"github.com/tympanix/nexus-cli/internal/config"
	"github.com/tympanix/nexus-cli/internal/nexusapi"
	"github.com/tympanix/nexus-cli/internal/progress"
	"github.com/tympanix/nexus-cli/internal/util"
)

func collectFiles(src string) ([]string, error) {
	return archive.CollectFilesWithGlob(src, "")
}

func uploadAptPackage(debFile, repository string, config *config.Config, opts *UploadOptions) error {
	info, err := os.Stat(debFile)
	if err != nil {
		return err
	}

	totalBytes := info.Size()
	bar := progress.NewProgressBar(totalBytes, "Uploading apt package", 0, 1, opts.QuietMode)

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	errChan := make(chan error, 1)
	go func() {
		defer pw.Close()
		err := nexusapi.BuildAptUploadForm(writer, debFile, bar)
		writer.Close()
		errChan <- err
	}()

	client := nexusapi.NewClient(config.NexusURL, config.Username, config.Password)
	contentType := nexusapi.GetFormDataContentType(writer)

	err = client.UploadComponent(repository, pr, contentType)
	if err != nil {
		return err
	}
	if goroutineErr := <-errChan; goroutineErr != nil {
		return goroutineErr
	}
	bar.Finish()
	if util.IsATTY() && !opts.QuietMode {
		fmt.Println()
	}
	opts.Logger.Printf("Uploaded apt package %s\n", filepath.Base(debFile))
	return nil
}

func uploadYumPackage(rpmFile, repository string, config *config.Config, opts *UploadOptions) error {
	info, err := os.Stat(rpmFile)
	if err != nil {
		return err
	}

	totalBytes := info.Size()
	bar := progress.NewProgressBar(totalBytes, "Uploading yum package", 0, 1, opts.QuietMode)

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	errChan := make(chan error, 1)
	go func() {
		defer pw.Close()
		err := nexusapi.BuildYumUploadForm(writer, rpmFile, bar)
		writer.Close()
		errChan <- err
	}()

	client := nexusapi.NewClient(config.NexusURL, config.Username, config.Password)
	contentType := nexusapi.GetFormDataContentType(writer)

	err = client.UploadComponent(repository, pr, contentType)
	if err != nil {
		return err
	}
	if goroutineErr := <-errChan; goroutineErr != nil {
		return goroutineErr
	}
	bar.Finish()
	if util.IsATTY() && !opts.QuietMode {
		fmt.Println()
	}
	opts.Logger.Printf("Uploaded yum package %s\n", filepath.Base(rpmFile))
	return nil
}

func uploadFiles(src, repository, subdir string, config *config.Config, opts *UploadOptions) error {
	// If compression is enabled, use compressed upload
	if opts.Compress {
		return uploadFilesCompressed(src, repository, subdir, config, opts)
	}

	// Original uncompressed upload logic
	filePaths, err := archive.CollectFilesWithGlob(src, opts.GlobPattern)
	if err != nil {
		return err
	}

	// Build a map of remote assets if checksum validation is enabled or skip-checksum is enabled
	// Skip this step if Force is enabled (always upload all files)
	var remoteAssets map[string]nexusapi.Asset
	if !opts.Force && (opts.SkipChecksum || opts.checksumValidator != nil) {
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
	totalBytesToUpload := int64(0)

	// Calculate total bytes for progress bar (validation + upload)
	totalBytes := int64(0)
	for _, filePath := range filePaths {
		info, err := os.Stat(filePath)
		if err != nil {
			return err
		}
		totalBytes += info.Size()
	}

	// Create a single progress bar for all operations
	bar := progress.NewProgressBar(totalBytes, "Processing files", 0, len(filePaths), opts.QuietMode)
	currentFile := 0

	for _, filePath := range filePaths {
		relPath, _ := filepath.Rel(src, filePath)
		relPath = filepath.ToSlash(relPath)
		info, err := os.Stat(filePath)
		if err != nil {
			return err
		}

		shouldSkip := false
		skipReason := ""

		// Check if file exists remotely and validate checksum (skip this check if Force is enabled)
		if !opts.Force && remoteAssets != nil {
			if asset, exists := remoteAssets[relPath]; exists {
				if opts.SkipChecksum {
					// For skip-checksum, just check existence and add file size to progress
					shouldSkip = true
					skipReason = "Skipped (file exists): %s\n"
					bar.Add64(info.Size())
				} else if opts.checksumValidator != nil {
					// Validate checksum with progress tracking
					valid, err := opts.checksumValidator.ValidateWithProgress(filePath, asset.Checksum, bar)
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
			currentFile++
			bar.Describe(fmt.Sprintf("[cyan][%d/%d][reset] Processing files", currentFile, len(filePaths)))
		} else {
			filesToUpload = append(filesToUpload, filePath)
			filesToUploadSizes = append(filesToUploadSizes, info.Size())
			totalBytesToUpload += info.Size()
		}
	}

	if len(filesToUpload) == 0 {
		bar.Finish()
		if util.IsATTY() && !opts.QuietMode {
			fmt.Println()
		}
		opts.Logger.Printf("All %d files already exist with matching checksums\n", len(filePaths))
		return nil
	}

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
	// Capture currentFile for use in goroutine
	fileCounter := currentFile
	go func() {
		defer pw.Close()
		// Callback to update progress bar description when each file completes
		onFileComplete := func(idx, total int) {
			fileCounter++
			bar.Describe(fmt.Sprintf("[cyan][%d/%d][reset] Processing files", fileCounter, len(filePaths)))
		}
		err := nexusapi.BuildRawUploadForm(writer, files, subdir, bar, nil, onFileComplete)
		writer.Close()
		errChan <- err
	}()

	client := nexusapi.NewClient(config.NexusURL, config.Username, config.Password)
	contentType := nexusapi.GetFormDataContentType(writer)

	err = client.UploadComponent(repository, pr, contentType)
	if err != nil {
		return err
	}
	if goroutineErr := <-errChan; goroutineErr != nil {
		return goroutineErr
	}
	bar.Finish()
	if util.IsATTY() && !opts.QuietMode {
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
func uploadFilesCompressed(src, repository, subdir string, config *config.Config, opts *UploadOptions) error {
	return uploadFilesCompressedWithArchiveName(src, repository, subdir, "", config, opts)
}

// uploadFilesCompressedWithArchiveName creates a compressed archive and uploads it as a single file with optional explicit name
func uploadFilesCompressedWithArchiveName(src, repository, subdir, explicitArchiveName string, config *config.Config, opts *UploadOptions) error {
	filePaths, err := archive.CollectFilesWithGlob(src, opts.GlobPattern)
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
	bar := progress.NewProgressBar(totalBytes, "Uploading compressed archive", 0, 1, opts.QuietMode)

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
		cappedBar := progress.NewCappingWriter(bar, totalBytes)
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
	if err != nil {
		return err
	}
	if goroutineErr := <-errChan; goroutineErr != nil {
		return goroutineErr
	}
	bar.Finish()
	if util.IsATTY() && !opts.QuietMode {
		fmt.Println()
	}
	opts.Logger.Printf("Uploaded compressed archive containing %d files from %s\n", len(filePaths), src)
	return nil
}

func UploadMain(srcs []string, dest string, config *config.Config, opts *UploadOptions) {
	processedDest, err := processKeyTemplateWrapper(dest, opts.KeyFromFile)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	if opts.KeyFromFile != "" {
		opts.Logger.Printf("Using key template: %s -> %s\n", dest, processedDest)
	}

	// For special package types (.deb, .rpm), only allow single source
	if len(srcs) == 1 {
		src := srcs[0]
		// Check if src is a single .deb file for APT package upload
		if info, err := os.Stat(src); err == nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(src), ".deb") {
			// APT package upload - repository is the destination
			repository := processedDest
			if strings.Contains(processedDest, "/") {
				fmt.Println("Error: APT package upload does not support subdirectories. Use only repository name as destination.")
				os.Exit(1)
			}
			if opts.Compress {
				fmt.Println("Error: APT package upload does not support compression.")
				os.Exit(1)
			}
			err := uploadAptPackage(src, repository, config, opts)
			if err != nil {
				fmt.Println("Upload error:", err)
				os.Exit(1)
			}
			return
		}

		// Check if src is a single .rpm file for YUM package upload
		if info, err := os.Stat(src); err == nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(src), ".rpm") {
			// YUM package upload - repository is the destination
			repository := processedDest
			if strings.Contains(processedDest, "/") {
				fmt.Println("Error: YUM package upload does not support subdirectories. Use only repository name as destination.")
				os.Exit(1)
			}
			if opts.Compress {
				fmt.Println("Error: YUM package upload does not support compression.")
				os.Exit(1)
			}
			err := uploadYumPackage(src, repository, config, opts)
			if err != nil {
				fmt.Println("Upload error:", err)
				os.Exit(1)
			}
			return
		}
	}

	repository := processedDest
	subdir := ""
	explicitArchiveName := ""

	if strings.Contains(processedDest, "/") {
		var ok bool
		repository, subdir, ok = util.ParseRepositoryPath(processedDest)
		if !ok {
			fmt.Println("Error: The dest argument must be in the form 'repository' or 'repository/folder'.")
			os.Exit(1)
		}

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
				opts.CompressionFormat = archive.DetectFromFilename(explicitArchiveName)
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
			opts.CompressionFormat = archive.DetectFromFilename(explicitArchiveName)
		}
	}

	// Default compression format if not set
	if opts.Compress && opts.CompressionFormat == "" {
		opts.CompressionFormat = archive.FormatGzip
	}

	// Upload each source
	for _, src := range srcs {
		err = uploadFilesWithArchiveName(src, repository, subdir, explicitArchiveName, config, opts)
		if err != nil {
			fmt.Println("Upload error:", err)
			os.Exit(1)
		}
	}
}

func uploadFilesWithArchiveName(src, repository, subdir, explicitArchiveName string, config *config.Config, opts *UploadOptions) error {
	// If compression is enabled, use compressed upload
	if opts.Compress {
		return uploadFilesCompressedWithArchiveName(src, repository, subdir, explicitArchiveName, config, opts)
	}

	return uploadFiles(src, repository, subdir, config, opts)
}
