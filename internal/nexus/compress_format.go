package nexus

import (
	"fmt"
	"io"
	"strings"
)

// CompressionFormat represents the compression format for archives
type CompressionFormat string

const (
	CompressionGzip CompressionFormat = "gzip"
	CompressionZstd CompressionFormat = "zstd"
	CompressionZip  CompressionFormat = "zip"
)

// String returns the string representation of the compression format
func (f CompressionFormat) String() string {
	return string(f)
}

// Extension returns the file extension for the compression format
func (f CompressionFormat) Extension() string {
	switch f {
	case CompressionGzip:
		return ".tar.gz"
	case CompressionZstd:
		return ".tar.zst"
	case CompressionZip:
		return ".zip"
	default:
		return ".tar.gz"
	}
}

// CreateArchive creates a compressed archive based on the format
func (f CompressionFormat) CreateArchive(srcDir string, writer io.Writer) error {
	return f.CreateArchiveWithGlob(srcDir, writer, "")
}

// CreateArchiveWithGlob creates a compressed archive based on the format with optional glob filtering
func (f CompressionFormat) CreateArchiveWithGlob(srcDir string, writer io.Writer, globPattern string) error {
	switch f {
	case CompressionGzip:
		return CreateTarGzWithGlob(srcDir, writer, globPattern)
	case CompressionZstd:
		return CreateTarZstWithGlob(srcDir, writer, globPattern)
	case CompressionZip:
		return CreateZipWithGlob(srcDir, writer, globPattern)
	default:
		return fmt.Errorf("unsupported compression format: %s", f)
	}
}

// ExtractArchive extracts a compressed archive based on the format
func (f CompressionFormat) ExtractArchive(reader io.Reader, destDir string) error {
	switch f {
	case CompressionGzip:
		return ExtractTarGz(reader, destDir)
	case CompressionZstd:
		return ExtractTarZst(reader, destDir)
	case CompressionZip:
		return ExtractZip(reader, destDir)
	default:
		return fmt.Errorf("unsupported compression format: %s", f)
	}
}

// ParseCompressionFormat parses a string into a CompressionFormat
func ParseCompressionFormat(s string) (CompressionFormat, error) {
	switch strings.ToLower(s) {
	case "gzip", "gz":
		return CompressionGzip, nil
	case "zstd", "zst":
		return CompressionZstd, nil
	case "zip":
		return CompressionZip, nil
	default:
		return "", fmt.Errorf("unsupported compression format '%s': must be one of: gzip, zstd, zip", s)
	}
}

// DetectCompressionFromFilename detects the compression format from a filename
func DetectCompressionFromFilename(filename string) CompressionFormat {
	if strings.HasSuffix(filename, ".tar.zst") {
		return CompressionZstd
	}
	if strings.HasSuffix(filename, ".zip") {
		return CompressionZip
	}
	// Default to gzip for .tar.gz or any other case
	return CompressionGzip
}
