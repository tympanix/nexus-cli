# Nexus CLI

A command-line tool for uploading and downloading files to/from a Nexus RAW repository.

## Features
- Upload all files from a directory to a Nexus RAW repository (with optional subdirectory)
- Download all files from a Nexus RAW folder recursively
- **Compression support**: Upload and download files as compressed tar.gz archives
- Parallel downloads for speed
- Small container image size using multi-stage build with scratch base

## Building

### Using Docker (Recommended)

Build the Docker image:

```bash
docker build -t nexuscli-go .
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
- Compression/decompression tests
- URL construction and encoding tests
- CLI flag parsing and override tests
- Logger functionality tests

## Usage

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

Upload files to a Nexus RAW repository:

```bash
nexuscli-go upload [flags] <directory> <repository[/subdir]>
```

**Flags:**
- `-z, --compress` - Compress files into a tar.gz archive before uploading
- `-q, --quiet` - Suppress all output
- `--url <url>` - URL to Nexus server (overrides NEXUS_URL env var)
- `--username <user>` - Username for authentication (overrides NEXUS_USER env var)
- `--password <pass>` - Password for authentication (overrides NEXUS_PASS env var)

**Examples:**

Upload individual files:
```bash
nexuscli-go upload ./files my-repo/path
```

Upload as a compressed archive:
```bash
nexuscli-go upload --compress ./files my-repo/path
```
This creates a tar.gz archive named `files.tar.gz` containing all files from the `./files` directory.

### Download

Download files from a Nexus RAW repository:

```bash
nexuscli-go download [flags] <repository/folder> <directory>
```

**Flags:**
- `-z, --compress` - Download and extract a compressed tar.gz archive
- `-c, --checksum <algorithm>` - Checksum algorithm for validation (sha1, sha256, sha512, md5; default: sha1)
- `-s, --skip-checksum` - Skip checksum validation (only check file existence)
- `-q, --quiet` - Suppress all output
- `--url <url>` - URL to Nexus server (overrides NEXUS_URL env var)
- `--username <user>` - Username for authentication (overrides NEXUS_USER env var)
- `--password <pass>` - Password for authentication (overrides NEXUS_PASS env var)

**Examples:**

Download individual files:
```bash
nexuscli-go download my-repo/path ./files
```

Download and extract a compressed archive:
```bash
nexuscli-go download --compress my-repo/path/files.tar.gz ./files
```
This downloads `files.tar.gz` and extracts all files to the `./files` directory.

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

Uploading compressed archives:
```bash
# Upload as compressed archive
nexuscli-go upload --compress ./my-files my-repo/backups

# Download and extract the archive
nexuscli-go download --compress my-repo/backups/my-files.tar.gz ./restored-files
```

Using Docker with CLI flags:
```bash
docker run --rm -v $(pwd):/data \
  nexuscli-go upload --url http://your-nexus:8081 --username myuser --password mypassword /data/<directory> <repository/subdir>
```

## Compression Feature

The compression feature allows you to upload and download files as tar.gz archives, which can be useful for:
- Reducing storage space in Nexus
- Faster uploads/downloads when dealing with many small files
- Preserving directory structure in a single artifact
- Atomic operations (one archive = one upload/download)

### How It Works

**Upload with compression (`-z` flag):**
1. All files from the source directory are collected recursively
2. Files are compressed on-the-fly into a tar.gz archive
3. The archive is named using the source directory name (e.g., `mydir` → `mydir.tar.gz`)
4. A single compressed archive is uploaded to Nexus
5. Progress bar shows compression and upload progress

**Download with compression (`-z` flag):**
1. The tar.gz archive is downloaded from Nexus
2. Files are extracted on-the-fly to the destination directory
3. Original directory structure and file permissions are preserved
4. Progress bar shows download and extraction progress

**Note:** When using compression, no intermediate files are created on disk. All compression and decompression happens in memory as data is streamed.

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
