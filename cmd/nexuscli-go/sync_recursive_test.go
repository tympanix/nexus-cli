package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tympanix/nexus-cli/internal/checksum"
)

func TestDepsSyncRecursivePaths(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	depsIniContent := `[defaults]
repository = builds
checksum = sha256
output_dir = ./local

[example]
path = test1/
recursive = true
`
	if err := os.WriteFile("deps.ini", []byte(depsIniContent), 0644); err != nil {
		t.Fatal(err)
	}

	testFileContent := []byte("test file content")
	testFilePath := filepath.Join("local", "test1", "sub", "subfile13.out")
	if err := os.MkdirAll(filepath.Dir(testFilePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFilePath, testFileContent, 0644); err != nil {
		t.Fatal(err)
	}

	testChecksum, err := checksum.ComputeChecksum(testFilePath, "sha256")
	if err != nil {
		t.Fatal(err)
	}

	lockFileContent := `[example]
test1/sub/subfile13.out = sha256:` + testChecksum + `
`
	if err := os.WriteFile("deps-lock.ini", []byte(lockFileContent), 0644); err != nil {
		t.Fatal(err)
	}

	rootCmd := buildRootCommand()
	rootCmd.SetArgs([]string{"deps", "env"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deps env failed: %v", err)
	}

	if _, err := os.Stat("deps.env"); os.IsNotExist(err) {
		t.Error("deps.env was not created")
	}
}
