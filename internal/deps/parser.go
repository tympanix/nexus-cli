package deps

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func validateOutputDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("output_dir cannot be empty")
	}

	cleanDir := filepath.Clean(dir)

	if cleanDir == "." {
		return fmt.Errorf("output_dir cannot be '.' (current directory) for safety reasons")
	}

	if cleanDir == "/" {
		return fmt.Errorf("output_dir cannot be '/' (root directory) for safety reasons")
	}

	return nil
}

func ParseDepsIni(filename string) (*DepsManifest, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filename, err)
	}
	defer file.Close()

	manifest := &DepsManifest{
		Defaults: Defaults{
			Repository: "",
			Checksum:   "sha256",
			OutputDir:  "./local",
			URL:        "",
		},
		Dependencies: make(map[string]*Dependency),
	}

	scanner := bufio.NewScanner(file)
	var currentSection string
	var currentDep *Dependency

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			sectionName := strings.TrimSpace(line[1 : len(line)-1])
			currentSection = sectionName

			if sectionName == "defaults" {
				continue
			}

			currentDep = &Dependency{
				Name:       sectionName,
				Repository: manifest.Defaults.Repository,
				Checksum:   manifest.Defaults.Checksum,
				OutputDir:  manifest.Defaults.OutputDir,
				URL:        manifest.Defaults.URL,
			}
			manifest.Dependencies[sectionName] = currentDep
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

		if currentSection == "defaults" {
			switch key {
			case "repository":
				manifest.Defaults.Repository = value
			case "checksum":
				manifest.Defaults.Checksum = value
			case "output_dir":
				if err := validateOutputDir(value); err != nil {
					return nil, fmt.Errorf("invalid output_dir in [defaults]: %w", err)
				}
				manifest.Defaults.OutputDir = value
			case "url":
				manifest.Defaults.URL = value
			}
		} else if currentDep != nil {
			switch key {
			case "repository":
				currentDep.Repository = value
			case "path":
				currentDep.Path = value
			case "version":
				currentDep.Version = value
			case "checksum":
				currentDep.Checksum = value
			case "output_dir":
				if err := validateOutputDir(value); err != nil {
					return nil, fmt.Errorf("invalid output_dir in [%s]: %w", currentSection, err)
				}
				currentDep.OutputDir = value
			case "dest":
				currentDep.Dest = value
			case "recursive":
				currentDep.Recursive = strings.ToLower(value) == "true"
			case "url":
				currentDep.URL = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", filename, err)
	}

	for name, dep := range manifest.Dependencies {
		if dep.Path == "" {
			return nil, fmt.Errorf("dependency %s is missing required 'path' field", name)
		}
		if dep.Repository == "" {
			return nil, fmt.Errorf("dependency %s is missing 'repository' (not set in defaults or dependency)", name)
		}
		if err := validateOutputDir(dep.OutputDir); err != nil {
			return nil, fmt.Errorf("dependency %s has invalid output_dir: %w", name, err)
		}
	}

	return manifest, nil
}

func WriteDepsIni(filename string, manifest *DepsManifest) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", filename, err)
	}
	defer file.Close()

	if manifest.Defaults.Repository != "" || manifest.Defaults.Checksum != "" || manifest.Defaults.OutputDir != "" || manifest.Defaults.URL != "" {
		fmt.Fprintf(file, "[defaults]\n")
		if manifest.Defaults.URL != "" {
			fmt.Fprintf(file, "url = %s\n", manifest.Defaults.URL)
		}
		if manifest.Defaults.Repository != "" {
			fmt.Fprintf(file, "repository = %s\n", manifest.Defaults.Repository)
		}
		if manifest.Defaults.Checksum != "" {
			fmt.Fprintf(file, "checksum = %s\n", manifest.Defaults.Checksum)
		}
		if manifest.Defaults.OutputDir != "" {
			fmt.Fprintf(file, "output_dir = %s\n", manifest.Defaults.OutputDir)
		}
		fmt.Fprintf(file, "\n")
	}

	for name, dep := range manifest.Dependencies {
		fmt.Fprintf(file, "[%s]\n", name)
		fmt.Fprintf(file, "path = %s\n", dep.Path)
		if dep.Version != "" {
			fmt.Fprintf(file, "version = %s\n", dep.Version)
		}
		if dep.URL != manifest.Defaults.URL && dep.URL != "" {
			fmt.Fprintf(file, "url = %s\n", dep.URL)
		}
		if dep.Repository != manifest.Defaults.Repository && dep.Repository != "" {
			fmt.Fprintf(file, "repository = %s\n", dep.Repository)
		}
		if dep.Checksum != manifest.Defaults.Checksum && dep.Checksum != "" {
			fmt.Fprintf(file, "checksum = %s\n", dep.Checksum)
		}
		if dep.OutputDir != manifest.Defaults.OutputDir && dep.OutputDir != "" {
			fmt.Fprintf(file, "output_dir = %s\n", dep.OutputDir)
		}
		if dep.Dest != "" {
			fmt.Fprintf(file, "dest = %s\n", dep.Dest)
		}
		if dep.Recursive {
			fmt.Fprintf(file, "recursive = true\n")
		}
		fmt.Fprintf(file, "\n")
	}

	return nil
}
