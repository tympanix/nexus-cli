# Nexus CLI

A command-line tool for uploading and downloading files to/from a Nexus RAW repository.

## Features
- Upload all files from a directory to a Nexus RAW repository (with optional subdirectory)
- Filter files using advanced glob patterns with support for multiple patterns and negation (e.g., `**/*.go,!**/*_test.go`)
- Download all files from a Nexus RAW folder recursively with optional glob pattern filtering
- Compression support: upload/download files as tar.gz, tar.zst, or zip archives
- Parallel downloads for speed
- Clear Unix-style console output with transfer statistics (files transferred, size, time, speed)
- Verbose and quiet output modes
- Shell autocompletion for bash, zsh, fish, and PowerShell with dynamic repository and path suggestions
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

#### DEB (Debian/Ubuntu)

```bash
sudo dpkg -i dist/nexus-cli_*_linux_amd64.deb
```

#### RPM (Red Hat/Fedora)

```bash
sudo rpm -i dist/nexus-cli_*_linux_amd64.rpm
```

#### Standalone Binary

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

### Shell Autocompletion

The CLI provides shell autocompletion support for bash, zsh, fish, and PowerShell. This includes:
- Repository name completion from your Nexus server
- Asset path completion for download commands
- File/directory completion for local paths

#### Setup

Generate and load the autocompletion script for your shell:

**Bash:**
```bash
# Generate completion script
nexuscli-go completion bash > /tmp/nexuscli-go-completion.bash

# Load in current session
source /tmp/nexuscli-go-completion.bash

# Add to your .bashrc for persistent completion
echo 'source /tmp/nexuscli-go-completion.bash' >> ~/.bashrc
```

**Zsh:**
```bash
# Generate completion script
nexuscli-go completion zsh > "${fpath[1]}/_nexuscli-go"

# Reload completions
autoload -U compinit && compinit
```

**Fish:**
```bash
# Generate and load completion
nexuscli-go completion fish | source

# Add to your fish config for persistent completion
nexuscli-go completion fish > ~/.config/fish/completions/nexuscli-go.fish
```

**PowerShell:**
```powershell
# Generate completion script
nexuscli-go completion powershell | Out-String | Invoke-Expression

# Add to PowerShell profile for persistent completion
nexuscli-go completion powershell >> $PROFILE
```

#### Features

- **Repository completion**: When typing repository names, press Tab to see available repositories from your Nexus server
- **Path completion**: When typing paths within repositories (e.g., `my-repo/path/`), press Tab to see available files and directories
- **Smart context**: Completion adapts based on which command you're using (upload vs download) and which argument you're completing

#### Example Usage

```bash
# Complete repository names
nexuscli-go download my-<TAB>
# Shows: my-repo, my-other-repo, my-backup-repo

# Complete paths within a repository
nexuscli-go download my-repo/ar<TAB>
# Shows: my-repo/artifacts/, my-repo/archives/

# Complete local paths for upload source
nexuscli-go upload ./<TAB>
# Shows local directories and files
```

### Authentication

You can authenticate with Nexus using environment variables or CLI flags:

#### Environment variables

- `NEXUS_URL` (default: http://localhost:8081)
- `NEXUS_USER` (default: admin)
- `NEXUS_PASS` (default: admin)

#### CLI flags (take precedence over environment variables)

- `--url <url>` - URL to Nexus server
- `--username <username>` - Username for Nexus authentication
- `--password <password>` - Password for Nexus authentication

### Global Options

These options are available for all commands:

- `--quiet` or `-q` - Suppress all output (no progress bars or informational messages)
- `--verbose` or `-v` - Enable verbose output with detailed information about operations

### Console Output

The CLI provides clear, Unix-style output for file transfer operations, similar to tools like `rsync`, `scp`, and `wget`:

**Normal mode** (default):
- Shows a header line indicating the action and target repository
- Displays per-file status when not showing a progress bar
- Shows a single progress bar for all files during actual transfer (when connected to a TTY)
- Provides a summary after completion with statistics: files transferred, skipped, failed, total size, elapsed time, and average speed

**Verbose mode** (`--verbose` or `-v`):
- Includes additional information such as total file count and total size in the header
- Shows detailed per-file messages for skipped files (with reason)
- Displays individual file paths as they are processed

**Quiet mode** (`--quiet` or `-q`):
- Suppresses all output including progress bars and summary

Example output (normal mode):
```
Uploading to my-repo/path
✓ file1.txt (1.2 KiB, 245.3 KiB/s)
✓ file2.txt (856 B, 198.7 KiB/s)
- file3.txt (skipped)

Files uploaded: 2, skipped: 1, size: 2.0 KiB, time: 1.2s, speed: 1.7 KiB/s
```

### Common Options

The following options are available for both upload and download commands:

#### Checksum validation

- `--checksum <algorithm>` or `-c <algorithm>` - Checksum algorithm to use for validation (sha1, sha256, sha512, md5). Default: sha1
- `--skip-checksum` or `-s` - Skip checksum validation and process files based on file existence only
- `--force` - Force processing all files regardless of existence or checksum match

#### Compression

- `--compress` or `-z` - Create/extract compressed archives
- `--compress-format <format>` - Compression format to use: `gzip` (default), `zstd`, or `zip`

##### Compression formats

- `gzip` (default) - Creates/extracts `.tar.gz` archives (widely compatible)
- `zstd` - Creates/extracts `.tar.zst` archives (better compression ratio and speed)
- `zip` - Creates/extracts `.zip` archives (widely compatible, no tar wrapper)

**For upload:** All files in the source directory are compressed into a single archive before uploading. This is useful for:
- Uploading many small files more efficiently
- Reducing network overhead
- Storing files as a single artifact in Nexus

**For download:** The CLI looks for a compressed archive in the specified path and extracts it to the destination directory. This is useful for:
- Downloading files that were uploaded with compression
- Extracting archives on-the-fly without storing the compressed file locally
- Faster downloads when dealing with many small files

You must specify the archive filename (with extension) as part of the path. The format is auto-detected from the file extension if `--compress-format` is not specified.

#### File filtering with glob patterns

- `--glob <pattern>` or `-g <pattern>` - Glob pattern(s) to filter files (supports multiple patterns and negation)

The `--glob` flag allows you to filter which files are processed using glob patterns. This works for both regular operations and compressed archives. The pattern is matched against file paths relative to the source directory.

##### Multiple patterns and negation

- Use commas to specify multiple patterns: `"**/*.go,**/*.md"`
- Use `!` prefix for negative matches (exclusions): `"**/*.go,!**/*_test.go"`
- Patterns are evaluated left-to-right: positive patterns include files, negative patterns exclude them

##### Supported glob patterns

- `*` - Matches any characters except `/` (directory separator)
- `**` - Matches any characters including `/` (matches directories recursively)
- `?` - Matches any single character
- `[...]` - Matches any character inside the brackets
- `{alt1,alt2}` - Matches any of the alternatives

##### Examples

```bash
# Filter only .txt files
nexuscli-go upload --glob "*.txt" ./files my-repo
nexuscli-go download --glob "**/*.txt" my-repo/files ./local-folder

# Filter all .go files anywhere in the directory tree
nexuscli-go upload --glob "**/*.go" ./files my-repo
nexuscli-go download --glob "**/*.go" my-repo/src ./local-folder

# Multiple file types using multiple patterns
nexuscli-go upload --glob "**/*.go,**/*.md,**/*.txt" ./files my-repo

# Exclude test files (using negation)
nexuscli-go upload --glob "**/*.go,!**/*_test.go" ./files my-repo
nexuscli-go download --glob "**/*.go,!**/*_test.go" my-repo/src ./local-folder

# Exclude specific directories
nexuscli-go upload --glob "!vendor/**,!node_modules/**" ./files my-repo
nexuscli-go download --glob "**/*,!vendor/**,!node_modules/**" my-repo/project ./local-folder

# Complex pattern: include source files but exclude tests and vendor
nexuscli-go upload --glob "**/*.go,**/*.md,!**/*_test.go,!vendor/**" ./files my-repo

# With compressed archives
nexuscli-go upload --compress --glob "**/*.go,!**/*_test.go" ./files my-repo/archive.tar.gz
```

#### Content-based caching with key templates

- `--key-from <file>` - Path to file to compute hash from for `{key}` template in path

The `--key-from` flag enables content-based caching by computing a SHA256 hash from a specified file and using it in the path. This is particularly useful for:
- Caching build artifacts based on dependency files (e.g., `package-lock.json`, `go.sum`)
- Versioning artifacts automatically based on file content
- Avoiding overwrites of existing cached artifacts

When using `--key-from`, you must include the `{key}` template placeholder in your path. The CLI will compute a SHA256 hash of the specified file and replace `{key}` with the hash value.

**Important:** The `{key}` placeholder is required when `--key-from` is specified. If the template is missing, the CLI will exit with an error.

##### Examples

```bash
# Upload with hash from dependency file
nexuscli-go upload --key-from package-lock.json ./node_modules my-repo/cache-{key}
# Uploads to: my-repo/cache-<sha256-hash>/

# Download with same key to retrieve cached version
nexuscli-go download --key-from package-lock.json my-repo/cache-{key} ./node_modules
# Downloads from: my-repo/cache-<sha256-hash>/

# With compression and key-based naming
nexuscli-go upload --compress --key-from package-lock.json ./node_modules my-repo/cache-{key}.tar.gz
nexuscli-go download --compress --key-from package-lock.json my-repo/cache-{key}.tar.gz ./node_modules

# Key in subdirectory path
nexuscli-go upload --key-from go.sum ./vendor my-repo/go-deps/{key}/vendor
nexuscli-go download --key-from go.sum my-repo/go-deps/{key}/vendor ./vendor
```

### Upload

```bash
nexuscli-go upload [options] <directory> <repository[/subdir]>
```

Uploads all files from a local directory to a Nexus RAW repository. Files can be uploaded individually or as a compressed archive.

See [Common Options](#common-options) for available flags: `--checksum`, `--skip-checksum`, `--force`, `--compress`, `--compress-format`, `--glob`, `--key-from`.

#### Examples

```bash
# Basic upload
nexuscli-go upload ./files my-repo/path

# Upload with compression
nexuscli-go upload --compress ./files my-repo/path/backup.tar.gz
nexuscli-go upload --compress --compress-format zstd ./files my-repo/path/backup.tar.zst
nexuscli-go upload --compress --compress-format zip ./files my-repo/path/backup.zip

# Upload filtered files
nexuscli-go upload --glob "**/*.go,!**/*_test.go" ./files my-repo
nexuscli-go upload --compress --glob "**/*.go" ./files my-repo/archive.tar.gz

# Upload with content-based caching
nexuscli-go upload --key-from package-lock.json ./node_modules my-repo/cache-{key}
nexuscli-go upload --compress --key-from package-lock.json ./node_modules my-repo/cache-{key}.tar.gz

# Force upload all files (ignore checksums)
nexuscli-go upload --force ./files my-repo/path

# Skip checksum validation (faster, but less safe)
nexuscli-go upload --skip-checksum ./files my-repo/path

# Use custom checksum algorithm
nexuscli-go upload --checksum sha256 ./files my-repo/path
```

### Download

```bash
nexuscli-go download [options] <repository/folder> <directory>
```

Downloads all files from a Nexus RAW repository folder recursively. Files can be downloaded individually or as a compressed archive that gets extracted.

See [Common Options](#common-options) for available flags: `--checksum`, `--skip-checksum`, `--force`, `--compress`, `--compress-format`, `--glob`, `--key-from`.

#### Download-specific options

- `--flatten` or `-f` - Download files without preserving the base path specified in the source argument
- `--delete` - Remove local files from the destination folder that are not present in Nexus

#### About the `--flatten` flag

By default, when downloading from `repository/path/to/folder`, the entire path structure is preserved locally. For example:
- File at `/path/to/folder/file.txt` in Nexus → saved to `<dest>/path/to/folder/file.txt` locally

With the `--flatten` flag enabled, the base path specified in the source argument is stripped:
- File at `/path/to/folder/file.txt` in Nexus → saved to `<dest>/file.txt` locally
- File at `/path/to/folder/subdir/file.txt` in Nexus → saved to `<dest>/subdir/file.txt` locally (subdirectories beyond the base path are preserved)

#### Examples

```bash
# Basic download
nexuscli-go download my-repo/path ./local-folder

# Download with flatten (remove base path)
nexuscli-go download --flatten my-repo/path ./local-folder

# Download with delete (sync local with remote)
nexuscli-go download --delete my-repo/path ./local-folder
nexuscli-go download --flatten --delete my-repo/path ./local-folder

# Download and extract compressed archive
nexuscli-go download --compress my-repo/path/backup.tar.gz ./local-folder
nexuscli-go download --compress my-repo/path/backup.tar.zst ./local-folder
nexuscli-go download --compress my-repo/path/backup.zip ./local-folder

# Download filtered files
nexuscli-go download --glob "**/*.go" my-repo/src ./local-folder
nexuscli-go download --glob "**/*.go,!**/*_test.go" my-repo/src ./local-folder
nexuscli-go download --flatten --glob "**/*.go" my-repo/src ./local-folder

# Download with content-based caching
nexuscli-go download --key-from package-lock.json my-repo/cache-{key} ./node_modules
nexuscli-go download --compress --key-from package-lock.json my-repo/cache-{key}.tar.gz ./node_modules

# Force download all files (ignore checksums)
nexuscli-go download --force my-repo/path ./local-folder

# Skip checksum validation (faster, but less safe)
nexuscli-go download --skip-checksum my-repo/path ./local-folder

# Use custom checksum algorithm
nexuscli-go download --checksum sha256 my-repo/path ./local-folder

# Using authentication flags
nexuscli-go download --url http://your-nexus:8081 --username myuser --password mypassword my-repo/path ./local-folder
```

## Exit Codes

The CLI uses different exit codes to indicate various outcomes:

- **0** - Success: Operation completed successfully
- **1** - Error: General errors including:
  - Invalid command-line arguments
  - API communication errors
  - Authentication failures
  - Download/upload failures
- **66** - No assets found: The API call succeeded, but returned zero assets
  - This exit code is specific to download operations
  - Indicates the repository path exists but contains no files
  - Distinguishes "empty folder" from "API error"

**Example usage in scripts:**

```bash
#!/bin/bash
nexuscli-go download my-repo/folder ./dest

if [ $? -eq 0 ]; then
  echo "Download successful"
elif [ $? -eq 66 ]; then
  echo "No files found in repository (this may be expected)"
  exit 0  # Treat as success in your script if desired
else
  echo "Download failed with error"
  exit 1
fi
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

#### Requirements

- Docker must be installed and running
- The test takes approximately 1-2 minutes to complete

#### Run the end-to-end test

```bash
make test-e2e
```

Or directly with Go:

```bash
go test -v -run TestEndToEndUploadDownload -timeout 15m ./internal/nexus
```

**Note:** The e2e test is automatically skipped when running `go test -short` or `make test-short`.
