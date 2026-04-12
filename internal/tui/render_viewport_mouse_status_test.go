package tui

import (
	"strings"
	"testing"
)

func TestViewportView_ShowsMouseSelectionOverlay(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"line zero", "line one", "line two"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 1, col: 3}

	view := m.viewportView()
	if !strings.Contains(view, "\033[48;5;240m") {
		t.Fatal("expected mouse selection background in viewport view")
	}
}

func TestViewportView_NoOverlayWithoutSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"line zero", "line one"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()

	view := m.viewportView()
	if strings.Contains(view, "\033[48;5;240m") {
		t.Fatal("expected no selection background without active selection")
	}
}

func TestStatusBar_ShowsMouseSelHints(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 2, col: 5}

	bar := m.renderStatusBar()
	if !strings.Contains(bar, "SELECT") {
		t.Fatal("expected SELECT in status bar when mouse selection active")
	}
}

func TestStatusBar_ShowsDragHint(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.mouseDragging = true

	bar := m.renderStatusBar()
	if !strings.Contains(bar, "SELECT") {
		t.Fatal("expected SELECT in status bar during drag")
	}
}
