package nexus

import (
	"archive/tar"
	"archive/zip"
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
	return CreateTarGzWithGlob(srcDir, writer, "")
}

// CreateTarGzWithGlob creates a tar.gz archive containing files from srcDir filtered by glob pattern.
// The archive is written to the provided writer on-the-fly.
// Files are stored in the archive with paths relative to srcDir.
func CreateTarGzWithGlob(srcDir string, writer io.Writer, globPattern string) error {
	gzipWriter := gzip.NewWriter(writer)

	if err := createTarArchive(srcDir, gzipWriter, globPattern); err != nil {
		gzipWriter.Close()
		return err
	}

	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
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

	return extractTar(gzipReader, destDir)
}

// CreateTarZst creates a tar.zst archive containing all files from srcDir.
// The archive is written to the provided writer on-the-fly.
// Files are stored in the archive with paths relative to srcDir.
func CreateTarZst(srcDir string, writer io.Writer) error {
	return CreateTarZstWithGlob(srcDir, writer, "")
}

// CreateTarZstWithGlob creates a tar.zst archive containing files from srcDir filtered by glob pattern.
// The archive is written to the provided writer on-the-fly.
// Files are stored in the archive with paths relative to srcDir.
func CreateTarZstWithGlob(srcDir string, writer io.Writer, globPattern string) error {
	zstdWriter, err := zstd.NewWriter(writer)
	if err != nil {
		return fmt.Errorf("failed to create zstd writer: %w", err)
	}

	if err := createTarArchive(srcDir, zstdWriter, globPattern); err != nil {
		zstdWriter.Close()
		return err
	}

	if err := zstdWriter.Close(); err != nil {
		return fmt.Errorf("failed to close zstd writer: %w", err)
	}

	return nil
}

// ExtractTarZst extracts a tar.zst archive from the provided reader to destDir.
// Files are extracted on-the-fly as they are read from the archive.
func ExtractTarZst(reader io.Reader, destDir string) error {
	zstdReader, err := zstd.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create zstd reader: %w", err)
	}
	defer zstdReader.Close()

	return extractTar(zstdReader, destDir)
}

// extractTar is a helper function that extracts tar content from any decompressed reader.
func extractTar(reader io.Reader, destDir string) error {
	tarReader := tar.NewReader(reader)

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

// createTarArchive is a helper function that creates a tar archive from files.
// It writes to any io.Writer (which may be a compression writer).
func createTarArchive(srcDir string, writer io.Writer, globPattern string) error {
	tarWriter := tar.NewWriter(writer)
	defer tarWriter.Close()

	files, err := collectFilesWithGlob(srcDir, globPattern)
	if err != nil {
		return fmt.Errorf("failed to collect files: %w", err)
	}

	for _, filePath := range files {
		if err := addFileToTar(tarWriter, srcDir, filePath); err != nil {
			return err
		}
	}

	return nil
}

// addFileToTar adds a single file to a tar archive
func addFileToTar(tarWriter *tar.Writer, srcDir string, filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	relPath, err := filepath.Rel(srcDir, filePath)
	if err != nil {
		return fmt.Errorf("failed to get relative path for %s: %w", filePath, err)
	}
	relPath = filepath.ToSlash(relPath)

	header := &tar.Header{
		Name:    relPath,
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header for %s: %w", relPath, err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	if _, err := io.Copy(tarWriter, file); err != nil {
		return fmt.Errorf("failed to write file %s to archive: %w", relPath, err)
	}

	return nil
}

// CreateZip creates a zip archive containing all files from srcDir.
// The archive is written to the provided writer on-the-fly.
// Files are stored in the archive with paths relative to srcDir.
func CreateZip(srcDir string, writer io.Writer) error {
	return CreateZipWithGlob(srcDir, writer, "")
}

// CreateZipWithGlob creates a zip archive containing files from srcDir filtered by glob pattern.
// The archive is written to the provided writer on-the-fly.
// Files are stored in the archive with paths relative to srcDir.
func CreateZipWithGlob(srcDir string, writer io.Writer, globPattern string) error {
	zipWriter := zip.NewWriter(writer)
	defer zipWriter.Close()

	files, err := collectFilesWithGlob(srcDir, globPattern)
	if err != nil {
		return fmt.Errorf("failed to collect files: %w", err)
	}

	for _, filePath := range files {
		if err := addFileToZip(zipWriter, srcDir, filePath); err != nil {
			return err
		}
	}

	return nil
}

// addFileToZip adds a single file to a zip archive
func addFileToZip(zipWriter *zip.Writer, srcDir string, filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	relPath, err := filepath.Rel(srcDir, filePath)
	if err != nil {
		return fmt.Errorf("failed to get relative path for %s: %w", filePath, err)
	}
	relPath = filepath.ToSlash(relPath)

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("failed to create zip header for %s: %w", relPath, err)
	}
	header.Name = relPath
	header.Method = zip.Deflate

	headerWriter, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create header for %s: %w", relPath, err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	if _, err := io.Copy(headerWriter, file); err != nil {
		return fmt.Errorf("failed to write file %s to archive: %w", relPath, err)
	}

	return nil
}

// ExtractZip extracts a zip archive from the provided reader to destDir.
// Files are extracted on-the-fly as they are read from the archive.
func ExtractZip(reader io.Reader, destDir string) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read zip data: %w", err)
	}

	zipReader, err := zip.NewReader(strings.NewReader(string(data)), int64(len(data)))
	if err != nil {
		return fmt.Errorf("failed to create zip reader: %w", err)
	}

	for _, file := range zipReader.File {
		if err := extractZipFile(file, destDir); err != nil {
			return err
		}
	}

	return nil
}

// extractZipFile extracts a single file from a zip archive
func extractZipFile(file *zip.File, destDir string) error {
	targetPath := filepath.Join(destDir, file.Name)

	if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(destDir)) {
		return fmt.Errorf("illegal file path in archive: %s", file.Name)
	}

	if file.FileInfo().IsDir() {
		return os.MkdirAll(targetPath, file.Mode())
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", targetPath, err)
	}

	fileReader, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file %s in archive: %w", file.Name, err)
	}
	defer fileReader.Close()

	outFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", targetPath, err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, fileReader); err != nil {
		return fmt.Errorf("failed to extract file %s: %w", targetPath, err)
	}

	if err := os.Chmod(targetPath, file.Mode()); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", targetPath, err)
	}

	return nil
}
