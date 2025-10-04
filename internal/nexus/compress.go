package nexus

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
)

// CreateTarGz creates a tar.gz archive containing all files from srcDir.
// The archive is written to the provided writer on-the-fly.
// Files are stored in the archive with paths relative to srcDir.
func CreateTarGz(srcDir string, writer io.Writer) error {
	gzipWriter := gzip.NewWriter(writer)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Collect all files
	files, err := collectFiles(srcDir)
	if err != nil {
		return fmt.Errorf("failed to collect files: %w", err)
	}

	for _, filePath := range files {
		// Get file info
		info, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("failed to stat file %s: %w", filePath, err)
		}

		// Get relative path for archive
		relPath, err := filepath.Rel(srcDir, filePath)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", filePath, err)
		}
		// Normalize to forward slashes for consistency
		relPath = filepath.ToSlash(relPath)

		// Create tar header
		header := &tar.Header{
			Name:    relPath,
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		}

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %s: %w", relPath, err)
		}

		// Write file content
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", filePath, err)
		}

		if _, err := io.Copy(tarWriter, file); err != nil {
			file.Close()
			return fmt.Errorf("failed to write file %s to archive: %w", relPath, err)
		}
		file.Close()
	}

	return nil
}

// CreateTarZstd creates a tar.zst archive containing all files from srcDir.
// The archive is written to the provided writer on-the-fly.
// Files are stored in the archive with paths relative to srcDir.
func CreateTarZstd(srcDir string, writer io.Writer) error {
	zstdWriter, err := zstd.NewWriter(writer)
	if err != nil {
		return fmt.Errorf("failed to create zstd writer: %w", err)
	}
	defer zstdWriter.Close()

	tarWriter := tar.NewWriter(zstdWriter)
	defer tarWriter.Close()

	// Collect all files
	files, err := collectFiles(srcDir)
	if err != nil {
		return fmt.Errorf("failed to collect files: %w", err)
	}

	for _, filePath := range files {
		// Get file info
		info, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("failed to stat file %s: %w", filePath, err)
		}

		// Get relative path for archive
		relPath, err := filepath.Rel(srcDir, filePath)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", filePath, err)
		}
		// Normalize to forward slashes for consistency
		relPath = filepath.ToSlash(relPath)

		// Create tar header
		header := &tar.Header{
			Name:    relPath,
			Size:    info.Size(),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		}

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header for %s: %w", relPath, err)
		}

		// Write file content
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", filePath, err)
		}

		if _, err := io.Copy(tarWriter, file); err != nil {
			file.Close()
			return fmt.Errorf("failed to write file %s to archive: %w", relPath, err)
		}
		file.Close()
	}

	return nil
}

// ExtractTarGz extracts a tar.gz archive from the provided reader to destDir.
// Files are extracted on-the-fly as they are read from the archive.
func ExtractTarGz(reader io.Reader, destDir string) error {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct target path
		targetPath := filepath.Join(destDir, header.Name)

		// Security check: ensure path doesn't escape destDir
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(destDir)) {
			return fmt.Errorf("illegal file path in archive: %s", header.Name)
		}

		// Create directories as needed
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", targetPath, err)
		}

		// Extract file
		if header.Typeflag == tar.TypeReg {
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to extract file %s: %w", targetPath, err)
			}
			outFile.Close()

			// Restore file mode
			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to set permissions on %s: %w", targetPath, err)
			}
		}
	}

	return nil
}

// ExtractTarZstd extracts a tar.zst archive from the provided reader to destDir.
// Files are extracted on-the-fly as they are read from the archive.
func ExtractTarZstd(reader io.Reader, destDir string) error {
	zstdReader, err := zstd.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create zstd reader: %w", err)
	}
	defer zstdReader.Close()

	tarReader := tar.NewReader(zstdReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct target path
		targetPath := filepath.Join(destDir, header.Name)

		// Security check: ensure path doesn't escape destDir
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(destDir)) {
			return fmt.Errorf("illegal file path in archive: %s", header.Name)
		}

		// Create directories as needed
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", targetPath, err)
		}

		// Extract file
		if header.Typeflag == tar.TypeReg {
			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to extract file %s: %w", targetPath, err)
			}
			outFile.Close()

			// Restore file mode
			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to set permissions on %s: %w", targetPath, err)
			}
		}
	}

	return nil
}

// GenerateArchiveName generates a name for the compressed archive based on repository, subdir, and compression type.
// Format: <repository>-<subdir>.tar.gz or <repository>.tar.gz if subdir is empty (for gzip)
// Format: <repository>-<subdir>.tar.zst or <repository>.tar.zst if subdir is empty (for zstd)
func GenerateArchiveName(repository, subdir, compressionType string) string {
	// Sanitize repository and subdir to create a valid filename
	sanitize := func(s string) string {
		// Replace slashes and other problematic characters
		s = strings.ReplaceAll(s, "/", "-")
		s = strings.ReplaceAll(s, "\\", "-")
		s = strings.ReplaceAll(s, " ", "_")
		return s
	}

	// Determine file extension based on compression type
	extension := ".tar.gz"
	if compressionType == "zstd" {
		extension = ".tar.zst"
	}

	if subdir == "" {
		return fmt.Sprintf("%s%s", sanitize(repository), extension)
	}
	return fmt.Sprintf("%s-%s%s", sanitize(repository), sanitize(subdir), extension)
}
