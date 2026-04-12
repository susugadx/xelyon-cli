package navigation

import (
	"path/filepath"
	"testing"
)

func TestBuildSnapshotPathMatcher_GetwdAndCleanHintFallbacks(t *testing.T) {
	t.Run("uses current working directory when invocation cwd is empty", func(t *testing.T) {
		tmp := t.TempDir()
		t.Chdir(tmp)

		matcher := buildSnapshotPathMatcher("", "", "pkg")
		if matcher == nil {
			t.Fatal("buildSnapshotPathMatcher() = nil, want directory matcher")
		}
		if !matcher("pkg/run.go") || !matcher("pkg/sub/other.go") || matcher("other/run.go") {
			t.Fatal("directory matcher should use cwd-relative clean hint")
		}
	})

	t.Run("falls back to clean hint when absolute hint is outside snapshot root", func(t *testing.T) {
		root := t.TempDir()
		outside := t.TempDir()

		matcher := buildSnapshotPathMatcher(root, outside, "pkg")
		if matcher == nil {
			t.Fatal("buildSnapshotPathMatcher() = nil, want clean-hint directory matcher")
		}
		if !matcher("pkg/run.go") || !matcher("pkg/sub/other.go") || matcher("other/run.go") {
			t.Fatal("fallback matcher should use clean relative hint")
		}
	})

	t.Run("absolute file hint outside root does not match root-relative candidates", func(t *testing.T) {
		root := t.TempDir()
		outsideFile := filepath.Join(t.TempDir(), "run.go")

		matcher := buildSnapshotPathMatcher(root, "", outsideFile)
		if matcher == nil {
			t.Fatal("buildSnapshotPathMatcher() = nil, want file matcher")
		}
		if matcher("pkg/run.go") {
			t.Fatal("file matcher for outside absolute path should not match unrelated root-relative file")
		}
		if !matcher(filepath.Clean(filepath.ToSlash(outsideFile))) {
			t.Fatal("file matcher should match the cleaned absolute hint path")
		}
	})
}
