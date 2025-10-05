package nexus

import (
	"os"
	"testing"
)

func TestUsernamePasswordFromEnvironment(t *testing.T) {
	// Save original values
	originalUser := os.Getenv("NEXUS_USER")
	originalPass := os.Getenv("NEXUS_PASS")
	defer func() {
		os.Setenv("NEXUS_USER", originalUser)
		os.Setenv("NEXUS_PASS", originalPass)
	}()

	// Test with environment variables
	os.Setenv("NEXUS_USER", "testuser")
	os.Setenv("NEXUS_PASS", "testpass")

	user := getenv("NEXUS_USER", "admin")
	pass := getenv("NEXUS_PASS", "admin")

	if user != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", user)
	}
	if pass != "testpass" {
		t.Errorf("Expected password 'testpass', got '%s'", pass)
	}

	// Test with no environment variables
	os.Unsetenv("NEXUS_USER")
	os.Unsetenv("NEXUS_PASS")

	user = getenv("NEXUS_USER", "admin")
	pass = getenv("NEXUS_PASS", "admin")

	if user != "admin" {
		t.Errorf("Expected default username 'admin', got '%s'", user)
	}
	if pass != "admin" {
		t.Errorf("Expected default password 'admin', got '%s'", pass)
	}
}

func TestGetenvFunction(t *testing.T) {
	// Save original value
	originalVal := os.Getenv("TEST_VAR")
	defer os.Setenv("TEST_VAR", originalVal)

	// Test with set value
	os.Setenv("TEST_VAR", "value1")
	result := getenv("TEST_VAR", "fallback")
	if result != "value1" {
		t.Errorf("Expected 'value1', got '%s'", result)
	}

	// Test with empty value
	os.Setenv("TEST_VAR", "")
	result = getenv("TEST_VAR", "fallback")
	if result != "fallback" {
		t.Errorf("Expected 'fallback', got '%s'", result)
	}

	// Test with unset value
	os.Unsetenv("TEST_VAR")
	result = getenv("TEST_VAR", "fallback")
	if result != "fallback" {
		t.Errorf("Expected 'fallback', got '%s'", result)
	}
}

// TestNewProgressBar tests the newProgressBar function
func TestNewProgressBar(t *testing.T) {
	tests := []struct {
		name        string
		totalBytes  int64
		description string
		currentFile int
		totalFiles  int
		quietMode   bool
	}{
		{
			name:        "normal mode with upload",
			totalBytes:  1024,
			description: "Uploading files",
			currentFile: 1,
			totalFiles:  3,
			quietMode:   false,
		},
		{
			name:        "normal mode with download",
			totalBytes:  2048,
			description: "Downloading files",
			currentFile: 2,
			totalFiles:  5,
			quietMode:   false,
		},
		{
			name:        "quiet mode",
			totalBytes:  512,
			description: "Testing files",
			currentFile: 1,
			totalFiles:  1,
			quietMode:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := newProgressBar(tt.totalBytes, tt.description, tt.currentFile, tt.totalFiles, tt.quietMode)
			if bar == nil {
				t.Errorf("newProgressBar returned nil")
			}
		})
	}
}
