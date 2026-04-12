package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMouseSelection_SurvivesResize(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 20)
	m.rebuildChrome()
	m.mouseSelAnchor = visualPosition{line: 3, col: 2}
	m.mouseSelEnd = visualPosition{line: 7, col: 5}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 25})
	m = updated.(Model)

	if !m.hasActiveMouseSelection() {
		t.Fatal("expected selection to survive resize")
	}
	if m.mouseSelAnchor.line != 3 || m.mouseSelEnd.line != 7 {
		t.Fatal("expected selection positions to be preserved")
	}
}

func TestMouseDrag_StopsOnResize(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 20)
	m.rebuildChrome()
	m.mouseDragging = true
	m.mouseAutoScrolling = true

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 25})
	m = updated.(Model)

	if m.mouseDragging {
		t.Fatal("expected drag to stop on resize")
	}
	if m.mouseAutoScrolling {
		t.Fatal("expected auto-scroll to stop on resize")
	}
}

func TestMouseSelection_SurvivesStreamUpdate(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 20)
	m.rebuildChrome()
	m.mouseSelAnchor = visualPosition{line: 3, col: 2}
	m.mouseSelEnd = visualPosition{line: 7, col: 5}

	m.appendContentLines("new line appended")

	if !m.hasActiveMouseSelection() {
		t.Fatal("expected selection to survive stream update")
	}
	if m.mouseSelAnchor.line != 3 || m.mouseSelEnd.line != 7 {
		t.Fatal("expected selection positions to be preserved after stream update")
	}
}

func TestWheelScroll_PreservedWithSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 50)
	m.vp.gotoTop()
	m.rebuildChrome()
	m.mouseSelAnchor = visualPosition{line: 5, col: 0}
	m.mouseSelEnd = visualPosition{line: 10, col: 5}

	oldOffset := m.vp.yOffset
	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	m = updated.(Model)

	if m.vp.yOffset <= oldOffset {
		t.Fatal("expected wheel scroll to work with active selection")
	}
	if !m.hasActiveMouseSelection() {
		t.Fatal("expected selection to survive wheel scroll")
	}
}
