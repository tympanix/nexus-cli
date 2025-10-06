package nexus

import (
	"bytes"
	"fmt"
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

// TestProgressBarCompletion tests that progress bar shows 100% and [n/n] when complete
func TestProgressBarCompletion(t *testing.T) {
	tests := []struct {
		name        string
		totalBytes  int64
		totalFiles  int
		description string
	}{
		{
			name:        "single file upload",
			totalBytes:  1024,
			totalFiles:  1,
			description: "Uploading files",
		},
		{
			name:        "multiple files upload",
			totalBytes:  5120,
			totalFiles:  5,
			description: "Uploading files",
		},
		{
			name:        "single file download",
			totalBytes:  2048,
			totalFiles:  1,
			description: "Downloading files",
		},
		{
			name:        "multiple files download",
			totalBytes:  10240,
			totalFiles:  10,
			description: "Downloading files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create progress bar in quiet mode to avoid output
			bar := newProgressBar(tt.totalBytes, tt.description, 0, tt.totalFiles, true)

			// Simulate writing all the bytes
			written := int64(0)
			chunkSize := int64(256) // Write in chunks of 256 bytes
			for written < tt.totalBytes {
				toWrite := chunkSize
				if written+toWrite > tt.totalBytes {
					toWrite = tt.totalBytes - written
				}
				bar.Write(make([]byte, toWrite))
				written += toWrite
			}

			// Simulate updating file count by updating description
			for i := 1; i <= tt.totalFiles; i++ {
				bar.Describe(fmt.Sprintf("[cyan][%d/%d][reset] %s", i, tt.totalFiles, tt.description))
			}

			// Finish the progress bar
			bar.Finish()

			// Check the state
			state := bar.State()

			// Verify completion - CurrentNum should equal Max (which means 100%)
			if state.CurrentNum != state.Max {
				t.Errorf("Expected CurrentNum to equal Max (%d), got %d", state.Max, state.CurrentNum)
			}

			// Verify all bytes were written
			if state.CurrentNum != tt.totalBytes {
				t.Errorf("Expected %d bytes written, got %d", tt.totalBytes, state.CurrentNum)
			}

			// Calculate percentage manually
			percentComplete := float64(state.CurrentNum) / float64(state.Max) * 100.0
			if percentComplete != 100.0 {
				t.Errorf("Expected 100%% completion, got %.2f%%", percentComplete)
			}

			// Verify description shows final file count [n/n]
			expectedDesc := fmt.Sprintf("[cyan][%d/%d][reset] %s", tt.totalFiles, tt.totalFiles, tt.description)
			if state.Description != expectedDesc {
				t.Errorf("Expected description '%s', got '%s'", expectedDesc, state.Description)
			}
		})
	}
}

// TestProgressBarWithCountCompletion tests progressBarWithCount completion
func TestProgressBarWithCountCompletion(t *testing.T) {
	tests := []struct {
		name        string
		totalBytes  int64
		totalFiles  int
		description string
	}{
		{
			name:        "single file",
			totalBytes:  1024,
			totalFiles:  1,
			description: "Downloading files",
		},
		{
			name:        "multiple files",
			totalBytes:  5120,
			totalFiles:  5,
			description: "Downloading files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create base progress bar in quiet mode
			baseBar := newProgressBar(tt.totalBytes, tt.description, 0, tt.totalFiles, true)
			var current int32
			bar := &progressBarWithCount{
				bar:          baseBar,
				current:      &current,
				total:        tt.totalFiles,
				description:  tt.description,
				showProgress: false,
			}

			// Simulate downloading files
			bytesPerFile := tt.totalBytes / int64(tt.totalFiles)
			for i := 0; i < tt.totalFiles; i++ {
				// Write bytes for this file
				bar.Add64(bytesPerFile)
				// Increment file count
				bar.incrementFile()
			}

			// Finish the progress bar
			bar.Finish()

			// Check the state
			state := baseBar.State()

			// Verify completion - CurrentNum should equal Max (which means 100%)
			if state.CurrentNum != state.Max {
				t.Errorf("Expected CurrentNum to equal Max (%d), got %d", state.Max, state.CurrentNum)
			}

			// Verify all bytes were written
			if state.CurrentNum != tt.totalBytes {
				t.Errorf("Expected %d bytes written, got %d", tt.totalBytes, state.CurrentNum)
			}

			// Calculate percentage manually
			percentComplete := float64(state.CurrentNum) / float64(state.Max) * 100.0
			if percentComplete != 100.0 {
				t.Errorf("Expected 100%% completion, got %.2f%%", percentComplete)
			}

			// Verify description shows final file count [n/n]
			expectedDesc := fmt.Sprintf("[cyan][%d/%d][reset] %s", tt.totalFiles, tt.totalFiles, tt.description)
			if state.Description != expectedDesc {
				t.Errorf("Expected description '%s', got '%s'", expectedDesc, state.Description)
			}
		})
	}
}
