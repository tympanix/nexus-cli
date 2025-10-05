package nexus

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

func TestNewChecksumValidator(t *testing.T) {
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
			validator, err := NewChecksumValidator(tt.algorithm)
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
	validator, err := NewChecksumValidator("sha1")
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
			validator, err := NewChecksumValidator(tt.algorithm)
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
	validator, err := NewChecksumValidator("sha1")
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	_, err = validator.Validate("/nonexistent/file.txt", nexusapi.Checksum{SHA1: "abc123"})
	if err == nil {
		t.Errorf("Expected error for non-existent file, got nil")
	}
}
