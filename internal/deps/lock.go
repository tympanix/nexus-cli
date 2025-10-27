package deps

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/go-ini/ini"
)

func hexToBase64(hexStr string) (string, error) {
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", fmt.Errorf("invalid hex string: %w", err)
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

func base64ToHex(b64Str string) (string, error) {
	bytes, err := base64.StdEncoding.DecodeString(b64Str)
	if err != nil {
		return "", fmt.Errorf("invalid base64 string: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func ParseLockFile(filename string) (*LockFile, error) {
	cfg, err := ini.Load(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filename, err)
	}

	lockFile := &LockFile{
		Dependencies: make(map[string]map[string]string),
	}

	for _, section := range cfg.Sections() {
		sectionName := section.Name()
		if sectionName == "DEFAULT" {
			continue
		}

		lockFile.Dependencies[sectionName] = make(map[string]string)
		for _, key := range section.Keys() {
			lockFile.Dependencies[sectionName][key.Name()] = key.String()
		}
	}

	return lockFile, nil
}

func WriteLockFile(filename string, lockFile *LockFile) error {
	cfg := ini.Empty()

	var depNames []string
	for depName := range lockFile.Dependencies {
		depNames = append(depNames, depName)
	}
	sort.Strings(depNames)

	for _, depName := range depNames {
		files := lockFile.Dependencies[depName]
		section, _ := cfg.NewSection(depName)

		var filePaths []string
		for filePath := range files {
			filePaths = append(filePaths, filePath)
		}
		sort.Strings(filePaths)

		for _, filePath := range filePaths {
			checksumStr := files[filePath]
			parts := strings.SplitN(checksumStr, ":", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid checksum format: %s", checksumStr)
			}
			algorithm := parts[0]
			hexChecksum := parts[1]

			base64Checksum, err := hexToBase64(hexChecksum)
			if err != nil {
				return fmt.Errorf("failed to convert checksum to base64 for %s: %w", filePath, err)
			}

			section.NewKey(filePath, fmt.Sprintf("%s:%s", algorithm, base64Checksum))
		}
	}

	if err := cfg.SaveTo(filename); err != nil {
		return fmt.Errorf("failed to create %s: %w", filename, err)
	}

	return nil
}

func VerifyLockFile(lockFile *LockFile, depName string, filePath string, algorithm string, actualChecksum string) error {
	if lockFile.Dependencies[depName] == nil {
		return fmt.Errorf("dependency %s not found in lock file", depName)
	}

	expectedChecksumStr, ok := lockFile.Dependencies[depName][filePath]
	if !ok {
		return fmt.Errorf("file %s not found in lock file for dependency %s", filePath, depName)
	}

	parts := strings.SplitN(expectedChecksumStr, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid checksum format in lock file: %s", expectedChecksumStr)
	}

	expectedAlgorithm := parts[0]
	base64Checksum := parts[1]

	expectedChecksum, err := base64ToHex(base64Checksum)
	if err != nil {
		return fmt.Errorf("invalid base64 checksum in lock file: %w", err)
	}

	if !strings.EqualFold(expectedAlgorithm, algorithm) {
		return fmt.Errorf("checksum algorithm mismatch: expected %s, got %s", expectedAlgorithm, algorithm)
	}

	if !strings.EqualFold(expectedChecksum, actualChecksum) {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", filePath, expectedChecksum, actualChecksum)
	}

	return nil
}
