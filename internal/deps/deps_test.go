package deps

import (
	"os"
	"strings"
	"testing"
)

func TestParseDepsIni(t *testing.T) {
	content := `[defaults]
url = http://nexus.example.com:8081
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

	if manifest.Defaults.URL != "http://nexus.example.com:8081" {
		t.Errorf("Expected URL 'http://nexus.example.com:8081', got '%s'", manifest.Defaults.URL)
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

	if exampleTxt.URL != "http://nexus.example.com:8081" {
		t.Errorf("Expected example_txt to inherit default URL, got '%s'", exampleTxt.URL)
	}
}

func TestParseDepsIniWithPerDependencyURL(t *testing.T) {
	content := `[defaults]
url = http://nexus-default.example.com:8081
repository = libs
checksum = sha256
output_dir = ./local

[example_txt]
path = docs/example-${version}.txt
version = 1.0.0
url = http://nexus-custom.example.com:8082

[libfoo_tar]
path = thirdparty/libfoo-${version}.tar.gz
version = 1.2.3
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

	exampleTxt := manifest.Dependencies["example_txt"]
	if exampleTxt == nil {
		t.Fatal("example_txt dependency not found")
	}
	if exampleTxt.URL != "http://nexus-custom.example.com:8082" {
		t.Errorf("Expected custom URL for example_txt, got '%s'", exampleTxt.URL)
	}

	libfooTar := manifest.Dependencies["libfoo_tar"]
	if libfooTar == nil {
		t.Fatal("libfoo_tar dependency not found")
	}
	if libfooTar.URL != "http://nexus-default.example.com:8081" {
		t.Errorf("Expected libfoo_tar to inherit default URL, got '%s'", libfooTar.URL)
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
			expected: "local/docs/example-1.0.0.txt",
		},
		{
			name: "recursive folder",
			dep: &Dependency{
				Path:      "docs/${version}/",
				Version:   "2025-10-15",
				OutputDir: "./local",
				Recursive: true,
			},
			expected: "local/docs/2025-10-15",
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

func TestLockFileDeterministicOutput(t *testing.T) {
	lockFile := &LockFile{
		Dependencies: map[string]map[string]string{
			"zeta": {
				"path/z.txt": "sha256:checksum_z",
			},
			"alpha": {
				"path/a.txt": "sha256:checksum_a",
			},
			"beta": {
				"path/c.txt": "sha256:checksum_c",
				"path/b.txt": "sha256:checksum_b",
				"path/a.txt": "sha256:checksum_a2",
			},
		},
	}

	var outputs []string
	for i := 0; i < 10; i++ {
		tmpfile, err := os.CreateTemp("", "deps-lock-*.ini")
		if err != nil {
			t.Fatal(err)
		}
		filename := tmpfile.Name()
		tmpfile.Close()
		defer os.Remove(filename)

		if err := WriteLockFile(filename, lockFile); err != nil {
			t.Fatalf("WriteLockFile failed: %v", err)
		}

		content, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		outputs = append(outputs, string(content))
	}

	for i := 1; i < len(outputs); i++ {
		if outputs[i] != outputs[0] {
			t.Errorf("Lock file output is not deterministic.\nFirst output:\n%s\n\nMismatched output:\n%s", outputs[0], outputs[i])
			break
		}
	}
}

func TestValidateOutputDir(t *testing.T) {
	tests := []struct {
		name      string
		outputDir string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid relative path",
			outputDir: "./local",
			wantErr:   false,
		},
		{
			name:      "valid nested path",
			outputDir: "./vendor/deps",
			wantErr:   false,
		},
		{
			name:      "valid absolute path",
			outputDir: "/tmp/deps",
			wantErr:   false,
		},
		{
			name:      "empty string",
			outputDir: "",
			wantErr:   true,
			errMsg:    "output_dir cannot be empty",
		},
		{
			name:      "current directory",
			outputDir: ".",
			wantErr:   true,
			errMsg:    "output_dir cannot be '.' (current directory)",
		},
		{
			name:      "current directory relative",
			outputDir: "./",
			wantErr:   true,
			errMsg:    "output_dir cannot be '.' (current directory)",
		},
		{
			name:      "root directory",
			outputDir: "/",
			wantErr:   true,
			errMsg:    "output_dir cannot be '/' (root directory)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOutputDir(tt.outputDir)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateOutputDir() expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateOutputDir() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateOutputDir() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestParseDepsIniWithEmptyOutputDir(t *testing.T) {
	content := `[defaults]
repository = libs
output_dir = 

[example_txt]
path = docs/example.txt
version = 1.0.0
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

	_, err = ParseDepsIni(tmpfile.Name())
	if err == nil {
		t.Fatal("ParseDepsIni should have failed with empty output_dir")
	}
	if !strings.Contains(err.Error(), "output_dir cannot be empty") {
		t.Errorf("Expected error about empty output_dir, got: %v", err)
	}
}

func TestParseDepsIniWithCurrentDirOutputDir(t *testing.T) {
	content := `[defaults]
repository = libs
output_dir = .

[example_txt]
path = docs/example.txt
version = 1.0.0
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

	_, err = ParseDepsIni(tmpfile.Name())
	if err == nil {
		t.Fatal("ParseDepsIni should have failed with '.' as output_dir")
	}
	if !strings.Contains(err.Error(), "output_dir cannot be '.' (current directory)") {
		t.Errorf("Expected error about current directory, got: %v", err)
	}
}

func TestParseDepsIniWithPerDependencyEmptyOutputDir(t *testing.T) {
	content := `[defaults]
repository = libs
output_dir = ./local

[example_txt]
path = docs/example.txt
version = 1.0.0
output_dir = 
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

	_, err = ParseDepsIni(tmpfile.Name())
	if err == nil {
		t.Fatal("ParseDepsIni should have failed with empty per-dependency output_dir")
	}
	if !strings.Contains(err.Error(), "output_dir cannot be empty") {
		t.Errorf("Expected error about empty output_dir, got: %v", err)
	}
}

func TestParseDepsIniWithValidCustomOutputDir(t *testing.T) {
	content := `[defaults]
repository = libs
output_dir = ./local

[example_txt]
path = docs/example.txt
version = 1.0.0
output_dir = ./custom

[libfoo_tar]
path = thirdparty/libfoo.tar.gz
version = 1.2.3
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

	exampleTxt := manifest.Dependencies["example_txt"]
	if exampleTxt.OutputDir != "./custom" {
		t.Errorf("Expected custom output_dir './custom', got '%s'", exampleTxt.OutputDir)
	}

	libfooTar := manifest.Dependencies["libfoo_tar"]
	if libfooTar.OutputDir != "./local" {
		t.Errorf("Expected default output_dir './local', got '%s'", libfooTar.OutputDir)
	}
}

func TestParseDepsIniWithInvalidDefaultKey(t *testing.T) {
	content := `[defaults]
repository = libs
output_dir = ./local
invalid_key = some_value

[example_txt]
path = docs/example.txt
version = 1.0.0
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

	_, err = ParseDepsIni(tmpfile.Name())
	if err == nil {
		t.Fatal("ParseDepsIni should have failed with invalid key in [defaults]")
	}
	if !strings.Contains(err.Error(), "unknown key 'invalid_key' in [defaults] section") {
		t.Errorf("Expected error about unknown key 'invalid_key' in [defaults], got: %v", err)
	}
}

func TestParseDepsIniWithInvalidDependencyKey(t *testing.T) {
	content := `[defaults]
repository = libs
output_dir = ./local

[example_txt]
path = docs/example.txt
version = 1.0.0
unknown_field = invalid

[libfoo_tar]
path = thirdparty/libfoo.tar.gz
version = 1.2.3
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

	_, err = ParseDepsIni(tmpfile.Name())
	if err == nil {
		t.Fatal("ParseDepsIni should have failed with invalid key in dependency section")
	}
	if !strings.Contains(err.Error(), "unknown key 'unknown_field' in [example_txt] section") {
		t.Errorf("Expected error about unknown key 'unknown_field' in [example_txt], got: %v", err)
	}
}

func TestParseDepsIniWithMultipleInvalidKeys(t *testing.T) {
	content := `[defaults]
repository = libs
output_dir = ./local
bad_key = value

[example_txt]
path = docs/example.txt
version = 1.0.0
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

	_, err = ParseDepsIni(tmpfile.Name())
	if err == nil {
		t.Fatal("ParseDepsIni should have failed with invalid key")
	}
	if !strings.Contains(err.Error(), "unknown key") {
		t.Errorf("Expected error about unknown key, got: %v", err)
	}
}

func TestParseDepsIniWithTypo(t *testing.T) {
	content := `[defaults]
repository = libs
output_dir = ./local

[example_txt]
path = docs/example.txt
version = 1.0.0
repositry = libs
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

	_, err = ParseDepsIni(tmpfile.Name())
	if err == nil {
		t.Fatal("ParseDepsIni should have failed with typo 'repositry' instead of 'repository'")
	}
	if !strings.Contains(err.Error(), "unknown key 'repositry'") {
		t.Errorf("Expected error about unknown key 'repositry', got: %v", err)
	}
}
