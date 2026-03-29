package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// --- stylePlainTextRangeWithCursor: range-based ANSI ---

func TestStyleWithCursor_CursorLine_RangeBasedANSI(t *testing.T) {
	// NAV cursor line: lineBg should be a single span, not per-character
	const cursorLineBg = "\033[48;5;236m"
	const cursorCharBg = "\033[48;5;255;38;5;16m"

	styled := stylePlainTextRangeWithCursor("ABCDE", 0, 0, "", 2, cursorCharBg, cursorLineBg)

	// lineBg should appear as spans, not N times
	lineBgCount := strings.Count(styled, cursorLineBg)
	if lineBgCount > 3 {
		t.Errorf("lineBg emitted %d times (per-char flicker); want <=3 transitions (styled=%q)", lineBgCount, styled)
	}

	// Total reset count should be minimal (transitions, not per-char)
	resetCount := strings.Count(styled, "\033[0m")
	if resetCount > 3 {
		t.Errorf("reset count = %d, want <=3 (styled=%q)", resetCount, styled)
	}

	// Width must match
	if w := lipgloss.Width(styled); w != 5 {
		t.Errorf("width = %d, want 5", w)
	}

	// Content preserved
	if p := stripANSI(styled); p != "ABCDE" {
		t.Errorf("plain = %q, want ABCDE", p)
	}
}

func TestStyleWithCursor_VisualCharWithCursor_RangeBased(t *testing.T) {
	const visualBg = "\033[48;5;240m"
	const visualCursorBg = "\033[48;5;255;38;5;16m"

	// Selection [1,4), cursor at 3
	styled := stylePlainTextRangeWithCursor("ABCDE", 1, 4, visualBg, 3, visualCursorBg, "")

	resetCount := strings.Count(styled, "\033[0m")
	if resetCount > 4 {
		t.Errorf("reset count = %d, want <=4 (styled=%q)", resetCount, styled)
	}
	if w := lipgloss.Width(styled); w != 5 {
		t.Errorf("width = %d, want 5", w)
	}
	if p := stripANSI(styled); p != "ABCDE" {
		t.Errorf("plain = %q, want ABCDE", p)
	}
}

func TestStyleWithCursor_VisualLineWithCursor_RangeBased(t *testing.T) {
	const visualBg = "\033[48;5;240m"
	const cursorBg = "\033[48;5;255;38;5;16m"

	// Full line selected [0,5), cursor at 2
	styled := stylePlainTextRangeWithCursor("ABCDE", 0, 5, visualBg, 2, cursorBg, "")

	resetCount := strings.Count(styled, "\033[0m")
	if resetCount > 3 {
		t.Errorf("reset count = %d, want <=3 (styled=%q)", resetCount, styled)
	}
	if w := lipgloss.Width(styled); w != 5 {
		t.Errorf("width = %d, want 5", w)
	}
}

func TestStyleWithCursor_EmptyString(t *testing.T) {
	const cursorLineBg = "\033[48;5;236m"
	const cursorCharBg = "\033[48;5;255;38;5;16m"

	// Cursor on empty line
	styled := stylePlainTextRangeWithCursor("", 0, 0, "", 0, cursorCharBg, cursorLineBg)
	if !strings.Contains(styled, cursorCharBg) {
		t.Errorf("empty line cursor should show cursorBg, got %q", styled)
	}

	// No cursor, with lineBg
	styled2 := stylePlainTextRangeWithCursor("", 0, 0, "", 5, cursorCharBg, cursorLineBg)
	if !strings.Contains(styled2, cursorLineBg) {
		t.Errorf("empty line with lineBg should show lineBg, got %q", styled2)
	}
}

func TestStyleWithCursor_CJKCursorPosition(t *testing.T) {
	const cursorBg = "\033[48;5;255;38;5;16m"
	const lineBg = "\033[48;5;236m"

	// "abc日def" → col: a(0) b(1) c(2) 日(3-4) d(5) e(6) f(7)
	// Cursor at col 3 (lands on 日)
	styled := stylePlainTextRangeWithCursor("abc日def", 0, 0, "", 3, cursorBg, lineBg)

	if !strings.Contains(styled, cursorBg) {
		t.Fatalf("CJK cursor should show cursorBg, got %q", styled)
	}
	// Width must match original (8 display cols)
	if w := lipgloss.Width(styled); w != 8 {
		t.Errorf("width = %d, want 8", w)
	}
	if p := stripANSI(styled); p != "abc日def" {
		t.Errorf("plain = %q, want abc日def", p)
	}
}

// --- viewportView NAV mode: line width correctness ---

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

// --- Non-cursor lines match normal rendering ---

func TestNavViewport_NonCursorLine_MatchesNormal(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"\033[31mcolored\033[0m text", "plain line", "another"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()

	// Normal rendering
	normalView := m.vp.view()
	normalLines := strings.SplitN(normalView, "\n", m.vp.height+1)

	// NAV mode with cursor on line 1
	m.navigationMode = true
	m.cursorLine = 1
	m.cursorCol = 0

	navView := m.viewportView()
	navLines := strings.SplitN(navView, "\n", m.vp.height+1)

	// Line 0 (not cursor line) should match normal exactly
	if navLines[0] != normalLines[0] {
		t.Errorf("non-cursor line 0 differs from normal:\n  normal: %q\n  nav:    %q", normalLines[0], navLines[0])
	}
	// Line 2 (not cursor line) should match normal exactly
	if navLines[2] != normalLines[2] {
		t.Errorf("non-cursor line 2 differs from normal:\n  normal: %q\n  nav:    %q", normalLines[2], navLines[2])
	}
}

// --- ANSI with NAV highlight: no extra resets ---

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

	// Count resets — with range-based approach, should be small
	resetCount := strings.Count(cursorLine, "\033[0m")
	// Expect: fillANSITextWidth wrapping adds some, styled adds <=3. Total should be small.
	plainLen := lipgloss.Width(stripANSI(cursorLine))
	if resetCount > plainLen/2 {
		t.Errorf("too many resets (%d) for %d-char line — likely per-char flicker (line=%q)",
			resetCount, plainLen, cursorLine)
	}
}
