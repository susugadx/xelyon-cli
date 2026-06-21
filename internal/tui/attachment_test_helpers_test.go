package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tuiattachments "github.com/susugadx/xelyon-cli/internal/tui/attachments"
)

func writeTempFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func fillDroppedFileAttachments(t *testing.T, m *Model, dir string, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		path := writeTempFile(t, dir, fmt.Sprintf("f%02d.txt", i), []byte("a"))
		if ok := m.appendAttachment(tuiattachments.Attachment{
			Kind:   tuiattachments.KindFile,
			Source: tuiattachments.SourceDroppedPath,
			Path:   path,
			Size:   1,
		}); !ok {
			t.Fatalf("appendAttachment() = false at index %d, want true", i)
		}
	}
}

func assertTransientStatus(t *testing.T, m Model, want string) {
	t.Helper()
	if got := m.transientStatus; got != want {
		t.Fatalf("transientStatus = %q, want %q", got, want)
	}
}

func withTempWorkingDir(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})
	return dir
}
