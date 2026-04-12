package history

import (
	"os"
	"strings"
	"testing"
)

func TestStorageMetadata_ErrorPathsAndPreviewSelection(t *testing.T) {
	t.Run("saveMetadata returns write error when metadata path is directory", func(t *testing.T) {
		storage := newTestStorage(t)
		session := NewSession("test-model")

		if err := os.MkdirAll(storage.metadataPath(session.ID), 0o755); err != nil {
			t.Fatalf("MkdirAll(metadataPath) error = %v", err)
		}

		if err := storage.saveMetadata(session); err == nil || !strings.Contains(err.Error(), "failed to write metadata") {
			t.Fatalf("saveMetadata() error = %v, want write metadata error", err)
		}
	})

	t.Run("loadMetadata returns read error when file is missing", func(t *testing.T) {
		storage := newTestStorage(t)

		if _, err := storage.loadMetadata("missing"); err == nil || !strings.Contains(err.Error(), "failed to read metadata") {
			t.Fatalf("loadMetadata() error = %v, want read metadata error", err)
		}
	})

	t.Run("saveMetadata uses first user message after skipping non-user entries", func(t *testing.T) {
		storage := newTestStorage(t)
		session := NewSession("test-model")
		session.Messages = []MessageEntry{
			{Role: "assistant", Content: "assistant reply"},
			{Role: "user", Content: "first user preview"},
			{Role: "user", Content: "second user preview"},
		}

		if err := storage.saveMetadata(session); err != nil {
			t.Fatalf("saveMetadata() error = %v", err)
		}

		meta, err := storage.loadMetadata(session.ID)
		if err != nil {
			t.Fatalf("loadMetadata() error = %v", err)
		}
		if meta.Preview != "first user preview" {
			t.Fatalf("Preview = %q, want first user message", meta.Preview)
		}
	})
}
