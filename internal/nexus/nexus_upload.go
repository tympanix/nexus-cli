package nexus

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"
)

// UploadOptions holds options for upload operations
type UploadOptions struct {
	Logger    Logger
	QuietMode bool
	Compress  bool
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

// createTarGz creates a tar.gz archive from the files in src directory
// Returns a reader for the archive and the total size
func createTarGz(src string, filePaths []string, bar *progressbar.ProgressBar) (io.Reader, error) {
	pr, pw := io.Pipe()
	
	go func() {
		defer pw.Close()
		
		gzw := gzip.NewWriter(pw)
		defer gzw.Close()
		
		tw := tar.NewWriter(gzw)
		defer tw.Close()
		
		for _, filePath := range filePaths {
			info, err := os.Stat(filePath)
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			
			relPath, _ := filepath.Rel(src, filePath)
			relPath = filepath.ToSlash(relPath)
			
			header := &tar.Header{
				Name:    relPath,
				Size:    info.Size(),
				Mode:    int64(info.Mode()),
				ModTime: info.ModTime(),
			}
			
			if err := tw.WriteHeader(header); err != nil {
				pw.CloseWithError(err)
				return
			}
			
			file, err := os.Open(filePath)
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			
			var reader io.Reader = file
			if bar != nil {
				reader = io.TeeReader(file, bar)
			}
			
			if _, err := io.Copy(tw, reader); err != nil {
				file.Close()
				pw.CloseWithError(err)
				return
			}
			file.Close()
		}
	}()
	
	return pr, nil
}

// uploadCompressed uploads files as a single tar.gz archive
func uploadCompressed(src, repository, subdir string, filePaths []string, config *Config, opts *UploadOptions, bar *progressbar.ProgressBar) error {
	// Determine archive name based on source directory
	archiveName := filepath.Base(src) + ".tar.gz"
	
	// Create tar.gz reader
	archiveReader, err := createTarGz(src, filePaths, bar)
	if err != nil {
		return err
	}
	
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	
	errChan := make(chan error, 1)
	go func() {
		defer pw.Close()
		
		part, err := writer.CreateFormFile("raw.asset1", archiveName)
		if err != nil {
			errChan <- err
			return
		}
		
		if _, err := io.Copy(part, archiveReader); err != nil {
			errChan <- err
			return
		}
		
		filename := archiveName
		if subdir != "" {
			filename = filepath.ToSlash(filepath.Join(subdir, archiveName))
		}
		_ = writer.WriteField("raw.asset1.filename", filename)
		
		writer.Close()
		errChan <- nil
	}()
	
	baseURL, err := url.Parse(config.NexusURL)
	if err != nil {
		return fmt.Errorf("invalid Nexus URL: %w", err)
	}
	baseURL.Path = "/service/rest/v1/components"
	query := baseURL.Query()
	query.Set("repository", repository)
	baseURL.RawQuery = query.Encode()
	
	req, err := http.NewRequest("POST", baseURL.String(), pr)
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
		bar.Finish()
		opts.Logger.Printf("Uploaded %d files from %s as compressed archive %s\n", len(filePaths), src, archiveName)
		return nil
	}
	
	respBody, _ := io.ReadAll(resp.Body)
	opts.Logger.Printf("Failed to upload compressed archive: %d %s\n", resp.StatusCode, string(respBody))
	return fmt.Errorf("upload failed with status %d", resp.StatusCode)
}


func uploadFiles(src, repository, subdir string, config *Config, opts *UploadOptions) error {
	filePaths, err := collectFiles(src)
	if err != nil {
		return err
	}
	
	if len(filePaths) == 0 {
		return fmt.Errorf("no files found in %s", src)
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

	// Create progress bar - write to io.Discard when disabled
	showProgress := isatty() && !opts.QuietMode
	progressWriter := io.Writer(os.Stdout)
	if !showProgress {
		progressWriter = io.Discard
	}
	bar := progressbar.NewOptions64(totalBytes,
		progressbar.OptionSetWriter(progressWriter),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetDescription("Uploading bytes"),
		progressbar.OptionFullWidth(),
	)

	if opts.Compress {
		return uploadCompressed(src, repository, subdir, filePaths, config, opts, bar)
	}

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

	baseURL, err := url.Parse(config.NexusURL)
	if err != nil {
		return fmt.Errorf("invalid Nexus URL: %w", err)
	}
	baseURL.Path = "/service/rest/v1/components"
	query := baseURL.Query()
	query.Set("repository", repository)
	baseURL.RawQuery = query.Encode()

	req, err := http.NewRequest("POST", baseURL.String(), pr)
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
		bar.Finish()
		opts.Logger.Printf("Uploaded %d files from %s\n", len(filePaths), src)
		return nil
	}
	respBody, _ := io.ReadAll(resp.Body)
	opts.Logger.Printf("Failed to upload files: %d %s\n", resp.StatusCode, string(respBody))
	return fmt.Errorf("upload failed with status %d", resp.StatusCode)
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
