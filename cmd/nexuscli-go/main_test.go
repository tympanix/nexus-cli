package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/tympanix/nexus-cli/internal/config"
	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

func TestCLIFlagsOverrideEnvVars(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "nexuscli-go-test")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("./nexuscli-go-test")

	tests := []struct {
		name        string
		envUser     string
		envPass     string
		envURL      string
		cliUser     string
		cliPass     string
		cliURL      string
		expectHelp  bool
		description string
	}{
		{
			name:        "environment variables only",
			envUser:     "env_user",
			envPass:     "env_pass",
			envURL:      "",
			cliUser:     "",
			cliPass:     "",
			cliURL:      "",
			expectHelp:  true,
			description: "Should use environment variables when no CLI flags provided",
		},
		{
			name:        "CLI flags only",
			envUser:     "",
			envPass:     "",
			envURL:      "",
			cliUser:     "cli_user",
			cliPass:     "cli_pass",
			cliURL:      "",
			expectHelp:  true,
			description: "Should use CLI flags when no environment variables set",
		},
		{
			name:        "CLI flags override environment",
			envUser:     "env_user",
			envPass:     "env_pass",
			envURL:      "",
			cliUser:     "cli_user",
			cliPass:     "cli_pass",
			cliURL:      "",
			expectHelp:  true,
			description: "CLI flags should override environment variables",
		},
		{
			name:        "partial override - username only",
			envUser:     "env_user",
			envPass:     "env_pass",
			envURL:      "",
			cliUser:     "cli_user",
			cliPass:     "",
			cliURL:      "",
			expectHelp:  true,
			description: "CLI username should override, password should use env",
		},
		{
			name:        "partial override - password only",
			envUser:     "env_user",
			envPass:     "env_pass",
			envURL:      "",
			cliUser:     "",
			cliPass:     "cli_pass",
			cliURL:      "",
			expectHelp:  true,
			description: "CLI password should override, username should use env",
		},
		{
			name:        "URL from environment only",
			envUser:     "",
			envPass:     "",
			envURL:      "http://env-nexus:8081",
			cliUser:     "",
			cliPass:     "",
			cliURL:      "",
			expectHelp:  true,
			description: "Should use URL from environment variable",
		},
		{
			name:        "CLI URL overrides environment URL",
			envUser:     "",
			envPass:     "",
			envURL:      "http://env-nexus:8081",
			cliUser:     "",
			cliPass:     "",
			cliURL:      "http://cli-nexus:8081",
			expectHelp:  true,
			description: "CLI URL should override environment URL",
		},
		{
			name:        "all CLI flags override all environment",
			envUser:     "env_user",
			envPass:     "env_pass",
			envURL:      "http://env-nexus:8081",
			cliUser:     "cli_user",
			cliPass:     "cli_pass",
			cliURL:      "http://cli-nexus:8081",
			expectHelp:  true,
			description: "All CLI flags should override all environment variables",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			oldUser := os.Getenv("NEXUS_USER")
			oldPass := os.Getenv("NEXUS_PASS")
			oldURL := os.Getenv("NEXUS_URL")
			defer func() {
				os.Setenv("NEXUS_USER", oldUser)
				os.Setenv("NEXUS_PASS", oldPass)
				os.Setenv("NEXUS_URL", oldURL)
			}()

			if tt.envUser != "" {
				os.Setenv("NEXUS_USER", tt.envUser)
			} else {
				os.Unsetenv("NEXUS_USER")
			}
			if tt.envPass != "" {
				os.Setenv("NEXUS_PASS", tt.envPass)
			} else {
				os.Unsetenv("NEXUS_PASS")
			}
			if tt.envURL != "" {
				os.Setenv("NEXUS_URL", tt.envURL)
			} else {
				os.Unsetenv("NEXUS_URL")
			}

			// Build command
			args := []string{"--help"}
			if tt.cliUser != "" {
				args = append([]string{"--username", tt.cliUser}, args...)
			}
			if tt.cliPass != "" {
				args = append([]string{"--password", tt.cliPass}, args...)
			}
			if tt.cliURL != "" {
				args = append([]string{"--url", tt.cliURL}, args...)
			}

			cmd := exec.Command("./nexuscli-go-test", args...)
			var stdout bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stdout

			// Run command
			err := cmd.Run()
			if err != nil && !tt.expectHelp {
				t.Errorf("Command failed: %v, output: %s", err, stdout.String())
			}

			// Verify help output contains the expected text
			output := stdout.String()
			if tt.expectHelp {
				if !strings.Contains(output, "Nexus CLI for upload and download") {
					t.Errorf("Expected help output, got: %s", output)
				}
			}
		})
	}
}

func TestVersionCommand(t *testing.T) {
	// Build the binary with a custom version
	buildCmd := exec.Command("go", "build", "-ldflags", "-X main.version=test-version", "-o", "nexuscli-go-test-version")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("./nexuscli-go-test-version")

	// Run version command
	cmd := exec.Command("./nexuscli-go-test-version", "version")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("Command failed: %v, output: %s", err, stdout.String())
	}

	output := stdout.String()
	expected := "nexuscli-go version test-version"
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain '%s', got: %s", expected, output)
	}
}

func TestKeyFromFlagExists(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "nexuscli-go-test-keyfrom")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("./nexuscli-go-test-keyfrom")

	tests := []struct {
		command string
		args    []string
	}{
		{
			command: "upload",
			args:    []string{"upload", "--help"},
		},
		{
			command: "download",
			args:    []string{"download", "--help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			cmd := exec.Command("./nexuscli-go-test-keyfrom", tt.args...)
			var stdout bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stdout

			cmd.Run()

			output := stdout.String()
			if !strings.Contains(output, "--key-from") {
				t.Errorf("Expected help output to contain --key-from flag for %s command, got: %s", tt.command, output)
			}
		})
	}
}

func TestVerboseFlag(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "nexuscli-go-test-verbose")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("./nexuscli-go-test-verbose")

	// Test that --verbose flag is available in help
	cmd := exec.Command("./nexuscli-go-test-verbose", "--help")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout

	if err := cmd.Run(); err != nil {
		t.Fatalf("Command failed: %v, output: %s", err, stdout.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "--verbose") {
		t.Errorf("Expected help output to contain '--verbose' flag, got: %s", output)
	}
	if !strings.Contains(output, "Enable verbose output") {
		t.Errorf("Expected help output to contain verbose flag description, got: %s", output)
	}
}

func TestDownloadExitCodes(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "nexuscli-go-test-exitcode")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("./nexuscli-go-test-exitcode")

	tests := []struct {
		name         string
		args         []string
		expectedExit int
		description  string
	}{
		{
			name:         "invalid src format",
			args:         []string{"download", "invalid-format", "/tmp/dest"},
			expectedExit: 1,
			description:  "Invalid src argument should exit with code 1",
		},
		{
			name:         "missing arguments",
			args:         []string{"download"},
			expectedExit: 1,
			description:  "Missing arguments should exit with code 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./nexuscli-go-test-exitcode", tt.args...)
			cmd.Env = append(os.Environ(),
				"NEXUS_URL=http://fake-nexus:8081",
				"NEXUS_USER=test",
				"NEXUS_PASS=test",
			)

			var stdout bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stdout

			err := cmd.Run()
			if err == nil && tt.expectedExit != 0 {
				t.Errorf("Expected command to fail with exit code %d, but it succeeded", tt.expectedExit)
				return
			}

			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode := exitErr.ExitCode()
				if exitCode != tt.expectedExit {
					t.Errorf("Expected exit code %d, got %d. Output: %s", tt.expectedExit, exitCode, stdout.String())
				}
			} else if tt.expectedExit != 0 {
				t.Errorf("Expected exit error with code %d, got: %v", tt.expectedExit, err)
			}
		})
	}
}

func TestAptPackageUpload(t *testing.T) {
	// Build the binary first
	buildCmd := exec.Command("go", "build", "-o", "nexuscli-go-test-apt")
	buildCmd.Dir = "."
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	defer os.Remove("./nexuscli-go-test-apt")

	// Create a test .deb file
	tempDir := t.TempDir()
	debFilePath := tempDir + "/test-package_1.0.0_amd64.deb"

	// Create test .deb file with some content
	debContent := []byte("fake deb package content for testing")
	err := os.WriteFile(debFilePath, debContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test deb file: %v", err)
	}

	tests := []struct {
		name         string
		args         []string
		expectedExit int
		checkOutput  func(output string) error
		description  string
	}{
		{
			name:         "upload deb file without repository",
			args:         []string{"upload", debFilePath},
			expectedExit: 1,
			checkOutput: func(output string) error {
				if !strings.Contains(output, "Error") {
					return fmt.Errorf("expected error message in output")
				}
				return nil
			},
			description: "Should fail when repository is not specified",
		},
		{
			name:         "upload deb file with subdirectory",
			args:         []string{"upload", debFilePath, "apt-repo/subdir"},
			expectedExit: 1,
			checkOutput: func(output string) error {
				if !strings.Contains(output, "subdirectories") {
					return fmt.Errorf("expected subdirectories error in output")
				}
				return nil
			},
			description: "Should fail when destination includes subdirectories",
		},
		{
			name:         "upload deb file with compress flag",
			args:         []string{"upload", "--compress", debFilePath, "apt-repo"},
			expectedExit: 1,
			checkOutput: func(output string) error {
				if !strings.Contains(output, "compression") {
					return fmt.Errorf("expected compression error in output")
				}
				return nil
			},
			description: "Should fail when compression is enabled for apt packages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("./nexuscli-go-test-apt", tt.args...)
			cmd.Env = append(os.Environ(),
				"NEXUS_URL=http://fake-nexus:8081",
				"NEXUS_USER=test",
				"NEXUS_PASS=test",
			)

			var stdout bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stdout

			err := cmd.Run()
			output := stdout.String()

			if tt.expectedExit == 0 {
				if err != nil {
					t.Errorf("Expected command to succeed, got error: %v. Output: %s", err, output)
				}
			} else {
				if err == nil {
					t.Errorf("Expected command to fail with exit code %d, but it succeeded. Output: %s", tt.expectedExit, output)
					return
				}

				if exitErr, ok := err.(*exec.ExitError); ok {
					exitCode := exitErr.ExitCode()
					if exitCode != tt.expectedExit {
						t.Errorf("Expected exit code %d, got %d. Output: %s", tt.expectedExit, exitCode, output)
					}
				}
			}

			if tt.checkOutput != nil {
				if err := tt.checkOutput(output); err != nil {
					t.Errorf("Output check failed: %v. Output: %s", err, output)
				}
			}
		})
	}
}

func TestParseRepoAndPath(t *testing.T) {
	tests := []struct {
		name         string
		arg          string
		expectedRepo string
		expectedPath string
	}{
		{
			name:         "repository only",
			arg:          "myrepo",
			expectedRepo: "myrepo",
			expectedPath: "",
		},
		{
			name:         "repository with slash",
			arg:          "myrepo/",
			expectedRepo: "myrepo",
			expectedPath: "",
		},
		{
			name:         "repository with path",
			arg:          "myrepo/path",
			expectedRepo: "myrepo",
			expectedPath: "path",
		},
		{
			name:         "repository with nested path",
			arg:          "myrepo/path/to/file",
			expectedRepo: "myrepo",
			expectedPath: "path/to/file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, path := parseRepoAndPath(tt.arg)
			if repo != tt.expectedRepo {
				t.Errorf("Expected repo '%s', got '%s'", tt.expectedRepo, repo)
			}
			if path != tt.expectedPath {
				t.Errorf("Expected path '%s', got '%s'", tt.expectedPath, path)
			}
		})
	}
}

func TestCompletionBehavior(t *testing.T) {
	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	server.AddRepository(nexusapi.Repository{Name: "myrepo", Format: "raw", Type: "hosted"})
	server.AddRepository(nexusapi.Repository{Name: "myrepo-test", Format: "raw", Type: "hosted"})
	server.AddRepository(nexusapi.Repository{Name: "other-repo", Format: "raw", Type: "hosted"})

	server.AddAsset("myrepo", "/artifacts/file1.txt", nexusapi.Asset{}, nil)
	server.AddAsset("myrepo", "/archives/file2.txt", nexusapi.Asset{}, nil)

	cfg := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	tests := []struct {
		name                    string
		toComplete              string
		expectedCompletions     []string
		expectsSlash            bool
		expectsNoSpaceDirective bool
	}{
		{
			name:                    "repository completion without slash",
			toComplete:              "my",
			expectedCompletions:     []string{"myrepo/", "myrepo-test/"},
			expectsSlash:            true,
			expectsNoSpaceDirective: true,
		},
		{
			name:                    "repository completion with partial name",
			toComplete:              "myrepo",
			expectedCompletions:     []string{"myrepo/", "myrepo-test/"},
			expectsSlash:            true,
			expectsNoSpaceDirective: true,
		},
		{
			name:                    "path completion after slash",
			toComplete:              "myrepo/",
			expectedCompletions:     []string{"myrepo/artifacts/", "myrepo/archives/"},
			expectsSlash:            false,
			expectsNoSpaceDirective: true,
		},
		{
			name:                    "path completion with prefix",
			toComplete:              "myrepo/ar",
			expectedCompletions:     []string{"myrepo/artifacts/", "myrepo/archives/"},
			expectsSlash:            false,
			expectsNoSpaceDirective: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, pathPrefix := parseRepoAndPath(tt.toComplete)
			var completions []string
			var directive cobra.ShellCompDirective

			if !strings.Contains(tt.toComplete, "/") {
				completions = getRepositoryCompletions(cfg, repo)
				for i := range completions {
					completions[i] = completions[i] + "/"
				}
				directive = cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
			} else {
				completions = getPathCompletions(cfg, repo, pathPrefix)
				for i := range completions {
					completions[i] = repo + completions[i]
				}
				hasDir := false
				for _, comp := range completions {
					if strings.HasSuffix(comp, "/") {
						hasDir = true
						break
					}
				}
				if hasDir {
					directive = cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
				} else {
					directive = cobra.ShellCompDirectiveNoFileComp
				}
			}

			if len(completions) != len(tt.expectedCompletions) {
				t.Errorf("Expected %d completions, got %d: %v", len(tt.expectedCompletions), len(completions), completions)
			}

			completionSet := make(map[string]bool)
			for _, comp := range completions {
				completionSet[comp] = true
			}

			for _, expected := range tt.expectedCompletions {
				if !completionSet[expected] {
					t.Errorf("Expected completion '%s' not found in completions: %v", expected, completions)
				}
			}

			if tt.expectsNoSpaceDirective {
				if directive&cobra.ShellCompDirectiveNoSpace == 0 {
					t.Errorf("Expected ShellCompDirectiveNoSpace to be set, but it wasn't")
				}
			}
		})
	}
}

func TestShellCompletionIntegration(t *testing.T) {
	server := nexusapi.NewMockNexusServer()
	defer server.Close()

	server.AddRepository(nexusapi.Repository{Name: "my-repo", Format: "raw", Type: "hosted"})
	server.AddRepository(nexusapi.Repository{Name: "my-other-repo", Format: "raw", Type: "hosted"})

	server.AddAsset("my-repo", "/files/test.txt", nexusapi.Asset{}, nil)

	cfg := &config.Config{
		NexusURL: server.URL,
		Username: "test",
		Password: "test",
	}

	tests := []struct {
		name                    string
		toComplete              string
		args                    []string
		expectedCompletions     []string
		expectedDirective       cobra.ShellCompDirective
		expectsNoSpaceDirective bool
	}{
		{
			name:                    "complete repository name",
			toComplete:              "my",
			args:                    []string{},
			expectedCompletions:     []string{"my-repo/", "my-other-repo/"},
			expectedDirective:       cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp,
			expectsNoSpaceDirective: true,
		},
		{
			name:                    "complete repository name exact match",
			toComplete:              "my-repo",
			args:                    []string{},
			expectedCompletions:     []string{"my-repo/"},
			expectedDirective:       cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp,
			expectsNoSpaceDirective: true,
		},
		{
			name:                    "complete path after slash",
			toComplete:              "my-repo/",
			args:                    []string{},
			expectedCompletions:     []string{"my-repo/files/"},
			expectedDirective:       cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp,
			expectsNoSpaceDirective: true,
		},
		{
			name:                    "complete path with prefix",
			toComplete:              "my-repo/files",
			args:                    []string{},
			expectedCompletions:     []string{"my-repo/files/"},
			expectedDirective:       cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp,
			expectsNoSpaceDirective: true,
		},
		{
			name:                    "complete file inside directory",
			toComplete:              "my-repo/files/",
			args:                    []string{},
			expectedCompletions:     []string{"my-repo/files/test.txt"},
			expectedDirective:       cobra.ShellCompDirectiveNoFileComp,
			expectsNoSpaceDirective: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, pathPrefix := parseRepoAndPath(tt.toComplete)
			var completions []string
			var directive cobra.ShellCompDirective

			if !strings.Contains(tt.toComplete, "/") {
				completions = getRepositoryCompletions(cfg, repo)
				for i := range completions {
					completions[i] = completions[i] + "/"
				}
				directive = cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
			} else {
				completions = getPathCompletions(cfg, repo, pathPrefix)
				for i := range completions {
					completions[i] = repo + completions[i]
				}
				hasDir := false
				for _, comp := range completions {
					if strings.HasSuffix(comp, "/") {
						hasDir = true
						break
					}
				}
				if hasDir {
					directive = cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
				} else {
					directive = cobra.ShellCompDirectiveNoFileComp
				}
			}

			if len(completions) != len(tt.expectedCompletions) {
				t.Errorf("Expected %d completions, got %d: %v", len(tt.expectedCompletions), len(completions), completions)
			}

			completionSet := make(map[string]bool)
			for _, comp := range completions {
				completionSet[comp] = true
			}

			for _, expected := range tt.expectedCompletions {
				if !completionSet[expected] {
					t.Errorf("Expected completion '%s' not found in completions: %v", expected, completions)
				}
			}

			if directive != tt.expectedDirective {
				t.Errorf("Expected directive %v, got %v", tt.expectedDirective, directive)
			}

			if tt.expectsNoSpaceDirective {
				if directive&cobra.ShellCompDirectiveNoSpace == 0 {
					t.Errorf("Expected ShellCompDirectiveNoSpace to be set, but it wasn't")
				}
			}
		})
	}
}
