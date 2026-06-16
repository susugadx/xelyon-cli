package resumecwd

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestMatchesAllowsLegacyOrUnknownWorkingDir(t *testing.T) {
	if !Matches("", filepath.Join(t.TempDir(), "current")) {
		t.Fatal("Matches() = false, want true for legacy session cwd")
	}
	if !Matches(filepath.Join(t.TempDir(), "session"), "") {
		t.Fatal("Matches() = false, want true for unknown current cwd")
	}
}

func TestSameNormalizedPathForOS_WindowsIgnoresPathCase(t *testing.T) {
	if !sameNormalizedPathForOS(`C:\Repo`, `c:\repo`, "windows") {
		t.Fatal("sameNormalizedPathForOS(windows) = false, want true for casing-only difference")
	}
}

func TestSameNormalizedPathForOS_NonWindowsPreservesCaseSensitivity(t *testing.T) {
	if sameNormalizedPathForOS("/tmp/Repo", "/tmp/repo", "linux") {
		t.Fatal("sameNormalizedPathForOS(linux) = true, want false for casing-only difference")
	}
	if sameNormalizedPathForOS("/tmp/Repo", "/tmp/repo", "darwin") {
		t.Fatal("sameNormalizedPathForOS(darwin) = true, want false for casing-only difference")
	}
}

func TestMatchesCleansEquivalentPaths(t *testing.T) {
	root := t.TempDir()
	sessionDir := filepath.Join(root, "repo", ".")
	currentDir := filepath.Join(root, "repo", "subdir", "..")
	if !Matches(sessionDir, currentDir) {
		t.Fatalf("Matches(%q, %q) = false, want true", sessionDir, currentDir)
	}
}

func TestMatchesKeepsNonWindowsCaseSensitive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows resume scope intentionally ignores path casing")
	}
	root := t.TempDir()
	sessionDir := filepath.Join(root, "Repo")
	currentDir := filepath.Join(root, "repo")
	if Matches(sessionDir, currentDir) {
		t.Fatalf("Matches(%q, %q) = true on this platform, want false unless running on case-insensitive Windows semantics", sessionDir, currentDir)
	}
}
