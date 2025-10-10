package checksum

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"

	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

// Validator interface for checksum validation
type Validator interface {
	Validate(filePath string, expected nexusapi.Checksum) (bool, error)
	ValidateWithProgress(filePath string, expected nexusapi.Checksum, progress io.Writer) (bool, error)
	Algorithm() string
}

type validator struct {
	algorithm string
	hashFunc  func() hash.Hash
	extractor func(nexusapi.Checksum) string
}

func (v *validator) Algorithm() string {
	return v.algorithm
}

func (v *validator) Validate(filePath string, expected nexusapi.Checksum) (bool, error) {
	return v.ValidateWithProgress(filePath, expected, io.Discard)
}

func (v *validator) ValidateWithProgress(filePath string, expected nexusapi.Checksum, progress io.Writer) (bool, error) {
	expectedChecksum := v.extractor(expected)
	if expectedChecksum == "" {
		return false, fmt.Errorf("no %s checksum available for validation", v.algorithm)
	}

	actualChecksum, err := v.computeChecksumWithProgress(filePath, progress)
	if err != nil {
		return false, err
	}

	return strings.EqualFold(actualChecksum, expectedChecksum), nil
}

func (v *validator) computeChecksum(filePath string) (string, error) {
	return v.computeChecksumWithProgress(filePath, io.Discard)
}

func (v *validator) computeChecksumWithProgress(filePath string, progress io.Writer) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := v.hashFunc()
	teeReader := io.TeeReader(file, progress)
	if _, err := io.Copy(h, teeReader); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// NewValidator creates a new checksum validator for the specified algorithm
func NewValidator(algorithm string) (Validator, error) {
	alg := strings.ToLower(algorithm)
	switch alg {
	case "sha1":
		return &validator{
			algorithm: "sha1",
			hashFunc:  sha1.New,
			extractor: func(c nexusapi.Checksum) string { return c.SHA1 },
		}, nil
	case "sha256":
		return &validator{
			algorithm: "sha256",
			hashFunc:  sha256.New,
			extractor: func(c nexusapi.Checksum) string { return c.SHA256 },
		}, nil
	case "sha512":
		return &validator{
			algorithm: "sha512",
			hashFunc:  sha512.New,
			extractor: func(c nexusapi.Checksum) string { return c.SHA512 },
		}, nil
	case "md5":
		return &validator{
			algorithm: "md5",
			hashFunc:  md5.New,
			extractor: func(c nexusapi.Checksum) string { return c.MD5 },
		}, nil
	default:
		return nil, fmt.Errorf("unsupported checksum algorithm '%s': must be one of: sha1, sha256, sha512, md5", algorithm)
	}
}

// ComputeChecksum computes the checksum of a file using the specified algorithm
func ComputeChecksum(filePath string, algorithm string) (string, error) {
	return ComputeChecksumWithProgress(filePath, algorithm, io.Discard)
}

// ComputeChecksumWithProgress computes the checksum of a file using the specified algorithm with progress tracking
func ComputeChecksumWithProgress(filePath string, algorithm string, progress io.Writer) (string, error) {
	var h hash.Hash
	switch strings.ToLower(algorithm) {
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	case "md5":
		h = md5.New()
	default:
		return "", fmt.Errorf("unsupported checksum algorithm '%s'", algorithm)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	teeReader := io.TeeReader(file, progress)
	if _, err := io.Copy(h, teeReader); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
