package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToolCache_EphemeralLoadSaveDoesNotTouchDisk(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	cacheDir := filepath.Join(tmpDir, ".xelyon", "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cacheFile := filepath.Join(cacheDir, "tool_cache.json")
	sentinel := []byte("not json{{{")
	if err := os.WriteFile(cacheFile, sentinel, 0o600); err != nil {
		t.Fatal(err)
	}

	cache := NewEphemeralToolCache()
	if err := cache.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	cache.SetFile(testFile, "content")
	if err := cache.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(got) != string(sentinel) {
		t.Fatalf("cache file = %q, want unchanged sentinel %q", string(got), string(sentinel))
	}
}
