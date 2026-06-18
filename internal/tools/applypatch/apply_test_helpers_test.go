package applypatch

import (
	"os"
	"path/filepath"
	"testing"
)

func withTempWorkdir(t *testing.T, fn func()) {
	t.Helper()

	root := t.TempDir()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", root, err)
	}
	defer func() {
		if err := os.Chdir(prev); err != nil {
			t.Fatalf("restore cwd error = %v", err)
		}
	}()

	fn()
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("content mismatch for %s:\n got: %q\nwant: %q", path, string(got), want)
	}
}

func assertPerm(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("perm mismatch for %s: got %o, want %o", path, got, want)
	}
}

func assertApplyResult(t *testing.T, got *ApplyResult, added, modified, deleted []string) {
	t.Helper()

	if got == nil {
		t.Fatal("ApplyResult should not be nil")
	}
	if !sameStrings(got.Added, added) {
		t.Fatalf("Added = %v, want %v", got.Added, added)
	}
	if !sameStrings(got.Modified, modified) {
		t.Fatalf("Modified = %v, want %v", got.Modified, modified)
	}
	if !sameStrings(got.Deleted, deleted) {
		t.Fatalf("Deleted = %v, want %v", got.Deleted, deleted)
	}
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
