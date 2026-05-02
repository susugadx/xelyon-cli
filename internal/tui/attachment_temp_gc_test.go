package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupStaleClipboardAttachmentTemps(t *testing.T) {
	root := t.TempDir()
	now := time.Now()

	oldDir := filepath.Join(root, clipboardAttachmentTempDirPrefix+"old")
	if err := os.MkdirAll(oldDir, 0755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", oldDir, err)
	}
	freshDir := filepath.Join(root, clipboardAttachmentTempDirPrefix+"fresh")
	if err := os.MkdirAll(freshDir, 0755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", freshDir, err)
	}
	otherDir := filepath.Join(root, "unrelated-temp-dir")
	if err := os.MkdirAll(otherDir, 0755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", otherDir, err)
	}

	oldTime := now.Add(-2 * staleClipboardAttachmentTTL)
	if err := os.Chtimes(oldDir, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes(%q) error = %v", oldDir, err)
	}
	freshTime := now.Add(-(staleClipboardAttachmentTTL / 2))
	if err := os.Chtimes(freshDir, freshTime, freshTime); err != nil {
		t.Fatalf("Chtimes(%q) error = %v", freshDir, err)
	}

	prevRoot := clipboardTempRootDir
	clipboardTempRootDir = func() string { return root }
	t.Cleanup(func() {
		clipboardTempRootDir = prevRoot
	})

	removed := cleanupStaleClipboardAttachmentTemps(now)
	if removed != 1 {
		t.Fatalf("cleanupStaleClipboardAttachmentTemps() removed = %d, want 1", removed)
	}
	if _, err := os.Stat(oldDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("os.Stat(%q) error = %v, want os.ErrNotExist", oldDir, err)
	}
	if _, err := os.Stat(freshDir); err != nil {
		t.Fatalf("os.Stat(%q) error = %v, want nil", freshDir, err)
	}
	if _, err := os.Stat(otherDir); err != nil {
		t.Fatalf("os.Stat(%q) error = %v, want nil", otherDir, err)
	}
}
