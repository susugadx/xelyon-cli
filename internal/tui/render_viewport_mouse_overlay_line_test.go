package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func TestRender_SelectionOnlyOnSelectedLines(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"line0", "line1", "line2", "line3", "line4"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 1, col: 0}
	m.mouseSelEnd = visualPosition{line: 3, col: 4}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)

	if strings.Contains(lines[0], theme.Viewport.MouseSelectionBg) {
		t.Error("line 0 should not have selection background")
	}
	for i := 1; i <= 3; i++ {
		if !strings.Contains(lines[i], theme.Viewport.MouseSelectionBg) {
			t.Errorf("line %d should have selection background", i)
		}
	}
	if strings.Contains(lines[4], theme.Viewport.MouseSelectionBg) {
		t.Error("line 4 should not have selection background")
	}
}

func TestRender_CJKBoundary_SelectionHighlightsWholeChar(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"ab日cd"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 2}
	m.mouseSelEnd = visualPosition{line: 0, col: 3}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)
	line0 := lines[0]

	if !strings.Contains(line0, theme.Viewport.MouseSelectionBg) {
		t.Fatal("expected selection background for CJK char")
	}
	plainBefore := stripANSI(line0)
	if !strings.HasPrefix(plainBefore, "ab") {
		t.Fatalf("expected plain content to start with 'ab', got %q", plainBefore)
	}
}

func TestRender_ANSIResetBalanced(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"Hello World"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 2}
	m.mouseSelEnd = visualPosition{line: 0, col: 7}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)
	line0 := lines[0]

	bgCount := strings.Count(line0, theme.Viewport.MouseSelectionBg)
	resetCount := strings.Count(line0, "\033[0m")
	if bgCount > 0 && resetCount < bgCount {
		t.Errorf("ANSI unbalanced: %d bg opens, %d resets", bgCount, resetCount)
	}
}

func TestRender_EmptyLineInSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"hello", "", "world"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 2, col: 4}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)

	for i := 0; i < 3; i++ {
		w := lipgloss.Width(lines[i])
		if w != m.vp.width {
			t.Errorf("line %d: width = %d, want %d", i, w, m.vp.width)
		}
	}
}

func TestRender_NonSelectedLineSameAsNormal(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"\033[31mcolored text\033[0m", "plain text", "another"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()

	normalView := m.vp.view()
	normalLines := helperSplitViewLines(normalView, m.vp.height)

	m.mouseSelAnchor = visualPosition{line: 1, col: 0}
	m.mouseSelEnd = visualPosition{line: 1, col: 5}
	selView := m.renderViewportWithMouseSelection()
	selLines := helperSplitViewLines(selView, m.vp.height)

	if selLines[0] != normalLines[0] {
		t.Errorf("non-selected line differs from normal:\n  normal: %q\n  select: %q", normalLines[0], selLines[0])
	}
	if selLines[2] != normalLines[2] {
		t.Errorf("non-selected line 2 differs from normal:\n  normal: %q\n  select: %q", normalLines[2], selLines[2])
	}
}

func TestRender_SelectedLineContentPreserved(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"Hello, World!"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 12}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)
	plain := stripANSI(lines[0])
	trimmed := strings.TrimRight(plain, " ")
	if trimmed != "Hello, World!" {
		t.Errorf("plain content = %q, want %q", trimmed, "Hello, World!")
	}
}

func TestRender_IntermediateLineFull(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"first", "INTERMEDIATE", "last"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 2}
	m.mouseSelEnd = visualPosition{line: 2, col: 2}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)

	plain1 := stripANSI(lines[1])
	trimmed1 := strings.TrimRight(plain1, " ")
	if trimmed1 != "INTERMEDIATE" {
		t.Errorf("intermediate plain = %q, want %q", trimmed1, "INTERMEDIATE")
	}
	if !strings.Contains(lines[1], theme.Viewport.MouseSelectionBg) {
		t.Error("intermediate line should have selection background")
	}
	w := lipgloss.Width(lines[1])
	if w != m.vp.width {
		t.Errorf("intermediate width = %d, want %d", w, m.vp.width)
	}
}
