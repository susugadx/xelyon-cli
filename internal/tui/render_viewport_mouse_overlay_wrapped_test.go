package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRender_WrappedLine_SelectionAcrossBoundary(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 20
	m.height = 15
	m.vp = lightViewport{width: 20, height: m.height - m.footerHeight()}
	m.padLineCache = strings.Repeat(" ", 20)
	m.rawLines = []string{"ABCDEFGHIJKLMNOPQRSTUVWXYZ1234"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()

	if len(m.layout.Rows) < 2 {
		t.Fatalf("expected >=2 visual rows, got %d", len(m.layout.Rows))
	}

	m.mouseSelAnchor = visualPosition{line: 0, col: 15}
	m.mouseSelEnd = visualPosition{line: 0, col: 25}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)

	for i := 0; i < 2; i++ {
		w := lipgloss.Width(lines[i])
		if w != 20 {
			t.Errorf("visual row %d: width = %d, want 20 (line=%q)", i, w, lines[i])
		}
	}
	if !strings.Contains(lines[0], mouseSelBg) {
		t.Error("visual row 0 should have selection background")
	}
	if !strings.Contains(lines[1], mouseSelBg) {
		t.Error("visual row 1 should have selection background")
	}
}

func TestRender_WrappedLine_SelectionOnlyOnFirstRow(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 20
	m.height = 15
	m.vp = lightViewport{width: 20, height: m.height - m.footerHeight()}
	m.padLineCache = strings.Repeat(" ", 20)
	m.rawLines = []string{"ABCDEFGHIJKLMNOPQRSTUVWXYZ1234"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()

	m.mouseSelAnchor = visualPosition{line: 0, col: 2}
	m.mouseSelEnd = visualPosition{line: 0, col: 8}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)

	if !strings.Contains(lines[0], mouseSelBg) {
		t.Error("visual row 0 should have selection background")
	}
	if strings.Contains(lines[1], mouseSelBg) {
		t.Error("visual row 1 should NOT have selection background")
	}
	for i := 0; i < 2; i++ {
		w := lipgloss.Width(lines[i])
		if w != 20 {
			t.Errorf("visual row %d: width = %d, want 20", i, w)
		}
	}
}

func TestRender_WrappedCJKLine_Width(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 20
	m.height = 15
	m.vp = lightViewport{width: 20, height: m.height - m.footerHeight()}
	m.padLineCache = strings.Repeat(" ", 20)
	m.rawLines = []string{"日本語テストABC全角文字"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()

	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	maxCol := m.maxCursorColForLine(0)
	m.mouseSelEnd = visualPosition{line: 0, col: maxCol}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)

	nRows := len(m.layout.Rows)
	for i := 0; i < nRows && i < len(lines); i++ {
		w := lipgloss.Width(lines[i])
		if w != 20 {
			t.Errorf("visual row %d: width = %d, want 20 (line=%q)", i, w, lines[i])
		}
	}

	var allPlain strings.Builder
	for i := 0; i < nRows && i < len(lines); i++ {
		allPlain.WriteString(strings.TrimRight(stripANSI(lines[i]), " "))
	}
	expectedPlain := stripANSI(m.rawLines[0])
	if allPlain.String() != expectedPlain {
		t.Errorf("concatenated plain = %q, want %q", allPlain.String(), expectedPlain)
	}
}
