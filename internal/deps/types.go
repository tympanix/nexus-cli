package deps

import (
	"fmt"
	"path/filepath"
	"strings"
)

type Defaults struct {
	Repository string
	Checksum   string
	OutputDir  string
	URL        string
}

type Dependency struct {
	Name       string
	Repository string
	Path       string
	Version    string
	Checksum   string
	OutputDir  string
	Dest       string
	Recursive  bool
	URL        string
}

func (d *Dependency) ExpandedPath() string {
	return expandVariables(d.Path, d.Version)
}

func (d *Dependency) LocalPath() string {
	if d.Dest != "" {
		return d.Dest
	}
	expanded := d.ExpandedPath()
	return filepath.Join(d.OutputDir, expanded)
}

func (d *Dependency) NexusPath() string {
	return d.ExpandedPath()
}

type DepsManifest struct {
	Defaults     Defaults
	Dependencies map[string]*Dependency
}

type LockFile struct {
	Dependencies map[string]map[string]string
}

type EnvExport struct {
	Name    string
	Version string
	Path    string
}

func expandVariables(template string, version string) string {
	result := template
	result = strings.ReplaceAll(result, "${version}", version)
	return result
}

func NormalizeName(name string) string {
	return strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
}

func (e *EnvExport) EnvName() string {
	return fmt.Sprintf("DEPS_%s_NAME", NormalizeName(e.Name))
}

func (e *EnvExport) EnvVersion() string {
	return fmt.Sprintf("DEPS_%s_VERSION", NormalizeName(e.Name))
}

func (e *EnvExport) EnvPath() string {
	return fmt.Sprintf("DEPS_%s_PATH", NormalizeName(e.Name))
}
