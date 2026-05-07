package tui

import "testing"

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
	toolBlock := m.toolBlocks[0]
	if toolBlock.block.lineStart != baseLines {
		t.Fatalf("lineStart = %d, want %d", toolBlock.block.lineStart, baseLines)
	}
	if toolBlock.block.lineCount != 1 {
		t.Fatalf("collapsed lineCount = %d, want 1", toolBlock.block.lineCount)
	}
}

func TestToolBlock_UpsertsToolResultByID(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})

	m.appendToolResult(ToolResult{
		ID:        "tool-1",
		Name:      "read_file",
		Summary:   "● running read_file a.go",
		Target:    "a.go",
		Status:    ToolStatusRunning,
		Collapsed: true,
	})
	m.appendToolResult(ToolResult{
		ID:        "tool-1",
		Name:      "read_file",
		Summary:   "✓ ok read_file a.go 12ms",
		Detail:    "file contents",
		Target:    "a.go",
		Status:    ToolStatusOK,
		Collapsed: true,
	})

	if len(m.toolBlocks) != 1 {
		t.Fatalf("toolBlocks len = %d, want 1", len(m.toolBlocks))
	}
	if got := m.toolBlocks[0].tool.Status; got != ToolStatusOK {
		t.Fatalf("status = %q, want ok", got)
	}
	if got := m.rawLines[m.toolBlocks[0].block.lineStart]; got != " ▶ ✓ ok read_file a.go 12ms" {
		t.Fatalf("summary line = %q, want updated ok summary", got)
	}
}

func TestToolBlock_ReusedCompletedIDAppendsNewBlock(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})

	m.appendToolResult(ToolResult{
		ID:        "call_rescue_001",
		Name:      "read_file",
		Summary:   "● running read_file first.go",
		Status:    ToolStatusRunning,
		Collapsed: true,
	})
	m.appendToolResult(ToolResult{
		ID:        "call_rescue_001",
		Name:      "read_file",
		Summary:   "✓ ok read_file first.go",
		Status:    ToolStatusOK,
		Collapsed: true,
	})
	m.appendToolResult(ToolResult{
		ID:        "call_rescue_001",
		Name:      "read_file",
		Summary:   "● running read_file second.go",
		Status:    ToolStatusRunning,
		Collapsed: true,
	})
	m.appendToolResult(ToolResult{
		ID:        "call_rescue_001",
		Name:      "read_file",
		Summary:   "✓ ok read_file second.go",
		Status:    ToolStatusOK,
		Collapsed: true,
	})

	if len(m.toolBlocks) != 2 {
		t.Fatalf("toolBlocks len = %d, want separate blocks for reused completed ID", len(m.toolBlocks))
	}
	if got := m.toolBlocks[0].tool.Summary; got != "✓ ok read_file first.go" {
		t.Fatalf("first block summary = %q, want first final preserved", got)
	}
	if got := m.toolBlocks[1].tool.Summary; got != "✓ ok read_file second.go" {
		t.Fatalf("second block summary = %q, want second final", got)
	}
}

func TestToolBlock_ErrorUpdateUsesExpandedFinalState(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})

	m.appendToolResult(ToolResult{
		ID:        "tool-1",
		Name:      "read_file",
		Summary:   "● running read_file a.go",
		Status:    ToolStatusRunning,
		Collapsed: true,
	})
	m.appendToolResult(ToolResult{
		ID:        "tool-1",
		Name:      "read_file",
		Summary:   "✕ error read_file a.go 12ms",
		Detail:    "Error: missing file",
		Error:     true,
		Status:    ToolStatusError,
		Collapsed: false,
	})

	if len(m.toolBlocks) != 1 {
		t.Fatalf("toolBlocks len = %d, want 1", len(m.toolBlocks))
	}
	toolBlock := m.toolBlocks[0]
	if toolBlock.tool.Collapsed {
		t.Fatal("error update should use expanded final state")
	}
	if toolBlock.block.lineCount != 2 {
		t.Fatalf("error block lineCount = %d, want summary + detail", toolBlock.block.lineCount)
	}
}

func TestToolBlock_UpdatePreservesBottomFollowWhenExpanded(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})
	m.vp.height = 2
	m.appendContentLines("line1", "line2", "line3")
	m.vp.gotoBottom()

	m.appendToolResult(ToolResult{
		ID:        "tool-1",
		Name:      "read_file",
		Summary:   "● running read_file a.go",
		Status:    ToolStatusRunning,
		Collapsed: true,
	})
	if !m.vp.atBottom() {
		t.Fatal("running block append should leave viewport at bottom")
	}

	m.appendToolResult(ToolResult{
		ID:        "tool-1",
		Name:      "read_file",
		Summary:   "✕ error read_file a.go 12ms",
		Detail:    "Error: missing file\nline2",
		Error:     true,
		Status:    ToolStatusError,
		Collapsed: false,
	})

	if !m.vp.atBottom() {
		t.Fatal("expanded tool update should preserve bottom follow")
	}
	if m.newOutput {
		t.Fatal("newOutput should remain false when expanded update follows bottom")
	}
}

func TestToolBlock_UpdateMarksNewOutputAndShiftsFollowingBlocksWhenNotFollowing(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})
	m.vp.height = 3
	m.appendContentLines("line1", "line2", "line3", "line4")
	m.appendToolResult(ToolResult{
		ID:        "tool-1",
		Name:      "read_file",
		Summary:   "● running read_file a.go",
		Status:    ToolStatusRunning,
		Collapsed: true,
	})
	m.appendToolResult(ToolResult{
		Name:      "search_code",
		Summary:   "second block",
		Detail:    "detail",
		Collapsed: true,
	})
	secondStart := m.toolBlocks[1].block.lineStart

	m.vp.gotoTop()
	m.newOutput = false
	m.appendToolResult(ToolResult{
		ID:        "tool-1",
		Name:      "read_file",
		Summary:   "✕ error read_file a.go 12ms",
		Detail:    "Error: missing file\nline2",
		Error:     true,
		Status:    ToolStatusError,
		Collapsed: false,
	})

	if m.vp.yOffset != 0 {
		t.Fatalf("viewport yOffset = %d, want unchanged top", m.vp.yOffset)
	}
	if !m.newOutput {
		t.Fatal("newOutput should be marked when expanded update happens away from bottom")
	}
	if got, want := m.toolBlocks[1].block.lineStart, secondStart+2; got != want {
		t.Fatalf("second block lineStart = %d, want %d", got, want)
	}
	if got := m.rawLines[m.toolBlocks[1].block.lineStart]; got != " ▶ second block" {
		t.Fatalf("second block summary line = %q, want shifted summary", got)
	}
}

func TestToolBlock_CompletionWithoutKnownIDAppends(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})

	m.appendToolResult(ToolResult{ID: "tool-1", Name: "read_file", Summary: "● running read_file", Status: ToolStatusRunning, Collapsed: true})
	m.appendToolResult(ToolResult{ID: "tool-2", Name: "read_file", Summary: "✓ ok read_file", Status: ToolStatusOK, Collapsed: true})

	if len(m.toolBlocks) != 2 {
		t.Fatalf("toolBlocks len = %d, want 2", len(m.toolBlocks))
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
	if m.toolBlocks[0].block.lineCount != 1 {
		t.Fatalf("collapsed lineCount = %d, want 1", m.toolBlocks[0].block.lineCount)
	}

	m.toggleToolBlock(0)
	if m.toolBlocks[0].tool.Collapsed {
		t.Fatal("block should be expanded after toggle")
	}
	expandedLineCount := m.toolBlocks[0].block.lineCount
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

	block1Start := m.toolBlocks[1].block.lineStart
	if block1Start != m.toolBlocks[0].block.lineStart+1 {
		t.Fatalf("second block lineStart = %d, want %d", block1Start, m.toolBlocks[0].block.lineStart+1)
	}

	m.toggleToolBlock(0)
	delta := m.toolBlocks[0].block.lineCount - 1
	expectedStart := block1Start + delta
	if m.toolBlocks[1].block.lineStart != expectedStart {
		t.Fatalf("after expand: second block lineStart = %d, want %d", m.toolBlocks[1].block.lineStart, expectedStart)
	}

	m.toggleToolBlock(0)
	if m.toolBlocks[1].block.lineStart != block1Start {
		t.Fatalf("after collapse: second block lineStart = %d, want %d", m.toolBlocks[1].block.lineStart, block1Start)
	}
}
