package deps

import (
	"fmt"
	"strings"

	"github.com/tympanix/nexus-cli/internal/nexusapi"
)

type ClientFactory func(url, username, password string) *nexusapi.Client

type Resolver struct {
	clientFactory ClientFactory
	username      string
	password      string
	defaultURL    string
}

func NewResolver(client *nexusapi.Client) *Resolver {
	return &Resolver{
		clientFactory: nexusapi.NewClient,
		username:      client.Username,
		password:      client.Password,
		defaultURL:    client.BaseURL,
	}
}

func (r *Resolver) ResolveDependency(dep *Dependency) (map[string]string, error) {
	files := make(map[string]string)

	url := dep.URL
	if url == "" {
		url = r.defaultURL
	}

	client := r.clientFactory(url, r.username, r.password)

	expandedPath := dep.ExpandedPath()

	if dep.Recursive {
		pathPrefix := strings.TrimSuffix(expandedPath, "/")
		assets, err := client.SearchAssets(dep.Repository, pathPrefix)
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
		asset, err := client.GetAssetByPath(dep.Repository, expandedPath)
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
