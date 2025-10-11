package operations

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/blakesmith/ar"
	"github.com/tympanix/nexus-cli/internal/archive"
	"github.com/tympanix/nexus-cli/internal/config"
	"github.com/tympanix/nexus-cli/internal/util"
)

var (
	e2eContainerID string
	e2eConfig      *config.Config
	e2eNexusURL    = "http://localhost:8081"
)

func TestMain(m *testing.M) {
	// Parse flags first
	flag.Parse()

	// Check if running in short mode or if Docker is unavailable
	if testing.Short() || !isDockerAvailable() {
		// Skip setup and run tests (they will skip individually)
		os.Exit(m.Run())
	}

	// Start Nexus container once for all tests
	containerID, err := startNexusContainer()
	if err != nil {
		fmt.Printf("Failed to start Nexus container: %v\n", err)
		os.Exit(1)
	}
	e2eContainerID = containerID

	// Wait for Nexus to be ready
	if !waitForNexus(e2eNexusURL, 5*time.Minute) {
		cleanupContainer(e2eContainerID)
		fmt.Println("Nexus did not become ready in time")
		os.Exit(1)
	}

	// Get admin password from container
	adminPassword, err := getAdminPassword(e2eContainerID)
	if err != nil {
		cleanupContainer(e2eContainerID)
		fmt.Printf("Failed to get admin password: %v\n", err)
		os.Exit(1)
	}

	e2eConfig = &config.Config{
		NexusURL: e2eNexusURL,
		Username: "admin",
		Password: adminPassword,
	}

	// Run tests
	exitCode := m.Run()

	// Cleanup
	cleanupContainer(e2eContainerID)

	os.Exit(exitCode)
}

// TestEndToEndUploadDownload tests the complete workflow of uploading and downloading files using a real Nexus instance
func TestEndToEndUploadDownload(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Check if Docker is available
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping end-to-end test")
	}

	// Use shared Nexus instance
	config := e2eConfig

	// Create a RAW repository
	repoName := "test-repo"
	if err := createRawRepository(config, repoName); err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create test files
	testDir, err := os.MkdirTemp("", "test-upload-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create some test files with known content
	testFiles := map[string]string{
		"file1.txt":        "Hello from file 1",
		"file2.txt":        "Content of file 2",
		"subdir/file3.txt": "Nested file content",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testDir, filename)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Upload files using the CLI
	uploadOpts := &UploadOptions{
		Logger:    util.NewLogger(os.Stdout),
		QuietMode: false,
	}

	uploadPath := repoName + "/test-folder"
	err = uploadFiles(testDir, repoName, "test-folder", config, uploadOpts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Give Nexus a moment to process the upload
	time.Sleep(2 * time.Second)

	// Create download directory
	downloadDir, err := os.MkdirTemp("", "test-download-*")
	if err != nil {
		t.Fatalf("Failed to create download directory: %v", err)
	}
	defer os.RemoveAll(downloadDir)

	// Download files using the CLI
	downloadOpts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            util.NewLogger(os.Stdout),
		QuietMode:         false,
	}

	status := downloadFolder(uploadPath, downloadDir, config, downloadOpts)
	if status != DownloadSuccess {
		t.Fatal("Download failed")
	}

	// Validate downloaded files match original content
	for filename, expectedContent := range testFiles {
		downloadedPath := filepath.Join(downloadDir, "/test-folder", filename)
		content, err := os.ReadFile(downloadedPath)
		if err != nil {
			t.Errorf("Failed to read downloaded file %s: %v", filename, err)
			continue
		}

		if string(content) != expectedContent {
			t.Errorf("Content mismatch for %s: expected %q, got %q", filename, expectedContent, string(content))
		}
	}
}

// isDockerAvailable checks if Docker is available on the system
func isDockerAvailable() bool {
	cmd := exec.Command("docker", "version")
	return cmd.Run() == nil
}

// startNexusContainer starts a Nexus Docker container and returns its ID
func startNexusContainer() (string, error) {
	// Check if container already exists and remove it
	checkCmd := exec.Command("docker", "ps", "-a", "-q", "-f", "name=nexus-test")
	output, _ := checkCmd.Output()
	if len(output) > 0 {
		stopCmd := exec.Command("docker", "rm", "-f", "nexus-test")
		stopCmd.Run()
	}

	// Start new container
	cmd := exec.Command("docker", "run", "-d", "-p", "8081:8081", "--name", "nexus-test", "sonatype/nexus3:3.75.1")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	containerID := strings.TrimSpace(string(output))
	fmt.Printf("Started Nexus container: %s\n", containerID)
	return containerID, nil
}

// waitForNexus waits for Nexus to become ready
func waitForNexus(nexusURL string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 5 * time.Second}

	fmt.Println("Waiting for Nexus to be ready...")

	for time.Now().Before(deadline) {
		resp, err := client.Get(nexusURL + "/service/rest/v1/status")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			fmt.Println("Nexus is ready!")
			return true
		}
		if resp != nil {
			resp.Body.Close()
		}

		fmt.Print(".")
		time.Sleep(5 * time.Second)
	}

	fmt.Println()
	return false
}

// getAdminPassword retrieves the admin password from the Nexus container
func getAdminPassword(containerID string) (string, error) {
	// Wait a bit for the password file to be created
	time.Sleep(10 * time.Second)

	// Try to read the admin password file from the container
	cmd := exec.Command("docker", "exec", containerID, "cat", "/nexus-data/admin.password")
	output, err := cmd.Output()
	if err != nil {
		// If the file doesn't exist, the default password might be "admin123"
		// or the password might have already been changed
		return "admin123", nil
	}

	password := strings.TrimSpace(string(output))
	fmt.Printf("Retrieved admin password from container\n")
	return password, nil
}

// createRawRepository creates a RAW repository in Nexus
func createRawRepository(config *config.Config, repoName string) error {
	// Wait a bit to ensure Nexus is fully initialized
	time.Sleep(5 * time.Second)

	// Create repository configuration
	repoConfig := map[string]interface{}{
		"name":   repoName,
		"online": true,
		"storage": map[string]interface{}{
			"blobStoreName":               "default",
			"strictContentTypeValidation": false,
			"writePolicy":                 "ALLOW",
		},
	}

	jsonData, err := json.Marshal(repoConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal repository config: %w", err)
	}

	// Create the repository via API
	req, err := http.NewRequest("POST", config.NexusURL+"/service/rest/v1/repositories/raw/hosted", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(config.Username, config.Password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create repository: status %d, body: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Created repository: %s\n", repoName)
	return nil
}

// generateGPGKey generates a GPG key pair for APT repository signing
func generateGPGKey() (privateKey string, err error) {
	// Create a temporary GPG home directory
	gnupgHome, err := os.MkdirTemp("", "gnupg-*")
	if err != nil {
		return "", fmt.Errorf("failed to create GPG home directory: %w", err)
	}
	defer os.RemoveAll(gnupgHome)

	// Generate GPG key batch configuration
	batchConfig := `%no-protection
Key-Type: RSA
Key-Length: 2048
Name-Real: Test Nexus Repository
Name-Email: test@nexus.local
Expire-Date: 0
`

	// Write batch config to a temporary file
	batchFile := filepath.Join(gnupgHome, "batch.txt")
	if err := os.WriteFile(batchFile, []byte(batchConfig), 0600); err != nil {
		return "", fmt.Errorf("failed to write GPG batch config: %w", err)
	}

	// Generate the key
	cmd := exec.Command("gpg", "--homedir", gnupgHome, "--batch", "--gen-key", batchFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to generate GPG key: %w, output: %s", err, string(output))
	}

	// List keys to get the key ID
	cmd = exec.Command("gpg", "--homedir", gnupgHome, "--list-keys", "--with-colons")
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list GPG keys: %w", err)
	}

	// Parse the key ID from the output
	var keyID string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "pub:") {
			fields := strings.Split(line, ":")
			if len(fields) > 4 {
				keyID = fields[4]
				break
			}
		}
	}

	if keyID == "" {
		return "", fmt.Errorf("failed to find generated key ID")
	}

	// Export the private key
	cmd = exec.Command("gpg", "--homedir", gnupgHome, "--armor", "--export-secret-keys", keyID)
	privateKeyBytes, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to export private key: %w", err)
	}

	return string(privateKeyBytes), nil
}

// createAptRepository creates an APT repository in Nexus
func createAptRepository(config *config.Config, repoName string) error {
	// Wait a bit to ensure Nexus is fully initialized
	time.Sleep(5 * time.Second)

	// Generate a GPG key for APT signing
	privateKey, err := generateGPGKey()
	if err != nil {
		return fmt.Errorf("failed to generate GPG key: %w", err)
	}

	// Create repository configuration for APT hosted repository
	repoConfig := map[string]interface{}{
		"name":   repoName,
		"online": true,
		"storage": map[string]interface{}{
			"blobStoreName":               "default",
			"strictContentTypeValidation": true,
			"writePolicy":                 "ALLOW",
		},
		"apt": map[string]interface{}{
			"distribution": "bionic",
		},
		"aptSigning": map[string]interface{}{
			"keypair":    privateKey,
			"passphrase": "",
		},
	}

	jsonData, err := json.Marshal(repoConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal repository config: %w", err)
	}

	// Create the repository via API
	req, err := http.NewRequest("POST", config.NexusURL+"/service/rest/v1/repositories/apt/hosted", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(config.Username, config.Password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create repository: status %d, body: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Created APT repository: %s\n", repoName)
	return nil
}

// cleanupContainer stops and removes the Nexus container
func cleanupContainer(containerID string) {
	if containerID == "" {
		return
	}

	fmt.Printf("Cleaning up container: %s\n", containerID)

	// Use context with timeout for cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop the container
	stopCmd := exec.CommandContext(ctx, "docker", "stop", containerID)
	stopCmd.Run()

	// Remove the container
	rmCmd := exec.Command("docker", "rm", containerID)
	rmCmd.Run()
}

// TestEndToEndUploadDownloadZstd tests the complete workflow of uploading and downloading files with zstd compression using a real Nexus instance
func TestEndToEndUploadDownloadZstd(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Check if Docker is available
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping end-to-end test")
	}

	// Use shared Nexus instance
	config := e2eConfig

	// Create a RAW repository
	repoName := "test-repo-zstd"
	if err := createRawRepository(config, repoName); err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create test files
	testDir, err := os.MkdirTemp("", "test-upload-zstd-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create some test files with known content
	testFiles := map[string]string{
		"file1.txt":        "Hello from file 1",
		"file2.txt":        "Content of file 2",
		"subdir/file3.txt": "Nested file content",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testDir, filename)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Upload files using zstd compression
	archiveName := "test-archive.tar.zst"
	uploadOpts := &UploadOptions{
		Logger:            util.NewLogger(os.Stdout),
		QuietMode:         false,
		Compress:          true,
		CompressionFormat: archive.FormatZstd,
	}

	// Upload with explicit archive name
	err = uploadFilesWithArchiveName(testDir, repoName, "test-folder", archiveName, config, uploadOpts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Give Nexus a moment to process the upload
	time.Sleep(2 * time.Second)

	// Create download directory
	downloadDir, err := os.MkdirTemp("", "test-download-zstd-*")
	if err != nil {
		t.Fatalf("Failed to create download directory: %v", err)
	}
	defer os.RemoveAll(downloadDir)

	// Download archive using the CLI
	downloadOpts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            util.NewLogger(os.Stdout),
		QuietMode:         false,
		Compress:          true,
		CompressionFormat: archive.FormatZstd,
	}

	status := downloadFolderCompressedWithArchiveName(repoName, "test-folder", archiveName, downloadDir, config, downloadOpts)
	if status != DownloadSuccess {
		t.Fatal("Download failed")
	}

	// Validate downloaded files match original content
	for filename, expectedContent := range testFiles {
		downloadedPath := filepath.Join(downloadDir, filename)
		content, err := os.ReadFile(downloadedPath)
		if err != nil {
			t.Errorf("Failed to read downloaded file %s: %v", filename, err)
			continue
		}

		if string(content) != expectedContent {
			t.Errorf("Content mismatch for %s: expected %q, got %q", filename, expectedContent, string(content))
		}
	}
}

// TestEndToEndUploadDownloadGzip tests the complete workflow of uploading and downloading files with gzip compression using a real Nexus instance
func TestEndToEndUploadDownloadGzip(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Check if Docker is available
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping end-to-end test")
	}

	// Use shared Nexus instance
	config := e2eConfig

	// Create a RAW repository
	repoName := "test-repo-gzip"
	if err := createRawRepository(config, repoName); err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create test files
	testDir, err := os.MkdirTemp("", "test-upload-gzip-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create some test files with known content
	testFiles := map[string]string{
		"file1.txt":        "Hello from file 1",
		"file2.txt":        "Content of file 2",
		"subdir/file3.txt": "Nested file content",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testDir, filename)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Upload files using gzip compression
	archiveName := "test-archive.tar.gz"
	uploadOpts := &UploadOptions{
		Logger:            util.NewLogger(os.Stdout),
		QuietMode:         false,
		Compress:          true,
		CompressionFormat: archive.FormatGzip,
	}

	// Upload with explicit archive name
	err = uploadFilesWithArchiveName(testDir, repoName, "test-folder", archiveName, config, uploadOpts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Give Nexus a moment to process the upload
	time.Sleep(2 * time.Second)

	// Create download directory
	downloadDir, err := os.MkdirTemp("", "test-download-gzip-*")
	if err != nil {
		t.Fatalf("Failed to create download directory: %v", err)
	}
	defer os.RemoveAll(downloadDir)

	// Download archive using the CLI
	downloadOpts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            util.NewLogger(os.Stdout),
		QuietMode:         false,
		Compress:          true,
		CompressionFormat: archive.FormatGzip,
	}

	status := downloadFolderCompressedWithArchiveName(repoName, "test-folder", archiveName, downloadDir, config, downloadOpts)
	if status != DownloadSuccess {
		t.Fatal("Download failed")
	}

	// Validate downloaded files match original content
	for filename, expectedContent := range testFiles {
		downloadedPath := filepath.Join(downloadDir, filename)
		content, err := os.ReadFile(downloadedPath)
		if err != nil {
			t.Errorf("Failed to read downloaded file %s: %v", filename, err)
			continue
		}

		if string(content) != expectedContent {
			t.Errorf("Content mismatch for %s: expected %q, got %q", filename, expectedContent, string(content))
		}
	}
}

// TestEndToEndUploadDownloadZip tests the complete workflow of uploading and downloading files with zip compression using a real Nexus instance
func TestEndToEndUploadDownloadZip(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Check if Docker is available
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping end-to-end test")
	}

	// Use shared Nexus instance
	config := e2eConfig

	// Create a RAW repository
	repoName := "test-repo-zip"
	if err := createRawRepository(config, repoName); err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create test files
	testDir, err := os.MkdirTemp("", "test-upload-zip-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create some test files with known content
	testFiles := map[string]string{
		"file1.txt":        "Hello from file 1",
		"file2.txt":        "Content of file 2",
		"subdir/file3.txt": "Nested file content",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(testDir, filename)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Upload files using zip compression
	archiveName := "test-archive.zip"
	uploadOpts := &UploadOptions{
		Logger:            util.NewLogger(os.Stdout),
		QuietMode:         false,
		Compress:          true,
		CompressionFormat: archive.FormatZip,
	}

	// Upload with explicit archive name
	err = uploadFilesWithArchiveName(testDir, repoName, "test-folder", archiveName, config, uploadOpts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Give Nexus a moment to process the upload
	time.Sleep(2 * time.Second)

	// Create download directory
	downloadDir, err := os.MkdirTemp("", "test-download-zip-*")
	if err != nil {
		t.Fatalf("Failed to create download directory: %v", err)
	}
	defer os.RemoveAll(downloadDir)

	// Download archive using the CLI
	downloadOpts := &DownloadOptions{
		ChecksumAlgorithm: "sha1",
		SkipChecksum:      false,
		Logger:            util.NewLogger(os.Stdout),
		QuietMode:         false,
		Compress:          true,
		CompressionFormat: archive.FormatZip,
	}

	status := downloadFolderCompressedWithArchiveName(repoName, "test-folder", archiveName, downloadDir, config, downloadOpts)
	if status != DownloadSuccess {
		t.Fatal("Download failed")
	}

	// Validate downloaded files match original content
	for filename, expectedContent := range testFiles {
		downloadedPath := filepath.Join(downloadDir, filename)
		content, err := os.ReadFile(downloadedPath)
		if err != nil {
			t.Errorf("Failed to read downloaded file %s: %v", filename, err)
			continue
		}

		if string(content) != expectedContent {
			t.Errorf("Content mismatch for %s: expected %q, got %q", filename, expectedContent, string(content))
		}
	}
}

// createDebPackage creates a simple .deb package for testing
func createDebPackage(outputPath, packageName, version, arch string) error {
	// Create temporary directory for package structure
	tmpDir, err := os.MkdirTemp("", "deb-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create DEBIAN directory
	debianDir := filepath.Join(tmpDir, "DEBIAN")
	if err := os.MkdirAll(debianDir, 0755); err != nil {
		return fmt.Errorf("failed to create DEBIAN dir: %w", err)
	}

	// Create control file
	controlContent := fmt.Sprintf(`Package: %s
Version: %s
Architecture: %s
Maintainer: Test <test@example.com>
Description: Test package for nexus-cli e2e testing
 This is a simple test package created for e2e testing purposes.
`, packageName, version, arch)

	controlFile := filepath.Join(debianDir, "control")
	if err := os.WriteFile(controlFile, []byte(controlContent), 0644); err != nil {
		return fmt.Errorf("failed to create control file: %w", err)
	}

	// Create some dummy content
	usrDir := filepath.Join(tmpDir, "usr", "share", "doc", packageName)
	if err := os.MkdirAll(usrDir, 0755); err != nil {
		return fmt.Errorf("failed to create usr dir: %w", err)
	}

	readmeContent := fmt.Sprintf("This is a test package: %s version %s\n", packageName, version)
	readmeFile := filepath.Join(usrDir, "README")
	if err := os.WriteFile(readmeFile, []byte(readmeContent), 0644); err != nil {
		return fmt.Errorf("failed to create README: %w", err)
	}

	// Build the .deb package using native Go implementation
	if err := buildDebPackage(tmpDir, outputPath); err != nil {
		return fmt.Errorf("failed to build deb package: %w", err)
	}

	return nil
}

// buildDebPackage creates a .deb package from a directory structure using native Go
func buildDebPackage(sourceDir, outputPath string) error {
	// Create the output file
	debFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer debFile.Close()

	// Create ar archive writer
	arWriter := ar.NewWriter(debFile)

	// 1. Add debian-binary file
	debianBinary := []byte("2.0\n")
	if err := arWriter.WriteGlobalHeader(); err != nil {
		return fmt.Errorf("failed to write ar global header: %w", err)
	}
	hdr := &ar.Header{
		Name:    "debian-binary",
		Size:    int64(len(debianBinary)),
		Mode:    0644,
		ModTime: time.Now(),
	}
	if err := arWriter.WriteHeader(hdr); err != nil {
		return fmt.Errorf("failed to write debian-binary header: %w", err)
	}
	if _, err := arWriter.Write(debianBinary); err != nil {
		return fmt.Errorf("failed to write debian-binary content: %w", err)
	}

	// 2. Create and add control.tar.gz
	controlTarGz := new(bytes.Buffer)
	if err := createControlTarGz(filepath.Join(sourceDir, "DEBIAN"), controlTarGz); err != nil {
		return fmt.Errorf("failed to create control.tar.gz: %w", err)
	}
	hdr = &ar.Header{
		Name:    "control.tar.gz",
		Size:    int64(controlTarGz.Len()),
		Mode:    0644,
		ModTime: time.Now(),
	}
	if err := arWriter.WriteHeader(hdr); err != nil {
		return fmt.Errorf("failed to write control.tar.gz header: %w", err)
	}
	if _, err := arWriter.Write(controlTarGz.Bytes()); err != nil {
		return fmt.Errorf("failed to write control.tar.gz content: %w", err)
	}

	// 3. Create and add data.tar.gz
	dataTarGz := new(bytes.Buffer)
	if err := createDataTarGz(sourceDir, dataTarGz); err != nil {
		return fmt.Errorf("failed to create data.tar.gz: %w", err)
	}
	hdr = &ar.Header{
		Name:    "data.tar.gz",
		Size:    int64(dataTarGz.Len()),
		Mode:    0644,
		ModTime: time.Now(),
	}
	if err := arWriter.WriteHeader(hdr); err != nil {
		return fmt.Errorf("failed to write data.tar.gz header: %w", err)
	}
	if _, err := arWriter.Write(dataTarGz.Bytes()); err != nil {
		return fmt.Errorf("failed to write data.tar.gz content: %w", err)
	}

	return nil
}

// createControlTarGz creates a tar.gz archive containing the DEBIAN control files
func createControlTarGz(debianDir string, writer io.Writer) error {
	gzWriter := gzip.NewWriter(writer)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return filepath.Walk(debianDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(debianDir, path)
		if err != nil {
			return err
		}

		header := &tar.Header{
			Name:    "./" + relPath,
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tarWriter, file)
		return err
	})
}

// createDataTarGz creates a tar.gz archive containing the package data files
func createDataTarGz(sourceDir string, writer io.Writer) error {
	gzWriter := gzip.NewWriter(writer)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Skip the DEBIAN directory
		if strings.HasPrefix(relPath, "DEBIAN") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = "./" + relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// TestEndToEndAptPackageUpload tests the complete workflow of uploading a .deb file to an APT repository using a real Nexus instance
func TestEndToEndAptPackageUpload(t *testing.T) {
	// Skip if running in short mode
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	// Check if Docker is available
	if !isDockerAvailable() {
		t.Skip("Docker is not available, skipping end-to-end test")
	}

	// Use shared Nexus instance
	config := e2eConfig

	// Create an APT repository
	repoName := "test-apt-repo"
	if err := createAptRepository(config, repoName); err != nil {
		t.Fatalf("Failed to create APT repository: %v", err)
	}

	// Create test directory
	testDir, err := os.MkdirTemp("", "test-apt-upload-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// Create a real .deb package
	debFile := filepath.Join(testDir, "test-package_1.0.0_amd64.deb")
	err = createDebPackage(debFile, "test-package", "1.0.0", "amd64")
	if err != nil {
		t.Fatalf("Failed to create deb package: %v", err)
	}

	// Verify the .deb file was created
	if _, err := os.Stat(debFile); os.IsNotExist(err) {
		t.Fatalf("Deb file was not created: %s", debFile)
	}

	// Upload the .deb package using the CLI
	uploadOpts := &UploadOptions{
		Logger:    util.NewLogger(os.Stdout),
		QuietMode: false,
	}

	err = uploadAptPackage(debFile, repoName, config, uploadOpts)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Give Nexus a moment to process the upload
	time.Sleep(2 * time.Second)

	// Verify the package was uploaded by querying Nexus API
	// Note: APT packages are stored differently than raw files, so we verify via search
	baseURL, err := url.Parse(config.NexusURL)
	if err != nil {
		t.Fatalf("Invalid Nexus URL: %v", err)
	}
	baseURL.Path = "/service/rest/v1/search/assets"
	query := baseURL.Query()
	query.Set("repository", repoName)
	query.Set("name", "test-package")
	baseURL.RawQuery = query.Encode()

	req, err := http.NewRequest("GET", baseURL.String(), nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.SetBasicAuth(config.Username, config.Password)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to query assets: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to query assets: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Decode the response
	var searchResp struct {
		Items []struct {
			Path       string `json:"path"`
			Repository string `json:"repository"`
			Format     string `json:"format"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify that the package was found
	if len(searchResp.Items) == 0 {
		t.Fatal("Uploaded package not found in repository")
	}

	found := false
	for _, item := range searchResp.Items {
		if item.Repository == repoName && item.Format == "apt" {
			found = true
			t.Logf("Found uploaded package: %s in repository %s", item.Path, item.Repository)
			break
		}
	}

	if !found {
		t.Errorf("Expected to find package in APT repository %s, but it was not found", repoName)
	}

	t.Log("Successfully uploaded and verified .deb package to APT repository")
}
