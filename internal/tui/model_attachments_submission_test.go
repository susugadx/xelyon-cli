package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestComposer_SubmitImageAttachmentUsesImageChat(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	dir := t.TempDir()
	imagePath := writeTempFile(t, dir, "screen.png", []byte("png"))

	updated, _ := m.Update(pasteKey(imagePath))
	m = updated.(Model)
	m.textInput.SetValue("describe")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("Enter should return send command")
	}
	_ = cmd()

	if got := agent.lastChatInput(); got != "" {
		t.Fatalf("lastChatInput() = %q, want empty", got)
	}
	if got := agent.lastImageChatInput(); got != "describe||"+imagePath {
		t.Fatalf("lastImageChatInput() = %q, want %q", got, "describe||"+imagePath)
	}
}

func TestComposer_SubmitFileAttachmentEmbedsAttachedContext(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	dir := t.TempDir()
	filePath := writeTempFile(t, dir, "notes.txt", []byte("line1\nline2"))

	updated, _ := m.Update(pasteKey(filePath))
	m = updated.(Model)
	m.textInput.SetValue("summarize")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("Enter should return send command")
	}
	_ = cmd()

	input := agent.lastChatInput()
	if !strings.Contains(input, "summarize") {
		t.Fatalf("chat input should contain base prompt, got %q", input)
	}
	if !strings.Contains(input, "Attached context:") {
		t.Fatalf("chat input should contain attached context header, got %q", input)
	}
	if !strings.Contains(input, "line1\nline2") {
		t.Fatalf("chat input should contain file content, got %q", input)
	}
}

func TestComposer_UnhandledSlashFallbackExcludesFileAttachment(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	dir := t.TempDir()
	filePath := writeTempFile(t, dir, "payload.txt", []byte("A\nB\nC"))

	updated, _ := m.Update(pasteKey(filePath))
	m = updated.(Model)
	m.textInput.SetValue("/tmp/unknown")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("Enter should return send command for unhandled slash")
	}
	_ = cmd()

	input := agent.lastChatInput()
	if got, want := input, "/tmp/unknown"; got != want {
		t.Fatalf("chat input = %q, want %q", got, want)
	}
	if got := len(agent.handledInputs); got != 1 {
		t.Fatalf("handledInputs length = %d, want 1", got)
	}
}

func TestComposer_UnhandledSlashFallbackExcludesImageAttachment(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	dir := t.TempDir()
	imagePath := writeTempFile(t, dir, "shot.png", []byte("png"))

	updated, _ := m.Update(pasteKey(imagePath))
	m = updated.(Model)
	m.textInput.SetValue("/tmp/unknown")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("Enter should return send command for unhandled slash")
	}
	_ = cmd()

	if got := agent.lastImageChatInput(); got != "" {
		t.Fatalf("lastImageChatInput() = %q, want empty", got)
	}
	if got, want := agent.lastChatInput(), "/tmp/unknown"; got != want {
		t.Fatalf("lastChatInput() = %q, want %q", got, want)
	}
}
