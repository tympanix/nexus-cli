package deps

const DefaultDepsIniTemplate = `[defaults]
url = http://localhost:8081
repository = libs
checksum = sha256
output_dir = ./local

[example_txt]
path = docs/example-${version}.txt
version = 1.0.0

[libfoo_tar]
path = thirdparty/libfoo-${version}.tar.gz
version = 1.2.3
checksum = sha512

[docs_folder]
path = docs/${version}/
version = 2025-10-15
recursive = true
`

func CreateTemplateIni(filename string) error {
	manifest := &DepsManifest{
		Defaults: Defaults{
			URL:        "http://localhost:8081",
			Repository: "libs",
			Checksum:   "sha256",
			OutputDir:  "./local",
		},
		Dependencies: map[string]*Dependency{
			"example_txt": {
				Name:       "example_txt",
				Path:       "docs/example-${version}.txt",
				Version:    "1.0.0",
				URL:        "http://localhost:8081",
				Repository: "libs",
				Checksum:   "sha256",
				OutputDir:  "./local",
			},
			"libfoo_tar": {
				Name:       "libfoo_tar",
				Path:       "thirdparty/libfoo-${version}.tar.gz",
				Version:    "1.2.3",
				URL:        "http://localhost:8081",
				Repository: "libs",
				Checksum:   "sha512",
				OutputDir:  "./local",
			},
			"docs_folder": {
				Name:       "docs_folder",
				Path:       "docs/${version}/",
				Version:    "2025-10-15",
				URL:        "http://localhost:8081",
				Repository: "libs",
				Checksum:   "sha256",
				OutputDir:  "./local",
				Recursive:  true,
			},
		},
	}
	return WriteDepsIni(filename, manifest)
}
