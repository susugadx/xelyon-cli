package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestToolBlock_TabFocusLastBlockClearsNewOutputAtBottom(t *testing.T) {
	m := NewModel(&stubAgent{statusLine: "ready"}, "")
	m.ready = true
	m.width = 20
	m.height = 8
	m.vp = lightViewport{width: 20, height: 4}
	m.navigationMode = true

	for i := 0; i < 20; i++ {
		m.appendContentLines(fmt.Sprintf("line-%d", i))
	}
	m.appendToolResult(ToolResult{Name: "tool", Summary: "last", Detail: "d", Collapsed: true})

	m.vp.gotoTop()
	m.newOutput = true
	m.chromeDirty = true
	m.rebuildChrome()
	if !strings.Contains(m.chromeCache, "↓ New") {
		t.Fatalf("chromeCache should include new output badge before focus jump, got %q", m.chromeCache)
	}

	updated, _ := m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	m.rebuildChrome()

	if !m.vp.atBottom() {
		t.Fatal("viewport should be at bottom after focusing the last tool block")
	}
	if m.newOutput {
		t.Fatal("newOutput should be cleared when block focus jump reaches bottom")
	}
	if strings.Contains(m.chromeCache, "↓ New") {
		t.Fatalf("chromeCache should clear new output badge after focus jump, got %q", m.chromeCache)
	}
}

func TestToolBlock_JMoveFocusClearsNewOutputBadgeInChrome(t *testing.T) {
	m := NewModel(&stubAgent{statusLine: "ready"}, "")
	m.ready = true
	m.width = 20
	m.height = 8
	m.vp = lightViewport{width: 20, height: 4}
	m.navigationMode = true

	for i := 0; i < 20; i++ {
		m.appendContentLines(fmt.Sprintf("line-%d", i))
	}
	for i := 0; i < 10; i++ {
		m.appendToolResult(ToolResult{
			Name:      fmt.Sprintf("tool-%d", i),
			Summary:   fmt.Sprintf("block-%d", i),
			Detail:    "x",
			Collapsed: true,
		})
	}

	m.vp.gotoTop()
	m.newOutput = true
	m.setBlockFocus(7)
	if m.vp.atBottom() {
		t.Fatal("precondition failed: focus should not be at bottom before j move")
	}
	m.chromeDirty = true
	m.rebuildChrome()
	if !strings.Contains(m.chromeCache, "↓ New") {
		t.Fatalf("chromeCache should include new output badge before j move, got %q", m.chromeCache)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	if !m.vp.atBottom() {
		t.Fatal("viewport should reach bottom after moving focus to last block")
	}
	if m.newOutput {
		t.Fatal("newOutput should be cleared after j move reaches bottom")
	}
	if strings.Contains(m.chromeCache, "↓ New") {
		t.Fatalf("chromeCache should clear new output badge after j move, got %q", m.chromeCache)
	}
}

func TestToolBlock_MouseWheelKeepsFocusedBlockState(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})
	m.navigationMode = true

	for i := 0; i < 40; i++ {
		m.appendContentLines(fmt.Sprintf("padding-%d", i))
	}
	m.appendToolResult(ToolResult{Name: "a", Summary: "first", Detail: "d", Collapsed: true})
	m.appendToolResult(ToolResult{Name: "b", Summary: "second", Detail: "d", Collapsed: true})
	m.setBlockFocus(1)

	focusedBefore := m.focusedBlock
	lineBefore := m.toolBlocks[focusedBefore].block.lineStart

	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelUp,
		Action: tea.MouseActionPress,
	})
	m = updated.(Model)

	if m.focusedBlock != focusedBefore {
		t.Fatalf("focusedBlock changed from %d to %d", focusedBefore, m.focusedBlock)
	}
	if m.toolBlocks[m.focusedBlock].block.lineStart != lineBefore {
		t.Fatalf("focused block lineStart changed from %d to %d", lineBefore, m.toolBlocks[m.focusedBlock].block.lineStart)
	}

	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.toolBlocks[focusedBefore].tool.Collapsed {
		t.Fatal("focused block should still toggle after wheel scroll")
	}
}

func TestToolBlock_ExpandedANSIDetailTruncatesWithoutBleedingAtNarrowWidth(t *testing.T) {
	m := NewModel(&stubAgent{statusLine: "ready"}, "")
	m.ready = true
	m.width = 6
	m.height = 8
	m.vp = lightViewport{width: 6, height: 4}
	m.padLineCache = strings.Repeat(" ", 6)

	m.appendToolResult(ToolResult{
		Name:      "bash",
		Summary:   "bash",
		Detail:    "\033[31mabcdefghi\033[0m",
		Collapsed: false,
	})
	m.rebuildChrome()

	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) < 2 {
		t.Fatalf("view should include detail line, got %q", view)
	}
	detailLine := lines[1]

	if lipgloss.Width(stripANSI(detailLine)) != 6 {
		t.Fatalf("detail line width = %d, want 6; line=%q", lipgloss.Width(stripANSI(detailLine)), detailLine)
	}
	if !strings.HasSuffix(detailLine, "\033[0m") {
		t.Fatalf("detail line should end with ANSI reset, got %q", detailLine)
	}
}

func TestToolBlock_MouseWheelKeepsExpandedANSIDetailStable(t *testing.T) {
	m := NewModel(&stubAgent{statusLine: "ready"}, "")
	m.ready = true
	m.navigationMode = true
	m.width = 8
	m.height = 8
	m.vp = lightViewport{width: 8, height: 4}
	m.padLineCache = strings.Repeat(" ", 8)

	for i := 0; i < 10; i++ {
		m.appendContentLines(fmt.Sprintf("pad-%d", i))
	}
	m.appendToolResult(ToolResult{
		Name:      "bash",
		Summary:   "bash",
		Detail:    "\033[32m0123456789abcdef\033[0m",
		Collapsed: false,
	})
	m.setBlockFocus(0)
	m.rebuildChrome()

	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	m = updated.(Model)

	if m.focusedBlock != 0 {
		t.Fatalf("focusedBlock = %d, want 0", m.focusedBlock)
	}
	if m.toolBlocks[0].tool.Collapsed {
		t.Fatal("expanded block should remain expanded after wheel")
	}

	view := m.View()
	if !strings.Contains(view, "▼ bash") {
		t.Fatalf("view should still show expanded block summary, got %q", view)
	}
}

func TestToolBlock_ScrollToBlock(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})

	for i := 0; i < 50; i++ {
		m.appendContentLines("padding line")
	}
	m.appendToolResult(ToolResult{Name: "a", Summary: "target", Detail: "d", Collapsed: true})
	m.vp.setLines(m.getVisualRowContents())

	m.vp.gotoTop()
	if m.vp.yOffset != 0 {
		t.Fatalf("yOffset after gotoTop = %d, want 0", m.vp.yOffset)
	}

	m.scrollToBlock(0)
	blockStart := m.toolBlocks[0].block.lineStart
	target := max(0, blockStart-2)
	maxY := m.vp.maxYOffset()
	if target > maxY {
		target = maxY
	}
	if m.vp.yOffset != target {
		t.Fatalf("yOffset after scrollToBlock = %d, want %d", m.vp.yOffset, target)
	}
	if m.vp.yOffset > blockStart || m.vp.yOffset+m.vp.height <= blockStart {
		t.Fatalf("block at line %d not visible in viewport [%d, %d)", blockStart, m.vp.yOffset, m.vp.yOffset+m.vp.height)
	}
}
