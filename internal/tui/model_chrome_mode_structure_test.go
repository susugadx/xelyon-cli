package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestView_NormalMode_Structure(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)
	verifyViewStructure(t, m, "normal mode")
}

func TestView_NavMode_Structure(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)
	m.navigationMode = true
	m.textInput.Blur()
	m.cursorLine = 5
	m.cursorCol = 0
	m.rebuildChrome()
	verifyViewStructure(t, m, "NAV mode")
}

func TestView_NavVisualChar_Structure(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)
	m.navigationMode = true
	m.textInput.Blur()
	m.cursorLine = 8
	m.cursorCol = 10
	m.visualMode = visualModeChar
	m.visualStart = visualPosition{line: 3, col: 5}
	m.rebuildChrome()
	verifyViewStructure(t, m, "NAV visual char")
}

func TestView_NavVisualLine_Structure(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)
	m.navigationMode = true
	m.textInput.Blur()
	m.cursorLine = 5
	m.cursorCol = 0
	m.visualMode = visualModeLine
	m.visualStart = visualPosition{line: 3, col: 0}
	m.rebuildChrome()
	verifyViewStructure(t, m, "NAV visual line")
}

func TestView_MouseSelection_Structure(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)
	m.mouseSelAnchor = visualPosition{line: 2, col: 0}
	m.mouseSelEnd = visualPosition{line: 10, col: 20}
	m.rebuildChrome()
	verifyViewStructure(t, m, "mouse selection")
}

func TestView_FooterViewportHeightConsistency(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	footer := m.footerHeight()
	vpHeight := m.vp.height
	if footer+vpHeight != m.height {
		t.Errorf("footer(%d) + vp.height(%d) = %d, want %d", footer, vpHeight, footer+vpHeight, m.height)
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m = updated.(Model)
	footer = m.footerHeight()
	vpHeight = m.vp.height
	if footer+vpHeight != m.height {
		t.Errorf("after resize: footer(%d) + vp.height(%d) = %d, want %d", footer, vpHeight, footer+vpHeight, m.height)
	}
	verifyViewStructure(t, m, "after resize")
}
