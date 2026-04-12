package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRender_EachLineExactWidth_ASCII(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"Hello World", "Second line here", "Third"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 2}
	m.mouseSelEnd = visualPosition{line: 2, col: 3}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != m.vp.width {
			t.Errorf("line %d: lipgloss.Width = %d, want %d", i, w, m.vp.width)
		}
	}
}

func TestRender_EachLineExactWidth_CJK(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"日本語テスト", "abc日本語def", "テスト完了"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 2, col: 7}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != m.vp.width {
			t.Errorf("line %d: lipgloss.Width = %d, want %d (line=%q)", i, w, m.vp.width, line)
		}
	}
}

func TestRender_EachLineExactWidth_ANSI(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{
		"\033[31mred text\033[0m normal",
		"\033[1;32mbold green\033[0m end",
		"plain text",
	}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 3}
	m.mouseSelEnd = visualPosition{line: 2, col: 5}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != m.vp.width {
			t.Errorf("line %d: lipgloss.Width = %d, want %d", i, w, m.vp.width)
		}
	}
}

func TestRender_EachLineExactWidth_MixedANSICJK(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{
		"\033[31m日本語\033[0mテスト",
		"\033[34mABC\033[0m全角ＤＥＦ",
	}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 2}
	m.mouseSelEnd = visualPosition{line: 1, col: 7}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != m.vp.width {
			t.Errorf("line %d: lipgloss.Width = %d, want %d (line=%q)", i, w, m.vp.width, line)
		}
	}
}

func TestStylePlainTextRange_RangeBasedANSI(t *testing.T) {
	styled := stylePlainTextRange("ABCDE", 1, 4, "\033[48;5;240m")

	bgCount := strings.Count(styled, "\033[48;5;240m")
	if bgCount != 1 {
		t.Errorf("expected 1 bg open, got %d (styled=%q) — per-character wrapping causes flicker", bgCount, styled)
	}
	resetCount := strings.Count(styled, "\033[0m")
	if resetCount != 1 {
		t.Errorf("expected 1 reset, got %d (styled=%q)", resetCount, styled)
	}
	if w := lipgloss.Width(styled); w != 5 {
		t.Errorf("width = %d, want 5", w)
	}
	if p := stripANSI(styled); p != "ABCDE" {
		t.Errorf("plain = %q, want %q", p, "ABCDE")
	}
}

func TestStylePlainTextRange_CJKRangeBased(t *testing.T) {
	styled := stylePlainTextRange("日本語テスト", 2, 6, "\033[48;5;240m")
	bgCount := strings.Count(styled, "\033[48;5;240m")
	if bgCount != 1 {
		t.Errorf("expected 1 bg open for CJK range, got %d (styled=%q)", bgCount, styled)
	}
	if p := stripANSI(styled); p != "日本語テスト" {
		t.Errorf("plain = %q, want 日本語テスト", p)
	}
}

func TestStylePlainTextRange_PreservesWidth(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"ASCII", "Hello, World!"},
		{"CJK", "日本語テスト完了"},
		{"Mixed", "abc日本語def"},
		{"Fullwidth", "ＡＢＣ１２３"},
		{"Empty", ""},
		{"SingleCJK", "日"},
		{"Space", "  spaces  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputWidth := lipgloss.Width(tt.input)
			styled := stylePlainTextRange(tt.input, 0, 9999, mouseSelBg)
			styledWidth := lipgloss.Width(styled)
			if styledWidth != inputWidth {
				t.Errorf("full highlight: width %d -> %d (styled=%q)", inputWidth, styledWidth, styled)
			}
			mid := inputWidth / 2
			if mid > 0 {
				styled2 := stylePlainTextRange(tt.input, mid, mid+1, mouseSelBg)
				styled2Width := lipgloss.Width(styled2)
				if styled2Width != inputWidth {
					t.Errorf("partial highlight: width %d -> %d", inputWidth, styled2Width)
				}
			}
		})
	}
}

func TestFillANSITextWidth_StyledOutputExactWidth(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
	}{
		{"ASCII short", "Hello", 80},
		{"CJK short", "日本語", 80},
		{"CJK exact", "日本語テスト日本語テ", 20},
		{"ANSI styled", stylePlainTextRange("abcdef", 2, 5, mouseSelBg), 80},
		{"CJK styled", stylePlainTextRange("日本語テスト", 2, 8, mouseSelBg), 80},
		{"Empty styled", stylePlainTextRange("", 0, 5, mouseSelBg), 80},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fillANSITextWidth(tt.input, tt.width, "")
			w := lipgloss.Width(result)
			if w != tt.width {
				t.Errorf("fillANSITextWidth width = %d, want %d (result=%q)", w, tt.width, result)
			}
		})
	}
}
