package nexus

import (
	"fmt"
	"strings"
)

func computeKeyFromFile(filePath string) (string, error) {
	return computeChecksum(filePath, "sha256")
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

func processKeyTemplate(input string, keyFromFile string) (string, error) {
	if keyFromFile == "" {
		return input, nil
	}

	if err := validateKeyTemplate(input, keyFromFile); err != nil {
		return "", err
	}

	keyValue, err := computeKeyFromFile(keyFromFile)
	if err != nil {
		return "", fmt.Errorf("failed to compute key from file %s: %w", keyFromFile, err)
	}

	return replaceKeyTemplate(input, keyValue), nil
}
