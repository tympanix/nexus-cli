package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/tympanix/nexus-cli/internal/nexus"
)

var version = "dev"

func main() {
	config := nexus.NewConfig()
	var logger nexus.Logger
	var quietMode bool
	var verboseMode bool

	uploadOpts := &nexus.UploadOptions{}
	var uploadCompressionFormat string
	var uploadChecksumAlg string

	downloadOpts := &nexus.DownloadOptions{
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
				config.NexusURL = cliURL
			}
			if cliUsername != "" {
				config.Username = cliUsername
			}
			if cliPassword != "" {
				config.Password = cliPassword
			}
			if quietMode {
				logger = nexus.NewLogger(io.Discard)
			} else if verboseMode {
				logger = nexus.NewVerboseLogger(os.Stdout)
			} else {
				logger = nexus.NewLogger(os.Stdout)
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
		Run: func(cmd *cobra.Command, args []string) {
			if uploadCompressionFormat != "" {
				format, err := nexus.ParseCompressionFormat(uploadCompressionFormat)
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
			nexus.UploadMain(src, dest, config, uploadOpts)
		},
	}
	uploadCmd.Flags().BoolVarP(&uploadOpts.Compress, "compress", "z", false, "Create and upload files as a compressed archive")
	uploadCmd.Flags().StringVar(&uploadCompressionFormat, "compress-format", "", "Compression format to use: gzip (default), zstd, or zip")
	uploadCmd.Flags().StringVarP(&uploadOpts.GlobPattern, "glob", "g", "", "Glob pattern(s) to filter files (e.g., '**/*.go', '**/*.go,**/*.md', '**/*.go,!**/*_test.go')")
	uploadCmd.Flags().StringVar(&uploadOpts.KeyFromFile, "key-from", "", "Path to file to compute hash from for {key} template in dest")
	uploadCmd.Flags().StringVarP(&uploadChecksumAlg, "checksum", "c", "sha1", "Checksum algorithm to use for validation (sha1, sha256, sha512, md5)")
	uploadCmd.Flags().BoolVarP(&uploadOpts.SkipChecksum, "skip-checksum", "s", false, "Skip checksum validation and upload files based on file existence")
	uploadCmd.Flags().BoolVarP(&uploadOpts.Force, "force", "F", false, "Force upload of all files, ignoring both checksum validation and file existence checks")

	var downloadCmd = &cobra.Command{
		Use:   "download <src> <dest>",
		Short: "Download a folder from Nexus RAW",
		Long:  "Download a folder from Nexus RAW\n\nExit codes:\n  0  - Success\n  1  - General error\n  66 - No files found",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			if downloadCompressionFormat != "" {
				format, err := nexus.ParseCompressionFormat(downloadCompressionFormat)
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
			nexus.DownloadMain(src, dest, config, downloadOpts)
		},
	}
	downloadCmd.Flags().StringVarP(&downloadChecksumAlg, "checksum", "c", "sha1", "Checksum algorithm to use for validation (sha1, sha256, sha512, md5)")
	downloadCmd.Flags().BoolVarP(&downloadOpts.SkipChecksum, "skip-checksum", "s", false, "Skip checksum validation and download files based on file existence")
	downloadCmd.Flags().BoolVarP(&downloadOpts.Flatten, "flatten", "f", false, "Download files without preserving the base path specified in the source argument")
	downloadCmd.Flags().BoolVar(&downloadOpts.DeleteExtra, "delete", false, "Remove local files from the destination folder that are not present in Nexus")
	downloadCmd.Flags().BoolVarP(&downloadOpts.Compress, "compress", "z", false, "Download and extract a compressed archive")
	downloadCmd.Flags().StringVar(&downloadCompressionFormat, "compress-format", "", "Compression format to use: gzip (default), zstd, or zip")
	downloadCmd.Flags().StringVar(&downloadOpts.KeyFromFile, "key-from", "", "Path to file to compute hash from for {key} template in src")
	downloadCmd.Flags().BoolVarP(&downloadOpts.Force, "force", "F", false, "Force download of all files, ignoring both checksum validation and file existence checks")

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
