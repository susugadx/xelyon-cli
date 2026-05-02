package tui

import (
	"fmt"
	"testing"
)

func fillDroppedFileAttachments(t *testing.T, m *Model, dir string, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		path := writeTempFile(t, dir, fmt.Sprintf("f%02d.txt", i), []byte("a"))
		if ok := m.appendAttachment(composerAttachment{
			Kind:   composerAttachmentFile,
			Source: composerAttachmentSourceDroppedPath,
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
