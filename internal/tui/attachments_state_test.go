package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestComposer_ClearComposerRemovesTemporaryClipboardAttachment(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	root := t.TempDir()
	clipDir, err := os.MkdirTemp(root, "clip-")
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}
	imagePath := filepath.Join(clipDir, clipboardAttachmentFileName)
	if err := os.WriteFile(imagePath, []byte("png"), 0644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", imagePath, err)
	}

	if ok := m.appendAttachment(composerAttachment{
		Kind:   composerAttachmentImage,
		Source: composerAttachmentSourceClipboardImage,
		Path:   imagePath,
	}); !ok {
		t.Fatal("appendAttachment() = false, want true")
	}

	m.clearComposer()

	if _, err := os.Stat(clipDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("os.Stat(%q) error = %v, want os.ErrNotExist", clipDir, err)
	}
}

func TestComposer_ClearComposerKeepsRegularFileAttachment(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	dir := t.TempDir()
	filePath := writeTempFile(t, dir, "notes.txt", []byte("hello"))

	if ok := m.appendAttachment(composerAttachment{
		Kind: composerAttachmentFile,
		Path: filePath,
	}); !ok {
		t.Fatal("appendAttachment() = false, want true")
	}

	m.clearComposer()

	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("os.Stat(%q) error = %v, want nil", filePath, err)
	}
}

func TestComposer_ClearComposerDoesNotRemoveDroppedClipboardNamedFile(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	dir := t.TempDir()
	filePath := writeTempFile(t, dir, clipboardAttachmentFileName, []byte("hello"))

	if ok := m.appendAttachment(composerAttachment{
		Kind:   composerAttachmentFile,
		Source: composerAttachmentSourceDroppedPath,
		Path:   filePath,
	}); !ok {
		t.Fatal("appendAttachment() = false, want true")
	}

	m.clearComposer()

	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("os.Stat(%q) error = %v, want nil", filePath, err)
	}
}

func TestComposer_AppendAttachmentRejectsWhenLimitReached(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	dir := t.TempDir()

	fillDroppedFileAttachments(t, &m, dir, maxComposerAttachments)

	extra := writeTempFile(t, dir, "extra.txt", []byte("x"))
	if ok := m.appendAttachment(composerAttachment{Kind: composerAttachmentFile, Path: extra}); ok {
		t.Fatal("appendAttachment() = true for 13th item, want false")
	}
	if got := len(m.attachments); got != maxComposerAttachments {
		t.Fatalf("attachments length = %d, want %d", got, maxComposerAttachments)
	}
}
