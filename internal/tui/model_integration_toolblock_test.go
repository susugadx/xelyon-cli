package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIIntegration_ToolBlockFocusExpandWheelCollapseSequence(t *testing.T) {
	m := NewModel(&stubAgent{statusLine: "ready"}, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 10, Height: 9})
	m = updated.(Model)
	m.navigationMode = true
	for i := 0; i < 12; i++ {
		m.appendContentLines(fmt.Sprintf("pad-%d", i))
	}
	m.appendToolResult(ToolResult{
		Name:      "bash",
		Summary:   "bash",
		Detail:    "\033[32m0123456789abcdef\033[0m",
		Collapsed: true,
	})

	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	updated, _ = m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelUp, Action: tea.MouseActionPress})
	m = updated.(Model)
	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.focusedBlock != 0 {
		t.Fatalf("focusedBlock = %d, want 0", m.focusedBlock)
	}
	if !m.toolBlocks[0].tool.Collapsed {
		t.Fatal("block should be collapsed again after second Enter")
	}
	if m.toolBlocks[0].block.lineStart <= m.toolBlocks[0].block.lineCount-1 {
		t.Fatalf("unexpected tool block lineStart = %d", m.toolBlocks[0].block.lineStart)
	}
	if m.vp.yOffset < 0 {
		t.Fatalf("viewport yOffset = %d, want >= 0", m.vp.yOffset)
	}
}
