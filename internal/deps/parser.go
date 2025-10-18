package deps

import (
	"fmt"
	"path/filepath"

	"github.com/go-ini/ini"
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
	cfg, err := ini.Load(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", filename, err)
	}

	manifest := &DepsManifest{
		Defaults: Defaults{
			Repository: "",
			Checksum:   "sha256",
			OutputDir:  "./local",
			URL:        "",
		},
		Dependencies: make(map[string]*Dependency),
	}

	validDefaultKeys := map[string]bool{
		"repository": true,
		"checksum":   true,
		"output_dir": true,
		"url":        true,
	}

	if cfg.HasSection("defaults") {
		defaultsSection := cfg.Section("defaults")

		for _, key := range defaultsSection.KeyStrings() {
			if !validDefaultKeys[key] {
				return nil, fmt.Errorf("unknown key '%s' in [defaults] section", key)
			}
		}

		if defaultsSection.HasKey("repository") {
			manifest.Defaults.Repository = defaultsSection.Key("repository").String()
		}
		if defaultsSection.HasKey("checksum") {
			manifest.Defaults.Checksum = defaultsSection.Key("checksum").String()
		}
		if defaultsSection.HasKey("output_dir") {
			manifest.Defaults.OutputDir = defaultsSection.Key("output_dir").String()
		}
		if defaultsSection.HasKey("url") {
			manifest.Defaults.URL = defaultsSection.Key("url").String()
		}
	}

	validDependencyKeys := map[string]bool{
		"repository": true,
		"path":       true,
		"version":    true,
		"checksum":   true,
		"output_dir": true,
		"dest":       true,
		"recursive":  true,
		"url":        true,
	}

	for _, section := range cfg.Sections() {
		sectionName := section.Name()
		if sectionName == "DEFAULT" || sectionName == "defaults" {
			continue
		}

		for _, key := range section.KeyStrings() {
			if !validDependencyKeys[key] {
				return nil, fmt.Errorf("unknown key '%s' in [%s] section", key, sectionName)
			}
		}

		dep := &Dependency{
			Name:       sectionName,
			Repository: manifest.Defaults.Repository,
			Checksum:   manifest.Defaults.Checksum,
			OutputDir:  manifest.Defaults.OutputDir,
			URL:        manifest.Defaults.URL,
		}

		if section.HasKey("repository") {
			dep.Repository = section.Key("repository").String()
		}
		if section.HasKey("path") {
			dep.Path = section.Key("path").String()
		}
		if section.HasKey("version") {
			dep.Version = section.Key("version").String()
		}
		if section.HasKey("checksum") {
			dep.Checksum = section.Key("checksum").String()
		}
		if section.HasKey("output_dir") {
			dep.OutputDir = section.Key("output_dir").String()
		}
		if section.HasKey("dest") {
			dep.Dest = section.Key("dest").String()
		}
		if section.HasKey("recursive") {
			dep.Recursive, _ = section.Key("recursive").Bool()
		}
		if section.HasKey("url") {
			dep.URL = section.Key("url").String()
		}

		manifest.Dependencies[sectionName] = dep
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
	cfg := ini.Empty()

	if manifest.Defaults.Repository != "" || manifest.Defaults.Checksum != "" || manifest.Defaults.OutputDir != "" || manifest.Defaults.URL != "" {
		defaultsSection, _ := cfg.NewSection("defaults")
		if manifest.Defaults.URL != "" {
			defaultsSection.NewKey("url", manifest.Defaults.URL)
		}
		if manifest.Defaults.Repository != "" {
			defaultsSection.NewKey("repository", manifest.Defaults.Repository)
		}
		if manifest.Defaults.Checksum != "" {
			defaultsSection.NewKey("checksum", manifest.Defaults.Checksum)
		}
		if manifest.Defaults.OutputDir != "" {
			defaultsSection.NewKey("output_dir", manifest.Defaults.OutputDir)
		}
	}

	for name, dep := range manifest.Dependencies {
		depSection, _ := cfg.NewSection(name)
		depSection.NewKey("path", dep.Path)
		if dep.Version != "" {
			depSection.NewKey("version", dep.Version)
		}
		if dep.URL != manifest.Defaults.URL && dep.URL != "" {
			depSection.NewKey("url", dep.URL)
		}
		if dep.Repository != manifest.Defaults.Repository && dep.Repository != "" {
			depSection.NewKey("repository", dep.Repository)
		}
		if dep.Checksum != manifest.Defaults.Checksum && dep.Checksum != "" {
			depSection.NewKey("checksum", dep.Checksum)
		}
		if dep.OutputDir != manifest.Defaults.OutputDir && dep.OutputDir != "" {
			depSection.NewKey("output_dir", dep.OutputDir)
		}
		if dep.Dest != "" {
			depSection.NewKey("dest", dep.Dest)
		}
		if dep.Recursive {
			depSection.NewKey("recursive", "true")
		}
	}

	if err := cfg.SaveTo(filename); err != nil {
		return fmt.Errorf("failed to create %s: %w", filename, err)
	}

	return nil
}
