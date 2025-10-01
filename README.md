# Nexus CLI

A command-line tool for uploading and downloading files to/from a Nexus RAW repository.

This repository contains both a Python implementation and a Go implementation of the CLI.

## Python Installation

```sh
pip install .
```

## Go Build

To build the Go implementation with production packages:

```sh
make build
```

This creates standalone binaries, DEB packages, and RPM packages in the `dist/` directory using [GoReleaser](https://goreleaser.com).

See [nexuscli-go/README.md](nexuscli-go/README.md) for more details on the Go implementation.

## Usage

Set the following environment variables for authentication and Nexus URL:

- `NEXUS_URL` (default: http://localhost:8081)
- `NEXUS_USER` (default: admin)
- `NEXUS_PASS` (default: admin)

### Upload

```
nexus upload <directory> <repository[/subdir]>
```

### Download

```
nexus download <repository/folder> <dest>
```
