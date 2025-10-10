package checksum

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

func TestNewValidator(t *testing.T) {
	tests := []struct {
		name      string
		algorithm string
		wantErr   bool
	}{
		{
			name:      "sha1",
			algorithm: "sha1",
			wantErr:   false,
		},
		{
			name:      "SHA1 uppercase",
			algorithm: "SHA1",
			wantErr:   false,
		},
		{
			name:      "sha256",
			algorithm: "sha256",
			wantErr:   false,
		},
		{
			name:      "sha512",
			algorithm: "sha512",
			wantErr:   false,
		},
		{
			name:      "md5",
			algorithm: "md5",
			wantErr:   false,
		},
		{
			name:      "unsupported algorithm",
			algorithm: "invalid",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewValidator(tt.algorithm)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for algorithm '%s', got nil", tt.algorithm)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error for algorithm '%s': %v", tt.algorithm, err)
			}
			if validator == nil {
				t.Errorf("Expected non-nil validator for algorithm '%s'", tt.algorithm)
			}
		})
	}
}

func TestChecksumValidatorAlgorithm(t *testing.T) {
	validator, err := NewValidator("sha1")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	if validator.Algorithm() != "sha1" {
		t.Errorf("Expected algorithm 'sha1', got '%s'", validator.Algorithm())
	}
}

func TestChecksumValidatorValidate(t *testing.T) {
	testContent := "test content for checksum validation"

	testDir, err := os.MkdirTemp("", "test-checksum-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name      string
		algorithm string
		checksums nexusapi.Checksum
		wantValid bool
		wantErr   bool
	}{
		{
			name:      "valid sha1",
			algorithm: "sha1",
			checksums: nexusapi.Checksum{
				SHA1: "d38a2973b20670764496e490a7f638302eb96602",
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name:      "invalid sha1",
			algorithm: "sha1",
			checksums: nexusapi.Checksum{
				SHA1: "wrongchecksum",
			},
			wantValid: false,
			wantErr:   false,
		},
		{
			name:      "valid sha256",
			algorithm: "sha256",
			checksums: nexusapi.Checksum{
				SHA256: "b873ee26f3d17e038e023b4a4a9c9e3379ecc018171760b986abdbc011e17746",
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name:      "valid md5",
			algorithm: "md5",
			checksums: nexusapi.Checksum{
				MD5: "1786a2d74a141e8ca2d371a0b519ebc3",
			},
			wantValid: true,
			wantErr:   false,
		},
		{
			name:      "missing checksum",
			algorithm: "sha512",
			checksums: nexusapi.Checksum{},
			wantValid: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator, err := NewValidator(tt.algorithm)
			if err != nil {
				t.Fatalf("Failed to create validator: %v", err)
			}

			valid, err := validator.Validate(testFile, tt.checksums)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if valid != tt.wantValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.wantValid, valid)
			}
		})
	}
}

func TestChecksumValidatorValidateNonExistentFile(t *testing.T) {
	validator, err := NewValidator("sha1")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	_, err = validator.Validate("/nonexistent/file.txt", nexusapi.Checksum{SHA1: "abc123"})
	if err == nil {
		t.Errorf("Expected error for non-existent file, got nil")
	}
}

func TestChecksumValidatorValidateWithProgress(t *testing.T) {
	testContent := "test content for checksum validation with progress tracking"

	testDir, err := os.MkdirTemp("", "test-checksum-progress-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	validator, err := NewValidator("sha1")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Track bytes written to progress writer
	progressTracker := &bytesCounter{}

	// Expected SHA1 for the test content
	expectedChecksum := nexusapi.Checksum{
		SHA1: "3d636cf6f895a3598b5847b04fb334ac95f6b23e",
	}

	valid, err := validator.ValidateWithProgress(testFile, expectedChecksum, progressTracker)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !valid {
		t.Errorf("Expected valid checksum, got invalid")
	}

	// Verify that progress was tracked
	expectedBytes := int64(len(testContent))
	if progressTracker.bytesWritten != expectedBytes {
		t.Errorf("Expected %d bytes tracked, got %d", expectedBytes, progressTracker.bytesWritten)
	}
}

func TestComputeChecksumWithProgress(t *testing.T) {
	testContent := "test content for compute checksum with progress"

	testDir, err := os.MkdirTemp("", "test-compute-checksum-progress-*")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	progressTracker := &bytesCounter{}

	checksum, err := ComputeChecksumWithProgress(testFile, "sha1", progressTracker)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify checksum is computed correctly
	expectedChecksum := "b8df69c649ab3e06e79f9cd4c934fa55d3106c01"
	if checksum != expectedChecksum {
		t.Errorf("Expected checksum %s, got %s", expectedChecksum, checksum)
	}

	// Verify that progress was tracked
	expectedBytes := int64(len(testContent))
	if progressTracker.bytesWritten != expectedBytes {
		t.Errorf("Expected %d bytes tracked, got %d", expectedBytes, progressTracker.bytesWritten)
	}
}

// bytesCounter is a simple io.Writer that counts bytes written
type bytesCounter struct {
	bytesWritten int64
}

func (bc *bytesCounter) Write(p []byte) (n int, err error) {
	bc.bytesWritten += int64(len(p))
	return len(p), nil
}
