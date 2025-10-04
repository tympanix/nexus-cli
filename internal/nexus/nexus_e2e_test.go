package nexus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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

	// Start Nexus container
	containerID, err := startNexusContainer()
	if err != nil {
		t.Fatalf("Failed to start Nexus container: %v", err)
	}
	defer cleanupContainer(containerID)

	// Wait for Nexus to be ready
	nexusURL := "http://localhost:8081"
	if !waitForNexus(nexusURL, 5*time.Minute) {
		t.Fatal("Nexus did not become ready in time")
	}

	// Get admin password from container
	adminPassword, err := getAdminPassword(containerID)
	if err != nil {
		t.Fatalf("Failed to get admin password: %v", err)
	}

	config := &Config{
		NexusURL: nexusURL,
		Username: "admin",
		Password: adminPassword,
	}

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
		Logger:    NewLogger(os.Stdout),
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
		Logger:            NewLogger(os.Stdout),
		QuietMode:         false,
	}

	success := downloadFolder(uploadPath, downloadDir, config, downloadOpts)
	if !success {
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
func createRawRepository(config *Config, repoName string) error {
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
