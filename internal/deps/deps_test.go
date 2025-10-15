package deps

import (
	"os"
	"testing"
)

func TestParseDepsIni(t *testing.T) {
	content := `[defaults]
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
	tmpfile, err := os.CreateTemp("", "deps-*.ini")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	manifest, err := ParseDepsIni(tmpfile.Name())
	if err != nil {
		t.Fatalf("ParseDepsIni failed: %v", err)
	}

	if manifest.Defaults.Repository != "libs" {
		t.Errorf("Expected repository 'libs', got '%s'", manifest.Defaults.Repository)
	}
	if manifest.Defaults.Checksum != "sha256" {
		t.Errorf("Expected checksum 'sha256', got '%s'", manifest.Defaults.Checksum)
	}
	if manifest.Defaults.OutputDir != "./local" {
		t.Errorf("Expected output_dir './local', got '%s'", manifest.Defaults.OutputDir)
	}

	if len(manifest.Dependencies) != 3 {
		t.Fatalf("Expected 3 dependencies, got %d", len(manifest.Dependencies))
	}

	exampleTxt := manifest.Dependencies["example_txt"]
	if exampleTxt == nil {
		t.Fatal("example_txt dependency not found")
	}
	if exampleTxt.Path != "docs/example-${version}.txt" {
		t.Errorf("Expected path 'docs/example-${version}.txt', got '%s'", exampleTxt.Path)
	}
	if exampleTxt.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", exampleTxt.Version)
	}

	libfooTar := manifest.Dependencies["libfoo_tar"]
	if libfooTar == nil {
		t.Fatal("libfoo_tar dependency not found")
	}
	if libfooTar.Checksum != "sha512" {
		t.Errorf("Expected checksum 'sha512', got '%s'", libfooTar.Checksum)
	}

	docsFolder := manifest.Dependencies["docs_folder"]
	if docsFolder == nil {
		t.Fatal("docs_folder dependency not found")
	}
	if !docsFolder.Recursive {
		t.Error("Expected docs_folder to be recursive")
	}
}

func TestExpandedPath(t *testing.T) {
	dep := &Dependency{
		Path:    "docs/example-${version}.txt",
		Version: "1.0.0",
	}

	expected := "docs/example-1.0.0.txt"
	if dep.ExpandedPath() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, dep.ExpandedPath())
	}
}

func TestLocalPath(t *testing.T) {
	tests := []struct {
		name      string
		dep       *Dependency
		expected  string
		recursive bool
	}{
		{
			name: "simple file",
			dep: &Dependency{
				Path:      "docs/example-${version}.txt",
				Version:   "1.0.0",
				OutputDir: "./local",
			},
			expected: "local/example-1.0.0.txt",
		},
		{
			name: "recursive folder",
			dep: &Dependency{
				Path:      "docs/${version}/",
				Version:   "2025-10-15",
				OutputDir: "./local",
				Recursive: true,
			},
			expected: "local/docs/",
		},
		{
			name: "with dest override",
			dep: &Dependency{
				Path:      "docs/example-${version}.txt",
				Version:   "1.0.0",
				OutputDir: "./local",
				Dest:      "./vendor/example.txt",
			},
			expected: "./vendor/example.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.dep.LocalPath()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"example_txt", "EXAMPLE_TXT"},
		{"libfoo-tar", "LIBFOO_TAR"},
		{"docs_folder", "DOCS_FOLDER"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeName(tt.input)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestEnvExport(t *testing.T) {
	export := &EnvExport{
		Name:    "example_txt",
		Version: "1.0.0",
		Path:    "./local/example-1.0.0.txt",
	}

	if export.EnvName() != "DEPS_EXAMPLE_TXT_NAME" {
		t.Errorf("Expected 'DEPS_EXAMPLE_TXT_NAME', got '%s'", export.EnvName())
	}
	if export.EnvVersion() != "DEPS_EXAMPLE_TXT_VERSION" {
		t.Errorf("Expected 'DEPS_EXAMPLE_TXT_VERSION', got '%s'", export.EnvVersion())
	}
	if export.EnvPath() != "DEPS_EXAMPLE_TXT_PATH" {
		t.Errorf("Expected 'DEPS_EXAMPLE_TXT_PATH', got '%s'", export.EnvPath())
	}
}

func TestLockFileRoundTrip(t *testing.T) {
	lockFile := &LockFile{
		Dependencies: map[string]map[string]string{
			"example_txt": {
				"docs/example-1.0.0.txt": "sha256:f6a4e3c9b12",
			},
			"libfoo_tar": {
				"thirdparty/libfoo-1.2.3.tar.gz": "sha512:a4c9d2e8abf",
			},
		},
	}

	tmpfile, err := os.CreateTemp("", "deps-lock-*.ini")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	if err := WriteLockFile(tmpfile.Name(), lockFile); err != nil {
		t.Fatalf("WriteLockFile failed: %v", err)
	}

	parsed, err := ParseLockFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("ParseLockFile failed: %v", err)
	}

	if len(parsed.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(parsed.Dependencies))
	}

	if parsed.Dependencies["example_txt"]["docs/example-1.0.0.txt"] != "sha256:f6a4e3c9b12" {
		t.Error("Checksum mismatch for example_txt")
	}
}
