package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tympanix/nexus-cli/internal/archive"
	"github.com/tympanix/nexus-cli/internal/checksum"
	"github.com/tympanix/nexus-cli/internal/config"
	"github.com/tympanix/nexus-cli/internal/deps"
	"github.com/tympanix/nexus-cli/internal/nexusapi"
	"github.com/tympanix/nexus-cli/internal/operations"
	"github.com/tympanix/nexus-cli/internal/util"
)

var version = "dev"

func depsInitMain() {
	filename := "deps.ini"
	if _, err := os.Stat(filename); err == nil {
		fmt.Printf("Error: %s already exists\n", filename)
		os.Exit(1)
	}
	if err := deps.CreateTemplateIni(filename); err != nil {
		fmt.Printf("Error creating %s: %v\n", filename, err)
		os.Exit(1)
	}
	fmt.Printf("Created %s\n", filename)
}

func depsLockMain(cfg *config.Config, logger util.Logger) {
	manifest, err := deps.ParseDepsIni("deps.ini")
	if err != nil {
		fmt.Printf("Error parsing deps.ini: %v\n", err)
		os.Exit(1)
	}

	url := cfg.NexusURL
	if manifest.Defaults.URL != "" {
		url = manifest.Defaults.URL
	}

	client := nexusapi.NewClient(url, cfg.Username, cfg.Password)
	resolver := deps.NewResolver(client)

	lockFile := &deps.LockFile{
		Dependencies: make(map[string]map[string]string),
	}

	logger.Printf("=== Resolving Dependencies ===\n")
	totalFiles := 0
	for name, dep := range manifest.Dependencies {
		depURL := url
		if dep.URL != "" {
			depURL = dep.URL
		}
		repo := dep.Repository
		if repo == "" {
			repo = manifest.Defaults.Repository
		}
		checksumAlg := dep.Checksum
		if checksumAlg == "" {
			checksumAlg = manifest.Defaults.Checksum
		}

		logger.Printf("\n[%s]\n", name)
		logger.Printf("  Repository: %s\n", repo)
		logger.Printf("  Path:       %s\n", dep.ExpandedPath())
		logger.Printf("  Checksum:   %s\n", checksumAlg)
		logger.Printf("  Server:     %s\n", depURL)

		files, err := resolver.ResolveDependency(dep)
		if err != nil {
			fmt.Printf("\nError resolving %s: %v\n", name, err)
			os.Exit(1)
		}
		lockFile.Dependencies[name] = files
		totalFiles += len(files)
		logger.Printf("  ✓ Resolved %d file(s)\n", len(files))
	}

	if err := deps.WriteLockFile("deps-lock.ini", lockFile); err != nil {
		fmt.Printf("Error writing deps-lock.ini: %v\n", err)
		os.Exit(1)
	}

	logger.Printf("\n=== Summary ===\n")
	logger.Printf("Dependencies resolved: %d\n", len(manifest.Dependencies))
	logger.Printf("Total files: %d\n", totalFiles)
	logger.Printf("Lock file: deps-lock.ini\n")
}

func depsSyncMain(cfg *config.Config, logger util.Logger, cleanupUntracked bool) {
	manifest, err := deps.ParseDepsIni("deps.ini")
	if err != nil {
		fmt.Printf("Error parsing deps.ini: %v\n", err)
		os.Exit(1)
	}

	lockFile, err := deps.ParseLockFile("deps-lock.ini")
	if err != nil {
		fmt.Printf("Error parsing deps-lock.ini: %v\n", err)
		os.Exit(1)
	}

	trackedFilesByOutputDir := make(map[string]map[string]bool)

	logger.Printf("=== Syncing Dependencies ===\n")
	totalFilesVerified := 0
	for name, dep := range manifest.Dependencies {
		lockedFiles, ok := lockFile.Dependencies[name]
		if !ok {
			fmt.Printf("\nError: dependency %s not found in deps-lock.ini\n", name)
			os.Exit(1)
		}

		depURL := cfg.NexusURL
		if dep.URL != "" {
			depURL = dep.URL
		} else if manifest.Defaults.URL != "" {
			depURL = manifest.Defaults.URL
		}

		repo := dep.Repository
		if repo == "" {
			repo = manifest.Defaults.Repository
		}
		checksumAlg := dep.Checksum
		if checksumAlg == "" {
			checksumAlg = manifest.Defaults.Checksum
		}

		logger.Printf("\n[%s]\n", name)
		logger.Printf("  Repository: %s\n", repo)
		logger.Printf("  Path:       %s\n", dep.ExpandedPath())
		logger.Printf("  Output:     %s\n", dep.OutputDir)
		logger.Printf("  Files:      %d\n", len(lockedFiles))
		logger.Printf("  Checksum:   %s\n", checksumAlg)

		downloadOpts := &operations.DownloadOptions{
			Logger:            logger,
			QuietMode:         false,
			ChecksumAlgorithm: dep.Checksum,
			Recursive:         dep.Recursive,
		}
		if err := downloadOpts.SetChecksumAlgorithm(dep.Checksum); err != nil {
			fmt.Printf("\nError setting checksum algorithm: %v\n", err)
			os.Exit(1)
		}

		src := path.Clean(path.Join(dep.Repository, dep.ExpandedPath()))
		dest := dep.OutputDir

		depCfg := &config.Config{
			NexusURL: depURL,
			Username: cfg.Username,
			Password: cfg.Password,
		}

		operations.DownloadMain(src, dest, depCfg, downloadOpts)

		for filePath := range lockedFiles {
			localPath := filepath.Join(dep.OutputDir, filePath)
			expectedChecksum := lockedFiles[filePath]
			parts := strings.SplitN(expectedChecksum, ":", 2)
			if len(parts) != 2 {
				fmt.Printf("\nError: invalid checksum format in deps-lock.ini: %s\n", expectedChecksum)
				os.Exit(1)
			}
			algorithm := parts[0]
			expected := parts[1]

			actualChecksum, err := checksum.ComputeChecksum(localPath, algorithm)
			if err != nil {
				fmt.Printf("\nError computing checksum for %s: %v\n", localPath, err)
				os.Exit(1)
			}

			if !strings.EqualFold(actualChecksum, expected) {
				fmt.Printf("\nError: checksum mismatch for %s\n", localPath)
				fmt.Printf("  Expected: %s\n", expected)
				fmt.Printf("  Got: %s\n", actualChecksum)
				os.Exit(1)
			}
		}

		totalFilesVerified += len(lockedFiles)

		if cleanupUntracked {
			if trackedFilesByOutputDir[dep.OutputDir] == nil {
				trackedFilesByOutputDir[dep.OutputDir] = make(map[string]bool)
			}
			for filePath := range lockedFiles {
				normalizedPath := strings.TrimLeft(filePath, "/")
				trackedFilesByOutputDir[dep.OutputDir][normalizedPath] = true
			}
		}
	}

	if cleanupUntracked {
		totalDeleted := 0
		for outputDir, trackedFiles := range trackedFilesByOutputDir {
			nDeleted := cleanupUntrackedFiles(outputDir, trackedFiles, logger)
			if nDeleted > 0 {
				totalDeleted += nDeleted
			}
		}
		if totalDeleted > 0 {
			logger.Printf("\nCleaned up %d untracked file(s)\n", totalDeleted)
		}
	}

	logger.Printf("\n=== Summary ===\n")
	logger.Printf("Dependencies synced: %d\n", len(manifest.Dependencies))
	logger.Printf("Total files verified: %d\n", totalFilesVerified)
	logger.Printf("Status: ✓ All checksums valid\n")
}

func cleanupUntrackedFiles(outputDir string, trackedFiles map[string]bool, logger util.Logger) int {
	nDeleted := 0

	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(outputDir, path)
		if err != nil {
			return err
		}

		relPath = filepath.ToSlash(relPath)

		if !trackedFiles[relPath] {
			logger.VerbosePrintf("Deleting untracked file: %s\n", relPath)
			if err := os.Remove(path); err != nil {
				logger.Printf("Failed to delete file %s: %v\n", relPath, err)
			} else {
				nDeleted++
			}
		}

		return nil
	})

	if err != nil {
		logger.Printf("Error walking directory: %v\n", err)
	}

	cleanupEmptyDirectories(outputDir, logger)

	return nDeleted
}

func cleanupEmptyDirectories(outputDir string, logger util.Logger) {
	filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == outputDir {
			return nil
		}

		if info.IsDir() {
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil
			}

			if len(entries) == 0 {
				logger.VerbosePrintf("Removing empty directory: %s\n", path)
				os.Remove(path)
			}
		}

		return nil
	})
}

func depsEnvMain(logger util.Logger) {
	manifest, err := deps.ParseDepsIni("deps.ini")
	if err != nil {
		fmt.Printf("Error parsing deps.ini: %v\n", err)
		os.Exit(1)
	}

	if err := deps.GenerateEnvFile("deps.env", manifest); err != nil {
		fmt.Printf("Error generating deps.env: %v\n", err)
		os.Exit(1)
	}

	logger.Printf("Generated deps.env\n")
}

func getRepositoryCompletions(cfg *config.Config, toComplete string) []string {
	client := nexusapi.NewClient(cfg.NexusURL, cfg.Username, cfg.Password)
	repos, err := client.ListRepositories()
	if err != nil {
		return nil
	}
	var completions []string
	for _, repo := range repos {
		if strings.HasPrefix(repo.Name, toComplete) {
			completions = append(completions, repo.Name)
		}
	}
	return completions
}

func getPathCompletions(cfg *config.Config, repository, pathPrefix string) []string {
	client := nexusapi.NewClient(cfg.NexusURL, cfg.Username, cfg.Password)
	paths, err := client.SearchAssetsForCompletion(repository, pathPrefix)
	if err != nil {
		return nil
	}
	return paths
}

func parseRepoAndPath(arg string) (string, string) {
	parts := strings.SplitN(arg, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
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
		Long:  "Upload a directory to Nexus RAW\n\nExit codes:\n  0 - Success\n  1 - General error",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return nil, cobra.ShellCompDirectiveDefault | cobra.ShellCompDirectiveFilterDirs
			}
			if len(args) == 1 {
				repo, pathPrefix := parseRepoAndPath(toComplete)
				if !strings.Contains(toComplete, "/") {
					completions := getRepositoryCompletions(cfg, repo)
					for i := range completions {
						completions[i] = completions[i] + "/"
					}
					return completions, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
				}
				completions := getPathCompletions(cfg, repo, pathPrefix)
				for i := range completions {
					completions[i] = path.Join(repo, completions[i])
				}
				hasDir := false
				for _, comp := range completions {
					if strings.HasSuffix(comp, "/") {
						hasDir = true
						break
					}
				}
				if hasDir {
					return completions, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
				}
				return completions, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
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
	uploadCmd.Flags().StringVarP(&uploadOpts.GlobPattern, "glob", "g", "", "Glob pattern(s) to filter files (e.g., '**/*.go', '**/*.go,**/*.md', '**/*.go,!**/*_test.go')")
	uploadCmd.Flags().StringVar(&uploadOpts.KeyFromFile, "key-from", "", "Path to file to compute hash from for {key} template in dest")
	uploadCmd.Flags().StringVarP(&uploadChecksumAlg, "checksum", "c", "sha1", "Checksum algorithm to use for validation (sha1, sha256, sha512, md5)")
	uploadCmd.Flags().BoolVarP(&uploadOpts.SkipChecksum, "skip-checksum", "s", false, "Skip checksum validation and upload files based on file existence")
	uploadCmd.Flags().BoolVar(&uploadOpts.Force, "force", false, "Force upload all files regardless of existence or checksum match")
	uploadCmd.Flags().BoolVarP(&uploadOpts.DryRun, "dry-run", "n", false, "Perform a dry-run without actually uploading files")

	var downloadCmd = &cobra.Command{
		Use:   "download <src> <dest>",
		Short: "Download a folder from Nexus RAW",
		Long:  "Download a folder from Nexus RAW\n\nExit codes:\n  0  - Success\n  1  - General error\n  66 - No files found",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				repo, pathPrefix := parseRepoAndPath(toComplete)
				if !strings.Contains(toComplete, "/") {
					completions := getRepositoryCompletions(cfg, repo)
					for i := range completions {
						completions[i] = completions[i] + "/"
					}
					return completions, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
				}
				completions := getPathCompletions(cfg, repo, pathPrefix)
				for i := range completions {
					completions[i] = path.Join(repo, completions[i])
				}
				hasDir := false
				for _, comp := range completions {
					if strings.HasSuffix(comp, "/") {
						hasDir = true
						break
					}
				}
				if hasDir {
					return completions, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
				}
				return completions, cobra.ShellCompDirectiveNoFileComp
			}
			if len(args) == 1 {
				return nil, cobra.ShellCompDirectiveDefault | cobra.ShellCompDirectiveFilterDirs
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
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
	downloadCmd.Flags().StringVarP(&downloadChecksumAlg, "checksum", "c", "sha1", "Checksum algorithm to use for validation (sha1, sha256, sha512, md5)")
	downloadCmd.Flags().BoolVarP(&downloadOpts.SkipChecksum, "skip-checksum", "s", false, "Skip checksum validation and download files based on file existence")
	downloadCmd.Flags().BoolVarP(&downloadOpts.Flatten, "flatten", "f", false, "Download files without preserving the base path specified in the source argument")
	downloadCmd.Flags().BoolVar(&downloadOpts.DeleteExtra, "delete", false, "Remove local files from the destination folder that are not present in Nexus")
	downloadCmd.Flags().BoolVarP(&downloadOpts.Compress, "compress", "z", false, "Download and extract a compressed archive")
	downloadCmd.Flags().StringVar(&downloadCompressionFormat, "compress-format", "", "Compression format to use: gzip (default), zstd, or zip")
	downloadCmd.Flags().StringVarP(&downloadOpts.GlobPattern, "glob", "g", "", "Glob pattern(s) to filter files (e.g., '**/*.go', '**/*.go,**/*.md', '**/*.go,!**/*_test.go')")
	downloadCmd.Flags().StringVar(&downloadOpts.KeyFromFile, "key-from", "", "Path to file to compute hash from for {key} template in src")
	downloadCmd.Flags().BoolVar(&downloadOpts.Force, "force", false, "Force download all files regardless of existence or checksum match")
	downloadCmd.Flags().BoolVarP(&downloadOpts.DryRun, "dry-run", "n", false, "Perform a dry-run without actually downloading files")
	downloadCmd.Flags().BoolVarP(&downloadOpts.Recursive, "recursive", "r", false, "Download folder recursively (default: false for single file download)")

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Long:  "Print the version number of nexuscli-go",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("nexuscli-go version %s\n", version)
		},
	}

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

	var depsSyncNoCleanup bool
	var depsSyncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Download dependencies and verify against deps-lock.ini",
		Long:  "Download dependencies from Nexus and verify checksums atomically (fails if out of sync)",
		Run: func(cmd *cobra.Command, args []string) {
			depsSyncMain(cfg, logger, !depsSyncNoCleanup)
		},
	}
	depsSyncCmd.Flags().BoolVar(&depsSyncNoCleanup, "no-cleanup", false, "Skip cleanup of untracked files from output directory")

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
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(depsCmd)

	return rootCmd
}

func main() {
	rootCmd := buildRootCommand()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
