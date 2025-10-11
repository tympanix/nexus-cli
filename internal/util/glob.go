package util

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type GlobPattern struct {
	positivePatterns []string
	negativePatterns []string
}

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
