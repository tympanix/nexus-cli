# Nexus CLI Go implementation

This folder contains a Go translation of the Python nexuscli tool. It provides the same upload and download features, using similar command-line arguments and environment variables for configuration.

## Usage

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

Build the Go CLI:

```
go build -o nexuscli-go
```

Run upload:

```
./nexuscli-go upload <directory> <repository/subdir>
```

Run download:

```
./nexuscli-go download <repository/folder> <directory>
```

Environment variables:
- `NEXUS_URL` (default: http://localhost:8081)
- `NEXUS_USER` (default: admin)
- `NEXUS_PASS` (default: admin)

## Features
- Upload all files from a directory to a Nexus RAW repository (with optional subdirectory)
- Download all files from a Nexus RAW folder recursively
- Parallel downloads for speed
- Small container image size using multi-stage build with scratch base

See the Python code for the original implementation.
