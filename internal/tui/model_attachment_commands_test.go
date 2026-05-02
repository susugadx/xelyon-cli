package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestComposer_AttachCommandAddsAttachment(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	dir := t.TempDir()
	filePath := writeTempFile(t, dir, "notes.txt", []byte("hello"))
	m.textInput.SetValue(`/attach "` + filePath + `"`)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("/attach should be handled locally, got cmd=%v", cmd)
	}
	if got := len(m.attachments); got != 1 {
		t.Fatalf("attachments length = %d, want 1", got)
	}
	if got := m.attachments[0].Source; got != composerAttachmentSourceCommand {
		t.Fatalf("attachments[0].Source = %v, want command source", got)
	}
	if got := m.attachments[0].Path; got != filePath {
		t.Fatalf("attachments[0].Path = %q, want %q", got, filePath)
	}
	if got := agent.lastChatInput(); got != "" {
		t.Fatalf("lastChatInput() = %q, want empty", got)
	}
	if len(m.messages) == 0 || m.messages[len(m.messages)-1].Content != `/attach "`+filePath+`"` {
		t.Fatalf("last message content = %#v, want attach command text", m.messages)
	}
}

func TestComposer_AttachCommandInvalidUsageDoesNotFallbackToChat(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.textInput.SetValue("/attach")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("/attach usage error should be handled locally, got cmd=%v", cmd)
	}
	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length = %d, want 0", got)
	}
	if got := agent.lastChatInput(); got != "" {
		t.Fatalf("lastChatInput() = %q, want empty", got)
	}
	if got := m.transientStatus; got != "Usage: /attach <path>" {
		t.Fatalf("transientStatus = %q, want usage message", got)
	}
}

func TestComposer_DetachCommandRemovesAttachmentByIndex(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	dir := t.TempDir()
	first := writeTempFile(t, dir, "a.txt", []byte("a"))
	second := writeTempFile(t, dir, "b.txt", []byte("b"))
	m.appendAttachment(composerAttachment{
		Kind:   composerAttachmentFile,
		Source: composerAttachmentSourceDroppedPath,
		Path:   first,
		Size:   1,
	})
	m.appendAttachment(composerAttachment{
		Kind:   composerAttachmentFile,
		Source: composerAttachmentSourceDroppedPath,
		Path:   second,
		Size:   1,
	})
	m.textInput.SetValue("/detach #1")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("/detach should be handled locally, got cmd=%v", cmd)
	}
	if got := len(m.attachments); got != 1 {
		t.Fatalf("attachments length = %d, want 1", got)
	}
	if got := m.attachments[0].Path; got != second {
		t.Fatalf("remaining attachment path = %q, want %q", got, second)
	}
}

func TestComposer_DetachCommandInvalidUsageDoesNotFallbackToChat(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.textInput.SetValue("/detach")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("/detach usage error should be handled locally, got cmd=%v", cmd)
	}
	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length = %d, want 0", got)
	}
	if got := agent.lastChatInput(); got != "" {
		t.Fatalf("lastChatInput() = %q, want empty", got)
	}
	if got := m.transientStatus; got != "Usage: /detach <index>" {
		t.Fatalf("transientStatus = %q, want usage message", got)
	}
}

func TestComposer_DetachAllCommandRemovesClipboardTempAttachment(t *testing.T) {
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
	m.textInput.SetValue("/detach-all")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("/detach-all should be handled locally, got cmd=%v", cmd)
	}
	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length = %d, want 0", got)
	}
	if _, err := os.Stat(clipDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("os.Stat(%q) error = %v, want os.ErrNotExist", clipDir, err)
	}
}

func TestComposer_DetachAllCommandInvalidUsageDoesNotFallbackToChat(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	dir := t.TempDir()
	filePath := writeTempFile(t, dir, "notes.txt", []byte("hello"))
	if ok := m.appendAttachment(composerAttachment{
		Kind:   composerAttachmentFile,
		Source: composerAttachmentSourceDroppedPath,
		Path:   filePath,
		Size:   5,
	}); !ok {
		t.Fatal("appendAttachment() = false, want true")
	}
	m.textInput.SetValue("/detach-all now")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("/detach-all usage error should be handled locally, got cmd=%v", cmd)
	}
	if got := len(m.attachments); got != 1 {
		t.Fatalf("attachments length = %d, want 1", got)
	}
	if got := m.attachments[0].Path; got != filePath {
		t.Fatalf("attachments[0].Path = %q, want %q", got, filePath)
	}
	if got := agent.lastChatInput(); got != "" {
		t.Fatalf("lastChatInput() = %q, want empty", got)
	}
	if got := m.transientStatus; got != "Usage: /detach-all" {
		t.Fatalf("transientStatus = %q, want usage message", got)
	}
}
