package configgen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOutputPathFromArgs(t *testing.T) {
	if got := OutputPathFromArgs([]string{"cmd"}, "default.yaml"); got != "default.yaml" {
		t.Fatalf("unexpected default output path: %s", got)
	}
	if got := OutputPathFromArgs([]string{"cmd", "custom.yaml"}, "default.yaml"); got != "custom.yaml" {
		t.Fatalf("unexpected explicit output path: %s", got)
	}
	if got := OutputPathFromArgs([]string{"cmd", "--", "custom.yaml"}, "default.yaml"); got != "custom.yaml" {
		t.Fatalf("unexpected explicit output path with --: %s", got)
	}
}

func TestReadFileIfExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	data, found, err := ReadFileIfExists(path)
	if err != nil {
		t.Fatalf("ReadFileIfExists(nonexistent) error: %v", err)
	}
	if found || data != nil {
		t.Fatalf("unexpected file result for missing file: found=%v data=%v", found, data)
	}

	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	data, found, err = ReadFileIfExists(path)
	if err != nil {
		t.Fatalf("ReadFileIfExists(existing) error: %v", err)
	}
	if !found || string(data) != "hello" {
		t.Fatalf("unexpected file result: found=%v data=%q", found, string(data))
	}
}
