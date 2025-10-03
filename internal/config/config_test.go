package config

import (
	"os"
	"testing"
)

func TestNew(t *testing.T) {
	// Save original values
	originalUser := os.Getenv("NEXUS_USER")
	originalPass := os.Getenv("NEXUS_PASS")
	originalURL := os.Getenv("NEXUS_URL")
	defer func() {
		os.Setenv("NEXUS_USER", originalUser)
		os.Setenv("NEXUS_PASS", originalPass)
		os.Setenv("NEXUS_URL", originalURL)
	}()

	// Test with environment variables
	os.Setenv("NEXUS_USER", "testuser")
	os.Setenv("NEXUS_PASS", "testpass")
	os.Setenv("NEXUS_URL", "http://test:8081")

	cfg := New()

	if cfg.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", cfg.Username)
	}
	if cfg.Password != "testpass" {
		t.Errorf("Expected password 'testpass', got '%s'", cfg.Password)
	}
	if cfg.NexusURL != "http://test:8081" {
		t.Errorf("Expected URL 'http://test:8081', got '%s'", cfg.NexusURL)
	}

	// Test with no environment variables (defaults)
	os.Unsetenv("NEXUS_USER")
	os.Unsetenv("NEXUS_PASS")
	os.Unsetenv("NEXUS_URL")

	cfg = New()

	if cfg.Username != "admin" {
		t.Errorf("Expected default username 'admin', got '%s'", cfg.Username)
	}
	if cfg.Password != "admin" {
		t.Errorf("Expected default password 'admin', got '%s'", cfg.Password)
	}
	if cfg.NexusURL != "http://localhost:8081" {
		t.Errorf("Expected default URL 'http://localhost:8081', got '%s'", cfg.NexusURL)
	}
}

func TestGetenv(t *testing.T) {
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
