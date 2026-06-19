package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	tuiattachments "github.com/susugadx/xelyon-cli/internal/tui/attachments"
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
	if got := m.attachments[0].Source; got != tuiattachments.SourceCommand {
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

func TestComposer_AttachCommandRespectsAttachmentLimit(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	dir := t.TempDir()

	fillDroppedFileAttachments(t, &m, dir, tuiattachments.MaxComposerAttachments)

	extra := writeTempFile(t, dir, "extra.txt", []byte("x"))
	m.textInput.SetValue("/attach " + extra)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("/attach limit error should be handled locally, got cmd=%v", cmd)
	}
	if got := len(m.attachments); got != tuiattachments.MaxComposerAttachments {
		t.Fatalf("attachments length = %d, want %d", got, tuiattachments.MaxComposerAttachments)
	}
	if got := agent.lastChatInput(); got != "" {
		t.Fatalf("lastChatInput() = %q, want empty", got)
	}
	wantStatus := fmt.Sprintf("Attachment limit reached (%d max)", tuiattachments.MaxComposerAttachments)
	assertTransientStatus(t, m, wantStatus)
}

func TestComposer_CommandWithUnterminatedQuoteShowsSyntaxError(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.textInput.SetValue(`/attach "broken`)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("unterminated quote should be handled locally, got cmd=%v", cmd)
	}
	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length = %d, want 0", got)
	}
	if got := agent.lastChatInput(); got != "" {
		t.Fatalf("lastChatInput() = %q, want empty", got)
	}
	assertTransientStatus(t, m, "Invalid command syntax: unmatched quote")
}

func TestComposer_UnhandledCommandWithUnterminatedQuoteFallsBackToChat(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.textInput.SetValue(`/note "unfinished`)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("unhandled malformed slash input should return chat send command")
	}
	_ = cmd()

	if got := agent.lastChatInput(); got != `/note "unfinished` {
		t.Fatalf("lastChatInput() = %q, want %q", got, `/note "unfinished`)
	}
}

func TestComposer_UnhandledMalformedSlashFallbackExcludesAttachments(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	dir := t.TempDir()
	filePath := writeTempFile(t, dir, "notes.txt", []byte("hello"))
	if ok := m.appendAttachment(tuiattachments.Attachment{
		Kind:   tuiattachments.KindFile,
		Source: tuiattachments.SourceDroppedPath,
		Path:   filePath,
		Size:   5,
	}); !ok {
		t.Fatal("appendAttachment() = false, want true")
	}
	m.textInput.SetValue(`/note "unfinished`)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("unhandled malformed slash input should return chat send command")
	}
	_ = cmd()

	if got := agent.lastChatInput(); got != `/note "unfinished` {
		t.Fatalf("lastChatInput() = %q, want %q", got, `/note "unfinished`)
	}
	if strings.Contains(agent.lastChatInput(), "Attached context:") {
		t.Fatalf("lastChatInput() should exclude attachment context, got %q", agent.lastChatInput())
	}
}

func TestComposer_DetachCommandRemovesAttachmentByIndex(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	dir := t.TempDir()
	first := writeTempFile(t, dir, "a.txt", []byte("a"))
	second := writeTempFile(t, dir, "b.txt", []byte("b"))
	m.appendAttachment(tuiattachments.Attachment{
		Kind:   tuiattachments.KindFile,
		Source: tuiattachments.SourceDroppedPath,
		Path:   first,
		Size:   1,
	})
	m.appendAttachment(tuiattachments.Attachment{
		Kind:   tuiattachments.KindFile,
		Source: tuiattachments.SourceDroppedPath,
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
	assertTransientStatus(t, m, "Usage: /detach <index>")
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
	if ok := m.appendAttachment(tuiattachments.Attachment{
		Kind:   tuiattachments.KindImage,
		Source: tuiattachments.SourceClipboardImage,
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
	if ok := m.appendAttachment(tuiattachments.Attachment{
		Kind:   tuiattachments.KindFile,
		Source: tuiattachments.SourceDroppedPath,
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
	assertTransientStatus(t, m, "Usage: /detach-all")
}
