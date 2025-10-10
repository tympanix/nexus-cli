package archive

import (
	"testing"
)

func TestParseCompressionFormat(t *testing.T) {
	tests := []struct {
		input       string
		expected    Format
		expectError bool
	}{
		{"gzip", FormatGzip, false},
		{"gz", FormatGzip, false},
		{"GZIP", FormatGzip, false},
		{"zstd", FormatZstd, false},
		{"zst", FormatZstd, false},
		{"ZSTD", FormatZstd, false},
		{"zip", FormatZip, false},
		{"ZIP", FormatZip, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			format, err := Parse(tt.input)
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
		format   Format
		expected string
	}{
		{FormatGzip, ".tar.gz"},
		{FormatZstd, ".tar.zst"},
		{FormatZip, ".zip"},
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
		expected Format
	}{
		{"archive.tar.gz", FormatGzip},
		{"backup-2024.tar.gz", FormatGzip},
		{"archive.tar.zst", FormatZstd},
		{"backup-2024.tar.zst", FormatZstd},
		{"archive.zip", FormatZip},
		{"backup-2024.zip", FormatZip},
		{"file.txt", FormatGzip}, // default
		{"", FormatGzip},         // default
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			format := DetectFromFilename(tt.filename)
			if format != tt.expected {
				t.Errorf("Expected format %q for filename %q, but got %q", tt.expected, tt.filename, format)
			}
		})
	}
}
