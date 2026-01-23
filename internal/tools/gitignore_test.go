package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitignoreHasBackupPattern(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
		wantErr bool
	}{
		{
			name:    "has *.bak pattern",
			content: "*.log\n*.bak\n*.tmp\n",
			want:    true,
			wantErr: false,
		},
		{
			name:    "has *.bak.* pattern",
			content: "*.log\n*.bak.*\n*.tmp\n",
			want:    true,
			wantErr: false,
		},
		{
			name:    "has backup pattern with glob",
			content: "**/*.bak\n",
			want:    true,
			wantErr: false,
		},
		{
			name:    "no backup pattern",
			content: "*.log\n*.tmp\nnode_modules/\n",
			want:    false,
			wantErr: false,
		},
		{
			name:    "empty file",
			content: "",
			want:    false,
			wantErr: false,
		},
		{
			name:    "only comments",
			content: "# This is a comment\n# Another comment\n",
			want:    false,
			wantErr: false,
		},
		{
			name:    "bak in comment is ignored",
			content: "# ignore *.bak files\n*.log\n",
			want:    false,
			wantErr: false,
		},
		{
			name:    "multiple patterns with bak",
			content: "*.log\n*.tmp\n# Backup files\n*.bak\n*.swp\n",
			want:    true,
			wantErr: false,
		},
		{
			name:    "whitespace around pattern",
			content: "  *.bak  \n",
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file with content
			dir := t.TempDir()
			gitignorePath := filepath.Join(dir, ".gitignore")

			if err := os.WriteFile(gitignorePath, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			got, err := gitignoreHasBackupPattern(gitignorePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("gitignoreHasBackupPattern() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("gitignoreHasBackupPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGitignoreHasBackupPattern_FileNotExist(t *testing.T) {
	_, err := gitignoreHasBackupPattern("/nonexistent/path/.gitignore")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestFindGitRoot(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(dir string) string
		wantRoot bool
	}{
		{
			name: "directory with .git",
			setup: func(dir string) string {
				gitDir := filepath.Join(dir, ".git")
				os.Mkdir(gitDir, 0755)
				return dir
			},
			wantRoot: true,
		},
		{
			name: "nested directory with .git at parent",
			setup: func(dir string) string {
				gitDir := filepath.Join(dir, ".git")
				os.Mkdir(gitDir, 0755)
				subDir := filepath.Join(dir, "subdir", "nested")
				os.MkdirAll(subDir, 0755)
				return subDir
			},
			wantRoot: true,
		},
		{
			name: "directory without .git",
			setup: func(dir string) string {
				// No .git created
				return dir
			},
			wantRoot: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			testDir := tt.setup(baseDir)

			got := findGitRoot(testDir)

			if tt.wantRoot {
				if got == "" {
					t.Errorf("findGitRoot(%q) = empty, want non-empty", testDir)
				}
			} else {
				if got != "" {
					t.Errorf("findGitRoot(%q) = %q, want empty", testDir, got)
				}
			}
		})
	}
}
