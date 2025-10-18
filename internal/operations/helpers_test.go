package operations

import "testing"

func TestGetRelativePath(t *testing.T) {
	tests := []struct {
		name      string
		assetPath string
		basePath  string
		want      string
	}{
		{
			name:      "no base path",
			assetPath: "/repo/subdir/file.txt",
			basePath:  "",
			want:      "repo/subdir/file.txt",
		},
		{
			name:      "with base path",
			assetPath: "/repo/subdir/file.txt",
			basePath:  "repo/subdir",
			want:      "file.txt",
		},
		{
			name:      "base path with trailing slash",
			assetPath: "/repo/subdir/file.txt",
			basePath:  "repo/subdir/",
			want:      "file.txt",
		},
		{
			name:      "base path with leading slash",
			assetPath: "/repo/subdir/file.txt",
			basePath:  "/repo/subdir",
			want:      "file.txt",
		},
		{
			name:      "nested path",
			assetPath: "repo/subdir/nested/file.txt",
			basePath:  "repo/subdir",
			want:      "nested/file.txt",
		},
		{
			name:      "no common prefix",
			assetPath: "/other/path/file.txt",
			basePath:  "repo/subdir",
			want:      "other/path/file.txt",
		},
		{
			name:      "exact match",
			assetPath: "/repo/subdir",
			basePath:  "repo/subdir",
			want:      "repo/subdir",
		},
		{
			name:      "path normalization with double slashes",
			assetPath: "//repo//subdir//file.txt",
			basePath:  "repo/subdir",
			want:      "file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getRelativePath(tt.assetPath, tt.basePath)
			if got != tt.want {
				t.Errorf("getRelativePath(%q, %q) = %q, want %q", tt.assetPath, tt.basePath, got, tt.want)
			}
		})
	}
}
