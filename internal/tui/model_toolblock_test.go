package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestToolBlock_AppendToolResultTracksLineStart(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})

	m.appendContentLines("line1", "line2", "line3")
	baseLines := len(m.rawLines)

	m.appendToolResult(ToolResult{
		Name:      "search_code",
		Summary:   "🔍 search_code: test",
		Detail:    "match1\nmatch2",
		Collapsed: true,
	})

	if len(m.toolBlocks) != 1 {
		t.Fatalf("toolBlocks len = %d, want 1", len(m.toolBlocks))
	}
	block := m.toolBlocks[0]
	if block.lineStart != baseLines {
		t.Fatalf("lineStart = %d, want %d", block.lineStart, baseLines)
	}
	if block.lineCount != 1 {
		t.Fatalf("collapsed lineCount = %d, want 1", block.lineCount)
	}
}

func TestToolBlock_ToggleExpandsAndCollapses(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})

	m.appendToolResult(ToolResult{
		Name:      "search_code",
		Summary:   "🔍 search_code: test",
		Detail:    "match1\nmatch2\nmatch3",
		Collapsed: true,
	})

	initialLineCount := len(m.rawLines)
	if m.toolBlocks[0].lineCount != 1 {
		t.Fatalf("collapsed lineCount = %d, want 1", m.toolBlocks[0].lineCount)
	}

	m.toggleToolBlock(0)
	if m.toolBlocks[0].tool.Collapsed {
		t.Fatal("block should be expanded after toggle")
	}
	expandedLineCount := m.toolBlocks[0].lineCount
	if expandedLineCount != 4 {
		t.Fatalf("expanded lineCount = %d, want 4", expandedLineCount)
	}
	if len(m.rawLines) != initialLineCount+(expandedLineCount-1) {
		t.Fatalf("rawLines len = %d, want %d", len(m.rawLines), initialLineCount+(expandedLineCount-1))
	}

	m.toggleToolBlock(0)
	if !m.toolBlocks[0].tool.Collapsed {
		t.Fatal("block should be collapsed after second toggle")
	}
	if len(m.rawLines) != initialLineCount {
		t.Fatalf("rawLines len after re-collapse = %d, want %d", len(m.rawLines), initialLineCount)
	}
}

func TestToolBlock_MultipleBlocksLineStartUpdated(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})

	m.appendToolResult(ToolResult{
		Name: "read_file", Summary: "📄 read_file: a.go",
		Detail: "content1\ncontent2", Collapsed: true,
	})
	m.appendToolResult(ToolResult{
		Name: "search_code", Summary: "🔍 search_code: test",
		Detail: "match1", Collapsed: true,
	})

	block1Start := m.toolBlocks[1].lineStart
	if block1Start != m.toolBlocks[0].lineStart+1 {
		t.Fatalf("second block lineStart = %d, want %d", block1Start, m.toolBlocks[0].lineStart+1)
	}

	m.toggleToolBlock(0)
	delta := m.toolBlocks[0].lineCount - 1
	expectedStart := block1Start + delta
	if m.toolBlocks[1].lineStart != expectedStart {
		t.Fatalf("after expand: second block lineStart = %d, want %d", m.toolBlocks[1].lineStart, expectedStart)
	}

	m.toggleToolBlock(0)
	if m.toolBlocks[1].lineStart != block1Start {
		t.Fatalf("after collapse: second block lineStart = %d, want %d", m.toolBlocks[1].lineStart, block1Start)
	}
}

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

func TestToolBlock_TabKeyEntersFocusAndToggles(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})
	m.navigationMode = true

	m.appendToolResult(ToolResult{
		Name: "search_code", Summary: "🔍 search_code: test",
		Detail: "match1\nmatch2", Collapsed: true,
	})

	updated, _ := m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.focusedBlock != 0 {
		t.Fatalf("after first Tab: focusedBlock = %d, want 0", m.focusedBlock)
	}

	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.toolBlocks[0].tool.Collapsed {
		t.Fatal("after second Tab: block should be expanded")
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
	lineBefore := m.toolBlocks[focusedBefore].lineStart

	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelUp,
		Action: tea.MouseActionPress,
	})
	m = updated.(Model)

	if m.focusedBlock != focusedBefore {
		t.Fatalf("focusedBlock changed from %d to %d", focusedBefore, m.focusedBlock)
	}
	if m.toolBlocks[m.focusedBlock].lineStart != lineBefore {
		t.Fatalf("focused block lineStart changed from %d to %d", lineBefore, m.toolBlocks[m.focusedBlock].lineStart)
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
	blockStart := m.toolBlocks[0].lineStart
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
