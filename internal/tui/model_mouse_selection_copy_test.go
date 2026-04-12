package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCtrlC_CopiesMouseSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"Hello, World!"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.rebuildChrome()
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 4}

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if len(agent.copyTexts) != 1 || agent.copyTexts[0] != "Hello" {
		t.Fatalf("copyTexts = %v, want [\"Hello\"]", agent.copyTexts)
	}
	if m.hasActiveMouseSelection() {
		t.Fatal("expected selection cleared after copy")
	}
}

func TestCtrlC_CopyTakesPriorityOverInterrupt(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	agent.setProcessing(true)
	m := newModelWithViewport(agent)
	m.rawLines = []string{"Hello"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.rebuildChrome()
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 4}

	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyCtrlC})

	if agent.copyCalls != 1 {
		t.Fatalf("copyCalls = %d, want 1 (copy should take priority over interrupt)", agent.copyCalls)
	}
	if agent.cancelCalls != 0 {
		t.Fatalf("cancelCalls = %d, want 0", agent.cancelCalls)
	}
}

func TestNavY_CopiesMouseSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"first", "second", "third"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.rebuildChrome()
	m.navigationMode = true
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 1, col: 5}

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)

	if len(agent.copyTexts) != 1 {
		t.Fatalf("copyTexts = %v, want 1 entry", agent.copyTexts)
	}
	expected := "first\nsecond"
	if agent.copyTexts[0] != expected {
		t.Fatalf("copyTexts[0] = %q, want %q", agent.copyTexts[0], expected)
	}
}

func TestCopyCommand_CopiesMouseSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"copy me"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.rebuildChrome()
	m.textInput.SetValue("/copy")
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 6}

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if len(agent.copyTexts) != 1 || agent.copyTexts[0] != "copy me" {
		t.Fatalf("copyTexts = %v, want [\"copy me\"]", agent.copyTexts)
	}
}
