package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestChromeDirty_AfterTextInputReset(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	m.textInput.SetValue("hello")
	m.rebuildChrome()
	before := m.chromeCache

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	after := m.chromeCache
	if before == after {
		t.Error("chromeCache should change after Enter (textInput reset)")
	}
}

func TestChromeDirty_AfterCommand(t *testing.T) {
	agent := &stubAgent{statusLine: "status-before"}
	m := setupModelForChromeTest(agent)

	m.textInput.SetValue("/test")
	m.rebuildChrome()
	beforeChrome := m.chromeCache

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	afterChrome := m.chromeCache
	if beforeChrome == afterChrome {
		t.Error("chromeCache should change after Enter (textInput was reset from '/test' to empty)")
	}
}

func TestChromeCache_ExactLineCount(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	lines := strings.Split(m.chromeCache, "\n")
	if len(lines) != m.footerHeight() {
		t.Errorf("chromeCache line count = %d, want %d (footerHeight)", len(lines), m.footerHeight())
	}

	m.navigationMode = true
	m.textInput.Blur()
	m.rebuildChrome()
	lines = strings.Split(m.chromeCache, "\n")
	if len(lines) != m.footerHeight() {
		t.Errorf("NAV chromeCache line count = %d, want %d", len(lines), m.footerHeight())
	}
}

func TestChromeDirty_BlockToggle(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)
	m.appendToolResult(ToolResult{Name: "test", Summary: "Tool", Detail: "line1\nline2\nline3"})
	m.rebuildChrome()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	verifyViewStructure(t, m, "after block toggle")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	verifyViewStructure(t, m, "after block toggle back")
}

func TestChromeDirty_BlockFocusMovement(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)
	m.appendToolResult(ToolResult{Name: "test1", Summary: "Tool 1", Detail: "d1"})
	m.appendToolResult(ToolResult{Name: "test2", Summary: "Tool 2", Detail: "d2"})
	m.rebuildChrome()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "j block focus move")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	verifyViewStructure(t, m, "arrow up block focus move")
}

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
