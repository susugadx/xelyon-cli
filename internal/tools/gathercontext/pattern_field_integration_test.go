package gathercontext

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGatherContext_PatternFieldLiteralSearchContracts(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		args     map[string]string
		contains []string
		excludes []string
	}{
		{
			name: "filename-like pattern stays search",
			files: map[string]string{
				"README.md": "# README.md\n",
			},
			args: map[string]string{
				"query": `pattern:"README.md"`,
			},
			contains: []string{"Route: Auto search", "README.md"},
			excludes: []string{"Route: Direct read"},
		},
		{
			name: "symbol-like pattern stays literal text search",
			files: map[string]string{
				"builder.go": "package sample\n\nfunc Builder() {}\n",
			},
			args: map[string]string{
				"query":       `pattern:"Builder"`,
				"file_filter": "go",
			},
			contains: []string{"Route: Auto search", "builder.go", "func Builder()"},
			excludes: []string{"Related Tests", "Definitions:"},
		},
		{
			name: "comma pattern is not split",
			files: map[string]string{
				"literal.txt": "foo,bar\n",
				"split.txt":   "foo only\n",
			},
			args: map[string]string{
				"query": `pattern:"foo,bar"`,
			},
			contains: []string{"Route: Auto search", "literal.txt"},
			excludes: []string{"split.txt", `Pattern 1/`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			withGatherContextWorkingDir(t, root)
			files := make(map[string]string, len(tt.files))
			for path, body := range tt.files {
				files[filepath.Join(root, path)] = body
			}
			writeGatherContextFiles(t, files)

			result, _ := runGatherContext(t, newGatherContextExecCtx(root), tt.args)
			for _, want := range tt.contains {
				if !strings.Contains(result, want) {
					t.Fatalf("expected %q in result, got:\n%s", want, result)
				}
			}
			for _, exclude := range tt.excludes {
				if strings.Contains(result, exclude) {
					t.Fatalf("did not expect %q in result, got:\n%s", exclude, result)
				}
			}
		})
	}
}
