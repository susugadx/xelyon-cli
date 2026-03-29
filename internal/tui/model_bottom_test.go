package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// verifyViewLines checks total line count and each line width.
// Returns the split lines for further inspection.
func verifyViewLines(t *testing.T, m Model, label string) []string {
	t.Helper()
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != m.height {
		t.Errorf("[%s] line count = %d, want %d (vp.height=%d, footer=%d)",
			label, len(lines), m.height, m.vp.height, m.footerHeight())
	}
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != m.width {
			t.Errorf("[%s] line %d: width = %d, want %d", label, i, w, m.width)
		}
	}
	return lines
}

// verifyFooterPosition checks that the footer (input dock + status bar) occupies
// exactly the last footerHeight() lines of View(). If the footer shifts up,
// transcript content would leak into the footer zone.
func verifyFooterPosition(t *testing.T, m Model, label string) {
	t.Helper()
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != m.height {
		t.Errorf("[%s] line count = %d, want %d", label, len(lines), m.height)
		return
	}

	footerStart := m.height - m.footerHeight()

	// The viewport zone (lines 0..footerStart-1) should NOT contain input dock bg
	// patterns from chromeCache. The footer zone should contain them.
	// Check: the footer start line is a padLine (gray bg, all spaces after strip)
	padLine := lines[footerStart]
	padPlain := strings.TrimRight(stripANSI(padLine), " ")
	if padPlain != "" {
		t.Errorf("[%s] footer start line (line %d) has non-space content: %q",
			label, footerStart, padPlain)
	}

	// Check: input line contains the prompt
	inputLine := lines[footerStart+1]
	if !strings.Contains(inputLine, inputPrompt) && !strings.Contains(inputLine, "Type your message") {
		t.Errorf("[%s] footer input line (line %d) missing prompt/placeholder: %q",
			label, footerStart+1, inputLine)
	}
}

func setupBottomTestModel(lineCount int) Model {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 60
	m.height = 20
	vph := m.height - m.footerHeight()
	m.vp = lightViewport{width: 60, height: vph}
	m.textInput.Width = m.width - inputPromptWidth - 1
	m.padLineCache = fillANSITextWidth("", m.width, "\033[48;5;236m")
	for i := range lineCount {
		m.appendContentLines(fmt.Sprintf("Line %03d: %s", i, strings.Repeat("content ", 3)))
	}
	m.rebuildChrome()
	return m
}

// --- Bottom reach tests ---

func TestBottom_G_FooterStaysFixed(t *testing.T) {
	m := setupBottomTestModel(50)

	// Enter NAV
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewLines(t, m, "enter NAV")
	verifyFooterPosition(t, m, "enter NAV")

	// Press G (go to bottom)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	verifyViewLines(t, m, "G")
	verifyFooterPosition(t, m, "G")
}

func TestBottom_RepeatedJ_FooterStaysFixed(t *testing.T) {
	m := setupBottomTestModel(50)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	// Press j many times until bottom
	for i := range 60 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = updated.(Model)
		label := fmt.Sprintf("j_%d", i)
		lines := verifyViewLines(t, m, label)
		_ = lines
	}
	// Final check at bottom
	verifyFooterPosition(t, m, "after j spam bottom")
}

func TestBottom_RepeatedDown_PastContent(t *testing.T) {
	m := setupBottomTestModel(50)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	// Go to bottom
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)

	// Keep pressing j past the bottom
	for i := range 20 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = updated.(Model)
		verifyViewLines(t, m, fmt.Sprintf("j_past_%d", i))
	}
	verifyFooterPosition(t, m, "j past bottom")
}

func TestBottom_GG_Then_G(t *testing.T) {
	m := setupBottomTestModel(50)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	// gg (top)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(Model)
	verifyViewLines(t, m, "gg")
	verifyFooterPosition(t, m, "gg")

	// G (bottom)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	verifyViewLines(t, m, "G after gg")
	verifyFooterPosition(t, m, "G after gg")
}

func TestBottom_DU_HalfPage(t *testing.T) {
	m := setupBottomTestModel(50)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	// d (half page down) multiple times
	for i := range 5 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
		m = updated.(Model)
		verifyViewLines(t, m, fmt.Sprintf("d_%d", i))
	}
	verifyFooterPosition(t, m, "after d spam")

	// u (half page up) multiple times
	for i := range 5 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
		m = updated.(Model)
		verifyViewLines(t, m, fmt.Sprintf("u_%d", i))
	}
	verifyFooterPosition(t, m, "after u spam")
}

func TestBottom_BlockTab_NearBottom(t *testing.T) {
	m := setupBottomTestModel(30)
	// Add tool blocks near the end
	m.appendToolResult(ToolResult{Name: "t1", Summary: "Tool near end", Detail: "detail"})
	m.appendToolResult(ToolResult{Name: "t2", Summary: "Last tool", Detail: "d1\nd2\nd3"})
	m.rebuildChrome()

	// Enter NAV
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	// Tab to last block (near bottom)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	verifyViewLines(t, m, "Tab first block")
	verifyFooterPosition(t, m, "Tab first block")

	// Shift+Tab
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(Model)
	verifyViewLines(t, m, "ShiftTab")
	verifyFooterPosition(t, m, "ShiftTab")
}

func TestBottom_NavThenEsc(t *testing.T) {
	m := setupBottomTestModel(50)

	// Enter NAV → G → ESC
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewLines(t, m, "bottom then ESC")
	verifyFooterPosition(t, m, "bottom then ESC")
}

func TestBottom_ShortContent_FooterStaysAtBottom(t *testing.T) {
	// Content shorter than viewport
	m := setupBottomTestModel(3)

	verifyViewLines(t, m, "short normal")
	verifyFooterPosition(t, m, "short normal")

	// NAV + G
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	verifyViewLines(t, m, "short G")
	verifyFooterPosition(t, m, "short G")
}

// --- viewport height + footer height = window height invariant ---

func TestInvariant_VpHeightPlusFooter_AllModes(t *testing.T) {
	m := setupBottomTestModel(50)

	check := func(label string) {
		t.Helper()
		if m.vp.height+m.footerHeight() != m.height {
			t.Errorf("[%s] vp.height(%d) + footer(%d) = %d, want %d",
				label, m.vp.height, m.footerHeight(), m.vp.height+m.footerHeight(), m.height)
		}
	}

	check("normal")

	// NAV
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	check("NAV")

	// Visual
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	check("visual")

	// G
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	check("G")

	// ESC back
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	check("back to normal")

	// Resize
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 15})
	m = updated.(Model)
	check("after resize")
}

// --- Trailing newline in content ---

func TestBottom_TrailingEmptyLines_FooterStaysFixed(t *testing.T) {
	m := setupBottomTestModel(10)
	// Add trailing empty lines
	m.appendContentLines("", "", "", "")
	m.rebuildChrome()

	// NAV + G
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	verifyViewLines(t, m, "trailing empties G")
	verifyFooterPosition(t, m, "trailing empties G")

	// j past bottom
	for range 5 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = updated.(Model)
	}
	verifyViewLines(t, m, "trailing empties j past")
	verifyFooterPosition(t, m, "trailing empties j past")
}

// --- No embedded newlines in any viewport line ---

func TestBottom_NoEmbeddedNewlines(t *testing.T) {
	m := setupBottomTestModel(50)

	// NAV + G
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)

	view := m.viewportView()
	viewLines := strings.Split(view, "\n")
	if len(viewLines) != m.vp.height {
		t.Fatalf("viewport line count = %d, want %d", len(viewLines), m.vp.height)
	}

	chrome := m.chromeCache
	chromeLines := strings.Split(chrome, "\n")
	if len(chromeLines) != m.footerHeight() {
		t.Fatalf("chrome line count = %d, want %d", len(chromeLines), m.footerHeight())
	}
}
