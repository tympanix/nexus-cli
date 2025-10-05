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
	// Create config from environment variables
	config := nexus.NewConfig()
	var logger nexus.Logger
	var quietMode bool

	var rootCmd = &cobra.Command{
		Use:   "nexuscli-go",
		Short: "Nexus CLI for upload and download",
		Long:  "Nexus CLI for upload and download\n\nExit codes:\n  0  - Success\n  1  - General error\n  66 - No files found (download only)",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cliURL, _ := cmd.Flags().GetString("url")
			cliUsername, _ := cmd.Flags().GetString("username")
			cliPassword, _ := cmd.Flags().GetString("password")
			quietMode, _ = cmd.Flags().GetBool("quiet")
			if cliURL != "" {
				config.NexusURL = cliURL
			}
			if cliUsername != "" {
				config.Username = cliUsername
			}
			if cliPassword != "" {
				config.Password = cliPassword
			}
			// Configure logger based on quiet mode
			if quietMode {
				logger = nexus.NewLogger(io.Discard)
			} else {
				logger = nexus.NewLogger(os.Stdout)
			}
		},
	}

	rootCmd.PersistentFlags().String("url", "", "URL to Nexus server (defaults to NEXUS_URL env var or 'http://localhost:8081')")
	rootCmd.PersistentFlags().String("username", "", "Username for Nexus authentication (defaults to NEXUS_USER env var or 'admin')")
	rootCmd.PersistentFlags().String("password", "", "Password for Nexus authentication (defaults to NEXUS_PASS env var or 'admin')")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress all output")

	var uploadCompress bool
	var uploadCompressionFormat string
	var uploadGlobPattern string
	var uploadCmd = &cobra.Command{
		Use:   "upload <src> <dest>",
		Short: "Upload a directory to Nexus RAW",
		Long:  "Upload a directory to Nexus RAW\n\nExit codes:\n  0 - Success\n  1 - General error",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			opts := &nexus.UploadOptions{
				Logger:      logger,
				QuietMode:   quietMode,
				Compress:    uploadCompress,
				GlobPattern: uploadGlobPattern,
			}
			// Parse compression format if provided
			if uploadCompressionFormat != "" {
				format, err := nexus.ParseCompressionFormat(uploadCompressionFormat)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				opts.CompressionFormat = format
			}
			src := args[0]
			dest := args[1]
			nexus.UploadMain(src, dest, config, opts)
		},
	}
	uploadCmd.Flags().BoolVarP(&uploadCompress, "compress", "z", false, "Create and upload files as a compressed archive")
	uploadCmd.Flags().StringVar(&uploadCompressionFormat, "compress-format", "", "Compression format to use: gzip (default), zstd, or zip")
	uploadCmd.Flags().StringVarP(&uploadGlobPattern, "glob", "g", "", "Glob pattern to filter files (e.g., '**/*.go', '*.txt')")

	var checksumAlg string
	var skipChecksumValidation bool
	var flattenPath bool
	var deleteExtra bool
	var compressDownload bool
	var downloadCompressionFormat string
	var downloadCmd = &cobra.Command{
		Use:   "download <src> <dest>",
		Short: "Download a folder from Nexus RAW",
		Long:  "Download a folder from Nexus RAW\n\nExit codes:\n  0  - Success\n  1  - General error\n  66 - No files found",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			skipChecksumValidation, _ = cmd.Flags().GetBool("skip-checksum")
			flattenPath, _ = cmd.Flags().GetBool("flatten")
			deleteExtra, _ = cmd.Flags().GetBool("delete")
			compressDownload, _ = cmd.Flags().GetBool("compress")
			opts := &nexus.DownloadOptions{
				ChecksumAlgorithm: "sha1", // default
				SkipChecksum:      skipChecksumValidation,
				Flatten:           flattenPath,
				DeleteExtra:       deleteExtra,
				Logger:            logger,
				QuietMode:         quietMode,
				Compress:          compressDownload,
			}
			// Parse compression format if provided
			if downloadCompressionFormat != "" {
				format, err := nexus.ParseCompressionFormat(downloadCompressionFormat)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				opts.CompressionFormat = format
			}
			src := args[0]
			dest := args[1]
			if err := opts.SetChecksumAlgorithm(checksumAlg); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			nexus.DownloadMain(src, dest, config, opts)
		},
	}
	downloadCmd.Flags().StringVarP(&checksumAlg, "checksum", "c", "sha1", "Checksum algorithm to use for validation (sha1, sha256, sha512, md5)")
	downloadCmd.Flags().BoolP("skip-checksum", "s", false, "Skip checksum validation and download files based on file existence")
	downloadCmd.Flags().BoolP("flatten", "f", false, "Download files without preserving the base path specified in the source argument")
	downloadCmd.Flags().Bool("delete", false, "Remove local files from the destination folder that are not present in Nexus")
	downloadCmd.Flags().BoolP("compress", "z", false, "Download and extract a compressed archive")
	downloadCmd.Flags().StringVar(&downloadCompressionFormat, "compress-format", "", "Compression format to use: gzip (default), zstd, or zip")

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
