package operations

import (
	"github.com/tympanix/nexus-cli/internal/archive"
	"github.com/tympanix/nexus-cli/internal/checksum"
	"github.com/tympanix/nexus-cli/internal/util"
)

// UploadOptions holds options for upload operations
type UploadOptions struct {
	ChecksumAlgorithm string
	SkipChecksum      bool
	Force             bool
	Logger            util.Logger
	QuietMode         bool
	Compress          bool           // Enable compression (tar.gz, tar.zst, or zip)
	CompressionFormat archive.Format // Compression format to use (gzip, zstd, or zip)
	GlobPattern       string         // Optional glob pattern(s) to filter files (comma-separated, supports negation with !)
	KeyFromFile       string         // Path to file to compute hash from for {key} template
	checksumValidator checksum.Validator
}

// SetChecksumAlgorithm validates and sets the checksum algorithm
// Returns an error if the algorithm is not supported
func (opts *UploadOptions) SetChecksumAlgorithm(algorithm string) error {
	validator, err := checksum.NewValidator(algorithm)
	if err != nil {
		return err
	}
	opts.ChecksumAlgorithm = validator.Algorithm()
	opts.checksumValidator = validator
	return nil
}

// DownloadOptions holds options for download operations
type DownloadOptions struct {
	ChecksumAlgorithm string
	SkipChecksum      bool
	Force             bool
	Logger            util.Logger
	QuietMode         bool
	Flatten           bool
	DeleteExtra       bool
	Compress          bool           // Enable decompression (tar.gz, tar.zst, or zip)
	CompressionFormat archive.Format // Compression format to use (gzip, zstd, or zip)
	KeyFromFile       string         // Path to file to compute hash from for {key} template
	checksumValidator checksum.Validator
}

// SetChecksumAlgorithm validates and sets the checksum algorithm
// Returns an error if the algorithm is not supported
func (opts *DownloadOptions) SetChecksumAlgorithm(algorithm string) error {
	validator, err := checksum.NewValidator(algorithm)
	if err != nil {
		return err
	}
	opts.ChecksumAlgorithm = validator.Algorithm()
	opts.checksumValidator = validator
	return nil
}

// DownloadStatus represents the exit status of a download operation
type DownloadStatus int

const (
	DownloadSuccess       DownloadStatus = 0
	DownloadError         DownloadStatus = 1
	DownloadNoAssetsFound DownloadStatus = 66
)
