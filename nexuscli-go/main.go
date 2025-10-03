package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "nexuscli-go",
		Short: "Nexus CLI for upload and download",
		Long:  "Nexus CLI for upload and download\n\nExit codes:\n  0  - Success\n  1  - General error\n  66 - No files found (download only)",
	}

	var uploadCmd = &cobra.Command{
		Use:   "upload <src> <dest>",
		Short: "Upload a directory to Nexus RAW",
		Long:  "Upload a directory to Nexus RAW\n\nExit codes:\n  0 - Success\n  1 - General error",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			quietMode, _ = cmd.Flags().GetBool("quiet")
			src := args[0]
			dest := args[1]
			uploadMain(src, dest)
		},
	}
	uploadCmd.Flags().BoolP("quiet", "q", false, "Suppress all output")

	var checksumAlg string
	var skipChecksumValidation bool
	var downloadCmd = &cobra.Command{
		Use:   "download <src> <dest>",
		Short: "Download a folder from Nexus RAW",
		Long:  "Download a folder from Nexus RAW\n\nExit codes:\n  0  - Success\n  1  - General error\n  66 - No files found",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			quietMode, _ = cmd.Flags().GetBool("quiet")
			skipChecksumValidation, _ = cmd.Flags().GetBool("skip-checksum")
			skipChecksum = skipChecksumValidation
			src := args[0]
			dest := args[1]
			if err := setChecksumAlgorithm(checksumAlg); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			downloadMain(src, dest)
		},
	}
	downloadCmd.Flags().StringVarP(&checksumAlg, "checksum", "c", "sha1", "Checksum algorithm to use for validation (sha1, sha256, sha512, md5)")
	downloadCmd.Flags().BoolP("skip-checksum", "s", false, "Skip checksum validation and download files based on file existence")
	downloadCmd.Flags().BoolP("quiet", "q", false, "Suppress all output")

	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(downloadCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
