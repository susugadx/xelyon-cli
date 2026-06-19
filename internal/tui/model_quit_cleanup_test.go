package tui

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	tuiattachments "github.com/susugadx/xelyon-cli/internal/tui/attachments"
)

func attachTemporaryClipboardImage(t *testing.T, m *Model) string {
	t.Helper()

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

	return clipDir
}

func TestComposer_QuitCommandCleansTemporaryAttachments(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	clipDir := attachTemporaryClipboardImage(t, &m)
	m.textInput.SetValue("/quit")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("quit command should return tea.Quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("quit command = %T, want tea.QuitMsg", cmd())
	}
	if _, err := os.Stat(clipDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("os.Stat(%q) error = %v, want os.ErrNotExist", clipDir, err)
	}
	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length after quit = %d, want 0", got)
	}
}

func TestComposer_CtrlCQuitCleansTemporaryAttachments(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	clipDir := attachTemporaryClipboardImage(t, &m)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)
	if cmd != nil {
		t.Fatal("first ctrl+c should not quit")
	}
	if _, err := os.Stat(clipDir); err != nil {
		t.Fatalf("first ctrl+c should not remove temp attachment: %v", err)
	}

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("second ctrl+c should return tea.Quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("second ctrl+c command = %T, want tea.QuitMsg", cmd())
	}
	if _, err := os.Stat(clipDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("os.Stat(%q) error = %v, want os.ErrNotExist", clipDir, err)
	}
	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length after ctrl+c quit = %d, want 0", got)
	}
}
