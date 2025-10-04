package nexus

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

// UploadOptions holds options for upload operations
type UploadOptions struct {
	Logger    Logger
	QuietMode bool
}

func collectFiles(src string) ([]string, error) {
	var files []string
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func uploadFiles(src, repository, subdir string, config *Config, opts *UploadOptions) error {
	filePaths, err := collectFiles(src)
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

func UploadMain(src, dest string, config *Config, opts *UploadOptions) {
	repository := dest
	subdir := ""
	if strings.Contains(dest, "/") {
		parts := strings.SplitN(dest, "/", 2)
		repository = parts[0]
		subdir = parts[1]
	}
	err := uploadFiles(src, repository, subdir, config, opts)
	if err != nil {
		fmt.Println("Upload error:", err)
		os.Exit(1)
	}
}
