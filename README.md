# Nexus CLI

A command-line tool for uploading and downloading files to/from a Nexus RAW repository.

## Building

To build with production packages:

```sh
make build
```

This creates standalone binaries, DEB packages, and RPM packages in the `dist/` directory using [GoReleaser](https://goreleaser.com).

For development builds and other options, see [nexuscli-go/README.md](nexuscli-go/README.md).

## Usage

Set the following environment variables for authentication and Nexus URL:

- `NEXUS_URL` (default: http://localhost:8081)
- `NEXUS_USER` (default: admin)
- `NEXUS_PASS` (default: admin)

### Upload

```
nexuscli-go upload <directory> <repository[/subdir]>
```

### Download

```
nexuscli-go download <repository/folder> <dest>
```
