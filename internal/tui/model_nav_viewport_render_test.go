package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestNavViewport_CursorLine_ExactWidth(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"Hello World", "\033[31mred text\033[0m", "日本語テスト"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.navigationMode = true
	m.cursorLine = 0
	m.cursorCol = 3

	view := m.viewportView()
	lines := strings.SplitN(view, "\n", m.vp.height+1)
	for i, line := range lines {
		if i >= m.vp.height {
			break
		}
		w := lipgloss.Width(line)
		if w != m.vp.width {
			t.Errorf("NAV line %d: width = %d, want %d (line=%q)", i, w, m.vp.width, line)
		}
	}
}

func TestNavViewport_CursorOnCJKLine_ExactWidth(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"abc", "日本語テスト完了", "def"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.navigationMode = true
	m.cursorLine = 1
	m.cursorCol = 4

	view := m.viewportView()
	lines := strings.SplitN(view, "\n", m.vp.height+1)
	for i, line := range lines {
		if i >= m.vp.height {
			break
		}
		w := lipgloss.Width(line)
		if w != m.vp.width {
			t.Errorf("NAV CJK line %d: width = %d, want %d", i, w, m.vp.width)
		}
	}
}

func TestNavViewport_VisualCharSelection_ExactWidth(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"first line", "second line", "third line"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.navigationMode = true
	m.visualMode = visualModeChar
	m.visualStart = visualPosition{line: 0, col: 3}
	m.cursorLine = 2
	m.cursorCol = 5

	view := m.viewportView()
	lines := strings.SplitN(view, "\n", m.vp.height+1)
	for i, line := range lines {
		if i >= m.vp.height {
			break
		}
		w := lipgloss.Width(line)
		if w != m.vp.width {
			t.Errorf("visual char line %d: width = %d, want %d", i, w, m.vp.width)
		}
	}
}

func TestNavViewport_VisualLineSelection_ExactWidth(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"first line", "second line", "third line"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.navigationMode = true
	m.visualMode = visualModeLine
	m.visualStart = visualPosition{line: 0, col: 0}
	m.cursorLine = 1
	m.cursorCol = 3

	view := m.viewportView()
	lines := strings.SplitN(view, "\n", m.vp.height+1)
	for i, line := range lines {
		if i >= m.vp.height {
			break
		}
		w := lipgloss.Width(line)
		if w != m.vp.width {
			t.Errorf("visual line %d: width = %d, want %d", i, w, m.vp.width)
		}
	}
}

func TestNavViewport_NonCursorLine_MatchesNormal(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"\033[31mcolored\033[0m text", "plain line", "another"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()

	normalView := m.vp.view()
	normalLines := strings.SplitN(normalView, "\n", m.vp.height+1)

	m.navigationMode = true
	m.cursorLine = 1
	m.cursorCol = 0

	navView := m.viewportView()
	navLines := strings.SplitN(navView, "\n", m.vp.height+1)

	if navLines[0] != normalLines[0] {
		t.Errorf("non-cursor line 0 differs from normal:\n  normal: %q\n  nav:    %q", normalLines[0], navLines[0])
	}
	if navLines[2] != normalLines[2] {
		t.Errorf("non-cursor line 2 differs from normal:\n  normal: %q\n  nav:    %q", normalLines[2], navLines[2])
	}
}

func TestNavViewport_ANSILine_CursorHighlight_MinimalResets(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"\033[31mred text here\033[0m"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.navigationMode = true
	m.cursorLine = 0
	m.cursorCol = 4

	view := m.viewportView()
	lines := strings.SplitN(view, "\n", m.vp.height+1)
	cursorLine := lines[0]

	resetCount := strings.Count(cursorLine, "\033[0m")
	plainLen := lipgloss.Width(stripANSI(cursorLine))
	if resetCount > plainLen/2 {
		t.Errorf("too many resets (%d) for %d-char line — likely per-char flicker (line=%q)",
			resetCount, plainLen, cursorLine)
	}
}
