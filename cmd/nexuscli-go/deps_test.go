package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/tympanix/nexus-cli/internal/archive"
	"github.com/tympanix/nexus-cli/internal/config"
	"github.com/tympanix/nexus-cli/internal/operations"
	"github.com/tympanix/nexus-cli/internal/util"
)

func TestDepsInitCommand(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	rootCmd := buildRootCommand()
	rootCmd.SetArgs([]string{"deps", "init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deps init failed: %v", err)
	}

	if _, err := os.Stat("deps.ini"); os.IsNotExist(err) {
		t.Error("deps.ini was not created")
	}

	content, err := os.ReadFile("deps.ini")
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "[defaults]") {
		t.Error("deps.ini missing [defaults] section")
	}
	if !strings.Contains(contentStr, "[example_txt]") {
		t.Error("deps.ini missing [example_txt] section")
	}
}

func TestDepsEnvCommand(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	rootCmd := buildRootCommand()
	rootCmd.SetArgs([]string{"deps", "init"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deps init failed: %v", err)
	}

	rootCmd = buildRootCommand()
	rootCmd.SetArgs([]string{"deps", "env"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("deps env failed: %v", err)
	}

	if _, err := os.Stat("deps.env"); os.IsNotExist(err) {
		t.Error("deps.env was not created")
	}

	content, err := os.ReadFile("deps.env")
	if err != nil {
		t.Fatal(err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "DEPS_EXAMPLE_TXT_NAME") {
		t.Error("deps.env missing DEPS_EXAMPLE_TXT_NAME")
	}
	if !strings.Contains(contentStr, "DEPS_EXAMPLE_TXT_VERSION") {
		t.Error("deps.env missing DEPS_EXAMPLE_TXT_VERSION")
	}
	if !strings.Contains(contentStr, "DEPS_EXAMPLE_TXT_PATH") {
		t.Error("deps.env missing DEPS_EXAMPLE_TXT_PATH")
	}
}

func TestDepsInitAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile("deps.ini", []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		rootCmd := buildRootCommand()
		rootCmd.SetArgs([]string{"deps", "init"})
		rootCmd.Execute()
		return
	}

	content, _ := os.ReadFile("deps.ini")
	if string(content) != "test" {
		t.Error("deps.ini was modified when it should not have been")
	}
}

func TestDepsEnvWithoutDepsIni(t *testing.T) {
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		rootCmd := buildRootCommand()
		rootCmd.SetArgs([]string{"deps", "env"})
		rootCmd.Execute()
		return
	}

	if _, err := os.Stat("deps.env"); err == nil {
		t.Error("deps.env should not have been created without deps.ini")
	}
}

func buildRootCommand() *cobra.Command {
	cfg := config.NewConfig()
	var logger util.Logger
	var quietMode bool
	var verboseMode bool

	uploadOpts := &operations.UploadOptions{}
	var uploadCompressionFormat string
	var uploadChecksumAlg string

	downloadOpts := &operations.DownloadOptions{
		ChecksumAlgorithm: "sha1",
	}
	var downloadCompressionFormat string
	var downloadChecksumAlg string

	var rootCmd = &cobra.Command{
		Use:   "nexuscli-go",
		Short: "Nexus CLI for upload and download",
		Long:  "Nexus CLI for upload and download\n\nExit codes:\n  0  - Success\n  1  - General error\n  66 - No files found (download only)",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cliURL, _ := cmd.Flags().GetString("url")
			cliUsername, _ := cmd.Flags().GetString("username")
			cliPassword, _ := cmd.Flags().GetString("password")
			quietMode, _ = cmd.Flags().GetBool("quiet")
			verboseMode, _ = cmd.Flags().GetBool("verbose")
			if cliURL != "" {
				cfg.NexusURL = cliURL
			}
			if cliUsername != "" {
				cfg.Username = cliUsername
			}
			if cliPassword != "" {
				cfg.Password = cliPassword
			}
			if quietMode {
				logger = util.NewLogger(io.Discard)
			} else if verboseMode {
				logger = util.NewVerboseLogger(os.Stdout)
			} else {
				logger = util.NewLogger(os.Stdout)
			}
			uploadOpts.Logger = logger
			uploadOpts.QuietMode = quietMode
			downloadOpts.Logger = logger
			downloadOpts.QuietMode = quietMode
		},
	}

	rootCmd.PersistentFlags().String("url", "", "URL to Nexus server (defaults to NEXUS_URL env var or 'http://localhost:8081')")
	rootCmd.PersistentFlags().String("username", "", "Username for Nexus authentication (defaults to NEXUS_USER env var or 'admin')")
	rootCmd.PersistentFlags().String("password", "", "Password for Nexus authentication (defaults to NEXUS_PASS env var or 'admin')")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress all output")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

	var uploadCmd = &cobra.Command{
		Use:   "upload <src> <dest>",
		Short: "Upload a directory to Nexus RAW",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if uploadCompressionFormat != "" {
				format, err := archive.Parse(uploadCompressionFormat)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				uploadOpts.CompressionFormat = format
			}
			src := args[0]
			dest := args[1]
			if !uploadOpts.SkipChecksum && uploadChecksumAlg != "" {
				if err := uploadOpts.SetChecksumAlgorithm(uploadChecksumAlg); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			}
			operations.UploadMain(src, dest, cfg, uploadOpts)
		},
	}
	uploadCmd.Flags().BoolVarP(&uploadOpts.Compress, "compress", "z", false, "Create and upload files as a compressed archive")
	uploadCmd.Flags().StringVar(&uploadCompressionFormat, "compress-format", "", "Compression format to use: gzip (default), zstd, or zip")
	uploadCmd.Flags().StringVarP(&uploadOpts.GlobPattern, "glob", "g", "", "Glob pattern(s) to filter files")
	uploadCmd.Flags().StringVar(&uploadOpts.KeyFromFile, "key-from", "", "Path to file to compute hash from for {key} template in dest")
	uploadCmd.Flags().StringVarP(&uploadChecksumAlg, "checksum", "c", "sha1", "Checksum algorithm to use for validation")
	uploadCmd.Flags().BoolVarP(&uploadOpts.SkipChecksum, "skip-checksum", "s", false, "Skip checksum validation")
	uploadCmd.Flags().BoolVar(&uploadOpts.Force, "force", false, "Force upload all files")
	uploadCmd.Flags().BoolVarP(&uploadOpts.DryRun, "dry-run", "n", false, "Perform a dry-run")

	var downloadCmd = &cobra.Command{
		Use:   "download <src> <dest>",
		Short: "Download a folder from Nexus RAW",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if downloadCompressionFormat != "" {
				format, err := archive.Parse(downloadCompressionFormat)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				downloadOpts.CompressionFormat = format
			}
			src := args[0]
			dest := args[1]
			if err := downloadOpts.SetChecksumAlgorithm(downloadChecksumAlg); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			operations.DownloadMain(src, dest, cfg, downloadOpts)
		},
	}
	downloadCmd.Flags().StringVarP(&downloadChecksumAlg, "checksum", "c", "sha1", "Checksum algorithm to use")
	downloadCmd.Flags().BoolVarP(&downloadOpts.SkipChecksum, "skip-checksum", "s", false, "Skip checksum validation")
	downloadCmd.Flags().BoolVarP(&downloadOpts.Flatten, "flatten", "f", false, "Download files without preserving path")
	downloadCmd.Flags().BoolVar(&downloadOpts.DeleteExtra, "delete", false, "Remove extra local files")
	downloadCmd.Flags().BoolVarP(&downloadOpts.Compress, "compress", "z", false, "Download and extract archive")
	downloadCmd.Flags().StringVar(&downloadCompressionFormat, "compress-format", "", "Compression format")
	downloadCmd.Flags().StringVarP(&downloadOpts.GlobPattern, "glob", "g", "", "Glob pattern(s) to filter files")
	downloadCmd.Flags().StringVar(&downloadOpts.KeyFromFile, "key-from", "", "Path to file for {key} template")
	downloadCmd.Flags().BoolVar(&downloadOpts.Force, "force", false, "Force download all files")
	downloadCmd.Flags().BoolVarP(&downloadOpts.DryRun, "dry-run", "n", false, "Perform a dry-run")

	var depsCmd = &cobra.Command{
		Use:   "deps",
		Short: "Dependency management commands",
		Long:  "Manage dependencies using deps.ini, deps-lock.ini, and deps.env files",
	}

	var depsInitCmd = &cobra.Command{
		Use:   "init",
		Short: "Create a template deps.ini file",
		Long:  "Create a template deps.ini file with example dependencies",
		Run: func(cmd *cobra.Command, args []string) {
			depsInitMain()
		},
	}

	var depsLockCmd = &cobra.Command{
		Use:   "lock",
		Short: "Resolve and update deps-lock.ini from deps.ini",
		Long:  "Resolve dependencies from Nexus and write checksums to deps-lock.ini",
		Run: func(cmd *cobra.Command, args []string) {
			depsLockMain(cfg, logger)
		},
	}

	var depsSyncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Download dependencies and verify against deps-lock.ini",
		Long:  "Download dependencies from Nexus and verify checksums atomically (fails if out of sync)",
		Run: func(cmd *cobra.Command, args []string) {
			depsSyncMain(cfg, logger)
		},
	}

	var depsEnvCmd = &cobra.Command{
		Use:   "env",
		Short: "Generate deps.env for shell/Makefile integration",
		Long:  "Generate deps.env file with DEPS_ prefixed variables for shell and Makefile integration",
		Run: func(cmd *cobra.Command, args []string) {
			depsEnvMain(logger)
		},
	}

	depsCmd.AddCommand(depsInitCmd)
	depsCmd.AddCommand(depsLockCmd)
	depsCmd.AddCommand(depsSyncCmd)
	depsCmd.AddCommand(depsEnvCmd)

	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(depsCmd)

	return rootCmd
}
