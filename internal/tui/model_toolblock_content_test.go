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
