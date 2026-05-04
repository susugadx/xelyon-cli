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
	if got := len(m.visibleAttachments()); got != 5 {
		t.Fatalf("visibleAttachments length = %d, want 5 (remaining footer budget)", got)
	}
	start := maxComposerAttachments - len(m.visibleAttachments()) + 1
	dock := stripANSI(m.renderInputDock())
	if !strings.Contains(dock, fmt.Sprintf("#%d", start)) || !strings.Contains(dock, fmt.Sprintf("#%d", maxComposerAttachments)) {
		t.Fatalf("renderInputDock() should keep global attachment numbering, got %q", dock)
	}
}
