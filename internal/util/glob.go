package util

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// GlobPattern represents a parsed glob pattern with support for positive and negative patterns.
// Positive patterns include matching files, while negative patterns (prefixed with !) exclude them.
type GlobPattern struct {
	positivePatterns []string
	negativePatterns []string
}

// ParseGlobPattern parses a comma-separated glob pattern string into a GlobPattern.
// Patterns can be positive (include) or negative (exclude, prefixed with !).
// Example: "**/*.go,!**/*_test.go" matches all .go files except test files.
func ParseGlobPattern(globPattern string) *GlobPattern {
	gp := &GlobPattern{}

	if globPattern == "" {
		return gp
	}

	patterns := strings.Split(globPattern, ",")
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if strings.HasPrefix(pattern, "!") {
			gp.negativePatterns = append(gp.negativePatterns, strings.TrimPrefix(pattern, "!"))
		} else {
			gp.positivePatterns = append(gp.positivePatterns, pattern)
		}
	}

	return gp
}

// Match checks if the given path matches the glob pattern.
// A path matches if:
// 1. At least one positive pattern matches (or no positive patterns exist)
// 2. No negative patterns match
// The path is automatically normalized to use forward slashes for consistent matching.
func (gp *GlobPattern) Match(path string) (bool, error) {
	path = filepath.ToSlash(path)

	matchesPositive := len(gp.positivePatterns) == 0
	for _, pattern := range gp.positivePatterns {
		matched, err := doublestar.Match(pattern, path)
		if err != nil {
			return false, fmt.Errorf("invalid glob pattern '%s': %w", pattern, err)
		}
		if matched {
			matchesPositive = true
			break
		}
	}

	if !matchesPositive {
		return false, nil
	}

	for _, pattern := range gp.negativePatterns {
		matched, err := doublestar.Match(pattern, path)
		if err != nil {
			return false, fmt.Errorf("invalid glob pattern '%s': %w", pattern, err)
		}
		if matched {
			return false, nil
		}
	}

	return true, nil
}

// FilterWithGlob filters a slice of items using glob patterns.
// The pathExtractor function is called for each item to extract the path to match.
// This generic function can work with any type (filesystem paths, Asset structs, etc.).
//
// Example with filesystem paths:
//
//	FilterWithGlob(filePaths, "**/*.go", func(path string) string { return path })
//
// Example with custom structs:
//
//	FilterWithGlob(assets, "**/*.tar.gz", func(asset Asset) string { return asset.Path })
func FilterWithGlob[T any](items []T, globPattern string, pathExtractor func(T) string) ([]T, error) {
	if globPattern == "" {
		return items, nil
	}

	gp := ParseGlobPattern(globPattern)
	var filtered []T

	for _, item := range items {
		path := pathExtractor(item)
		matched, err := gp.Match(path)
		if err != nil {
			return nil, err
		}
		if matched {
			filtered = append(filtered, item)
		}
	}

	return filtered, nil
}
