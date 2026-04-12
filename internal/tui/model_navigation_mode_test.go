package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNavMode_EscEntersNavWhenInputEmpty(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.vp = lightViewport{width: 10, height: 5, yOffset: 3}
	setModelRawLines(&m, 20)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(Model)
	if !got.navigationMode {
		t.Fatal("Esc with empty input should enter navigation mode")
	}
	if got.cursorLine != 3 {
		t.Fatalf("cursorLine = %d, want 3", got.cursorLine)
	}
}

func TestNavMode_EscDoesNotEnterNavWhenInputHasText(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.textInput.SetValue("hello")

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(Model)
	if got.navigationMode {
		t.Fatal("Esc with text in input should NOT enter navigation mode")
	}
}

func TestNavMode_QExitsNav(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	got := updated.(Model)
	if got.navigationMode {
		t.Fatal("q should exit navigation mode")
	}
}

func TestNavMode_CtrlCWorksInNav(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true

	updated, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyCtrlC})
	got := updated.(Model)
	if cmd != nil {
		t.Fatal("first ctrl+c should not quit")
	}
	if !got.lastInterrupt.After(time.Now().Add(-time.Second)) {
		t.Fatal("lastInterrupt should be set")
	}
}

func TestInputMode_EscEntersNavModeAtViewportTopContinuationRow(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 5
	m.height = 8
	m.vp = lightViewport{width: 5, height: 4}
	m.rawLines = []string{"abcdefghij"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.yOffset = 1
	m.textInput.SetValue("")

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if !m.navigationMode {
		t.Fatal("navigationMode = false, want true")
	}
	if m.cursorLine != 0 {
		t.Fatalf("cursorLine = %d, want 0", m.cursorLine)
	}
	if m.cursorCol != 5 {
		t.Fatalf("cursorCol = %d, want 5", m.cursorCol)
	}

	lines := strings.Split(m.viewportView(), "\n")
	if len(lines) == 0 {
		t.Fatal("viewportView returned no lines")
	}
	if !strings.Contains(lines[0], "\033[48;5;255;38;5;16mf\033[0m") {
		t.Fatalf("top visible continuation row should keep cursor on f, got %q", lines[0])
	}
}
