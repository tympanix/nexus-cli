package deps

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type tomlConfig struct {
	Defaults     Defaults               `toml:"defaults"`
	Dependencies map[string]*Dependency `toml:"dependencies"`
}

func ParseDepsIni(filename string) (*DepsManifest, error) {
	var config tomlConfig
	if _, err := toml.DecodeFile(filename, &config); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", filename, err)
	}

	manifest := &DepsManifest{
		Defaults: Defaults{
			Repository: config.Defaults.Repository,
			Checksum:   config.Defaults.Checksum,
			OutputDir:  config.Defaults.OutputDir,
			URL:        config.Defaults.URL,
		},
		Dependencies: make(map[string]*Dependency),
	}

	if manifest.Defaults.Checksum == "" {
		manifest.Defaults.Checksum = "sha256"
	}
	if manifest.Defaults.OutputDir == "" {
		manifest.Defaults.OutputDir = "./local"
	}

	for name, dep := range config.Dependencies {
		dep.Name = name
		if dep.Repository == "" {
			dep.Repository = manifest.Defaults.Repository
		}
		if dep.Checksum == "" {
			dep.Checksum = manifest.Defaults.Checksum
		}
		if dep.OutputDir == "" {
			dep.OutputDir = manifest.Defaults.OutputDir
		}
		if dep.URL == "" {
			dep.URL = manifest.Defaults.URL
		}
		manifest.Dependencies[name] = dep
	}

	for name, dep := range manifest.Dependencies {
		if dep.Path == "" {
			return nil, fmt.Errorf("dependency %s is missing required 'path' field", name)
		}
		if dep.Repository == "" {
			return nil, fmt.Errorf("dependency %s is missing 'repository' (not set in defaults or dependency)", name)
		}
	}

	return manifest, nil
}

func WriteDepsIni(filename string, manifest *DepsManifest) error {
	config := tomlConfig{
		Defaults:     manifest.Defaults,
		Dependencies: make(map[string]*Dependency),
	}

	for name, dep := range manifest.Dependencies {
		newDep := &Dependency{
			Path:      dep.Path,
			Version:   dep.Version,
			Recursive: dep.Recursive,
			Dest:      dep.Dest,
		}
		if dep.URL != manifest.Defaults.URL && dep.URL != "" {
			newDep.URL = dep.URL
		}
		if dep.Repository != manifest.Defaults.Repository && dep.Repository != "" {
			newDep.Repository = dep.Repository
		}
		if dep.Checksum != manifest.Defaults.Checksum && dep.Checksum != "" {
			newDep.Checksum = dep.Checksum
		}
		if dep.OutputDir != manifest.Defaults.OutputDir && dep.OutputDir != "" {
			newDep.OutputDir = dep.OutputDir
		}
		config.Dependencies[name] = newDep
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", filename, err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("failed to encode TOML: %w", err)
	}

	return nil
}
