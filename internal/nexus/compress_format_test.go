package nexus

import (
	"testing"
)

func TestParseCompressionFormat(t *testing.T) {
	tests := []struct {
		input       string
		expected    CompressionFormat
		expectError bool
	}{
		{"gzip", CompressionGzip, false},
		{"gz", CompressionGzip, false},
		{"GZIP", CompressionGzip, false},
		{"zstd", CompressionZstd, false},
		{"zst", CompressionZstd, false},
		{"ZSTD", CompressionZstd, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			format, err := ParseCompressionFormat(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for input %q, but got none", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input %q: %v", tt.input, err)
				}
				if format != tt.expected {
					t.Errorf("Expected format %q for input %q, but got %q", tt.expected, tt.input, format)
				}
			}
		})
	}
}

func TestCompressionFormatExtension(t *testing.T) {
	tests := []struct {
		format   CompressionFormat
		expected string
	}{
		{CompressionGzip, ".tar.gz"},
		{CompressionZstd, ".tar.zst"},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			ext := tt.format.Extension()
			if ext != tt.expected {
				t.Errorf("Expected extension %q for format %q, but got %q", tt.expected, tt.format, ext)
			}
		})
	}
}

func TestDetectCompressionFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		expected CompressionFormat
	}{
		{"archive.tar.gz", CompressionGzip},
		{"backup-2024.tar.gz", CompressionGzip},
		{"archive.tar.zst", CompressionZstd},
		{"backup-2024.tar.zst", CompressionZstd},
		{"file.txt", CompressionGzip}, // default
		{"", CompressionGzip},         // default
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			format := DetectCompressionFromFilename(tt.filename)
			if format != tt.expected {
				t.Errorf("Expected format %q for filename %q, but got %q", tt.expected, tt.filename, format)
			}
		})
	}
}
