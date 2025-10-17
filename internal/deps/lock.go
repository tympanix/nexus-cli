package deps

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

type tomlLockFile struct {
	Deps map[string]map[string]string `toml:"deps"`
}

func ParseLockFile(filename string) (*LockFile, error) {
	var lockConfig tomlLockFile
	if _, err := toml.DecodeFile(filename, &lockConfig); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	lockFile := &LockFile{
		Dependencies: lockConfig.Deps,
	}

	if lockFile.Dependencies == nil {
		lockFile.Dependencies = make(map[string]map[string]string)
	}

	return lockFile, nil
}

func WriteLockFile(filename string, lockFile *LockFile) error {
	var depNames []string
	for depName := range lockFile.Dependencies {
		depNames = append(depNames, depName)
	}
	sort.Strings(depNames)

	sortedDeps := make(map[string]map[string]string)
	for _, depName := range depNames {
		files := lockFile.Dependencies[depName]
		var filePaths []string
		for filePath := range files {
			filePaths = append(filePaths, filePath)
		}
		sort.Strings(filePaths)

		sortedFiles := make(map[string]string)
		for _, filePath := range filePaths {
			sortedFiles[filePath] = files[filePath]
		}
		sortedDeps[depName] = sortedFiles
	}

	lockConfig := tomlLockFile{
		Deps: sortedDeps,
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", filename, err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(lockConfig); err != nil {
		return fmt.Errorf("failed to encode TOML: %w", err)
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
	expectedChecksum := parts[1]

	if !strings.EqualFold(expectedAlgorithm, algorithm) {
		return fmt.Errorf("checksum algorithm mismatch: expected %s, got %s", expectedAlgorithm, algorithm)
	}

	if !strings.EqualFold(expectedChecksum, actualChecksum) {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", filePath, expectedChecksum, actualChecksum)
	}

	return nil
}
