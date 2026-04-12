package tui

import "testing"

func TestToolBlockSummaryLineReflectsFocusAndCollapse(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})
	m.appendToolResult(ToolResult{
		Name:      "search_code",
		Summary:   "search_code: test",
		Detail:    "line1",
		Collapsed: true,
	})

	if got := m.toolBlockSummaryLine(0); got != " ▶ search_code: test" {
		t.Fatalf("collapsed summary = %q, want %q", got, " ▶ search_code: test")
	}

	m.setBlockFocus(0)
	if got := m.toolBlockSummaryLine(0); got != "→▶ search_code: test" {
		t.Fatalf("focused summary = %q, want %q", got, "→▶ search_code: test")
	}

	m.toolBlocks[0].tool.Collapsed = false
	if got := m.toolBlockSummaryLine(0); got != "→▼ search_code: test" {
		t.Fatalf("expanded summary = %q, want %q", got, "→▼ search_code: test")
	}
}
