package navigation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/lsp"
)

func TestLSPLocationFilePath_ResolvesExistingAndFallbackPaths(t *testing.T) {
	rootDir := t.TempDir()
	invocationDir := filepath.Join(rootDir, "workspace")
	rootOnlyDir := filepath.Join(rootDir, "root")
	if err := os.MkdirAll(filepath.Join(invocationDir, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(rootOnlyDir, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(invocationDir, "pkg", "invocation.go"), []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rootOnlyDir, "pkg", "root.go"), []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		file          string
		rootPath      string
		invocationCWD string
		want          string
	}{
		{
			name:          "absolute path is preserved",
			file:          filepath.Join(rootDir, "abs.go"),
			rootPath:      rootOnlyDir,
			invocationCWD: invocationDir,
			want:          filepath.Join(rootDir, "abs.go"),
		},
		{
			name:          "existing invocation path wins",
			file:          "pkg/invocation.go",
			rootPath:      rootOnlyDir,
			invocationCWD: invocationDir,
			want:          filepath.Join(invocationDir, "pkg", "invocation.go"),
		},
		{
			name:          "existing root path is used",
			file:          "pkg/root.go",
			rootPath:      rootOnlyDir,
			invocationCWD: invocationDir,
			want:          filepath.Join(rootOnlyDir, "pkg", "root.go"),
		},
		{
			name:          "missing file falls back to invocation cwd join",
			file:          "pkg/missing.go",
			rootPath:      rootOnlyDir,
			invocationCWD: invocationDir,
			want:          filepath.Join(invocationDir, "pkg", "missing.go"),
		},
		{
			name:          "missing file falls back to root join",
			file:          "pkg/missing.go",
			rootPath:      rootOnlyDir,
			invocationCWD: "",
			want:          filepath.Join(rootOnlyDir, "pkg", "missing.go"),
		},
		{
			name:          "empty bases return cleaned relative path",
			file:          "pkg/missing.go",
			rootPath:      "",
			invocationCWD: "",
			want:          filepath.Clean(filepath.FromSlash("pkg/missing.go")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lspLocationFilePath(tt.file, tt.rootPath, tt.invocationCWD); got != tt.want {
				t.Fatalf("lspLocationFilePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLSPAdapter_NewAndToAbsPath(t *testing.T) {
	if got := NewLSPAdapter(nil, "/workspace"); got != nil {
		t.Fatalf("NewLSPAdapter(nil) = %+v, want nil", got)
	}

	adapter := NewLSPAdapter(&lsp.Client{}, "/workspace")
	if adapter == nil {
		t.Fatal("NewLSPAdapter() should return adapter for non-nil client")
	}

	tests := []struct {
		name string
		path string
		want string
	}{
		{"absolute path is preserved", "/tmp/file.go", "/tmp/file.go"},
		{"relative path joins root", "pkg/file.go", filepath.Join("/workspace", "pkg/file.go")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := adapter.toAbsPath(tt.path); got != tt.want {
				t.Fatalf("toAbsPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLSPAdapter_ConvertLocationsWithoutRootPreservesAbsolutePaths(t *testing.T) {
	adapter := NewLSPAdapter(&lsp.Client{}, "")
	locations := adapter.convertLocations([]lsp.Location{{
		URI: "file:///tmp/example.go",
		Range: lsp.Range{
			Start: lsp.Position{Line: 0, Character: 0},
			End:   lsp.Position{Line: 0, Character: 4},
		},
	}})

	if len(locations) != 1 {
		t.Fatalf("len(locations) = %d, want 1", len(locations))
	}
	if locations[0].File != filepath.Clean("/tmp/example.go") {
		t.Fatalf("File = %q, want absolute path", locations[0].File)
	}
}
