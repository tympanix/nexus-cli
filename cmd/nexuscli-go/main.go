package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tympanix/nexus-cli/internal/archive"
	"github.com/tympanix/nexus-cli/internal/config"
	"github.com/tympanix/nexus-cli/internal/nexusapi"
	"github.com/tympanix/nexus-cli/internal/operations"
	"github.com/tympanix/nexus-cli/internal/util"
)

var version = "dev"

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

func main() {
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

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Long:  "Print the version number of nexuscli-go",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("nexuscli-go version %s\n", version)
		},
	}

	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
