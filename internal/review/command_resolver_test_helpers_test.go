package review

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func createCommandResolverTestExecutable(t *testing.T, dir, base string) string {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", dir, err)
	}

	name := base
	content := "#!/bin/sh\nexit 0\n"
	mode := os.FileMode(0o755)

	if runtime.GOOS == "windows" {
		name = base + ".cmd"
		content = "@echo off\r\nexit /b 0\r\n"
		mode = 0o644
	}

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) error = %v", path, err)
	}
	return filepath.Clean(absPath)
}
