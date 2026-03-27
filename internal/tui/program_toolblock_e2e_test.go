package tui

import (
	"testing"
)

func TestTUIProgramE2E_ToolBlockSequence(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	model := NewModel(agent, "")

	final := setup(10, 9).
		nav().
		pads(12).
		tool(ToolResult{Name: "bash", Summary: "bash", Detail: "\033[32m0123456789abcdef\033[0m", Collapsed: true}).
		build().
		appendScript(keys("<Tab><Enter>")).
		append(sendMsg(wheelUp())).
		appendScript(keys("<Enter>")).
		run(t, model)

	if !final.ready {
		t.Fatal("model should be ready after window sizing")
	}
	if final.focusedBlock != 0 {
		t.Fatalf("focusedBlock = %d, want 0", final.focusedBlock)
	}
	if !final.toolBlocks[0].tool.Collapsed {
		t.Fatal("tool block should be collapsed after second Enter")
	}
	if final.vp.yOffset < 0 {
		t.Fatalf("viewport yOffset = %d, want >= 0", final.vp.yOffset)
	}
	lines := normalizedViewLines(final.View())
	if len(lines) == 0 {
		t.Fatal("view should not be empty")
	}
	assertGoldenText(t, "tool_block_status", lines[len(lines)-1])
}

func TestTUIProgramE2E_MultiToolBlockNavigation(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	model := NewModel(agent, "")

	final := setup(14, 10).
		nav().
		pads(6).
		tool(ToolResult{Name: "read_file", Summary: "first", Detail: "one", Collapsed: true}).
		tool(ToolResult{Name: "search_code", Summary: "second", Detail: "two", Collapsed: true}).
		build().
		appendScript(keys("<Tab>k<Enter>")).
		run(t, model)

	if final.focusedBlock != 0 {
		t.Fatalf("focusedBlock = %d, want 0", final.focusedBlock)
	}
	if final.toolBlocks[0].tool.Collapsed {
		t.Fatal("first block should be expanded after Enter")
	}
	if final.toolBlocks[1].tool.Collapsed != true {
		t.Fatal("second block should remain collapsed")
	}
	if final.cursorLine != final.toolBlocks[0].lineStart {
		t.Fatalf("cursorLine = %d, want %d", final.cursorLine, final.toolBlocks[0].lineStart)
	}
}
