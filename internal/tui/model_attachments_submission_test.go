package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

type submissionCleanupProbeAgent struct {
	*stubAgent

	observePath               string
	imagePathExistsAtSend     bool
	chatPathExistsAtSend      bool
	imagePathObservedForCheck string
	mu                        sync.RWMutex
}

func (a *submissionCleanupProbeAgent) setImagePathObserved(path string, exists bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.imagePathObservedForCheck = path
	a.imagePathExistsAtSend = exists
}

func (a *submissionCleanupProbeAgent) setChatPathObserved(exists bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.chatPathExistsAtSend = exists
}

func (a *submissionCleanupProbeAgent) imagePathExistedAtSend() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.imagePathExistsAtSend
}

func (a *submissionCleanupProbeAgent) chatPathExistedAtSend() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.chatPathExistsAtSend
}

func (a *submissionCleanupProbeAgent) Chat(input string) {
	if a.observePath != "" {
		_, err := os.Stat(a.observePath)
		a.setChatPathObserved(err == nil)
	}
	a.stubAgent.Chat(input)
}

func (a *submissionCleanupProbeAgent) ChatWithImagePath(input string, imagePath string) {
	_, err := os.Stat(imagePath)
	a.setImagePathObserved(imagePath, err == nil)
	a.stubAgent.ChatWithImagePath(input, imagePath)
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

func TestComposer_SubmitClipboardImageAttachmentKeepsPathUntilSendAndCleansAfterwards(t *testing.T) {
	probe := &submissionCleanupProbeAgent{
		stubAgent: &stubAgent{statusLine: "ready"},
	}
	m := newModelWithViewport(probe)
	clipDir := attachTemporaryClipboardImage(t, &m)
	imagePath := filepath.Join(clipDir, clipboardAttachmentFileName)

	m.textInput.SetValue("inspect")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("Enter should return send command")
	}
	_ = cmd()

	if got := probe.lastImageChatInput(); got != "inspect||"+imagePath {
		t.Fatalf("lastImageChatInput() = %q, want %q", got, "inspect||"+imagePath)
	}
	if !probe.imagePathExistedAtSend() {
		t.Fatalf("expected image path %q to exist when ChatWithImagePath was called", imagePath)
	}
	if _, err := os.Stat(clipDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("os.Stat(%q) error = %v, want os.ErrNotExist", clipDir, err)
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

func TestComposer_UnhandledSlashFallbackCleansClipboardImageAfterSend(t *testing.T) {
	clipDir := ""
	probe := &submissionCleanupProbeAgent{
		stubAgent:   &stubAgent{statusLine: "ready"},
		observePath: "",
	}
	m := newModelWithViewport(probe)
	clipDir = attachTemporaryClipboardImage(t, &m)
	probe.observePath = filepath.Join(clipDir, clipboardAttachmentFileName)

	m.textInput.SetValue("/tmp/unknown")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("Enter should return send command for unhandled slash")
	}
	_ = cmd()

	if got := probe.lastChatInput(); got != "/tmp/unknown" {
		t.Fatalf("lastChatInput() = %q, want %q", got, "/tmp/unknown")
	}
	if !probe.chatPathExistedAtSend() {
		t.Fatalf("expected clipboard attachment file %q to exist when Chat was called", probe.imagePathObservedForCheck)
	}
	if _, err := os.Stat(clipDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("os.Stat(%q) error = %v, want os.ErrNotExist", clipDir, err)
	}
}
