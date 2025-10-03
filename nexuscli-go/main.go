package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	// Create config from environment variables
	config := NewConfig()
	var logger Logger
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
				// Use /dev/null for logging when quiet mode is enabled
				devNull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
				logger = NewLogger(devNull)
			} else {
				logger = NewLogger(os.Stdout)
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
			opts := &UploadOptions{
				Logger:    logger,
				QuietMode: quietMode,
			}
			src := args[0]
			dest := args[1]
			uploadMain(src, dest, config, opts)
		},
	}

	var checksumAlg string
	var skipChecksumValidation bool
	var downloadCmd = &cobra.Command{
		Use:   "download <src> <dest>",
		Short: "Download a folder from Nexus RAW",
		Long:  "Download a folder from Nexus RAW\n\nExit codes:\n  0  - Success\n  1  - General error\n  66 - No files found",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			skipChecksumValidation, _ = cmd.Flags().GetBool("skip-checksum")
			opts := &DownloadOptions{
				ChecksumAlgorithm: "sha1", // default
				SkipChecksum:      skipChecksumValidation,
				Logger:            logger,
				QuietMode:         quietMode,
			}
			src := args[0]
			dest := args[1]
			if err := opts.setChecksumAlgorithm(checksumAlg); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			downloadMain(src, dest, config, opts)
		},
	}
	downloadCmd.Flags().StringVarP(&checksumAlg, "checksum", "c", "sha1", "Checksum algorithm to use for validation (sha1, sha256, sha512, md5)")
	downloadCmd.Flags().BoolP("skip-checksum", "s", false, "Skip checksum validation and download files based on file existence")

	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(downloadCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
