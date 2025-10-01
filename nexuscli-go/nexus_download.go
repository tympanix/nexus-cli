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

	"github.com/schollz/progressbar/v3"
)

type Asset struct {
	DownloadURL string `json:"downloadUrl"`
	Path        string `json:"path"`
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

func downloadAsset(asset Asset, destDir string, wg *sync.WaitGroup, errCh chan error) {
	defer wg.Done()
	path := strings.TrimLeft(asset.Path, "/")
	localPath := filepath.Join(destDir, path)
	os.MkdirAll(filepath.Dir(localPath), 0755)
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
	_, err = io.Copy(f, res.Body)
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
	// Calculate total bytes to download
	totalBytes := int64(0)
	assetSizes := make([]int64, len(assets))
	for i, asset := range assets {
		resp, err := http.NewRequest("HEAD", asset.DownloadURL, nil)
		if err != nil {
			fmt.Println("Error preparing HEAD request:", err)
			return false
		}
		resp.SetBasicAuth(username, password)
		res, err := http.DefaultClient.Do(resp)
		if err != nil {
			fmt.Println("Error getting asset size:", err)
			return false
		}
		if res.StatusCode != 200 {
			fmt.Printf("Failed to get size for %s: %d\n", asset.Path, res.StatusCode)
			res.Body.Close()
			return false
		}
		size := res.ContentLength
		assetSizes[i] = size
		totalBytes += size
		res.Body.Close()
	}
	bar := progressbar.DefaultBytes(totalBytes, "Downloading bytes")

	var wg sync.WaitGroup
	errCh := make(chan error, len(assets))
	for i, asset := range assets {
		wg.Add(1)
		go func(asset Asset, size int64) {
			downloadAssetWithBar(asset, destDir, &wg, errCh, bar)
		}(asset, assetSizes[i])
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

func downloadAssetWithBar(asset Asset, destDir string, wg *sync.WaitGroup, errCh chan error, bar *progressbar.ProgressBar) {
	defer wg.Done()
	path := strings.TrimLeft(asset.Path, "/")
	localPath := filepath.Join(destDir, path)
	os.MkdirAll(filepath.Dir(localPath), 0755)
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
	reader := io.TeeReader(res.Body, bar)
	_, err = io.Copy(f, reader)
	if err != nil {
		errCh <- err
	}
}

func downloadMain(src, dest string) {
	success := downloadFolder(src, dest)
	if !success {
		os.Exit(1)
	}
}
