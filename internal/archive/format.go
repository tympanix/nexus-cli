package archive

import (
	"fmt"
	"io"
	"strings"
)

// Format represents the compression format for archives
type Format string

const (
	FormatGzip Format = "gzip"
	FormatZstd Format = "zstd"
	FormatZip  Format = "zip"
)

// String returns the string representation of the compression format
func (f Format) String() string {
	return string(f)
}

// Extension returns the file extension for the compression format
func (f Format) Extension() string {
	switch f {
	case FormatGzip:
		return ".tar.gz"
	case FormatZstd:
		return ".tar.zst"
	case FormatZip:
		return ".zip"
	default:
		return ".tar.gz"
	}
}

// CreateArchive creates a compressed archive based on the format
func (f Format) CreateArchive(srcDir string, writer io.Writer) error {
	return f.CreateArchiveWithGlob(srcDir, writer, "")
}

// CreateArchiveWithGlob creates a compressed archive based on the format with optional glob filtering
func (f Format) CreateArchiveWithGlob(srcDir string, writer io.Writer, globPattern string) error {
	switch f {
	case FormatGzip:
		return CreateTarGzWithGlob(srcDir, writer, globPattern)
	case FormatZstd:
		return CreateTarZstWithGlob(srcDir, writer, globPattern)
	case FormatZip:
		return CreateZipWithGlob(srcDir, writer, globPattern)
	default:
		return fmt.Errorf("unsupported compression format: %s", f)
	}
}

// ExtractArchive extracts a compressed archive based on the format
func (f Format) ExtractArchive(reader io.Reader, destDir string) error {
	switch f {
	case FormatGzip:
		return ExtractTarGz(reader, destDir)
	case FormatZstd:
		return ExtractTarZst(reader, destDir)
	case FormatZip:
		return ExtractZip(reader, destDir)
	default:
		return fmt.Errorf("unsupported compression format: %s", f)
	}
}

// Parse parses a string into a Format
func Parse(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "gzip", "gz":
		return FormatGzip, nil
	case "zstd", "zst":
		return FormatZstd, nil
	case "zip":
		return FormatZip, nil
	default:
		return "", fmt.Errorf("unsupported compression format '%s': must be one of: gzip, zstd, zip", s)
	}
}

// DetectFromFilename detects the compression format from a filename
func DetectFromFilename(filename string) Format {
	if strings.HasSuffix(filename, ".tar.zst") {
		return FormatZstd
	}
	if strings.HasSuffix(filename, ".zip") {
		return FormatZip
	}
	// Default to gzip for .tar.gz or any other case
	return FormatGzip
}
