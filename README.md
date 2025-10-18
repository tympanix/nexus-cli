# Nexus CLI

A command-line tool for uploading and downloading files to/from a Nexus RAW repository.

## Features
- Upload all files from a directory to a Nexus RAW repository (with optional subdirectory)
- Filter files using advanced glob patterns with support for multiple patterns and negation (e.g., `**/*.txt,!**/*_backup.txt`)
- Download files from a Nexus RAW folder (single file or recursively) with optional glob pattern filtering
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

## Running Tests

Run the unit tests:

```bash
make test
```

The test suite includes configuration, upload/download functionality, URL construction, CLI flag parsing, logger, and compression tests.

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
nexuscli-go completion bash > /tmp/nexuscli-go-completion.bash
source /tmp/nexuscli-go-completion.bash
```

**Zsh:**
```bash
nexuscli-go completion zsh > "${fpath[1]}/_nexuscli-go"
autoload -U compinit && compinit
```

**Fish:**
```bash
nexuscli-go completion fish > ~/.config/fish/completions/nexuscli-go.fish
```

**PowerShell:**
```powershell
nexuscli-go completion powershell | Out-String | Invoke-Expression
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

- Use commas to specify multiple patterns: `"**/*.txt,**/*.md"`
- Use `!` prefix for negative matches (exclusions): `"**/*.txt,!**/*_backup.txt"`
- Patterns are evaluated left-to-right: positive patterns include files, negative patterns exclude them

##### Supported glob patterns

- `*` - Matches any characters except `/` (directory separator)
- `**` - Matches any characters including `/` (matches directories recursively)
- `?` - Matches any single character
- `[...]` - Matches any character inside the brackets
- `{alt1,alt2}` - Matches any of the alternatives

##### Examples

```bash
# Filter specific file types
nexuscli-go upload --glob "*.txt" ./files my-repo
nexuscli-go download --glob "**/*.json" my-repo/config ./local-folder

# Multiple file types
nexuscli-go upload --glob "**/*.md,**/*.txt" ./files my-repo

# Exclude patterns (using negation)
nexuscli-go upload --glob "**/*.txt,!**/*_backup.txt" ./files my-repo

# Exclude specific directories
nexuscli-go upload --glob "!vendor/**,!node_modules/**" ./files my-repo

# With compressed archives
nexuscli-go upload --compress --glob "**/*.json,**/*.yml" ./files my-repo/config.tar.gz
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

# Upload filtered files
nexuscli-go upload --glob "**/*.txt,!**/*_backup.txt" ./files my-repo

# Upload with content-based caching
nexuscli-go upload --key-from package-lock.json ./node_modules my-repo/cache-{key}

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

- `--recursive` or `-r` - Download folder recursively (default: false for single file download)
- `--flatten` or `-f` - Download files without preserving the base path specified in the source argument
- `--delete` - Remove local files from the destination folder that are not present in Nexus

#### About the `--recursive` flag

By default, the download command downloads a single file specified by the exact path. To download all files in a folder recursively, use the `--recursive` or `-r` flag.

**Without `--recursive` (default - single file mode):**
- Downloads only the exact file specified by the path
- Example: `nexuscli-go download my-repo/path/to/file.txt ./local` downloads only `file.txt`

**With `--recursive` flag:**
- Downloads all files within the specified folder and its subdirectories
- Example: `nexuscli-go download --recursive my-repo/path/to/folder ./local` downloads all files under `folder/`

#### About the `--flatten` flag

By default, when downloading from `repository/path/to/folder`, the entire path structure is preserved locally. For example:
- File at `/path/to/folder/file.txt` in Nexus → saved to `<dest>/path/to/folder/file.txt` locally

With the `--flatten` flag enabled, the base path specified in the source argument is stripped:
- File at `/path/to/folder/file.txt` in Nexus → saved to `<dest>/file.txt` locally
- File at `/path/to/folder/subdir/file.txt` in Nexus → saved to `<dest>/subdir/file.txt` locally (subdirectories beyond the base path are preserved)

#### Examples

```bash
# Basic download (single file)
nexuscli-go download my-repo/path/file.txt ./local-folder

# Download folder recursively
nexuscli-go download --recursive my-repo/path ./local-folder

# Download with flatten (remove base path)
nexuscli-go download --flatten my-repo/path ./local-folder

# Download and extract compressed archive
nexuscli-go download --compress my-repo/path/backup.tar.gz ./local-folder

# Download filtered files
nexuscli-go download --glob "**/*.json" my-repo/config ./local-folder

# Download with content-based caching
nexuscli-go download --key-from package-lock.json my-repo/cache-{key} ./node_modules

# Using authentication flags
nexuscli-go download --url http://your-nexus:8081 --username myuser --password mypassword my-repo/path ./local-folder
```

## Dependency Management

Nexus CLI provides a dependency management system for managing external dependencies stored in Nexus repositories. This is useful for:
- Managing build-time dependencies (libraries, tools, SDKs)
- Reproducible builds with locked versions and checksums
- CI/CD pipelines that need to fetch specific artifacts
- Team collaboration with consistent dependency versions

The dependency management system uses three files:
- `deps.ini` - Dependency manifest (version-controlled)
- `deps-lock.ini` - Lock file with resolved checksums (version-controlled)
- `deps.env` - Generated environment variables for shell/Makefile integration (not version-controlled)

### File Formats

#### deps.ini

The `deps.ini` file defines your project's dependencies. It uses INI format with a `[defaults]` section and one section per dependency.

**Format:**
```ini
[defaults]
url = <default-nexus-url>
repository = <default-repository-name>
checksum = <default-checksum-algorithm>
output_dir = <default-output-directory>

[dependency-name]
path = <path-in-nexus>
version = <version-string>
url = <nexus-url>                     # optional, overrides default
repository = <repository-name>        # optional, overrides default
checksum = <checksum-algorithm>       # optional, overrides default
output_dir = <output-directory>       # optional, overrides default
dest = <custom-local-path>            # optional, overrides computed path
recursive = <true|false>              # optional, download folder recursively
```

**Fields:**
- `url` - Nexus server URL (optional, defaults to environment variable `NEXUS_URL`)
- `repository` - Nexus repository name (required in defaults or per-dependency)
- `path` - Path to file or folder in Nexus, supports `${version}` variable substitution
- `version` - Version string, substituted into `${version}` in path
- `checksum` - Checksum algorithm: `sha1`, `sha256` (default), `sha512`, or `md5`
- `output_dir` - Local directory where dependencies are downloaded (default: `./local`). Must be a non-empty subdirectory path. Cannot be `.` (current directory) or `/` (root directory) for safety reasons.
- `dest` - Custom local path (overrides the computed path based on output_dir)
- `recursive` - If `true`, downloads entire folder recursively (for path ending in `/`)

**Example:**
```ini
[defaults]
url = http://localhost:8081
repository = libs
checksum = sha256
output_dir = ./local

[example_txt]
path = docs/example-${version}.txt
version = 1.0.0

[libfoo_tar]
path = thirdparty/libfoo-${version}.tar.gz
version = 1.2.3
checksum = sha512

[docs_folder]
path = docs/${version}/
version = 2025-10-15
recursive = true
```

In this example:
- All dependencies use the Nexus server at `http://localhost:8081`
- `example_txt` downloads `docs/example-1.0.0.txt` to `./local/example-1.0.0.txt`
- `libfoo_tar` downloads `thirdparty/libfoo-1.2.3.tar.gz` to `./local/libfoo-1.2.3.tar.gz` using SHA-512 checksums
- `docs_folder` recursively downloads all files from `docs/2025-10-15/` to `./local/docs/`

**Example with per-dependency URLs:**
```ini
[defaults]
url = http://nexus-primary.example.com:8081
repository = libs
checksum = sha256
output_dir = ./local

[internal_lib]
path = internal/lib-${version}.tar.gz
version = 1.0.0

[external_lib]
path = external/lib-${version}.tar.gz
version = 2.5.0
url = http://nexus-external.example.com:8082
```

In this example:
- `internal_lib` downloads from `http://nexus-primary.example.com:8081` (default URL)
- `external_lib` downloads from `http://nexus-external.example.com:8082` (custom URL)

#### deps-lock.ini

The `deps-lock.ini` file contains resolved file paths and their checksums. It is generated by `nexuscli-go deps lock` and should be committed to version control alongside `deps.ini`.

**Format:**
```ini
[dependency-name]
<file-path> = <algorithm>:<checksum>
<file-path> = <algorithm>:<checksum>
...
```

**Example:**
```ini
[example_txt]
docs/example-1.0.0.txt = sha256:f6a4e3c9b12a8d7e4f1c2b3a4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e

[libfoo_tar]
thirdparty/libfoo-1.2.3.tar.gz = sha512:a4c9d2e8abf7c6b5d4a3e2f1a0b9c8d7e6f5a4b3c2d1e0f9a8b7c6d5e4f3a2b1c0

[docs_folder]
docs/2025-10-15/readme.md = sha256:abcd1234ef567890abcd1234ef567890abcd1234ef567890abcd1234ef567890
docs/2025-10-15/guide.pdf = sha256:ef125678abcd9012ef125678abcd9012ef125678abcd9012ef125678abcd9012
```

This file ensures that every team member and CI/CD system downloads identical files with verified checksums.

#### deps.env

The `deps.env` file contains shell-compatible environment variables generated from `deps.ini`. It is created by `nexuscli-go deps env` and typically not committed to version control.

**Format:**
```bash
DEPS_<NAME>_NAME="<dependency-name>"
DEPS_<NAME>_VERSION="<version>"
DEPS_<NAME>_PATH="<local-path>"
```

**Example:**
```bash
DEPS_EXAMPLE_TXT_NAME="example_txt"
DEPS_EXAMPLE_TXT_VERSION="1.0.0"
DEPS_EXAMPLE_TXT_PATH="local/example-1.0.0.txt"

DEPS_LIBFOO_TAR_NAME="libfoo_tar"
DEPS_LIBFOO_TAR_VERSION="1.2.3"
DEPS_LIBFOO_TAR_PATH="local/libfoo-1.2.3.tar.gz"

DEPS_DOCS_FOLDER_NAME="docs_folder"
DEPS_DOCS_FOLDER_VERSION="2025-10-15"
DEPS_DOCS_FOLDER_PATH="local/docs/"
```

You can source this file in shell scripts or Makefiles:
```bash
# In bash
source deps.env
echo "Using libfoo version: $DEPS_LIBFOO_TAR_VERSION"
```

```makefile
# In Makefile
include deps.env
build:
	gcc -L$(DEPS_LIBFOO_TAR_PATH) main.c
```

### Commands

#### nexuscli-go deps init

Creates a template `deps.ini` file with example dependencies.

```bash
nexuscli-go deps init
```

This generates `deps.ini` in the current directory. Edit the file to define your actual dependencies.

#### nexuscli-go deps lock

Resolves dependencies from Nexus and generates `deps-lock.ini` with checksums.

```bash
nexuscli-go deps lock
```

This command:
1. Reads `deps.ini`
2. Queries Nexus for each dependency to find matching files
3. Retrieves checksums from Nexus
4. Writes `deps-lock.ini` with all file paths and checksums

Run this command whenever you update `deps.ini` or want to update to newer versions of dependencies.

#### nexuscli-go deps sync

Downloads dependencies from Nexus and verifies them against `deps-lock.ini`.

```bash
nexuscli-go deps sync
```

This command:
1. Reads both `deps.ini` and `deps-lock.ini`
2. Downloads each dependency to the specified local path
3. Verifies downloaded files match the checksums in `deps-lock.ini`
4. Removes untracked files from output directories (enabled by default)
5. Fails immediately if any checksum mismatch is detected

This ensures atomic verification - all files are verified against the lock file, guaranteeing consistency.

**Options:**
- `--no-cleanup` - Skip cleanup of untracked files from output directories (cleanup is enabled by default).


#### nexuscli-go deps env

Generates `deps.env` file with environment variables for shell/Makefile integration.

```bash
nexuscli-go deps env
```

This reads `deps.ini` and creates `deps.env` with `DEPS_*` prefixed variables for each dependency.

### Typical Workflow

**Initial setup:**

```bash
# Create deps.ini
nexuscli-go deps init

# Edit deps.ini to define your dependencies
vim deps.ini

# Resolve and lock dependencies
nexuscli-go deps lock

# Commit both files to version control
git add deps.ini deps-lock.ini
git commit -m "Add dependency manifest"
```

**Daily development:**

```bash
# Download and verify dependencies
nexuscli-go deps sync

# Optional: Generate environment variables
nexuscli-go deps env
source deps.env
```

**Updating dependencies:**

```bash
# Edit deps.ini to change versions
vim deps.ini

# Resolve new versions and update lock file
nexuscli-go deps lock

# Verify new dependencies
nexuscli-go deps sync

# Commit updated files
git add deps.ini deps-lock.ini
git commit -m "Update dependency versions"
```

**CI/CD pipeline example:**

```bash
#!/bin/bash
set -e

# Download and verify all dependencies
nexuscli-go deps sync

# Generate environment variables
nexuscli-go deps env

# Source variables for use in build
source deps.env

# Build using dependencies
make build
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

Run unit tests:

```bash
make test-short
```

### End-to-End Tests

Run the end-to-end test (requires Docker):

```bash
make test-e2e
```

The e2e test starts a Nexus Docker container, uploads test files, downloads them to verify functionality, and cleans up.
