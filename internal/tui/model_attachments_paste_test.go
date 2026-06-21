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
	if m.attachments[0].Kind != tuiattachments.KindImage {
		t.Fatalf("attachments[0].Kind = %v, want image", m.attachments[0].Kind)
	}
	if m.attachments[0].Source != tuiattachments.SourceDroppedPath {
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
	if got := m.attachments[0].Kind; got != tuiattachments.KindImage {
		t.Fatalf("attachments[0].Kind = %v, want image", got)
	}
	if got := m.attachments[0].Source; got != tuiattachments.SourceClipboardImage {
		t.Fatalf("attachments[0].Source = %v, want clipboard image source", got)
	}
}

func TestComposer_PasteExistingPathWhenAttachmentLimitReachedDoesNotFallbackToText(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	dir := t.TempDir()

	fillDroppedFileAttachments(t, &m, dir, tuiattachments.MaxComposerAttachments)

	extra := writeTempFile(t, dir, "overflow.txt", []byte("overflow"))
	updated, _ := m.Update(pasteKey(extra))
	m = updated.(Model)

	if got := len(m.attachments); got != tuiattachments.MaxComposerAttachments {
		t.Fatalf("attachments length = %d, want %d", got, tuiattachments.MaxComposerAttachments)
	}
	if got := m.textInput.Value(); got != "" {
		t.Fatalf("textInput.Value() = %q, want empty", got)
	}
	wantStatus := fmt.Sprintf("Attachment limit reached (%d max)", tuiattachments.MaxComposerAttachments)
	assertTransientStatus(t, m, wantStatus)
}

func TestComposer_CtrlVPasteClipboardImageWhenAttachmentLimitReachedCleansTemp(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	stubClipboardRead(t, "", nil)
	dir := t.TempDir()

	fillDroppedFileAttachments(t, &m, dir, tuiattachments.MaxComposerAttachments)

	clipDir, err := os.MkdirTemp(dir, "clip-")
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}
	imagePath := filepath.Join(clipDir, clipboardAttachmentFileName)
	if err := os.WriteFile(imagePath, []byte("png"), 0644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", imagePath, err)
	}

	prevSaver := saveClipboardImageForPaste
	saveClipboardImageForPaste = func() (string, error) {
		return imagePath, nil
	}
	t.Cleanup(func() {
		saveClipboardImageForPaste = prevSaver
	})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlV})
	m = updated.(Model)

	if got := len(m.attachments); got != tuiattachments.MaxComposerAttachments {
		t.Fatalf("attachments length = %d, want %d", got, tuiattachments.MaxComposerAttachments)
	}
	wantStatus := fmt.Sprintf("Attachment limit reached (%d max)", tuiattachments.MaxComposerAttachments)
	assertTransientStatus(t, m, wantStatus)
	if _, err := os.Stat(clipDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("os.Stat(%q) error = %v, want os.ErrNotExist", clipDir, err)
	}
}

func TestComposer_PasteTooManyPathsShowsLimitWithoutTextFallback(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	dir := t.TempDir()

	paths := make([]string, 0, tuiattachments.MaxComposerAttachments+1)
	for i := 0; i < tuiattachments.MaxComposerAttachments+1; i++ {
		path := writeTempFile(t, dir, fmt.Sprintf("f%02d.txt", i), []byte("a"))
		paths = append(paths, path)
	}

	updated, _ := m.Update(pasteKey(strings.Join(paths, "\n")))
	m = updated.(Model)

	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length = %d, want 0", got)
	}
	if got := m.textInput.Value(); got != "" {
		t.Fatalf("textInput.Value() = %q, want empty", got)
	}
	wantStatus := fmt.Sprintf("Attachment limit reached (%d max)", tuiattachments.MaxComposerAttachments)
	assertTransientStatus(t, m, wantStatus)
}

func TestComposer_PasteTooManyDuplicatePathsShowsLimitWithoutTextFallback(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	dir := t.TempDir()

	path := writeTempFile(t, dir, "same.txt", []byte("a"))
	lines := make([]string, 0, tuiattachments.MaxComposerAttachments+1)
	for i := 0; i < tuiattachments.MaxComposerAttachments+1; i++ {
		lines = append(lines, path)
	}

	updated, _ := m.Update(pasteKey(strings.Join(lines, "\n")))
	m = updated.(Model)

	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length = %d, want 0", got)
	}
	if got := m.textInput.Value(); got != "" {
		t.Fatalf("textInput.Value() = %q, want empty", got)
	}
	wantStatus := fmt.Sprintf("Attachment limit reached (%d max)", tuiattachments.MaxComposerAttachments)
	assertTransientStatus(t, m, wantStatus)
}

func TestComposer_PasteMixedExistingPathAndPlainTextFallsBackToText(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	dir := t.TempDir()
	path := writeTempFile(t, dir, "notes.txt", []byte("x"))
	payload := path + "\nplain text"

	updated, _ := m.Update(pasteKey(payload))
	m = updated.(Model)

	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length = %d, want 0", got)
	}
	if got := m.buildComposerPayload(); got != payload {
		t.Fatalf("buildComposerPayload() = %q, want %q", got, payload)
	}
}

func TestComposer_PasteNonExistentPathFallsBackToText(t *testing.T) {
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

func TestComposer_PasteUnterminatedQuoteTextFallsBackToText(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	payload := `"/tmp/not-closed`
	updated, _ := m.Update(pasteKey(payload))
	m = updated.(Model)

	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length = %d, want 0", got)
	}
	if got := m.textInput.Value(); got != payload {
		t.Fatalf("textInput.Value() = %q, want %q", got, payload)
	}
	if got := m.transientStatus; got != "" {
		t.Fatalf("transientStatus = %q, want empty", got)
	}
}

func TestComposer_PasteApostropheTextFallsBackToText(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	payload := "don't panic"
	updated, _ := m.Update(pasteKey(payload))
	m = updated.(Model)

	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length = %d, want 0", got)
	}
	if got := m.textInput.Value(); got != payload {
		t.Fatalf("textInput.Value() = %q, want %q", got, payload)
	}
}

func TestComposer_PasteURLFallsBackToText(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	url := "https://example.com/docs"
	updated, _ := m.Update(pasteKey(url))
	m = updated.(Model)

	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length = %d, want 0", got)
	}
	if got := m.textInput.Value(); got != url {
		t.Fatalf("textInput.Value() = %q, want %q", got, url)
	}
}

func TestComposer_PasteSlashContainingTextFallsBackToText(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	payload := "a/b testing"
	updated, _ := m.Update(pasteKey(payload))
	m = updated.(Model)

	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length = %d, want 0", got)
	}
	if got := m.textInput.Value(); got != payload {
		t.Fatalf("textInput.Value() = %q, want %q", got, payload)
	}
}

func TestComposer_PasteBareRelativeFilenameAttachesFile(t *testing.T) {
	dir := withTempWorkingDir(t)

	tests := []struct {
		name      string
		fileName  string
		pasteText string
	}{
		{name: "with extension", fileName: "notes.txt", pasteText: "notes.txt"},
		{name: "without extension", fileName: "Makefile", pasteText: "Makefile"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			agent := &stubAgent{statusLine: "ready"}
			m := newModelWithViewport(agent)

			path := writeTempFile(t, dir, tc.fileName, []byte("hello"))
			updated, _ := m.Update(pasteKey(tc.pasteText))
			m = updated.(Model)

			if got := len(m.attachments); got != 1 {
				t.Fatalf("attachments length = %d, want 1", got)
			}
			if got := m.attachments[0].Path; got != path {
				t.Fatalf("attachments[0].Path = %q, want %q", got, path)
			}
			if got := m.textInput.Value(); got != "" {
				t.Fatalf("textInput.Value() = %q, want empty", got)
			}
		})
	}
}

func TestComposer_PasteBareWordWithoutExistingFileFallsBackToText(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	_ = withTempWorkingDir(t)

	payload := "README"
	updated, _ := m.Update(pasteKey(payload))
	m = updated.(Model)

	if got := len(m.attachments); got != 0 {
		t.Fatalf("attachments length = %d, want 0", got)
	}
	if got := m.textInput.Value(); got != payload {
		t.Fatalf("textInput.Value() = %q, want %q", got, payload)
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
