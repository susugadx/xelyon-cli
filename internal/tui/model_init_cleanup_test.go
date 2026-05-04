package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestModelInit_CleansStaleClipboardAttachmentTemps(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, clipboardAttachmentTempDirPrefix+"legacy")
	if err := os.MkdirAll(oldDir, 0755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", oldDir, err)
	}
	oldTime := time.Now().Add(-2 * staleClipboardAttachmentTTL)
	if err := os.Chtimes(oldDir, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes(%q) error = %v", oldDir, err)
	}

	prevRoot := clipboardTempRootDir
	clipboardTempRootDir = func() string { return root }
	t.Cleanup(func() {
		clipboardTempRootDir = prevRoot
	})

	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	_ = m.Init()

	if _, err := os.Stat(oldDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("os.Stat(%q) error = %v, want os.ErrNotExist", oldDir, err)
	}
}
