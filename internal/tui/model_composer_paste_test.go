package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestComposer_ShortSingleLinePasteStaysInInput(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	updated, _ := m.Update(pasteKey("short paste"))
	m = updated.(Model)

	if got := m.textInput.Value(); got != "short paste" {
		t.Fatalf("textInput.Value() = %q, want %q", got, "short paste")
	}
	if len(m.pasteBlocks) != 0 {
		t.Fatalf("pasteBlocks length = %d, want 0", len(m.pasteBlocks))
	}
	if got := m.footerHeight(); got != statusBarHeight+inputHeight {
		t.Fatalf("footerHeight() = %d, want %d", got, statusBarHeight+inputHeight)
	}
}

func TestComposer_MultilinePasteCreatesFoldedBlock(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	updated, _ := m.Update(pasteKey("line1\nline2"))
	m = updated.(Model)

	if got := len(m.pasteBlocks); got != 1 {
		t.Fatalf("pasteBlocks length = %d, want 1", got)
	}
	if got := m.textInput.Value(); got != "" {
		t.Fatalf("textInput.Value() = %q, want empty", got)
	}
	if got := m.footerHeight(); got != statusBarHeight+inputHeight+1 {
		t.Fatalf("footerHeight() = %d, want %d", got, statusBarHeight+inputHeight+1)
	}
	if got := m.vp.height; got != m.height-m.footerHeight() {
		t.Fatalf("vp.height = %d, want %d", got, m.height-m.footerHeight())
	}

	dock := m.renderInputDock()
	if !strings.Contains(stripANSI(dock), "[Pasted Content 11 chars, 2 lines] #1") {
		t.Fatalf("renderInputDock() should contain folded paste summary, got %q", dock)
	}
	if strings.Contains(dock, "line1\nline2") {
		t.Fatalf("renderInputDock() should not contain raw pasted content, got %q", dock)
	}
}

func TestComposer_LongSingleLinePasteCreatesFoldedBlock(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	longLine := strings.Repeat("x", pasteBlockFoldThreshold)

	updated, _ := m.Update(pasteKey(longLine))
	m = updated.(Model)

	if got := len(m.pasteBlocks); got != 1 {
		t.Fatalf("pasteBlocks length = %d, want 1", got)
	}
	if got := m.textInput.Value(); got != "" {
		t.Fatalf("textInput.Value() = %q, want empty", got)
	}
	if strings.Contains(m.renderInputDock(), longLine) {
		t.Fatal("renderInputDock() should not inline a folded long paste")
	}
}

func TestComposer_PrefixTextStaysVisibleWithFoldedPaste(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	m.textInput.SetValue("Explain this:")
	updated, _ := m.Update(pasteKey("line1\nline2"))
	m = updated.(Model)

	dock := stripANSI(m.renderInputDock())
	prefixIndex := strings.Index(dock, "Explain this:")
	pasteIndex := strings.Index(dock, "[Pasted Content 11 chars, 2 lines] #1")
	if prefixIndex < 0 {
		t.Fatalf("renderInputDock() should contain the prefix text, got %q", dock)
	}
	if pasteIndex < 0 {
		t.Fatalf("renderInputDock() should contain the folded paste summary, got %q", dock)
	}
	if prefixIndex >= pasteIndex {
		t.Fatalf("prefix text should render before folded paste summary, got %q", dock)
	}
	if got := m.buildComposerPayload(); got != "Explain this:line1\nline2" {
		t.Fatalf("buildComposerPayload() = %q, want %q", got, "Explain this:line1\nline2")
	}
}

func TestComposer_CtrlVPasteCreatesFoldedBlockAndPreservesContent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	stubClipboardRead(t, "line1\tvalue\nline2", nil)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlV})
	m = updated.(Model)

	if got := len(m.pasteBlocks); got != 1 {
		t.Fatalf("pasteBlocks length = %d, want 1", got)
	}
	if got := m.textInput.Value(); got != "" {
		t.Fatalf("textInput.Value() = %q, want empty", got)
	}
	if got := m.buildComposerPayload(); got != "line1\tvalue\nline2" {
		t.Fatalf("buildComposerPayload() = %q, want %q", got, "line1\tvalue\nline2")
	}
	if !strings.Contains(stripANSI(m.renderInputDock()), "[Pasted Content 17 chars, 2 lines] #1") {
		t.Fatalf("renderInputDock() should contain folded paste summary, got %q", m.renderInputDock())
	}
}

func TestComposer_BackspaceRemovesLastPasteBlockWhenInputIsEmpty(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	m.textInput.SetValue("alpha")
	updated, _ := m.Update(pasteKey("one\ntwo"))
	m = updated.(Model)

	m.textInput.SetValue("beta")
	updated, _ = m.Update(pasteKey("three\nfour"))
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(Model)

	if got := len(m.pasteBlocks); got != 1 {
		t.Fatalf("pasteBlocks length = %d after removing last block, want 1", got)
	}
	if got := m.textInput.Value(); got != "beta" {
		t.Fatalf("textInput.Value() = %q after removing last block, want %q", got, "beta")
	}
	if got := m.buildComposerPayload(); got != "alphaone\ntwobeta" {
		t.Fatalf("buildComposerPayload() = %q, want %q", got, "alphaone\ntwobeta")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(Model)

	if got := len(m.pasteBlocks); got != 1 {
		t.Fatalf("pasteBlocks length = %d after backspacing text, want 1", got)
	}
	if got := m.textInput.Value(); got != "bet" {
		t.Fatalf("textInput.Value() = %q after backspacing text, want %q", got, "bet")
	}
}

func TestComposer_BackspaceRemovesLastPasteBlockAtInputStartWithTrailingText(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	m.textInput.SetValue("beforeafter")
	m.textInput.SetCursor(len("before"))
	updated, _ := m.Update(pasteKey("one\ntwo"))
	m = updated.(Model)

	if got := m.textInput.Value(); got != "after" {
		t.Fatalf("textInput.Value() after paste = %q, want %q", got, "after")
	}
	if got := m.textInput.Position(); got != 0 {
		t.Fatalf("textInput.Position() after paste = %d, want 0", got)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(Model)

	if got := len(m.pasteBlocks); got != 0 {
		t.Fatalf("pasteBlocks length = %d after removing block, want 0", got)
	}
	if got := m.textInput.Value(); got != "beforeafter" {
		t.Fatalf("textInput.Value() after removing block = %q, want %q", got, "beforeafter")
	}
	if got := m.textInput.Position(); got != len("before") {
		t.Fatalf("textInput.Position() after removing block = %d, want %d", got, len("before"))
	}
	if got := m.buildComposerPayload(); got != "beforeafter" {
		t.Fatalf("buildComposerPayload() = %q, want %q", got, "beforeafter")
	}
}
