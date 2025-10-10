package util

import (
	"fmt"
	"os"
	"strings"
)

// IsATTY checks if stdout is a terminal
func IsATTY() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// ParseRepositoryPath splits a repository path (e.g., "repository/folder" or "repository/folder/")
// into repository name and path, normalizing trailing slashes.
// Returns repository, path, and whether the parse was successful.
func ParseRepositoryPath(repoPath string) (repository string, path string, ok bool) {
	parts := strings.SplitN(repoPath, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	repository = parts[0]
	path = strings.TrimRight(parts[1], "/")
	return repository, path, true
}

func computeKeyFromFile(filePath string, checksumFunc func(string, string) (string, error)) (string, error) {
	return checksumFunc(filePath, "sha256")
}

func replaceKeyTemplate(input string, keyValue string) string {
	return strings.ReplaceAll(input, "{key}", keyValue)
}

func validateKeyTemplate(input string, keyFromFile string) error {
	if keyFromFile != "" && !strings.Contains(input, "{key}") {
		return fmt.Errorf("when --key-from is specified, the path must contain the {key} template placeholder")
	}
	return nil
}

// ProcessKeyTemplate processes key templates in the input string
// checksumFunc is a function that computes checksums (typically from the checksum package)
func ProcessKeyTemplate(input string, keyFromFile string, checksumFunc func(string, string) (string, error)) (string, error) {
	if keyFromFile == "" {
		return input, nil
	}

	if err := validateKeyTemplate(input, keyFromFile); err != nil {
		return "", err
	}

	keyValue, err := computeKeyFromFile(keyFromFile, checksumFunc)
	if err != nil {
		return "", fmt.Errorf("failed to compute key from file %s: %w", keyFromFile, err)
	}

	return replaceKeyTemplate(input, keyValue), nil
}
