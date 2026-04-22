package navigation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveNavigationSourceBase_UsesProvidedBaseOrProcessCWD(t *testing.T) {
	baseDir := t.TempDir()
	if got := resolveNavigationSourceBase(baseDir); got != filepath.Clean(baseDir) {
		t.Fatalf("resolveNavigationSourceBase(base) = %q, want %q", got, filepath.Clean(baseDir))
	}

	cwd := t.TempDir()
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	if got := resolveNavigationSourceBase(""); got != filepath.Clean(cwd) {
		t.Fatalf("resolveNavigationSourceBase(\"\") = %q, want %q", got, filepath.Clean(cwd))
	}
}

func TestResolveNavigationRelativeFilePath_PrefersInvocationThenRoot(t *testing.T) {
	root := t.TempDir()
	invocation := filepath.Join(root, "workspace")
	if err := os.MkdirAll(filepath.Join(invocation, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	invocationFile := filepath.Join(invocation, "pkg", "invocation.go")
	rootFile := filepath.Join(root, "pkg", "root.go")
	if err := os.WriteFile(invocationFile, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rootFile, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		file          string
		invocationCWD string
		rootPath      string
		want          string
	}{
		{
			name:          "absolute input is preserved",
			file:          invocationFile,
			invocationCWD: invocation,
			rootPath:      root,
			want:          filepath.Clean(invocationFile),
		},
		{
			name:          "existing invocation path wins",
			file:          "pkg/invocation.go",
			invocationCWD: invocation,
			rootPath:      root,
			want:          invocationFile,
		},
		{
			name:          "existing root path is used",
			file:          "pkg/root.go",
			invocationCWD: invocation,
			rootPath:      root,
			want:          rootFile,
		},
		{
			name:          "missing path falls back to invocation join",
			file:          "pkg/missing.go",
			invocationCWD: invocation,
			rootPath:      root,
			want:          filepath.Join(invocation, "pkg", "missing.go"),
		},
		{
			name:          "missing path falls back to root join",
			file:          "pkg/missing.go",
			invocationCWD: "",
			rootPath:      root,
			want:          filepath.Join(root, "pkg", "missing.go"),
		},
		{
			name:          "empty bases keep relative path",
			file:          "pkg/missing.go",
			invocationCWD: "",
			rootPath:      "",
			want:          filepath.Clean(filepath.FromSlash("pkg/missing.go")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveNavigationRelativeFilePath(tt.file, tt.invocationCWD, tt.rootPath); got != tt.want {
				t.Fatalf("resolveNavigationRelativeFilePath() = %q, want %q", got, tt.want)
			}
		})
	}
}
