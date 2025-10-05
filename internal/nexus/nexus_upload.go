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
	Logger            Logger
	QuietMode         bool
	Compress          bool              // Enable compression (tar.gz or tar.zst)
	CompressionFormat CompressionFormat // Compression format to use (gzip or zstd)
	GlobPattern       string            // Optional glob pattern to filter files
}

func collectFiles(src string) ([]string, error) {
	return collectFilesWithGlob(src, "")
}

func collectFilesWithGlob(src string, globPattern string) ([]string, error) {
	var files []string
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			// If glob pattern is provided, filter files
			if globPattern != "" {
				relPath, err := filepath.Rel(src, path)
				if err != nil {
					return err
				}
				// Normalize to forward slashes for consistent matching
				relPath = filepath.ToSlash(relPath)

				// Match against the glob pattern
				matched, err := doublestar.Match(globPattern, relPath)
				if err != nil {
					return fmt.Errorf("invalid glob pattern '%s': %w", globPattern, err)
				}
				if !matched {
					return nil
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
	// Calculate total bytes to upload
	totalBytes := int64(0)
	fileSizes := make([]int64, len(filePaths))
	for i, filePath := range filePaths {
		info, err := os.Stat(filePath)
		if err != nil {
			return err
		}
		fileSizes[i] = info.Size()
		totalBytes += info.Size()
	}

	bar := newProgressBar(totalBytes, "Uploading bytes", opts.QuietMode)

	// Prepare file upload information
	files := make([]nexusapi.FileUpload, len(filePaths))
	for i, filePath := range filePaths {
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
		err := nexusapi.BuildRawUploadForm(writer, files, subdir, bar)
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
	bar.Finish()
	opts.Logger.Printf("Uploaded %d files from %s\n", len(filePaths), src)
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

	// Calculate total bytes for progress
	totalBytes := int64(0)
	for _, filePath := range filePaths {
		info, err := os.Stat(filePath)
		if err != nil {
			return err
		}
		totalBytes += info.Size()
	}

	// Require explicit archive name
	if explicitArchiveName == "" {
		ext := opts.CompressionFormat.Extension()
		return fmt.Errorf("when using --compress, you must specify the %s filename in the destination path (e.g., repo/path/archive%s)", ext, ext)
	}

	archiveName := explicitArchiveName
	opts.Logger.VerbosePrintf("Creating compressed archive: %s (format: %s)\n", archiveName, opts.CompressionFormat)

	bar := newProgressBar(totalBytes, "Compressing bytes", opts.QuietMode)

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

		// Create compressed archive with progress tracking
		progressWriter := io.MultiWriter(part, bar)
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
	opts.Logger.Printf("Uploaded compressed archive containing %d files from %s\n", len(filePaths), src)
	return nil
}

func UploadMain(src, dest string, config *Config, opts *UploadOptions) {
	repository := dest
	subdir := ""
	explicitArchiveName := ""

	if strings.Contains(dest, "/") {
		parts := strings.SplitN(dest, "/", 2)
		repository = parts[0]
		subdir = parts[1]

		// If compress is enabled and dest ends with .tar.gz or .tar.zst, treat it as explicit archive name
		if opts.Compress && (strings.HasSuffix(subdir, ".tar.gz") || strings.HasSuffix(subdir, ".tar.zst")) {
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
	} else if opts.Compress && (strings.HasSuffix(dest, ".tar.gz") || strings.HasSuffix(dest, ".tar.zst")) {
		// Repository name ends with .tar.gz or .tar.zst, treat it as explicit archive name
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

	err := uploadFilesWithArchiveName(src, repository, subdir, explicitArchiveName, config, opts)
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
