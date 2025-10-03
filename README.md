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

```
nexuscli-go upload [--url <url>] [--username <user>] [--password <pass>] <directory> <repository[/subdir]>
```

### Download

```
nexuscli-go download [--url <url>] [--username <user>] [--password <pass>] <repository/folder> <dest>
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
