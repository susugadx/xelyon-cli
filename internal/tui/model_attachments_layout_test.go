package tui

import (
	"fmt"
	"strings"
	"testing"
)

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
	if got := m.visibleCompactChipRowCount(); got != 1 {
		t.Fatalf("visibleCompactChipRowCount() = %d, want 1", got)
	}
	dock := stripANSI(m.renderInputDock())
	if !strings.Contains(dock, "#1") {
		t.Fatalf("renderInputDock() should keep compact attachment numbering, got %q", dock)
	}
}

func TestComposer_CompactChipsHiddenWhenNoFooterRowsFit(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.applyChatWindowSize(80, statusBarHeight+inputHeight+minChatViewportHeight)

	dir := t.TempDir()
	path := writeTempFile(t, dir, "notes.txt", []byte("a"))
	m.appendAttachment(composerAttachment{Kind: composerAttachmentFile, Path: path, Size: 1})

	if got := m.visibleCompactChipRowCount(); got != 0 {
		t.Fatalf("visibleCompactChipRowCount() = %d, want 0 with no footer expansion rows", got)
	}
	if got := m.footerHeight(); got != statusBarHeight+inputHeight {
		t.Fatalf("footerHeight() = %d, want fixed footer height %d", got, statusBarHeight+inputHeight)
	}
	dock := stripANSI(m.renderInputDock())
	if strings.Contains(dock, "[file") {
		t.Fatalf("renderInputDock() should hide compact chips when no rows fit, got %q", dock)
	}
}
