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

func TestComposer_PasteExistingPathWhenAttachmentLimitReachedDoesNotFallbackToText(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	dir := t.TempDir()

	fillDroppedFileAttachments(t, &m, dir, maxComposerAttachments)

	extra := writeTempFile(t, dir, "overflow.txt", []byte("overflow"))
	updated, _ := m.Update(pasteKey(extra))
	m = updated.(Model)

	if got := len(m.attachments); got != maxComposerAttachments {
		t.Fatalf("attachments length = %d, want %d", got, maxComposerAttachments)
	}
	if got := m.textInput.Value(); got != "" {
		t.Fatalf("textInput.Value() = %q, want empty", got)
	}
	wantStatus := fmt.Sprintf("Attachment limit reached (%d max)", maxComposerAttachments)
	assertTransientStatus(t, m, wantStatus)
}

func TestComposer_CtrlVPasteClipboardImageWhenAttachmentLimitReachedCleansTemp(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	stubClipboardRead(t, "", nil)
	dir := t.TempDir()

	fillDroppedFileAttachments(t, &m, dir, maxComposerAttachments)

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

	if got := len(m.attachments); got != maxComposerAttachments {
		t.Fatalf("attachments length = %d, want %d", got, maxComposerAttachments)
	}
	wantStatus := fmt.Sprintf("Attachment limit reached (%d max)", maxComposerAttachments)
	assertTransientStatus(t, m, wantStatus)
	if _, err := os.Stat(clipDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("os.Stat(%q) error = %v, want os.ErrNotExist", clipDir, err)
	}
}

func TestComposer_PasteTooManyPathsShowsLimitWithoutTextFallback(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	dir := t.TempDir()

	paths := make([]string, 0, maxComposerAttachments+1)
	for i := 0; i < maxComposerAttachments+1; i++ {
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
	wantStatus := fmt.Sprintf("Attachment limit reached (%d max)", maxComposerAttachments)
	assertTransientStatus(t, m, wantStatus)
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
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

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
	for i := 0; i < maxComposerAttachments; i++ {
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
	start := maxComposerAttachments - len(m.visibleAttachments()) + 1
	dock := stripANSI(m.renderInputDock())
	if !strings.Contains(dock, fmt.Sprintf("#%d", start)) || !strings.Contains(dock, fmt.Sprintf("#%d", maxComposerAttachments)) {
		t.Fatalf("renderInputDock() should keep global attachment numbering, got %q", dock)
	}
}
