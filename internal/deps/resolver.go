package deps

import (
	"fmt"
	"strings"

	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

type Resolver struct {
	client *nexusapi.Client
}

func NewResolver(client *nexusapi.Client) *Resolver {
	return &Resolver{client: client}
}

func (r *Resolver) ResolveDependency(dep *Dependency) (map[string]string, error) {
	files := make(map[string]string)

	expandedPath := dep.ExpandedPath()

	if dep.Recursive {
		pathPrefix := strings.TrimSuffix(expandedPath, "/")
		assets, err := r.client.SearchAssets(dep.Repository, pathPrefix)
		if err != nil {
			return nil, fmt.Errorf("failed to search assets for %s: %w", dep.Name, err)
		}

		if len(assets) == 0 {
			return nil, fmt.Errorf("no assets found for dependency %s at path %s", dep.Name, expandedPath)
		}

		for _, asset := range assets {
			checksum := r.getChecksumForAlgorithm(asset.Checksum, dep.Checksum)
			if checksum == "" {
				return nil, fmt.Errorf("no %s checksum available for asset %s", dep.Checksum, asset.Path)
			}
			files[asset.Path] = fmt.Sprintf("%s:%s", dep.Checksum, checksum)
		}
	} else {
		asset, err := r.client.GetAssetByPath(dep.Repository, expandedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get asset for %s: %w", dep.Name, err)
		}

		checksum := r.getChecksumForAlgorithm(asset.Checksum, dep.Checksum)
		if checksum == "" {
			return nil, fmt.Errorf("no %s checksum available for asset %s", dep.Checksum, asset.Path)
		}
		files[asset.Path] = fmt.Sprintf("%s:%s", dep.Checksum, checksum)
	}

	return files, nil
}

func (r *Resolver) getChecksumForAlgorithm(checksum nexusapi.Checksum, algorithm string) string {
	switch strings.ToLower(algorithm) {
	case "sha1":
		return checksum.SHA1
	case "sha256":
		return checksum.SHA256
	case "sha512":
		return checksum.SHA512
	case "md5":
		return checksum.MD5
	default:
		return ""
	}
}
