package pathpolicy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLexical(t *testing.T) {
	baseDir := filepath.Join(string(filepath.Separator), "repo", "work")

	t.Run("relative candidate is resolved from base dir and cleaned", func(t *testing.T) {
		got := ResolveLexical(baseDir, filepath.Join("child", "..", "target.txt"))
		want := filepath.Join(baseDir, "target.txt")
		if got != want {
			t.Fatalf("ResolveLexical() = %q, want %q", got, want)
		}
	})

	t.Run("absolute candidate ignores base dir and is cleaned", func(t *testing.T) {
		candidate := filepath.Join(string(filepath.Separator), "tmp", "repo", "..", "target.txt")
		got := ResolveLexical(baseDir, candidate)
		want := filepath.Join(string(filepath.Separator), "tmp", "target.txt")
		if got != want {
			t.Fatalf("ResolveLexical() = %q, want %q", got, want)
		}
	})
}

func TestIsWithinRoot(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "repo")

	tests := []struct {
		name         string
		resolvedPath string
		want         bool
	}{
		{
			name:         "root itself is allowed",
			resolvedPath: root,
			want:         true,
		},
		{
			name:         "path under root is allowed",
			resolvedPath: filepath.Join(root, "child", "file.txt"),
			want:         true,
		},
		{
			name:         "path outside root is rejected",
			resolvedPath: filepath.Join(filepath.Dir(root), "outside", "file.txt"),
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := IsWithinRoot(root, tt.resolvedPath)
			if err != nil {
				t.Fatalf("IsWithinRoot() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("IsWithinRoot() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolveWithinRootLexically(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "repo")
	baseDir := filepath.Join(root, "work")

	t.Run("inside candidate is returned with outsideRoot false", func(t *testing.T) {
		resolved, outsideRoot, err := ResolveWithinRootLexically(root, baseDir, filepath.Join("missing", "file.txt"))
		if err != nil {
			t.Fatalf("ResolveWithinRootLexically() error = %v", err)
		}
		if outsideRoot {
			t.Fatal("outsideRoot = true, want false for path inside root")
		}
		want := filepath.Join(baseDir, "missing", "file.txt")
		if resolved != want {
			t.Fatalf("resolved = %q, want %q", resolved, want)
		}
	})

	t.Run("outside candidate is returned with outsideRoot true", func(t *testing.T) {
		candidate := filepath.Join("..", "..", "outside.txt")
		resolved, outsideRoot, err := ResolveWithinRootLexically(root, baseDir, candidate)
		if err != nil {
			t.Fatalf("ResolveWithinRootLexically() error = %v", err)
		}
		if !outsideRoot {
			t.Fatal("outsideRoot = false, want true for path outside root")
		}
		want := filepath.Join(filepath.Dir(root), "outside.txt")
		if resolved != want {
			t.Fatalf("resolved = %q, want %q", resolved, want)
		}
	})
}

func TestCheckSymlinkResolutionWithinRoot(t *testing.T) {
	tempDir := t.TempDir()
	root := filepath.Join(tempDir, "repo")
	insideTarget := filepath.Join(root, "inside")
	outsideTarget := filepath.Join(tempDir, "outside")
	for _, dir := range []string{insideTarget, outsideTarget} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("MkdirAll(%q): %v", dir, err)
		}
	}

	insideLink := filepath.Join(root, "inside-link")
	if err := os.Symlink(insideTarget, insideLink); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
	outsideLink := filepath.Join(root, "outside-link")
	if err := os.Symlink(outsideTarget, outsideLink); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
	brokenLink := filepath.Join(root, "broken-link")
	if err := os.Symlink(filepath.Join(tempDir, "missing-target"), brokenLink); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	tests := []struct {
		name         string
		resolvedPath string
		wantOutside  bool
	}{
		{
			name:         "symlink resolving inside root is allowed",
			resolvedPath: insideLink,
			wantOutside:  false,
		},
		{
			name:         "symlink resolving outside root is rejected",
			resolvedPath: outsideLink,
			wantOutside:  true,
		},
		{
			name:         "missing path leaves failure to caller",
			resolvedPath: filepath.Join(root, "missing-file"),
			wantOutside:  false,
		},
		{
			name:         "unresolvable symlink leaves failure to caller",
			resolvedPath: brokenLink,
			wantOutside:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outsideRoot, err := CheckSymlinkResolutionWithinRoot(root, tt.resolvedPath)
			if err != nil {
				t.Fatalf("CheckSymlinkResolutionWithinRoot() error = %v", err)
			}
			if outsideRoot != tt.wantOutside {
				t.Fatalf("outsideRoot = %v, want %v", outsideRoot, tt.wantOutside)
			}
		})
	}
}
