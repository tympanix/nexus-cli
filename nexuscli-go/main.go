package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: nexuscli-go <upload|download> [options]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "upload":
		uploadCmd := flag.NewFlagSet("upload", flag.ExitOnError)
		src := uploadCmd.String("src", "", "Directory to upload")
		dest := uploadCmd.String("dest", "", "Destination in the form 'repository/subdir' (subdir optional)")
		uploadCmd.Parse(os.Args[2:])
		if *src == "" || *dest == "" {
			fmt.Println("upload requires --src and --dest")
			os.Exit(1)
		}
		uploadMain(*src, *dest)
	case "download":
		downloadCmd := flag.NewFlagSet("download", flag.ExitOnError)
		src := downloadCmd.String("src", "", "Nexus RAW folder to download (e.g. 'myrepo/folder' or 'myrepo/folder/subfolder')")
		dest := downloadCmd.String("dest", "", "Destination directory to save files")
		downloadCmd.Parse(os.Args[2:])
		if *src == "" || *dest == "" {
			fmt.Println("download requires --src and --dest")
			os.Exit(1)
		}
		downloadMain(*src, *dest)
	default:
		fmt.Println("Unknown command:", os.Args[1])
		os.Exit(1)
	}
}
