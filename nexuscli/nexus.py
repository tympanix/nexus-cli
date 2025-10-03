#!/usr/bin/env python3
import argparse
import sys
from .nexus_upload import main as upload_main
from .nexus_download import main as download_main

def main():
    parser = argparse.ArgumentParser(
        description="Nexus CLI with upload and download subcommands.",
        epilog="Exit codes: 0=success, 1=general error, 2=command-line usage error, 66=no files found"
    )
    subparsers = parser.add_subparsers(dest="command", required=True)

    # Upload subcommand
    upload_parser = subparsers.add_parser(
        "upload",
        help="Upload all files from a directory to Nexus RAW repository.",
        epilog="Exit codes: 0=success, 1=general error"
    )
    upload_parser.add_argument("src", help="Directory to upload")
    upload_parser.add_argument("dest", help="Destination in the form 'repository/subdir' (subdir optional)")
    upload_parser.add_argument("-q", "--quiet", action="store_true", help="Suppress all output")
    upload_parser.set_defaults(func=upload_main)

    # Download subcommand
    download_parser = subparsers.add_parser(
        "download",
        help="Download all files from a Nexus RAW folder recursively.",
        epilog="Exit codes: 0=success, 1=general error, 66=no files found"
    )
    download_parser.add_argument("src", help="Nexus RAW folder to download (e.g. 'myrepo/folder' or 'myrepo/folder/subfolder')")
    download_parser.add_argument("dest", help="Destination directory to save files")
    download_parser.add_argument("-q", "--quiet", action="store_true", help="Suppress all output")
    download_parser.set_defaults(func=download_main)

    args = parser.parse_args()
    args.func(args)

if __name__ == "__main__":
    main()
