# Nexus CLI

A command-line tool for uploading and downloading files to/from a Nexus RAW repository.

## Installation

```sh
pip install .
```

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
