package deps

import (
	"os"
	"testing"

	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

func TestResolverWithMockServer(t *testing.T) {
	mockServer := nexusapi.NewMockNexusServer()
	defer mockServer.Close()

	mockServer.AddAsset("libs", "/docs/example-1.0.0.txt", nexusapi.Asset{
		Checksum: nexusapi.Checksum{
			SHA256: "f6a4e3c9b12",
		},
	}, nil)

	mockServer.AddAsset("libs", "/thirdparty/libfoo-1.2.3.tar.gz", nexusapi.Asset{
		Checksum: nexusapi.Checksum{
			SHA512: "a4c9d2e8abf",
		},
	}, nil)

	mockServer.AddAsset("libs", "/docs/2025-10-15/readme.md", nexusapi.Asset{
		Checksum: nexusapi.Checksum{
			SHA256: "abcd1234",
		},
	}, nil)
	mockServer.AddAsset("libs", "/docs/2025-10-15/guide.pdf", nexusapi.Asset{
		Checksum: nexusapi.Checksum{
			SHA256: "ef125678",
		},
	}, nil)

	client := nexusapi.NewClient(mockServer.URL, "admin", "admin")
	resolver := NewResolver(client)

	t.Run("resolve single file", func(t *testing.T) {
		dep := &Dependency{
			Name:       "example_txt",
			Repository: "libs",
			Path:       "/docs/example-${version}.txt",
			Version:    "1.0.0",
			Checksum:   "sha256",
		}

		files, err := resolver.ResolveDependency(dep)
		if err != nil {
			t.Fatalf("ResolveDependency failed: %v", err)
		}

		if len(files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(files))
		}

		expectedChecksum := "sha256:f6a4e3c9b12"
		if files["docs/example-1.0.0.txt"] != expectedChecksum {
			t.Errorf("Expected checksum '%s', got '%s'", expectedChecksum, files["docs/example-1.0.0.txt"])
		}
	})

	t.Run("resolve recursive folder", func(t *testing.T) {
		dep := &Dependency{
			Name:       "docs_folder",
			Repository: "libs",
			Path:       "/docs/${version}/",
			Version:    "2025-10-15",
			Checksum:   "sha256",
			Recursive:  true,
		}

		files, err := resolver.ResolveDependency(dep)
		if err != nil {
			t.Fatalf("ResolveDependency failed: %v", err)
		}

		if len(files) != 2 {
			t.Errorf("Expected 2 files, got %d", len(files))
		}

		if files["docs/2025-10-15/readme.md"] != "sha256:abcd1234" {
			t.Error("readme.md checksum mismatch")
		}
		if files["docs/2025-10-15/guide.pdf"] != "sha256:ef125678" {
			t.Error("guide.pdf checksum mismatch")
		}
	})
}

func TestCreateTemplateIni(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "deps-template-*.ini")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	if err := CreateTemplateIni(tmpfile.Name()); err != nil {
		t.Fatalf("CreateTemplateIni failed: %v", err)
	}

	manifest, err := ParseDepsIni(tmpfile.Name())
	if err != nil {
		t.Fatalf("ParseDepsIni failed: %v", err)
	}

	if len(manifest.Dependencies) != 3 {
		t.Errorf("Expected 3 dependencies in template, got %d", len(manifest.Dependencies))
	}

	if manifest.Dependencies["example_txt"] == nil {
		t.Error("example_txt not found in template")
	}
	if manifest.Dependencies["libfoo_tar"] == nil {
		t.Error("libfoo_tar not found in template")
	}
	if manifest.Dependencies["docs_folder"] == nil {
		t.Error("docs_folder not found in template")
	}
}

func TestGenerateEnvFile(t *testing.T) {
	manifest := &DepsManifest{
		Defaults: Defaults{
			Repository: "libs",
			Checksum:   "sha256",
			OutputDir:  "./local",
		},
		Dependencies: map[string]*Dependency{
			"example_txt": {
				Name:       "example_txt",
				Path:       "/docs/example-${version}.txt",
				Version:    "1.0.0",
				Repository: "libs",
				OutputDir:  "./local",
			},
		},
	}

	tmpfile, err := os.CreateTemp("", "deps-*.env")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	if err := GenerateEnvFile(tmpfile.Name(), manifest); err != nil {
		t.Fatalf("GenerateEnvFile failed: %v", err)
	}

	content, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expectedContent := "DEPS_EXAMPLE_TXT_NAME=\"example_txt\"\nDEPS_EXAMPLE_TXT_VERSION=\"1.0.0\"\nDEPS_EXAMPLE_TXT_PATH=\"local/docs/example-1.0.0.txt\"\n\n"
	if string(content) != expectedContent {
		t.Errorf("Expected:\n%s\nGot:\n%s", expectedContent, string(content))
	}
}

func TestResolverWithPerDependencyURL(t *testing.T) {
	mockServer1 := nexusapi.NewMockNexusServer()
	defer mockServer1.Close()

	mockServer2 := nexusapi.NewMockNexusServer()
	defer mockServer2.Close()

	mockServer1.AddAsset("libs", "/docs/example-1.0.0.txt", nexusapi.Asset{
		Checksum: nexusapi.Checksum{
			SHA256: "checksum1",
		},
	}, nil)

	mockServer2.AddAsset("libs", "/external/lib-2.0.0.tar.gz", nexusapi.Asset{
		Checksum: nexusapi.Checksum{
			SHA256: "checksum2",
		},
	}, nil)

	client := nexusapi.NewClient(mockServer1.URL, "admin", "admin")
	resolver := NewResolver(client)

	t.Run("dependency with default URL", func(t *testing.T) {
		dep := &Dependency{
			Name:       "example_txt",
			Repository: "libs",
			Path:       "/docs/example-${version}.txt",
			Version:    "1.0.0",
			Checksum:   "sha256",
			URL:        "",
		}

		files, err := resolver.ResolveDependency(dep)
		if err != nil {
			t.Fatalf("ResolveDependency failed: %v", err)
		}

		if len(files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(files))
		}

		expectedChecksum := "sha256:checksum1"
		if files["docs/example-1.0.0.txt"] != expectedChecksum {
			t.Errorf("Expected checksum '%s', got '%s'", expectedChecksum, files["docs/example-1.0.0.txt"])
		}
	})

	t.Run("dependency with custom URL", func(t *testing.T) {
		dep := &Dependency{
			Name:       "external_lib",
			Repository: "libs",
			Path:       "external/lib-${version}.tar.gz",
			Version:    "2.0.0",
			Checksum:   "sha256",
			URL:        mockServer2.URL,
		}

		files, err := resolver.ResolveDependency(dep)
		if err != nil {
			t.Fatalf("ResolveDependency failed: %v", err)
		}

		if len(files) != 1 {
			t.Errorf("Expected 1 file, got %d", len(files))
		}

		expectedChecksum := "sha256:checksum2"
		if files["external/lib-2.0.0.tar.gz"] != expectedChecksum {
			t.Errorf("Expected checksum '%s', got '%s'", expectedChecksum, files["external/lib-2.0.0.tar.gz"])
		}
	})
}
