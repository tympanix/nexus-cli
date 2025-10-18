package operations

import (
	"path"
	"strings"

	"github.com/tympanix/nexus-cli/internal/checksum"
	"github.com/tympanix/nexus-cli/internal/util"
)

func processKeyTemplateWrapper(input string, keyFromFile string) (string, error) {
	return util.ProcessKeyTemplate(input, keyFromFile, checksum.ComputeChecksum)
}

// getRelativePath returns the relative path from basePath to assetPath using path.Clean for normalization.
// Both paths are cleaned and normalized before computing the relative portion.
func getRelativePath(assetPath, basePath string) string {
	// Clean and normalize the asset path
	cleanAsset := path.Clean("/" + strings.TrimLeft(assetPath, "/"))
	cleanAsset = strings.TrimLeft(cleanAsset, "/")

	// If no base path, return the cleaned asset path
	if basePath == "" {
		return cleanAsset
	}

	// Clean and normalize the base path
	cleanBase := path.Clean("/" + strings.TrimLeft(basePath, "/"))
	cleanBase = strings.TrimLeft(cleanBase, "/")

	// Check if asset path starts with base path
	if strings.HasPrefix(cleanAsset, cleanBase+"/") {
		return cleanAsset[len(cleanBase)+1:]
	}

	return cleanAsset
}
