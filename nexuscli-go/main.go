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
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cliUsername, _ := cmd.Flags().GetString("username")
			cliPassword, _ := cmd.Flags().GetString("password")
			if cliUsername != "" {
				username = cliUsername
			}
			if cliPassword != "" {
				password = cliPassword
			}
		},
	}

	rootCmd.PersistentFlags().String("username", "", "Username for Nexus authentication (defaults to NEXUS_USER env var or 'admin')")
	rootCmd.PersistentFlags().String("password", "", "Password for Nexus authentication (defaults to NEXUS_PASS env var or 'admin')")

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
	var downloadCmd = &cobra.Command{
		Use:   "download <src> <dest>",
		Short: "Download a folder from Nexus RAW",
		Long:  "Download a folder from Nexus RAW\n\nExit codes:\n  0  - Success\n  1  - General error\n  66 - No files found",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			quietMode, _ = cmd.Flags().GetBool("quiet")
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
	downloadCmd.Flags().BoolP("quiet", "q", false, "Suppress all output")

	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(downloadCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
