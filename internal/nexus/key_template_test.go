package nexus

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeKeyFromFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "key-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	content := "test content for hashing"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	hash1, err := computeKeyFromFile(testFile)
	if err != nil {
		t.Fatalf("Failed to compute key: %v", err)
	}

	if hash1 == "" {
		t.Error("Expected non-empty hash")
	}

	hash2, err := computeKeyFromFile(testFile)
	if err != nil {
		t.Fatalf("Failed to compute key second time: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Hash should be consistent: got %s and %s", hash1, hash2)
	}

	if len(hash1) != 64 {
		t.Errorf("Expected SHA256 hash length of 64, got %d", len(hash1))
	}
}

func TestReplaceKeyTemplate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		keyValue string
		expected string
	}{
		{
			name:     "single occurrence",
			input:    "repo/cache-{key}/data",
			keyValue: "abc123",
			expected: "repo/cache-abc123/data",
		},
		{
			name:     "multiple occurrences",
			input:    "repo/{key}/data/{key}",
			keyValue: "xyz789",
			expected: "repo/xyz789/data/xyz789",
		},
		{
			name:     "no template",
			input:    "repo/data",
			keyValue: "abc123",
			expected: "repo/data",
		},
		{
			name:     "at start",
			input:    "{key}/data",
			keyValue: "start",
			expected: "start/data",
		},
		{
			name:     "at end",
			input:    "repo/{key}",
			keyValue: "end",
			expected: "repo/end",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceKeyTemplate(tt.input, tt.keyValue)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestValidateKeyTemplate(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		keyFromFile string
		expectError bool
	}{
		{
			name:        "valid with key template",
			input:       "repo/cache-{key}/data",
			keyFromFile: "/path/to/file",
			expectError: false,
		},
		{
			name:        "invalid without key template",
			input:       "repo/cache/data",
			keyFromFile: "/path/to/file",
			expectError: true,
		},
		{
			name:        "valid when no key-from specified",
			input:       "repo/cache/data",
			keyFromFile: "",
			expectError: false,
		},
		{
			name:        "valid when no key-from and has template",
			input:       "repo/cache-{key}/data",
			keyFromFile: "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateKeyTemplate(tt.input, tt.keyFromFile)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestProcessKeyTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "key-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "package-lock.json")
	content := `{"version": "1.0.0"}`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	expectedHash, err := computeKeyFromFile(testFile)
	if err != nil {
		t.Fatalf("Failed to compute expected hash: %v", err)
	}

	tests := []struct {
		name        string
		input       string
		keyFromFile string
		expected    string
		expectError bool
	}{
		{
			name:        "with key template",
			input:       "repo/node_modules-{key}.tar.gz",
			keyFromFile: testFile,
			expected:    "repo/node_modules-" + expectedHash + ".tar.gz",
			expectError: false,
		},
		{
			name:        "without key-from",
			input:       "repo/data/file.txt",
			keyFromFile: "",
			expected:    "repo/data/file.txt",
			expectError: false,
		},
		{
			name:        "missing template with key-from",
			input:       "repo/data/file.txt",
			keyFromFile: testFile,
			expected:    "",
			expectError: true,
		},
		{
			name:        "nonexistent file",
			input:       "repo/{key}/data",
			keyFromFile: "/nonexistent/file",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processKeyTemplate(tt.input, tt.keyFromFile)
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if !tt.expectError && result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
