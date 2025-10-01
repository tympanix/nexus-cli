# Nexus CLI Go implementation

This folder contains a Go translation of the Python nexuscli tool. It provides the same upload and download features, using similar command-line arguments and environment variables for configuration.

## Usage

Build the Go CLI:

```
go build -o nexuscli-go
```

Run upload:

```
./nexuscli-go upload --src <directory> --dest <repository/subdir>
```

Run download:

```
./nexuscli-go download --src <repository/folder> --dest <directory>
```

Environment variables:
- `NEXUS_URL` (default: http://localhost:8081)
- `NEXUS_USER` (default: admin)
- `NEXUS_PASS` (default: admin)

## Features
- Upload all files from a directory to a Nexus RAW repository (with optional subdirectory)
- Download all files from a Nexus RAW folder recursively
- Parallel downloads for speed

See the Python code for the original implementation.
