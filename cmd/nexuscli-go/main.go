package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/tympanix/nexus-cli/internal/nexus"
)

var version = "dev"

type CLIConfig struct {
	url         string
	username    string
	password    string
	quiet       bool
	verbose     bool
	nexusConfig *nexus.Config
	logger      nexus.Logger
}

type UploadCLIOptions struct {
	compress          bool
	compressionFormat string
	globPattern       string
	keyFrom           string
	checksumAlg       string
	skipChecksum      bool
}

type DownloadCLIOptions struct {
	checksumAlg       string
	skipChecksum      bool
	flatten           bool
	deleteExtra       bool
	compress          bool
	compressionFormat string
	keyFrom           string
}

func main() {
	cliConfig := &CLIConfig{
		nexusConfig: nexus.NewConfig(),
	}
	uploadOpts := &UploadCLIOptions{}
	downloadOpts := &DownloadCLIOptions{}

	var rootCmd = &cobra.Command{
		Use:   "nexuscli-go",
		Short: "Nexus CLI for upload and download",
		Long:  "Nexus CLI for upload and download\n\nExit codes:\n  0  - Success\n  1  - General error\n  66 - No files found (download only)",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cliConfig.url, _ = cmd.Flags().GetString("url")
			cliConfig.username, _ = cmd.Flags().GetString("username")
			cliConfig.password, _ = cmd.Flags().GetString("password")
			cliConfig.quiet, _ = cmd.Flags().GetBool("quiet")
			cliConfig.verbose, _ = cmd.Flags().GetBool("verbose")

			if cliConfig.url != "" {
				cliConfig.nexusConfig.NexusURL = cliConfig.url
			}
			if cliConfig.username != "" {
				cliConfig.nexusConfig.Username = cliConfig.username
			}
			if cliConfig.password != "" {
				cliConfig.nexusConfig.Password = cliConfig.password
			}

			if cliConfig.quiet {
				cliConfig.logger = nexus.NewLogger(io.Discard)
			} else if cliConfig.verbose {
				cliConfig.logger = nexus.NewVerboseLogger(os.Stdout)
			} else {
				cliConfig.logger = nexus.NewLogger(os.Stdout)
			}
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
		Run: func(cmd *cobra.Command, args []string) {
			opts := &nexus.UploadOptions{
				Logger:       cliConfig.logger,
				QuietMode:    cliConfig.quiet,
				Compress:     uploadOpts.compress,
				GlobPattern:  uploadOpts.globPattern,
				KeyFromFile:  uploadOpts.keyFrom,
				SkipChecksum: uploadOpts.skipChecksum,
			}
			if uploadOpts.compressionFormat != "" {
				format, err := nexus.ParseCompressionFormat(uploadOpts.compressionFormat)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				opts.CompressionFormat = format
			}
			src := args[0]
			dest := args[1]
			if !uploadOpts.skipChecksum && uploadOpts.checksumAlg != "" {
				if err := opts.SetChecksumAlgorithm(uploadOpts.checksumAlg); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			}
			nexus.UploadMain(src, dest, cliConfig.nexusConfig, opts)
		},
	}
	uploadCmd.Flags().BoolVarP(&uploadOpts.compress, "compress", "z", false, "Create and upload files as a compressed archive")
	uploadCmd.Flags().StringVar(&uploadOpts.compressionFormat, "compress-format", "", "Compression format to use: gzip (default), zstd, or zip")
	uploadCmd.Flags().StringVarP(&uploadOpts.globPattern, "glob", "g", "", "Glob pattern(s) to filter files (e.g., '**/*.go', '**/*.go,**/*.md', '**/*.go,!**/*_test.go')")
	uploadCmd.Flags().StringVar(&uploadOpts.keyFrom, "key-from", "", "Path to file to compute hash from for {key} template in dest")
	uploadCmd.Flags().StringVarP(&uploadOpts.checksumAlg, "checksum", "c", "sha1", "Checksum algorithm to use for validation (sha1, sha256, sha512, md5)")
	uploadCmd.Flags().BoolVarP(&uploadOpts.skipChecksum, "skip-checksum", "s", false, "Skip checksum validation and upload files based on file existence")

	var downloadCmd = &cobra.Command{
		Use:   "download <src> <dest>",
		Short: "Download a folder from Nexus RAW",
		Long:  "Download a folder from Nexus RAW\n\nExit codes:\n  0  - Success\n  1  - General error\n  66 - No files found",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			opts := &nexus.DownloadOptions{
				ChecksumAlgorithm: "sha1",
				SkipChecksum:      downloadOpts.skipChecksum,
				Flatten:           downloadOpts.flatten,
				DeleteExtra:       downloadOpts.deleteExtra,
				Logger:            cliConfig.logger,
				QuietMode:         cliConfig.quiet,
				Compress:          downloadOpts.compress,
				KeyFromFile:       downloadOpts.keyFrom,
			}
			if downloadOpts.compressionFormat != "" {
				format, err := nexus.ParseCompressionFormat(downloadOpts.compressionFormat)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				opts.CompressionFormat = format
			}
			src := args[0]
			dest := args[1]
			if err := opts.SetChecksumAlgorithm(downloadOpts.checksumAlg); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			nexus.DownloadMain(src, dest, cliConfig.nexusConfig, opts)
		},
	}
	downloadCmd.Flags().StringVarP(&downloadOpts.checksumAlg, "checksum", "c", "sha1", "Checksum algorithm to use for validation (sha1, sha256, sha512, md5)")
	downloadCmd.Flags().BoolVarP(&downloadOpts.skipChecksum, "skip-checksum", "s", false, "Skip checksum validation and download files based on file existence")
	downloadCmd.Flags().BoolVarP(&downloadOpts.flatten, "flatten", "f", false, "Download files without preserving the base path specified in the source argument")
	downloadCmd.Flags().BoolVar(&downloadOpts.deleteExtra, "delete", false, "Remove local files from the destination folder that are not present in Nexus")
	downloadCmd.Flags().BoolVarP(&downloadOpts.compress, "compress", "z", false, "Download and extract a compressed archive")
	downloadCmd.Flags().StringVar(&downloadOpts.compressionFormat, "compress-format", "", "Compression format to use: gzip (default), zstd, or zip")
	downloadCmd.Flags().StringVar(&downloadOpts.keyFrom, "key-from", "", "Path to file to compute hash from for {key} template in src")

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
