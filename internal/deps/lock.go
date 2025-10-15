package deps

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ParseLockFile(filename string) (*LockFile, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filename, err)
	}
	defer file.Close()

	lockFile := &LockFile{
		Dependencies: make(map[string]map[string]string),
	}

	scanner := bufio.NewScanner(file)
	var currentSection string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			sectionName := strings.TrimSpace(line[1 : len(line)-1])
			currentSection = sectionName
			if lockFile.Dependencies[currentSection] == nil {
				lockFile.Dependencies[currentSection] = make(map[string]string)
			}
			continue
		}

		if !strings.Contains(line, "=") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if currentSection != "" {
			lockFile.Dependencies[currentSection][key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filename, err)
	}

	return lockFile, nil
}

func WriteLockFile(filename string, lockFile *LockFile) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", filename, err)
	}
	defer file.Close()

	for depName, files := range lockFile.Dependencies {
		fmt.Fprintf(file, "[%s]\n", depName)
		for filePath, checksum := range files {
			fmt.Fprintf(file, "%s = %s\n", filePath, checksum)
		}
		fmt.Fprintf(file, "\n")
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
