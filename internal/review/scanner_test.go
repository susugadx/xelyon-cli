package review

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestScanner_ScanFromChanges(t *testing.T) {
	scanner := NewScanner()
	ctx := context.Background()

	tests := []struct {
		name        string
		changeStack []tools.FileChange
		opt         ScanOptions
		wantPaths   []string
	}{
		{
			name:        "empty changeStack",
			changeStack: []tools.FileChange{},
			opt:         ScanOptions{},
			wantPaths:   nil,
		},
		{
			name: "single file",
			changeStack: []tools.FileChange{
				{FilePath: "file.go", Tool: "write_file"},
			},
			opt:       ScanOptions{},
			wantPaths: []string{"file.go"},
		},
		{
			name: "duplicate files",
			changeStack: []tools.FileChange{
				{FilePath: "file.go", Tool: "write_file"},
				{FilePath: "file.go", Tool: "str_replace"},
			},
			opt:       ScanOptions{},
			wantPaths: []string{"file.go"},
		},
		{
			name: "multiple files sorted",
			changeStack: []tools.FileChange{
				{FilePath: "b.go", Tool: "write_file"},
				{FilePath: "a.go", Tool: "write_file"},
			},
			opt:       ScanOptions{},
			wantPaths: []string{"a.go", "b.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, err := scanner.ScanFromChanges(ctx, tt.changeStack, tt.opt)
			if err != nil {
				t.Fatalf("ScanFromChanges() error = %v", err)
			}

			if len(targets) != len(tt.wantPaths) {
				t.Fatalf("ScanFromChanges() returned %d targets, want %d", len(targets), len(tt.wantPaths))
			}

			for i, want := range tt.wantPaths {
				if targets[i].Path != want {
					t.Errorf("targets[%d].Path = %q, want %q", i, targets[i].Path, want)
				}
			}
		})
	}
}

func TestGuessChangeTypeFromTool(t *testing.T) {
	tests := []struct {
		name    string
		changes []tools.FileChange
		path    string
		want    string
	}{
		{
			name: "write_file is A",
			changes: []tools.FileChange{
				{FilePath: "new.go", Tool: "write_file"},
			},
			path: "new.go",
			want: "A",
		},
		{
			name: "delete_file is D",
			changes: []tools.FileChange{
				{FilePath: "old.go", Tool: "delete_file"},
			},
			path: "old.go",
			want: "D",
		},
		{
			name: "move_file is R",
			changes: []tools.FileChange{
				{FilePath: "moved.go", Tool: "move_file"},
			},
			path: "moved.go",
			want: "R",
		},
		{
			name: "str_replace is M",
			changes: []tools.FileChange{
				{FilePath: "modified.go", Tool: "str_replace"},
			},
			path: "modified.go",
			want: "M",
		},
		{
			name: "newest change wins",
			changes: []tools.FileChange{
				{FilePath: "file.go", Tool: "write_file"},
				{FilePath: "file.go", Tool: "str_replace"},
			},
			path: "file.go",
			want: "M",
		},
		{
			name: "path not found",
			changes: []tools.FileChange{
				{FilePath: "other.go", Tool: "write_file"},
			},
			path: "missing.go",
			want: "M",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := guessChangeTypeFromTool(tt.changes, tt.path)
			if got != tt.want {
				t.Errorf("guessChangeTypeFromTool() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUniqueFilePaths(t *testing.T) {
	tests := []struct {
		name    string
		changes []tools.FileChange
		want    []string
	}{
		{
			name:    "empty",
			changes: []tools.FileChange{},
			want:    []string{},
		},
		{
			name: "single",
			changes: []tools.FileChange{
				{FilePath: "file.go"},
			},
			want: []string{"file.go"},
		},
		{
			name: "duplicates removed",
			changes: []tools.FileChange{
				{FilePath: "file.go"},
				{FilePath: "file.go"},
			},
			want: []string{"file.go"},
		},
		{
			name: "sorted output",
			changes: []tools.FileChange{
				{FilePath: "z.go"},
				{FilePath: "a.go"},
				{FilePath: "m.go"},
			},
			want: []string{"a.go", "m.go", "z.go"},
		},
		{
			name: "empty paths ignored",
			changes: []tools.FileChange{
				{FilePath: "file.go"},
				{FilePath: ""},
				{FilePath: "  "},
			},
			want: []string{"file.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uniqueFilePaths(tt.changes)
			if len(got) != len(tt.want) {
				t.Fatalf("uniqueFilePaths() = %v, want %v", got, tt.want)
			}
			for i, w := range tt.want {
				if got[i] != w {
					t.Errorf("got[%d] = %q, want %q", i, got[i], w)
				}
			}
		})
	}
}

func TestParseNameOnly(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want []string
	}{
		{
			name: "single file",
			out:  "file.go\n",
			want: []string{"file.go"},
		},
		{
			name: "multiple files",
			out:  "a.go\nb.go\nc.go\n",
			want: []string{"a.go", "b.go", "c.go"},
		},
		{
			name: "empty lines ignored",
			out:  "file.go\n\n\n",
			want: []string{"file.go"},
		},
		{
			name: "whitespace trimmed",
			out:  "  file.go  \n",
			want: []string{"file.go"},
		},
		{
			name: "empty input",
			out:  "",
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNameOnly(tt.out)
			if len(got) != len(tt.want) {
				t.Fatalf("parseNameOnly() = %v, want %v", got, tt.want)
			}
			for i, w := range tt.want {
				if got[i] != w {
					t.Errorf("got[%d] = %q, want %q", i, got[i], w)
				}
			}
		})
	}
}

func TestTrimDiffToMaxLines(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		maxLines int
		wantLen  int
		wantEnd  string
	}{
		{
			name:     "under limit",
			diff:     "line1\nline2\nline3",
			maxLines: 10,
			wantLen:  3,
		},
		{
			name:     "at limit",
			diff:     "line1\nline2\nline3",
			maxLines: 3,
			wantLen:  3,
		},
		{
			name:     "over limit truncated",
			diff:     "line1\nline2\nline3\nline4\nline5",
			maxLines: 3,
			wantLen:  4, // 3 lines + truncation marker
			wantEnd:  "... (truncated)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimDiffToMaxLines(tt.diff, tt.maxLines)
			lines := len(splitLines(got))

			if lines > tt.wantLen+1 { // Allow for truncation marker
				t.Errorf("trimDiffToMaxLines() returned %d lines, want <= %d", lines, tt.wantLen)
			}

			if tt.wantEnd != "" && !containsString(got, tt.wantEnd) {
				t.Errorf("trimDiffToMaxLines() should contain %q", tt.wantEnd)
			}
		})
	}
}

func TestSplitUnifiedDiffByFile(t *testing.T) {
	tests := []struct {
		name      string
		patch     string
		wantFiles int
		wantPaths []string
	}{
		{
			name:      "empty patch",
			patch:     "",
			wantFiles: 0,
		},
		{
			name: "single file",
			patch: `diff --git a/file.go b/file.go
--- a/file.go
+++ b/file.go
@@ -1,3 +1,4 @@
+added line`,
			wantFiles: 1,
			wantPaths: []string{"file.go"},
		},
		{
			name: "multiple files",
			patch: `diff --git a/a.go b/a.go
--- a/a.go
+++ b/a.go
@@ -1 +1 @@
-old
+new
diff --git a/b.go b/b.go
--- a/b.go
+++ b/b.go
@@ -1 +1 @@
-old
+new`,
			wantFiles: 2,
			wantPaths: []string{"a.go", "b.go"},
		},
		{
			name: "new file mode",
			patch: `diff --git a/new.go b/new.go
new file mode 100644
--- /dev/null
+++ b/new.go
@@ -0,0 +1 @@
+package main`,
			wantFiles: 1,
			wantPaths: []string{"new.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitUnifiedDiffByFile(tt.patch)
			if len(got) != tt.wantFiles {
				t.Fatalf("splitUnifiedDiffByFile() returned %d files, want %d", len(got), tt.wantFiles)
			}

			for i, want := range tt.wantPaths {
				if i < len(got) && got[i].Path != want {
					t.Errorf("got[%d].Path = %q, want %q", i, got[i].Path, want)
				}
			}
		})
	}
}

func TestParseDiffGitHeader(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantNewPath string
		wantOldPath string
	}{
		{
			name:        "simple",
			line:        "diff --git a/file.go b/file.go",
			wantNewPath: "file.go",
			wantOldPath: "file.go",
		},
		{
			name:        "with path",
			line:        "diff --git a/internal/foo.go b/internal/foo.go",
			wantNewPath: "internal/foo.go",
			wantOldPath: "internal/foo.go",
		},
		{
			name:        "rename",
			line:        "diff --git a/old.go b/new.go",
			wantNewPath: "new.go",
			wantOldPath: "old.go",
		},
		{
			name:        "short line",
			line:        "xyz",
			wantNewPath: "",
			wantOldPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newPath, oldPath := parseDiffGitHeader(tt.line)
			if newPath != tt.wantNewPath {
				t.Errorf("newPath = %q, want %q", newPath, tt.wantNewPath)
			}
			if oldPath != tt.wantOldPath {
				t.Errorf("oldPath = %q, want %q", oldPath, tt.wantOldPath)
			}
		})
	}
}

// Helper functions

func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	var lines []string
	for _, line := range []byte(s) {
		if line == '\n' {
			lines = append(lines, "")
		}
	}
	return lines
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
