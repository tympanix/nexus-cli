# Nexus CLI - Copilot Instructions

## Project Overview

Nexus CLI is a command-line tool for uploading and downloading files to/from a Nexus RAW repository. The project includes two implementations:
- **Python implementation** (`nexuscli/`) - The original implementation
- **Go implementation** (`nexuscli-go/`) - A translation of the Python tool with similar functionality

## Project Structure

```
nexus-cli/
├── nexuscli/              # Python implementation
│   ├── __init__.py
│   ├── __main__.py
│   ├── nexus.py           # Main CLI entry point
│   ├── nexus_upload.py    # Upload functionality
│   └── nexus_download.py  # Download functionality
├── nexuscli-go/           # Go implementation
│   ├── main.go            # Main CLI entry point
│   ├── config.go          # Configuration management
│   ├── nexus_upload.go    # Upload functionality
│   └── nexus_download.go  # Download functionality
├── pyproject.toml         # Python project configuration
├── Makefile               # Build automation
└── README.md              # User documentation
```

## Key Features

1. **Upload**: Upload all files from a directory to a Nexus RAW repository (with optional subdirectory)
2. **Download**: Download all files from a Nexus RAW folder recursively
3. **Authentication**: Uses environment variables for Nexus credentials
4. **Progress tracking**: Shows progress bars during upload/download operations
5. **Checksum validation**: Go implementation supports multiple checksum algorithms (SHA1, SHA256, SHA512, MD5)

## Environment Variables

Both implementations use the same environment variables:
- `NEXUS_URL` - Nexus server URL (default: `http://localhost:8081`)
- `NEXUS_USER` - Username for authentication (default: `admin`)
- `NEXUS_PASS` - Password for authentication (default: `admin`)

## Python Implementation

### Installation
```bash
pip install .
```

### Command Format
```bash
nexus upload <directory> <repository[/subdir]>
nexus download <repository/folder> <dest>
```

### Key Dependencies
- `requests` - HTTP client
- `tqdm` - Progress bars
- `requests-toolbelt` - Multipart file uploads

### Code Style
- Use Python 3.8+ features
- Type hints are used where appropriate (`Optional`, `List`, etc.)
- Follow PEP 8 conventions
- No explicit comments unless necessary for clarity

## Go Implementation

### Build
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

### Python Development
1. Create a virtual environment: `python3 -m venv venv`
2. Activate: `. venv/bin/activate`
3. Install: `pip install .`
4. Run: `nexus <command> <args>`

Or use the Makefile:
```bash
make venv
```

### Go Development
1. Navigate to `nexuscli-go/`
2. Build: `go build -o nexuscli-go`
3. Run: `./nexuscli-go <command> <args>`

## Common Patterns

### Upload Flow
1. Collect all files from source directory recursively
2. Create multipart form data with file paths and content
3. Send POST request to `/service/rest/v1/components?repository={repository}`
4. Show progress during upload
5. Report success/failure

### Download Flow
1. Query Nexus API for assets in the specified path (using pagination with continuation tokens)
2. For each asset (in parallel using ThreadPoolExecutor/goroutines):
   - Check if file exists locally with matching checksum (skip if match - Go only)
   - Download file to local path
   - Show individual progress bar per file
3. Both implementations support parallel downloads for efficiency
4. Handle pagination automatically for large asset lists

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
1. Maintain compatibility between Python and Go implementations where applicable
2. Keep command-line interfaces similar
3. Update both READMEs if changing usage or behavior
4. Test both upload and download functionality
5. Ensure environment variables work correctly
6. Verify progress bars display correctly
7. Handle edge cases (empty directories, missing files, network errors)
8. Follow the Conventional Commits specification for commit messages

## Notes

- The Go implementation supports configurable checksum algorithms via `--checksum` flag
- The Python implementation uses SHA1 by default (Nexus standard)
- Both implementations support subdirectories in the repository path
- File paths are preserved relative to source directory during upload
- Downloads create necessary parent directories automatically
