package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMouseDrag_BasicFlow(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 20)
	m.vp.gotoTop()
	m.rebuildChrome()

	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
		X:      5, Y: 2,
	})
	m = updated.(Model)
	if !m.mouseDragging {
		t.Fatal("expected mouseDragging after press")
	}
	if m.mouseSelAnchor.line < 0 {
		t.Fatal("expected anchor to be set")
	}

	updated, _ = m.Update(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionMotion,
		X:      10, Y: 5,
	})
	m = updated.(Model)
	if !m.mouseDragging {
		t.Fatal("expected still dragging")
	}
	if m.mouseSelAnchor == m.mouseSelEnd {
		t.Fatal("expected selection to differ from anchor")
	}

	updated, _ = m.Update(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
		X:      10, Y: 5,
	})
	m = updated.(Model)
	if m.mouseDragging {
		t.Fatal("expected dragging to stop after release")
	}
	if !m.hasActiveMouseSelection() {
		t.Fatal("expected active selection after drag")
	}
}

func TestMouseDrag_ClickClearsSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 20)
	m.vp.gotoTop()
	m.rebuildChrome()

	m.mouseSelAnchor = visualPosition{line: 1, col: 0}
	m.mouseSelEnd = visualPosition{line: 3, col: 5}

	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
		X:      2, Y: 4,
	})
	m = updated.(Model)
	updated, _ = m.Update(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
		X:      2, Y: 4,
	})
	m = updated.(Model)

	if m.hasActiveMouseSelection() {
		t.Fatal("expected selection cleared after click without drag")
	}
}

func TestMouseDrag_InputDockIgnored(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 20)
	m.vp.gotoTop()
	m.rebuildChrome()

	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
		X:      5, Y: m.vp.height + 1,
	})
	m = updated.(Model)
	if m.mouseDragging {
		t.Fatal("should not start drag in input dock area")
	}
}

func TestMouseDrag_ClearsKeyboardVisualSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 20)
	m.vp.gotoTop()
	m.rebuildChrome()
	m.navigationMode = true
	m.visualMode = visualModeChar
	m.visualStart = visualPosition{line: 1, col: 0}

	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
		X:      2, Y: 3,
	})
	m = updated.(Model)

	if m.visualMode != visualModeOff {
		t.Fatal("expected keyboard visual selection to be cleared on drag start")
	}
}
