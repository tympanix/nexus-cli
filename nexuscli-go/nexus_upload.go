package main

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"
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

	// Create progress bar - write to /dev/null when disabled
	showProgress := isatty() && !opts.QuietMode
	progressWriter := os.Stdout
	if !showProgress {
		progressWriter, _ = os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	}
	bar := progressbar.NewOptions64(totalBytes,
		progressbar.OptionSetWriter(progressWriter),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetDescription("Uploading bytes"),
	)

	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	// Write multipart form in a goroutine
	errChan := make(chan error, 1)
	go func() {
		defer pw.Close()
		for idx, filePath := range filePaths {
			relPath, _ := filepath.Rel(src, filePath)
			relPath = filepath.ToSlash(relPath)
			f, err := os.Open(filePath)
			if err != nil {
				errChan <- err
				return
			}
			defer f.Close()
			part, err := writer.CreateFormFile(fmt.Sprintf("raw.asset%d", idx+1), filepath.Base(filePath))
			if err != nil {
				errChan <- err
				return
			}
			reader := io.TeeReader(f, bar)
			if _, err := io.Copy(part, reader); err != nil {
				errChan <- err
				return
			}
			_ = writer.WriteField(fmt.Sprintf("raw.asset%d.filename", idx+1), relPath)
		}
		if subdir != "" {
			_ = writer.WriteField("raw.directory", subdir)
		}
		writer.Close()
		errChan <- nil
	}()

	uploadEndpoint := fmt.Sprintf("%s/service/rest/v1/components?repository=%s", config.NexusURL, repository)
	req, err := http.NewRequest("POST", uploadEndpoint, pr)
	if err != nil {
		return err
	}
	req.SetBasicAuth(config.Username, config.Password)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if goroutineErr := <-errChan; goroutineErr != nil {
		return goroutineErr
	}
	if resp.StatusCode == 204 {
		opts.Logger.Printf("Uploaded %d files from %s\n", len(filePaths), src)
	} else {
		respBody, _ := io.ReadAll(resp.Body)
		opts.Logger.Printf("Failed to upload files: %d %s\n", resp.StatusCode, string(respBody))
	}
	return nil
}

func uploadMain(src, dest string, config *Config, opts *UploadOptions) {
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
