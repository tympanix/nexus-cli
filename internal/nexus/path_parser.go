package nexus

import (
	"strings"
)

// ParseRepositoryPath splits a repository path (e.g., "repository/folder" or "repository/folder/")
// into repository name and path, normalizing trailing slashes.
// Returns repository, path, and whether the parse was successful.
func ParseRepositoryPath(repoPath string) (repository string, path string, ok bool) {
	parts := strings.SplitN(repoPath, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	repository = parts[0]
	path = strings.TrimRight(parts[1], "/")
	return repository, path, true
}
