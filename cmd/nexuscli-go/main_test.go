package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
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
