package deps

import (
	"fmt"
	"os"
)

func GenerateEnvFile(filename string, manifest *DepsManifest) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", filename, err)
	}
	defer file.Close()

	for name, dep := range manifest.Dependencies {
		export := &EnvExport{
			Name:    name,
			Version: dep.Version,
			Path:    dep.LocalPath(),
		}

		fmt.Fprintf(file, "%s=\"%s\"\n", export.EnvName(), export.Name)
		fmt.Fprintf(file, "%s=\"%s\"\n", export.EnvVersion(), export.Version)
		fmt.Fprintf(file, "%s=\"%s\"\n", export.EnvPath(), export.Path)
		fmt.Fprintf(file, "\n")
	}

	return nil
}
