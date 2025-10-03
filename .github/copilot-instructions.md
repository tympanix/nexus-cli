# Nexus CLI - Copilot Instructions

## Project Overview

Nexus CLI is a command-line tool for uploading and downloading files to/from a Nexus RAW repository, implemented in Go.

## Project Structure

```
nexus-cli/
├── nexuscli-go/           # Go implementation
│   ├── main.go            # Main CLI entry point
│   ├── config.go          # Configuration management
│   ├── nexus_upload.go    # Upload functionality
│   └── nexus_download.go  # Download functionality
├── Makefile               # Build automation
└── README.md              # User documentation
```

## Key Features

1. **Upload**: Upload all files from a directory to a Nexus RAW repository (with optional subdirectory)
2. **Download**: Download all files from a Nexus RAW folder recursively
3. **Authentication**: Uses environment variables for Nexus credentials
4. **Progress tracking**: Shows progress bars during upload/download operations
5. **Checksum validation**: Supports multiple checksum algorithms (SHA1, SHA256, SHA512, MD5)
6. **Parallel downloads**: Downloads multiple files concurrently for improved performance

## Environment Variables

The following environment variables are used for configuration:
- `NEXUS_URL` - Nexus server URL (default: `http://localhost:8081`)
- `NEXUS_USER` - Username for authentication (default: `admin`)
- `NEXUS_PASS` - Password for authentication (default: `admin`)

## Build

### Production Build
From the root of the repository:
```bash
make build
```

This uses [GoReleaser](https://goreleaser.com) to create standalone binaries, DEB packages, and RPM packages in the `dist/` directory.

### Development Build
```bash
cd nexuscli-go
go build -o nexuscli-go
```

### Command Format
```bash
./nexuscli-go upload <src> <dest>
./nexuscli-go download --checksum <algorithm> <src> <dest>
```

### Key Dependencies
- `github.com/spf13/cobra` - CLI framework
- `github.com/schollz/progressbar/v3` - Progress bars

### Code Style
- Follow Go conventions (gofmt, golint)
- Use standard library where possible
- Error handling should be explicit
- No explicit comments unless necessary for clarity

## Development Workflow

1. Navigate to `nexuscli-go/`
2. Build: `go build -o nexuscli-go`
3. Run: `./nexuscli-go <command> <args>`

For production builds, use `make build` from the repository root.

## Common Patterns

### Upload Flow
1. Collect all files from source directory recursively
2. Create multipart form data with file paths and content
3. Send POST request to `/service/rest/v1/components?repository={repository}`
4. Show progress during upload
5. Report success/failure

### Download Flow
1. Query Nexus API for assets in the specified path (using pagination with continuation tokens)
2. For each asset (in parallel using goroutines):
   - Check if file exists locally with matching checksum (skip if match)
   - Download file to local path
   - Show individual progress bar per file
3. Handle pagination automatically for large asset lists

## API Endpoints

- **Upload**: `POST /service/rest/v1/components?repository={repository}`
- **List Assets**: `GET /service/rest/v1/search/assets?repository={repository}&group={path}`
- **Download**: `GET {asset.downloadUrl}` (from asset object)

## Important Conventions

1. **Path Handling**: Always normalize paths to use forward slashes (`/`) for consistency with Nexus
2. **Repository Format**: Destination format is `repository[/subdir]` where subdir is optional
3. **Authentication**: Always use Basic Auth with username/password from environment
4. **Error Handling**: Print clear error messages and use appropriate exit codes
5. **Progress Reporting**: Use progress bars for long-running operations
6. **File Operations**: Handle file handles properly (open/close) to avoid resource leaks

## Testing Considerations

- No existing test infrastructure in this repository
- Manual testing requires a running Nexus instance
- Test with various file structures (nested directories, single files, multiple files)
- Verify checksum validation works correctly

## Commit Conventions

This project uses [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/#specification) for commit messages.

### Commit Message Format
```
<type>: <description>

[optional body]

[optional footer(s)]
```

### Common Types
- `feat` - A new feature
- `fix` - A bug fix
- `docs` - Documentation only changes
- `style` - Changes that do not affect the meaning of the code
- `refactor` - A code change that neither fixes a bug nor adds a feature
- `perf` - A code change that improves performance
- `test` - Adding missing tests or correcting existing tests
- `chore` - Changes to the build process or auxiliary tools

### Notes
- The optional "scope" in commit messages is seldomly used for this project
- Keep commit messages concise and descriptive
- Use the imperative mood in the subject line (e.g., "add feature" not "added feature")

### Examples
```
feat: add support for custom timeout configuration
fix: handle empty directory uploads correctly
docs: update README with new environment variables
refactor: simplify checksum validation logic
```

## Making Changes

When making changes to this project:
1. Test both upload and download functionality
2. Ensure environment variables work correctly
3. Verify progress bars display correctly
4. Handle edge cases (empty directories, missing files, network errors)
5. Follow the Conventional Commits specification for commit messages
6. Update README.md if changing usage or behavior

## Notes

- Supports configurable checksum algorithms via `--checksum` flag (sha1, sha256, sha512, md5)
- SHA1 is used by default (Nexus standard)
- Supports subdirectories in the repository path
- File paths are preserved relative to source directory during upload
- Downloads create necessary parent directories automatically
- Parallel downloads improve performance for large file sets
