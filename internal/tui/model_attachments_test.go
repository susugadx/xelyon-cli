package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func writeTempFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func TestComposer_PasteExistingImagePathAttachesImage(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	dir := t.TempDir()
	imagePath := writeTempFile(t, dir, "ui shot.png", []byte("png"))

	updated, _ := m.Update(pasteKey("\"" + imagePath + "\""))
	m = updated.(Model)

	if got := len(m.attachments); got != 1 {
		t.Fatalf("attachments length = %d, want 1", got)
	}
	if m.attachments[0].Kind != composerAttachmentImage {
		t.Fatalf("attachments[0].Kind = %v, want image", m.attachments[0].Kind)
	}
	if m.attachments[0].Source != composerAttachmentSourceDroppedPath {
		t.Fatalf("attachments[0].Source = %v, want dropped path source", m.attachments[0].Source)
	}
	if got := m.attachments[0].Path; got != imagePath {
		t.Fatalf("attachments[0].Path = %q, want %q", got, imagePath)
	}
	if got := m.textInput.Value(); got != "" {
		t.Fatalf("textInput.Value() = %q, want empty", got)
	}
	if got := len(m.composer.PasteBlocks); got != 0 {
		t.Fatalf("pasteBlocks length = %d, want 0", got)
	}
}

func TestComposer_BackspaceRemovesAttachment(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	dir := t.TempDir()
	filePath := writeTempFile(t, dir, "notes.txt", []byte("hello"))

	updated, _ := m.Update(pasteKey(filePath))
	m = updated.(Model)
	if got := len(m.attachments); got != 1 {
		t.Fatalf("attachments length before backspace = %d, want 1", got)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(Model)
	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length after backspace = %d, want 0", got)
	}
}

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

func TestComposer_UnhandledSlashFallbackPreservesFileAttachment(t *testing.T) {
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
	if !strings.Contains(input, "Attached context:") {
		t.Fatalf("chat input should include attached context in slash fallback, got %q", input)
	}
	if !strings.Contains(input, "A\nB\nC") {
		t.Fatalf("chat input should include attached file body in slash fallback, got %q", input)
	}
	if got := len(agent.handledInputs); got != 1 {
		t.Fatalf("handledInputs length = %d, want 1", got)
	}
}

func TestComposer_UnhandledSlashFallbackPreservesImageAttachment(t *testing.T) {
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

	if got := agent.lastImageChatInput(); got != "/tmp/unknown||"+imagePath {
		t.Fatalf("lastImageChatInput() = %q, want %q", got, "/tmp/unknown||"+imagePath)
	}
}

func TestComposer_CtrlVPasteFallsBackToClipboardImage(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	stubClipboardRead(t, "", nil)

	dir := t.TempDir()
	imagePath := writeTempFile(t, dir, "clip.png", []byte("png"))
	prevSaver := saveClipboardImageForPaste
	saveClipboardImageForPaste = func() (string, error) {
		return imagePath, nil
	}
	t.Cleanup(func() {
		saveClipboardImageForPaste = prevSaver
	})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlV})
	m = updated.(Model)

	if got := len(m.attachments); got != 1 {
		t.Fatalf("attachments length = %d, want 1", got)
	}
	if got := m.attachments[0].Kind; got != composerAttachmentImage {
		t.Fatalf("attachments[0].Kind = %v, want image", got)
	}
	if got := m.attachments[0].Source; got != composerAttachmentSourceClipboardImage {
		t.Fatalf("attachments[0].Source = %v, want clipboard image source", got)
	}
}

func TestComposer_PasteNonExistentPathFallsBackToTextPaste(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	missing := filepath.Join(t.TempDir(), "missing.txt")
	updated, _ := m.Update(pasteKey(missing))
	m = updated.(Model)

	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length = %d, want 0", got)
	}
	if got := m.textInput.Value(); got != missing {
		t.Fatalf("textInput.Value() = %q, want %q", got, missing)
	}
}

func TestComposer_CtrlVPasteWithClipboardErrorDoesNotAttach(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	stubClipboardRead(t, "", errors.New("clipboard unavailable"))

	prevSaver := saveClipboardImageForPaste
	saveClipboardImageForPaste = func() (string, error) {
		return "", errors.New("no image")
	}
	t.Cleanup(func() {
		saveClipboardImageForPaste = prevSaver
	})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlV})
	m = updated.(Model)
	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length = %d, want 0", got)
	}
}

func TestComposer_AttachmentRowsRespectFooterBudgetWithComposerRows(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	for i := 0; i < 20; i++ {
		m.handleComposerPaste("line1\nline2")
	}

	dir := t.TempDir()
	for i := 0; i < 20; i++ {
		path := writeTempFile(t, dir, fmt.Sprintf("f%02d.txt", i), []byte("a"))
		m.appendAttachment(composerAttachment{Kind: composerAttachmentFile, Path: path, Size: 1})
	}

	if got := m.footerHeight(); got > m.height {
		t.Fatalf("footerHeight() = %d, want <= model height %d", got, m.height)
	}
	if got := len(m.visibleComposerRows()); got != 20 {
		t.Fatalf("visibleComposerRows length = %d, want 20", got)
	}
	if got := len(m.visibleAttachments()); got != 5 {
		t.Fatalf("visibleAttachments length = %d, want 5 (remaining footer budget)", got)
	}
	dock := stripANSI(m.renderInputDock())
	if !strings.Contains(dock, "#16") || !strings.Contains(dock, "#20") {
		t.Fatalf("renderInputDock() should keep global attachment numbering, got %q", dock)
	}
}
