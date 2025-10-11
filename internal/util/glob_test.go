package util

import (
	"testing"
)

func TestParseGlobPattern(t *testing.T) {
	tests := []struct {
		name         string
		globPattern  string
		wantPositive []string
		wantNegative []string
	}{
		{
			name:         "empty pattern",
			globPattern:  "",
			wantPositive: nil,
			wantNegative: nil,
		},
		{
			name:         "single positive pattern",
			globPattern:  "**/*.go",
			wantPositive: []string{"**/*.go"},
			wantNegative: nil,
		},
		{
			name:         "single negative pattern",
			globPattern:  "!**/*.txt",
			wantPositive: nil,
			wantNegative: []string{"**/*.txt"},
		},
		{
			name:         "multiple positive patterns",
			globPattern:  "**/*.go,**/*.md",
			wantPositive: []string{"**/*.go", "**/*.md"},
			wantNegative: nil,
		},
		{
			name:         "mixed positive and negative patterns",
			globPattern:  "**/*.go,!**/*_test.go",
			wantPositive: []string{"**/*.go"},
			wantNegative: []string{"**/*_test.go"},
		},
		{
			name:         "pattern with spaces",
			globPattern:  "**/*.go, **/*.md, !**/*.txt",
			wantPositive: []string{"**/*.go", "**/*.md"},
			wantNegative: []string{"**/*.txt"},
		},
		{
			name:         "pattern with empty elements",
			globPattern:  "**/*.go,,**/*.md",
			wantPositive: []string{"**/*.go", "**/*.md"},
			wantNegative: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := ParseGlobPattern(tt.globPattern)

			if len(gp.positivePatterns) != len(tt.wantPositive) {
				t.Errorf("ParseGlobPattern() positive patterns count = %d, want %d",
					len(gp.positivePatterns), len(tt.wantPositive))
			}
			for i, want := range tt.wantPositive {
				if i >= len(gp.positivePatterns) || gp.positivePatterns[i] != want {
					t.Errorf("ParseGlobPattern() positive pattern[%d] = %v, want %v",
						i, gp.positivePatterns[i], want)
				}
			}

			if len(gp.negativePatterns) != len(tt.wantNegative) {
				t.Errorf("ParseGlobPattern() negative patterns count = %d, want %d",
					len(gp.negativePatterns), len(tt.wantNegative))
			}
			for i, want := range tt.wantNegative {
				if i >= len(gp.negativePatterns) || gp.negativePatterns[i] != want {
					t.Errorf("ParseGlobPattern() negative pattern[%d] = %v, want %v",
						i, gp.negativePatterns[i], want)
				}
			}
		})
	}
}

func TestGlobPatternMatch(t *testing.T) {
	tests := []struct {
		name        string
		globPattern string
		path        string
		want        bool
		wantErr     bool
	}{
		{
			name:        "empty pattern matches all",
			globPattern: "",
			path:        "any/path.go",
			want:        true,
		},
		{
			name:        "simple wildcard match",
			globPattern: "**/*.go",
			path:        "main.go",
			want:        true,
		},
		{
			name:        "simple wildcard no match",
			globPattern: "**/*.go",
			path:        "main.txt",
			want:        false,
		},
		{
			name:        "recursive wildcard match",
			globPattern: "**/*.go",
			path:        "pkg/util/helper.go",
			want:        true,
		},
		{
			name:        "multiple positive patterns - first matches",
			globPattern: "**/*.go,**/*.md",
			path:        "main.go",
			want:        true,
		},
		{
			name:        "multiple positive patterns - second matches",
			globPattern: "**/*.go,**/*.md",
			path:        "README.md",
			want:        true,
		},
		{
			name:        "multiple positive patterns - none match",
			globPattern: "**/*.go,**/*.md",
			path:        "data.txt",
			want:        false,
		},
		{
			name:        "negative pattern excludes match",
			globPattern: "**/*.go,!**/*_test.go",
			path:        "main_test.go",
			want:        false,
		},
		{
			name:        "negative pattern doesn't exclude non-matching",
			globPattern: "**/*.go,!**/*_test.go",
			path:        "main.go",
			want:        true,
		},
		{
			name:        "single character wildcard",
			globPattern: "file?.txt",
			path:        "file1.txt",
			want:        true,
		},
		{
			name:        "single character wildcard no match",
			globPattern: "file?.txt",
			path:        "file10.txt",
			want:        false,
		},
		{
			name:        "root level only pattern",
			globPattern: "*.go",
			path:        "main.go",
			want:        true,
		},
		{
			name:        "root level only pattern no match subdirectory",
			globPattern: "*.go",
			path:        "pkg/main.go",
			want:        false,
		},
		{
			name:        "complex pattern with subdirectory",
			globPattern: "**/*,!vendor/**",
			path:        "vendor/pkg/lib.go",
			want:        false,
		},
		{
			name:        "complex pattern without excluded subdirectory",
			globPattern: "**/*,!vendor/**",
			path:        "pkg/lib.go",
			want:        true,
		},
		{
			name:        "path with forward slashes",
			globPattern: "**/*.go",
			path:        "internal/util/helper.go",
			want:        true,
		},
		{
			name:        "path with backslashes normalized",
			globPattern: "**/*.go",
			path:        "internal\\util\\helper.go",
			want:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gp := ParseGlobPattern(tt.globPattern)
			got, err := gp.Match(tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("GlobPattern.Match() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GlobPattern.Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterWithGlob(t *testing.T) {
	type testItem struct {
		path string
		data string
	}

	items := []testItem{
		{path: "main.go", data: "main code"},
		{path: "main_test.go", data: "test code"},
		{path: "README.md", data: "docs"},
		{path: "pkg/util/helper.go", data: "util code"},
		{path: "pkg/util/helper_test.go", data: "util test"},
		{path: "data.txt", data: "data"},
	}

	tests := []struct {
		name        string
		globPattern string
		wantPaths   []string
	}{
		{
			name:        "empty pattern returns all",
			globPattern: "",
			wantPaths:   []string{"main.go", "main_test.go", "README.md", "pkg/util/helper.go", "pkg/util/helper_test.go", "data.txt"},
		},
		{
			name:        "filter .go files",
			globPattern: "**/*.go",
			wantPaths:   []string{"main.go", "main_test.go", "pkg/util/helper.go", "pkg/util/helper_test.go"},
		},
		{
			name:        "filter .go files excluding tests",
			globPattern: "**/*.go,!**/*_test.go",
			wantPaths:   []string{"main.go", "pkg/util/helper.go"},
		},
		{
			name:        "filter .go and .md files",
			globPattern: "**/*.go,**/*.md",
			wantPaths:   []string{"main.go", "main_test.go", "README.md", "pkg/util/helper.go", "pkg/util/helper_test.go"},
		},
		{
			name:        "filter root level files only",
			globPattern: "*",
			wantPaths:   []string{"main.go", "main_test.go", "README.md", "data.txt"},
		},
		{
			name:        "exclude .txt files",
			globPattern: "**/*,!**/*.txt",
			wantPaths:   []string{"main.go", "main_test.go", "README.md", "pkg/util/helper.go", "pkg/util/helper_test.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered, err := FilterWithGlob(items, tt.globPattern, func(item testItem) string {
				return item.path
			})

			if err != nil {
				t.Errorf("FilterWithGlob() error = %v", err)
				return
			}

			if len(filtered) != len(tt.wantPaths) {
				t.Errorf("FilterWithGlob() filtered count = %d, want %d", len(filtered), len(tt.wantPaths))
			}

			for i, wantPath := range tt.wantPaths {
				if i >= len(filtered) {
					t.Errorf("FilterWithGlob() missing item at index %d, want path %s", i, wantPath)
					continue
				}
				if filtered[i].path != wantPath {
					t.Errorf("FilterWithGlob() item[%d].path = %s, want %s", i, filtered[i].path, wantPath)
				}
			}
		})
	}
}

func TestFilterWithGlobInvalidPattern(t *testing.T) {
	type testItem struct {
		path string
	}

	items := []testItem{{path: "test.go"}}

	_, err := FilterWithGlob(items, "[invalid", func(item testItem) string {
		return item.path
	})

	if err == nil {
		t.Error("FilterWithGlob() expected error for invalid pattern, got nil")
	}
}
