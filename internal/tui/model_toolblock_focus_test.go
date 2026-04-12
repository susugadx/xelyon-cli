package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestToolBlock_MoveBlockFocusClampsRange(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})

	m.appendToolResult(ToolResult{Name: "a", Summary: "a", Detail: "a", Collapsed: true})
	m.appendToolResult(ToolResult{Name: "b", Summary: "b", Detail: "b", Collapsed: true})
	m.appendToolResult(ToolResult{Name: "c", Summary: "c", Detail: "c", Collapsed: true})

	m.setBlockFocus(1)
	if m.focusedBlock != 1 {
		t.Fatalf("focusedBlock = %d, want 1", m.focusedBlock)
	}

	m.moveBlockFocus(-1)
	if m.focusedBlock != 0 {
		t.Fatalf("after move to -1: focusedBlock = %d, want 0", m.focusedBlock)
	}
	m.moveBlockFocus(-1)
	if m.focusedBlock != 0 {
		t.Fatalf("after move to -1 again: focusedBlock = %d, want 0 (clamped)", m.focusedBlock)
	}

	m.moveBlockFocus(100)
	if m.focusedBlock != 2 {
		t.Fatalf("after move to 100: focusedBlock = %d, want 2 (clamped)", m.focusedBlock)
	}
}

func TestToolBlock_FocusIndicatorReflected(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})

	m.appendToolResult(ToolResult{Name: "a", Summary: "test-summary", Detail: "d", Collapsed: true})

	firstLine := m.rawLines[m.toolBlocks[0].lineStart]
	if firstLine[0] != ' ' {
		t.Fatalf("unfocused indicator = %q, want space", string(firstLine[0]))
	}

	m.setBlockFocus(0)
	firstLine = m.rawLines[m.toolBlocks[0].lineStart]
	if !strings.HasPrefix(firstLine, "→") {
		t.Fatalf("focused line = %q, want → prefix", firstLine)
	}

	m.clearBlockFocus()
	firstLine = m.rawLines[m.toolBlocks[0].lineStart]
	if firstLine[0] != ' ' {
		t.Fatalf("after clear: indicator = %q, want space", string(firstLine[0]))
	}
}

func TestToolBlock_TabKeyMovesFocusAndEnterToggles(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})
	m.navigationMode = true

	m.appendToolResult(ToolResult{
		Name: "read_file", Summary: "first",
		Detail: "match1", Collapsed: true,
	})
	m.appendToolResult(ToolResult{
		Name: "search_code", Summary: "second",
		Detail: "match2", Collapsed: true,
	})

	updated, _ := m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.focusedBlock != 1 {
		t.Fatalf("after first Tab: focusedBlock = %d, want 1", m.focusedBlock)
	}

	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.focusedBlock != 1 {
		t.Fatalf("after second Tab at edge: focusedBlock = %d, want 1", m.focusedBlock)
	}
	if !m.toolBlocks[1].tool.Collapsed {
		t.Fatal("Tab should not toggle block collapse state")
	}

	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.toolBlocks[1].tool.Collapsed {
		t.Fatal("Enter should toggle focused block")
	}
}

func TestToolBlock_EnterFallbackTogglesFocusedBlock(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})
	m.navigationMode = true

	m.appendToolResult(ToolResult{
		Name: "read_file", Summary: "first",
		Detail: "match1", Collapsed: true,
	})
	m.setBlockFocus(0)

	updated, _ := m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'\r'}})
	m = updated.(Model)

	if m.toolBlocks[0].tool.Collapsed {
		t.Fatal("Enter fallback should toggle focused block")
	}
}

func TestToolBlock_ShiftTabMovesFocusBackward(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})
	m.navigationMode = true

	m.appendToolResult(ToolResult{Name: "a", Summary: "first", Detail: "a", Collapsed: true})
	m.appendToolResult(ToolResult{Name: "b", Summary: "second", Detail: "b", Collapsed: true})

	updated, _ := m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(Model)
	if m.focusedBlock != 0 {
		t.Fatalf("after first Shift+Tab: focusedBlock = %d, want 0", m.focusedBlock)
	}

	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.focusedBlock != 1 {
		t.Fatalf("after Tab: focusedBlock = %d, want 1", m.focusedBlock)
	}

	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(Model)
	if m.focusedBlock != 0 {
		t.Fatalf("after second Shift+Tab: focusedBlock = %d, want 0", m.focusedBlock)
	}
}

func TestToolBlock_FocusedBlockMovesWithJKAndArrows(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})
	m.navigationMode = true

	m.appendToolResult(ToolResult{Name: "a", Summary: "first", Detail: "a", Collapsed: true})
	m.appendToolResult(ToolResult{Name: "b", Summary: "second", Detail: "b", Collapsed: true})
	m.appendToolResult(ToolResult{Name: "c", Summary: "third", Detail: "c", Collapsed: true})

	updated, _ := m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.focusedBlock != 2 {
		t.Fatalf("after Tab: focusedBlock = %d, want 2", m.focusedBlock)
	}

	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(Model)
	if m.focusedBlock != 1 {
		t.Fatalf("after k: focusedBlock = %d, want 1", m.focusedBlock)
	}

	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.focusedBlock != 0 {
		t.Fatalf("after ↑: focusedBlock = %d, want 0", m.focusedBlock)
	}

	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.focusedBlock != 1 {
		t.Fatalf("after ↓: focusedBlock = %d, want 1", m.focusedBlock)
	}

	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.focusedBlock != 2 {
		t.Fatalf("after j: focusedBlock = %d, want 2", m.focusedBlock)
	}
}

func TestToolBlock_EscClearsBlockFocusBeforeExitingNav(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})
	m.navigationMode = true

	m.appendToolResult(ToolResult{Name: "a", Summary: "a", Detail: "d", Collapsed: true})
	m.setBlockFocus(0)

	updated, _ := m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.focusedBlock != -1 {
		t.Fatalf("after Esc: focusedBlock = %d, want -1", m.focusedBlock)
	}
	if !m.navigationMode {
		t.Fatal("after Esc with focus: should still be in NAV mode")
	}

	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.navigationMode {
		t.Fatal("after second Esc: should exit NAV mode")
	}
}
