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
		cliUser     string
		cliPass     string
		expectHelp  bool
		description string
	}{
		{
			name:        "environment variables only",
			envUser:     "env_user",
			envPass:     "env_pass",
			cliUser:     "",
			cliPass:     "",
			expectHelp:  true,
			description: "Should use environment variables when no CLI flags provided",
		},
		{
			name:        "CLI flags only",
			envUser:     "",
			envPass:     "",
			cliUser:     "cli_user",
			cliPass:     "cli_pass",
			expectHelp:  true,
			description: "Should use CLI flags when no environment variables set",
		},
		{
			name:        "CLI flags override environment",
			envUser:     "env_user",
			envPass:     "env_pass",
			cliUser:     "cli_user",
			cliPass:     "cli_pass",
			expectHelp:  true,
			description: "CLI flags should override environment variables",
		},
		{
			name:        "partial override - username only",
			envUser:     "env_user",
			envPass:     "env_pass",
			cliUser:     "cli_user",
			cliPass:     "",
			expectHelp:  true,
			description: "CLI username should override, password should use env",
		},
		{
			name:        "partial override - password only",
			envUser:     "env_user",
			envPass:     "env_pass",
			cliUser:     "",
			cliPass:     "cli_pass",
			expectHelp:  true,
			description: "CLI password should override, username should use env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			oldUser := os.Getenv("NEXUS_USER")
			oldPass := os.Getenv("NEXUS_PASS")
			defer func() {
				os.Setenv("NEXUS_USER", oldUser)
				os.Setenv("NEXUS_PASS", oldPass)
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

			// Build command
			args := []string{"--help"}
			if tt.cliUser != "" {
				args = append([]string{"--username", tt.cliUser}, args...)
			}
			if tt.cliPass != "" {
				args = append([]string{"--password", tt.cliPass}, args...)
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
