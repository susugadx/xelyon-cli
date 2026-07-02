package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHeadlessGitChangedFileStateForPath_DoesNotFollowSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on Windows")
	}
	if _, err := os.Stat("/dev/zero"); err != nil {
		t.Skipf("/dev/zero unavailable: %v", err)
	}

	dir := t.TempDir()
	linkPath := filepath.Join(dir, "zero-link")
	if err := os.Symlink("/dev/zero", linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	state := headlessGitChangedFileStateForPath(dir, "zero-link", "??")

	if !state.exists {
		t.Fatal("state.exists = false, want true for symlink")
	}
	if state.mode&os.ModeSymlink == 0 {
		t.Fatalf("state.mode = %v, want symlink mode", state.mode)
	}
	if state.linkTarget != "/dev/zero" {
		t.Fatalf("state.linkTarget = %q, want /dev/zero", state.linkTarget)
	}
	if state.hash != "" {
		t.Fatalf("state.hash = %q, want no hash for symlink", state.hash)
	}
}

func TestHeadlessGitChangedFileStateForPath_RecordsPermissionBits(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits differ on Windows")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "script.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	before := headlessGitChangedFileStateForPath(dir, "script.sh", " M")
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	after := headlessGitChangedFileStateForPath(dir, "script.sh", " M")

	if before.mode.Perm() != 0o644 {
		t.Fatalf("before.mode.Perm() = %#o, want 0644", before.mode.Perm())
	}
	if after.mode.Perm() != 0o755 {
		t.Fatalf("after.mode.Perm() = %#o, want 0755", after.mode.Perm())
	}
	if before == after {
		t.Fatal("permission-only mutation produced identical changed-file states")
	}
}

func TestHeadlessGitChangedFileStateForPath_DoesNotHashLargeRegularFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.bin")
	size := headlessGitChangedFileHashLimitBytes + 1
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.Truncate(path, size); err != nil {
		t.Fatalf("Truncate() error = %v", err)
	}

	state := headlessGitChangedFileStateForPath(dir, "large.bin", "??")

	if !state.exists {
		t.Fatal("state.exists = false, want true")
	}
	if state.size != size {
		t.Fatalf("state.size = %d, want %d", state.size, size)
	}
	if !strings.HasPrefix(state.hash, "too_large:") {
		t.Fatalf("state.hash = %q, want too_large marker", state.hash)
	}
}
