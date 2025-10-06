package nexus

import (
	"bytes"
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

// TestCappingWriter tests the cappingWriter functionality
func TestCappingWriter(t *testing.T) {
	tests := []struct {
		name            string
		maxBytes        int64
		writes          []int
		expectedWritten int64
		expectedBuf     int
	}{
		{
			name:            "writes under max",
			maxBytes:        100,
			writes:          []int{30, 40, 20},
			expectedWritten: 90,
			expectedBuf:     90,
		},
		{
			name:            "writes exactly at max",
			maxBytes:        100,
			writes:          []int{50, 50},
			expectedWritten: 100,
			expectedBuf:     100,
		},
		{
			name:            "writes exceed max",
			maxBytes:        100,
			writes:          []int{50, 60},
			expectedWritten: 100,
			expectedBuf:     100,
		},
		{
			name:            "writes far exceed max",
			maxBytes:        50,
			writes:          []int{30, 40, 50},
			expectedWritten: 50,
			expectedBuf:     50,
		},
		{
			name:            "single write exceeds max",
			maxBytes:        20,
			writes:          []int{100},
			expectedWritten: 20,
			expectedBuf:     20,
		},
		{
			name:            "continue writing after reaching max",
			maxBytes:        30,
			writes:          []int{20, 20, 20},
			expectedWritten: 30,
			expectedBuf:     30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			cw := newCappingWriter(&buf, tt.maxBytes)

			for _, size := range tt.writes {
				data := make([]byte, size)
				n, err := cw.Write(data)
				if err != nil {
					t.Fatalf("Write failed: %v", err)
				}
				if n != size {
					t.Errorf("Write returned %d, expected %d", n, size)
				}
			}

			if cw.bytesWritten != tt.expectedWritten {
				t.Errorf("bytesWritten = %d, expected %d", cw.bytesWritten, tt.expectedWritten)
			}

			if int64(buf.Len()) != int64(tt.expectedBuf) {
				t.Errorf("buffer length = %d, expected %d", buf.Len(), tt.expectedBuf)
			}
		})
	}
}

// TestCappingWriterWithProgressBar tests capping writer with actual progress bar
func TestCappingWriterWithProgressBar(t *testing.T) {
	// Create a progress bar with max 100 bytes
	bar := newProgressBar(100, "Testing", 1, 1, true) // quiet mode to avoid output

	// Wrap it with capping writer
	cw := newCappingWriter(bar, 100)

	// Write 50 bytes
	cw.Write(make([]byte, 50))
	state := bar.State()
	if state.CurrentNum != 50 {
		t.Errorf("After 50 bytes: current=%d, expected 50", state.CurrentNum)
	}

	// Write 60 more bytes (would be 110 total, but should cap at 100)
	cw.Write(make([]byte, 60))
	state = bar.State()
	if state.CurrentNum != 100 {
		t.Errorf("After 110 bytes written: current=%d, expected 100 (capped)", state.CurrentNum)
	}

	// Write more bytes - should still be capped at 100
	cw.Write(make([]byte, 50))
	state = bar.State()
	if state.CurrentNum != 100 {
		t.Errorf("After 160 bytes written: current=%d, expected 100 (capped)", state.CurrentNum)
	}

	bar.Finish()
}
