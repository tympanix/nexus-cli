package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/tympanix/nexus-cli/internal/nexus"
)

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

	var uploadCmd = &cobra.Command{
		Use:   "upload <src> <dest>",
		Short: "Upload a directory to Nexus RAW",
		Long:  "Upload a directory to Nexus RAW\n\nExit codes:\n  0 - Success\n  1 - General error",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			compress, _ := cmd.Flags().GetBool("compress")
			opts := &nexus.UploadOptions{
				Logger:    logger,
				QuietMode: quietMode,
				Compress:  compress,
			}
			src := args[0]
			dest := args[1]
			nexus.UploadMain(src, dest, config, opts)
		},
	}
	uploadCmd.Flags().BoolP("compress", "z", false, "Compress files into a tar.gz archive before uploading")

	var checksumAlg string
	var skipChecksumValidation bool
	var downloadCmd = &cobra.Command{
		Use:   "download <src> <dest>",
		Short: "Download a folder from Nexus RAW",
		Long:  "Download a folder from Nexus RAW\n\nExit codes:\n  0  - Success\n  1  - General error\n  66 - No files found",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			skipChecksumValidation, _ = cmd.Flags().GetBool("skip-checksum")
			compress, _ := cmd.Flags().GetBool("compress")
			opts := &nexus.DownloadOptions{
				ChecksumAlgorithm: "sha1", // default
				SkipChecksum:      skipChecksumValidation,
				Logger:            logger,
				QuietMode:         quietMode,
				Compress:          compress,
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
	downloadCmd.Flags().BoolP("compress", "z", false, "Download and extract compressed tar.gz archive")

	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(downloadCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
