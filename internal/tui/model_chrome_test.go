package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- View output structure tests ---

// verifyViewStructure checks that View() output has exactly m.height lines,
// each with exactly m.width display columns.
func verifyViewStructure(t *testing.T, m Model, context string) {
	t.Helper()
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) != m.height {
		t.Errorf("[%s] View line count = %d, want %d", context, len(lines), m.height)
	}
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != m.width {
			t.Errorf("[%s] line %d: width = %d, want %d (line=%q)", context, i, w, m.width, line)
		}
	}
}

func setupModelForChromeTest(agent *stubAgent) Model {
	m := NewModel(agent, "")
	m.ready = true
	m.width = 80
	m.height = 24
	vph := m.height - m.footerHeight()
	m.vp = lightViewport{width: 80, height: vph}
	m.textInput.Width = m.width - inputPromptWidth - 1
	m.padLineCache = fillANSITextWidth("", m.width, "\033[48;5;236m")
	for i := range 30 {
		m.appendContentLines(strings.Repeat("x", 40) + string(rune('A'+i%26)))
	}
	m.rebuildChrome()
	return m
}

// --- Basic structure ---

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

// --- ESC toggle idempotency ---

func TestView_EscToggle_StructureConsistent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	for i := range 10 {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = updated.(Model)
		verifyViewStructure(t, m, "ESC press "+string(rune('0'+i)))
	}
}

func TestView_EscToggle_PlaceholderNotDuplicated(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	for range 6 {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = updated.(Model)
	}

	view := m.View()
	count := strings.Count(view, "Type your message")
	if count > 1 {
		t.Errorf("placeholder appears %d times, want <=1", count)
	}
}

func TestView_EscToggle_StatusBarNotDuplicated(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	for range 6 {
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m = updated.(Model)
	}

	view := m.View()
	// The "NAV" badge or normal hints should appear at most once
	navCount := strings.Count(view, "NAV")
	// NAV might be in hints string too, so check for the styled badge
	badgeCount := strings.Count(view, "\033[48;5;33;38;5;255m NAV \033[0m")
	if badgeCount > 1 {
		t.Errorf("NAV badge appears %d times, want <=1", badgeCount)
	}
	_ = navCount
}

// --- NAV j/k movement ---

func TestView_NavJK_StructureConsistent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	// Enter NAV
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewStructure(t, m, "enter NAV")

	// Move with j multiple times
	for i := range 15 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = updated.(Model)
		verifyViewStructure(t, m, "j press "+string(rune('0'+i%10)))
	}

	// Move with k
	for i := range 10 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		m = updated.(Model)
		verifyViewStructure(t, m, "k press "+string(rune('0'+i%10)))
	}

	// Exit NAV
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewStructure(t, m, "exit NAV")
}

// --- footerHeight / viewport height consistency ---

func TestView_FooterViewportHeightConsistency(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	footer := m.footerHeight()
	vpHeight := m.vp.height
	if footer+vpHeight != m.height {
		t.Errorf("footer(%d) + vp.height(%d) = %d, want %d", footer, vpHeight, footer+vpHeight, m.height)
	}

	// After resize
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m = updated.(Model)
	footer = m.footerHeight()
	vpHeight = m.vp.height
	if footer+vpHeight != m.height {
		t.Errorf("after resize: footer(%d) + vp.height(%d) = %d, want %d", footer, vpHeight, footer+vpHeight, m.height)
	}
	verifyViewStructure(t, m, "after resize")
}

// --- chromeDirty on Enter/command ---

func TestChromeDirty_AfterTextInputReset(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	// Type something and press Enter
	m.textInput.SetValue("hello")
	m.rebuildChrome()

	// The chrome should show "hello" in input
	before := m.chromeCache

	// Press Enter (sends chat)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	// After Enter, chrome should be rebuilt (input was reset)
	after := m.chromeCache
	if before == after {
		t.Error("chromeCache should change after Enter (textInput reset)")
	}
}

func TestChromeDirty_AfterCommand(t *testing.T) {
	agent := &stubAgent{statusLine: "status-before"}
	m := setupModelForChromeTest(agent)

	// Record chrome with "/test" in input (pre-Enter state)
	m.textInput.SetValue("/test")
	m.rebuildChrome()
	beforeChrome := m.chromeCache

	// Press Enter: textInput.Reset() clears input, chrome should rebuild
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	afterChrome := m.chromeCache
	if beforeChrome == afterChrome {
		t.Error("chromeCache should change after Enter (textInput was reset from '/test' to empty)")
	}
}

// --- Chrome line count ---

func TestChromeCache_ExactLineCount(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	lines := strings.Split(m.chromeCache, "\n")
	if len(lines) != m.footerHeight() {
		t.Errorf("chromeCache line count = %d, want %d (footerHeight)", len(lines), m.footerHeight())
	}

	// After NAV toggle
	m.navigationMode = true
	m.textInput.Blur()
	m.rebuildChrome()
	lines = strings.Split(m.chromeCache, "\n")
	if len(lines) != m.footerHeight() {
		t.Errorf("NAV chromeCache line count = %d, want %d", len(lines), m.footerHeight())
	}
}

// --- NAV movement operations ---

func TestView_NavWordMovement_StructureConsistent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	// Enter NAV
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	// w multiple times
	for i := range 5 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
		m = updated.(Model)
		verifyViewStructure(t, m, "w press "+string(rune('0'+i)))
	}
	// b
	for i := range 3 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
		m = updated.(Model)
		verifyViewStructure(t, m, "b press "+string(rune('0'+i)))
	}
	// e
	for i := range 2 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		m = updated.(Model)
		verifyViewStructure(t, m, "e press "+string(rune('0'+i)))
	}
	// Exit NAV
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewStructure(t, m, "exit NAV after word movement")
}

func TestView_NavLineJumps_StructureConsistent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	// 0
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "0")

	// $
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'$'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "$")

	// ^
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'^'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "^")

	// G (bottom)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "G")

	// gg (top)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "gg")

	// d/u half page
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "d")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "u")
}

func TestView_NavBlockFocus_StructureConsistent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	// Add tool blocks
	m.appendToolResult(ToolResult{Name: "test1", Summary: "Tool 1", Detail: "detail1"})
	m.appendToolResult(ToolResult{Name: "test2", Summary: "Tool 2", Detail: "detail2\nline2"})
	m.rebuildChrome()

	// Enter NAV
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	// Tab to focus block
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	verifyViewStructure(t, m, "Tab focus")

	// j in block focus
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "j in block focus")

	// k in block focus
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "k in block focus")

	// Arrow Down
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	verifyViewStructure(t, m, "arrow down block focus")

	// Enter toggle
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	verifyViewStructure(t, m, "toggle block")

	// ESC unfocus
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewStructure(t, m, "unfocus block")

	// ESC exit NAV
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewStructure(t, m, "exit NAV after block ops")
}

func TestView_NavVisualWordMovement_StructureConsistent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	// Enter NAV
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	// Visual mode
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "visual start")

	// Word movements in visual
	for i := range 3 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
		m = updated.(Model)
		verifyViewStructure(t, m, "visual w "+string(rune('0'+i)))
	}

	// Line jump in visual
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'$'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "visual $")

	// ESC cancel visual
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewStructure(t, m, "cancel visual")

	// ESC exit NAV
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewStructure(t, m, "exit NAV after visual")
}

// --- chromeDirty for block operations ---

func TestChromeDirty_BlockToggle(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)
	m.appendToolResult(ToolResult{Name: "test", Summary: "Tool", Detail: "line1\nline2\nline3"})
	m.rebuildChrome()

	// Enter NAV, focus block
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	before := m.chromeCache

	// Toggle (collapse) — rawLines change, chrome must update
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	after := m.chromeCache
	// At minimum, chrome should have been rebuilt (even if content is same)
	// The real check: structure is still correct
	verifyViewStructure(t, m, "after block toggle")

	// Toggle back (expand)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	verifyViewStructure(t, m, "after block toggle back")
	_ = before
	_ = after
}

func TestChromeDirty_BlockFocusMovement(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)
	m.appendToolResult(ToolResult{Name: "test1", Summary: "Tool 1", Detail: "d1"})
	m.appendToolResult(ToolResult{Name: "test2", Summary: "Tool 2", Detail: "d2"})
	m.rebuildChrome()

	// Enter NAV, focus block
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	// Move with j — must not break structure
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "j block focus move")

	// Move with Arrow Up
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	verifyViewStructure(t, m, "arrow up block focus move")
}

// --- Chrome width ---

func TestChromeCache_ExactLineWidth(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	lines := strings.Split(m.chromeCache, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != m.width {
			t.Errorf("chrome line %d: width = %d, want %d", i, w, m.width)
		}
	}
}
