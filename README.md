# Nexus CLI

A command-line tool for uploading and downloading files to/from a Nexus RAW repository.

## Features
- Upload all files from a directory to a Nexus RAW repository (with optional subdirectory)
- Download all files from a Nexus RAW folder recursively
- Compression support: upload/download files as tar.gz archives
- Parallel downloads for speed
- Small container image size using multi-stage build with scratch base

## Building

### Using Docker (Recommended)

Build the Docker image:

```bash
docker build -t nexuscli-go .
```

To build with a specific version:

```bash
docker build --build-arg VERSION=1.0.0 -t nexuscli-go:1.0.0 .
```

> **Note**: The Docker build downloads dependencies during the build process. If you encounter certificate issues in restricted environments, ensure your Docker daemon has proper internet access and CA certificates.

Run upload:

```bash
docker run --rm -v $(pwd):/data \
  -e NEXUS_URL=http://your-nexus:8081 \
  -e NEXUS_USER=admin \
  -e NEXUS_PASS=admin \
  nexuscli-go upload /data/<directory> <repository/subdir>
```

Run download:

```bash
docker run --rm -v $(pwd):/data \
  -e NEXUS_URL=http://your-nexus:8081 \
  -e NEXUS_USER=admin \
  -e NEXUS_PASS=admin \
  nexuscli-go download <repository/folder> /data/<directory>
```

### Native Build

To build the Go CLI locally for development:

```bash
go build -o nexuscli-go ./cmd/nexuscli-go
```

To build with a specific version:

```bash
go build -ldflags "-X main.version=1.0.0" -o nexuscli-go ./cmd/nexuscli-go
```

### Production Build with Packages

From the root of the repository, use the Makefile to build production packages:

```bash
make build
```

This will use [GoReleaser](https://goreleaser.com) to build:
- Standalone binaries for Linux, macOS, and Windows (amd64 and arm64)
- DEB packages for Debian/Ubuntu-based systems
- RPM packages for Red Hat/Fedora-based systems
- Archives (tar.gz) for all platforms

All artifacts are placed in the `dist/` directory.

**Note:** GoReleaser automatically injects the version based on Git tags. When building from a tagged commit (e.g., `v1.0.0`), the version will be set accordingly. For development builds without tags, the version will default to a snapshot version.

### Installing from Packages

**DEB (Debian/Ubuntu):**
```bash
sudo dpkg -i dist/nexus-cli_*_linux_amd64.deb
```

**RPM (Red Hat/Fedora):**
```bash
sudo rpm -i dist/nexus-cli_*_linux_amd64.rpm
```

**Standalone Binary:**
```bash
./dist/nexuscli-go_linux_amd64_v1/nexuscli-go
```

## Running Tests

To run the unit tests:

```bash
make test
```

This will run all tests with verbose output. Alternatively, you can run tests directly using Go:

```bash
go test -v ./...
```

The test suite includes:
- Configuration tests (environment variables and defaults)
- Upload/download functionality tests
- URL construction and encoding tests
- CLI flag parsing and override tests
- Logger functionality tests
- Compression and decompression tests
- Round-trip compression tests

## Usage

### Version

Check the version of the CLI:

```bash
nexuscli-go version
```

### Authentication

You can authenticate with Nexus using environment variables or CLI flags:

**Environment variables:**
- `NEXUS_URL` (default: http://localhost:8081)
- `NEXUS_USER` (default: admin)
- `NEXUS_PASS` (default: admin)

**CLI flags (take precedence over environment variables):**
- `--url <url>` - URL to Nexus server
- `--username <username>` - Username for Nexus authentication
- `--password <password>` - Password for Nexus authentication

### Upload

```bash
nexuscli-go upload [--url <url>] [--username <user>] [--password <pass>] [--compress] <directory> <repository[/subdir]>
```

**Upload options:**
- `--compress` or `-z` - Create and upload files as a compressed tar.gz archive

**About the `--compress` flag:**

When the `--compress` flag is used, all files in the source directory are compressed into a single tar.gz archive before uploading. The archive is named based on the repository and subdirectory (e.g., `my-repo-path.tar.gz`). This is useful for:
- Uploading many small files more efficiently
- Reducing network overhead
- Storing files as a single artifact in Nexus

**Example:**
```bash
nexuscli-go upload --compress ./files my-repo/path
# Creates and uploads: my-repo-path.tar.gz
```

### Download

```bash
nexuscli-go download [--url <url>] [--username <user>] [--password <pass>] [--flatten] [--compress] <repository/folder> <directory>
```

**Download options:**
- `--checksum <algorithm>` or `-c <algorithm>` - Checksum algorithm to use for validation (sha1, sha256, sha512, md5). Default: sha1
- `--skip-checksum` or `-s` - Skip checksum validation and download files based on file existence only
- `--flatten` or `-f` - Download files without preserving the base path specified in the source argument
- `--delete` - Remove local files from the destination folder that are not present in Nexus
- `--compress` or `-z` - Download and extract a compressed tar.gz archive

**About the `--flatten` flag:**

By default, when downloading from `repository/path/to/folder`, the entire path structure is preserved locally. For example:
- File at `/path/to/folder/file.txt` in Nexus → saved to `<dest>/path/to/folder/file.txt` locally

With the `--flatten` flag enabled, the base path specified in the source argument is stripped:
- File at `/path/to/folder/file.txt` in Nexus → saved to `<dest>/file.txt` locally
- File at `/path/to/folder/subdir/file.txt` in Nexus → saved to `<dest>/subdir/file.txt` locally (subdirectories beyond the base path are preserved)

**About the `--compress` flag:**

When the `--compress` flag is used with download, the CLI looks for a tar.gz archive in the specified path and extracts it to the destination directory. This is useful for:
- Downloading files that were uploaded with compression
- Extracting archives on-the-fly without storing the compressed file locally
- Faster downloads when dealing with many small files

The archive name is expected to follow the pattern: `<repository>-<subdir>.tar.gz`

**Example:**
```bash
nexuscli-go download --compress my-repo/path ./local-folder
# Downloads and extracts: my-repo-path.tar.gz
```

**Examples:**

Using environment variables:
```bash
export NEXUS_URL=http://your-nexus:8081
export NEXUS_USER=myuser
export NEXUS_PASS=mypassword
nexuscli-go upload ./files my-repo/path
```

Using CLI flags:
```bash
nexuscli-go upload --url http://your-nexus:8081 --username myuser --password mypassword ./files my-repo/path
```

Download with flatten flag:
```bash
# Without flatten: files are saved with full path structure (my-repo/path/subdir/file.txt)
nexuscli-go download my-repo/path ./local-folder

# With flatten: files are saved without the base path (subdir/file.txt)
nexuscli-go download --flatten my-repo/path ./local-folder
```

Download with delete flag:
```bash
# Downloads files from Nexus and removes local files that are not present in Nexus
nexuscli-go download --delete my-repo/path ./local-folder

# Can be combined with other flags
nexuscli-go download --flatten --delete my-repo/path ./local-folder
```

Upload and download with compression:
```bash
# Upload files as a compressed archive
nexuscli-go upload --compress ./files my-repo/artifacts

# Download and extract the compressed archive
nexuscli-go download --compress my-repo/artifacts ./local-folder
```

Using Docker with CLI flags:
```bash
docker run --rm -v $(pwd):/data \
  nexuscli-go upload --url http://your-nexus:8081 --username myuser --password mypassword /data/<directory> <repository/subdir>
```

## Testing

### Unit Tests

Run unit tests with:

```bash
make test-short
```

Or directly with Go:

```bash
go test -v -short ./...
```

### End-to-End Tests

An end-to-end test is provided that uses a real Nexus instance running in Docker. This test:
- Starts a Sonatype Nexus Docker container
- Waits for Nexus to be ready
- Creates a RAW repository
- Uploads test files using the CLI
- Downloads the files to a new location
- Validates that the downloaded content matches the uploaded content
- Cleans up the Docker container

**Requirements:**
- Docker must be installed and running
- The test takes approximately 1-2 minutes to complete

**Run the end-to-end test:**

```bash
make test-e2e
```

Or directly with Go:

```bash
go test -v -run TestEndToEndUploadDownload -timeout 15m ./internal/nexus
```

**Note:** The e2e test is automatically skipped when running `go test -short` or `make test-short`.
