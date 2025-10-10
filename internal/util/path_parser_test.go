package util

import "testing"

func TestParseRepositoryPath(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantRepository string
		wantPath       string
		wantOk         bool
	}{
		{
			name:           "simple path without trailing slash",
			input:          "builds/test9",
			wantRepository: "builds",
			wantPath:       "test9",
			wantOk:         true,
		},
		{
			name:           "simple path with trailing slash",
			input:          "builds/test9/",
			wantRepository: "builds",
			wantPath:       "test9",
			wantOk:         true,
		},
		{
			name:           "nested path without trailing slash",
			input:          "repository/folder/subfolder",
			wantRepository: "repository",
			wantPath:       "folder/subfolder",
			wantOk:         true,
		},
		{
			name:           "nested path with trailing slash",
			input:          "repository/folder/subfolder/",
			wantRepository: "repository",
			wantPath:       "folder/subfolder",
			wantOk:         true,
		},
		{
			name:           "multiple trailing slashes",
			input:          "builds/test9///",
			wantRepository: "builds",
			wantPath:       "test9",
			wantOk:         true,
		},
		{
			name:           "only repository name",
			input:          "repository",
			wantRepository: "",
			wantPath:       "",
			wantOk:         false,
		},
		{
			name:           "empty string",
			input:          "",
			wantRepository: "",
			wantPath:       "",
			wantOk:         false,
		},
		{
			name:           "repository with empty path",
			input:          "repository/",
			wantRepository: "repository",
			wantPath:       "",
			wantOk:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRepository, gotPath, gotOk := ParseRepositoryPath(tt.input)
			if gotRepository != tt.wantRepository {
				t.Errorf("ParseRepositoryPath() repository = %v, want %v", gotRepository, tt.wantRepository)
			}
			if gotPath != tt.wantPath {
				t.Errorf("ParseRepositoryPath() path = %v, want %v", gotPath, tt.wantPath)
			}
			if gotOk != tt.wantOk {
				t.Errorf("ParseRepositoryPath() ok = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}
