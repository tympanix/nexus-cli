package nexus

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

type ChecksumValidator interface {
	Validate(filePath string, expected nexusapi.Checksum) (bool, error)
	Algorithm() string
}

type checksumValidator struct {
	algorithm string
	hashFunc  func() hash.Hash
	extractor func(nexusapi.Checksum) string
}

func (v *checksumValidator) Algorithm() string {
	return v.algorithm
}

func (v *checksumValidator) Validate(filePath string, expected nexusapi.Checksum) (bool, error) {
	expectedChecksum := v.extractor(expected)
	if expectedChecksum == "" {
		return false, fmt.Errorf("no %s checksum available for validation", v.algorithm)
	}

	actualChecksum, err := v.computeChecksum(filePath)
	if err != nil {
		return false, err
	}

	return strings.EqualFold(actualChecksum, expectedChecksum), nil
}

func (v *checksumValidator) computeChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	h := v.hashFunc()
	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func NewChecksumValidator(algorithm string) (ChecksumValidator, error) {
	alg := strings.ToLower(algorithm)
	switch alg {
	case "sha1":
		return &checksumValidator{
			algorithm: "sha1",
			hashFunc:  sha1.New,
			extractor: func(c nexusapi.Checksum) string { return c.SHA1 },
		}, nil
	case "sha256":
		return &checksumValidator{
			algorithm: "sha256",
			hashFunc:  sha256.New,
			extractor: func(c nexusapi.Checksum) string { return c.SHA256 },
		}, nil
	case "sha512":
		return &checksumValidator{
			algorithm: "sha512",
			hashFunc:  sha512.New,
			extractor: func(c nexusapi.Checksum) string { return c.SHA512 },
		}, nil
	case "md5":
		return &checksumValidator{
			algorithm: "md5",
			hashFunc:  md5.New,
			extractor: func(c nexusapi.Checksum) string { return c.MD5 },
		}, nil
	default:
		return nil, fmt.Errorf("unsupported checksum algorithm '%s': must be one of: sha1, sha256, sha512, md5", algorithm)
	}
}

func computeChecksum(filePath string, algorithm string) (string, error) {
	var h hash.Hash
	alg := strings.ToLower(algorithm)
	switch alg {
	case "sha1":
		h = sha1.New()
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	case "md5":
		h = md5.New()
	default:
		return "", fmt.Errorf("unsupported checksum algorithm '%s': must be one of: sha1, sha256, sha512, md5", algorithm)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(h, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
