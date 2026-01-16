package agent

import (
	"testing"
)

func TestShouldVerify_GoFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		want     bool
		fileType string
	}{
		{
			name:     "standard go file",
			filePath: "main.go",
			want:     true,
			fileType: "go",
		},
		{
			name:     "go test file",
			filePath: "main_test.go",
			want:     true,
			fileType: "go",
		},
		{
			name:     "go file in subdirectory",
			filePath: "internal/agent/agent.go",
			want:     true,
			fileType: "go",
		},
		{
			name:     "go file uppercase extension",
			filePath: "main.GO",
			want:     true,
			fileType: "go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldVerify(tt.filePath)

			if result.NeedsVerify != tt.want {
				t.Errorf("ShouldVerify(%q).NeedsVerify = %v, want %v", tt.filePath, result.NeedsVerify, tt.want)
			}

			if result.FileType != tt.fileType {
				t.Errorf("ShouldVerify(%q).FileType = %q, want %q", tt.filePath, result.FileType, tt.fileType)
			}

			if len(result.ChangedFiles) != 1 || result.ChangedFiles[0] != tt.filePath {
				t.Errorf("ShouldVerify(%q).ChangedFiles = %v, want [%q]", tt.filePath, result.ChangedFiles, tt.filePath)
			}
		})
	}
}

func TestShouldVerify_NonGoFile(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
	}{
		{
			name:     "javascript file",
			filePath: "script.js",
		},
		{
			name:     "typescript file",
			filePath: "component.ts",
		},
		{
			name:     "python file",
			filePath: "script.py",
		},
		{
			name:     "rust file",
			filePath: "main.rs",
		},
		{
			name:     "text file",
			filePath: "readme.txt",
		},
		{
			name:     "markdown file",
			filePath: "README.md",
		},
		{
			name:     "yaml file",
			filePath: "config.yaml",
		},
		{
			name:     "json file",
			filePath: "package.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldVerify(tt.filePath)

			if result.NeedsVerify {
				t.Errorf("ShouldVerify(%q).NeedsVerify = true, want false", tt.filePath)
			}

			// FileType should be empty for non-Go files
			if result.FileType != "" {
				t.Errorf("ShouldVerify(%q).FileType = %q, want empty", tt.filePath, result.FileType)
			}
		})
	}
}

func TestShouldVerify_NoExtension(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
	}{
		{
			name:     "no extension",
			filePath: "Makefile",
		},
		{
			name:     "dockerfile",
			filePath: "Dockerfile",
		},
		{
			name:     "hidden file",
			filePath: ".gitignore",
		},
		{
			name:     "path with no extension",
			filePath: "bin/xelyon",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldVerify(tt.filePath)

			if result.NeedsVerify {
				t.Errorf("ShouldVerify(%q).NeedsVerify = true, want false", tt.filePath)
			}
		})
	}
}

func TestShouldVerify_EmptyPath(t *testing.T) {
	result := ShouldVerify("")

	if result.NeedsVerify {
		t.Error("ShouldVerify(\"\").NeedsVerify = true, want false")
	}
}

func TestContainsString_Found(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		s     string
		want  bool
	}{
		{
			name:  "first element",
			slice: []string{"a", "b", "c"},
			s:     "a",
			want:  true,
		},
		{
			name:  "middle element",
			slice: []string{"a", "b", "c"},
			s:     "b",
			want:  true,
		},
		{
			name:  "last element",
			slice: []string{"a", "b", "c"},
			s:     "c",
			want:  true,
		},
		{
			name:  "single element slice",
			slice: []string{"only"},
			s:     "only",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsString(tt.slice, tt.s)
			if got != tt.want {
				t.Errorf("containsString(%v, %q) = %v, want %v", tt.slice, tt.s, got, tt.want)
			}
		})
	}
}

func TestContainsString_NotFound(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		s     string
	}{
		{
			name:  "not in slice",
			slice: []string{"a", "b", "c"},
			s:     "d",
		},
		{
			name:  "case sensitive mismatch",
			slice: []string{"A", "B", "C"},
			s:     "a",
		},
		{
			name:  "partial match not counted",
			slice: []string{"abc", "def"},
			s:     "ab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsString(tt.slice, tt.s)
			if got {
				t.Errorf("containsString(%v, %q) = true, want false", tt.slice, tt.s)
			}
		})
	}
}

func TestContainsString_EmptySlice(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		s     string
	}{
		{
			name:  "empty slice",
			slice: []string{},
			s:     "a",
		},
		{
			name:  "nil slice",
			slice: nil,
			s:     "a",
		},
		{
			name:  "empty slice with empty string",
			slice: []string{},
			s:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsString(tt.slice, tt.s)
			if got {
				t.Errorf("containsString(%v, %q) = true, want false", tt.slice, tt.s)
			}
		})
	}
}

func TestContainsString_EmptyString(t *testing.T) {
	// Test when looking for empty string in slice that contains empty string
	slice := []string{"a", "", "b"}
	got := containsString(slice, "")
	if !got {
		t.Errorf("containsString(%v, \"\") = false, want true", slice)
	}
}

func TestVerifyResult_DefaultValues(t *testing.T) {
	result := ShouldVerify("test.txt")

	// Check default values
	if result.FmtResult != "" {
		t.Errorf("VerifyResult.FmtResult should be empty by default, got %q", result.FmtResult)
	}

	if result.TestResult != "" {
		t.Errorf("VerifyResult.TestResult should be empty by default, got %q", result.TestResult)
	}

	if result.TestPassed {
		t.Error("VerifyResult.TestPassed should be false by default")
	}
}
