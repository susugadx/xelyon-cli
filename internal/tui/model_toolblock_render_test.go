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

	if got := m.toolBlockSummaryLine(0); got != "  search_code: test" {
		t.Fatalf("collapsed summary = %q, want %q", got, "  search_code: test")
	}

	m.setBlockFocus(0)
	if got := m.toolBlockSummaryLine(0); got != "▶ search_code: test" {
		t.Fatalf("focused summary = %q, want %q", got, "▶ search_code: test")
	}

	m.toolBlocks[0].tool.Collapsed = false
	if got := m.toolBlockSummaryLine(0); got != "▶ search_code: test" {
		t.Fatalf("expanded summary = %q, want %q", got, "▶ search_code: test")
	}
}

func TestToolBlockSummaryLineRemovesToolTypeIcon(t *testing.T) {
	tests := []struct {
		name    string
		summary string
		want    string
	}{
		{name: "search", summary: "🔍 search_code: test", want: "  search_code: test"},
		{name: "gather context", summary: "🧭 gather_context: query", want: "  gather_context: query"},
		{name: "str replace", summary: "✏️ str_replace: a.go:30", want: "  str_replace: a.go:30"},
		{name: "spawn agent", summary: "🚀 spawn_agent: inspect files", want: "  spawn_agent: inspect files"},
		{name: "wait agent", summary: "⏳ wait_agent: 2 agents", want: "  wait_agent: 2 agents"},
		{name: "git", summary: "📦 git_status: short", want: "  git_status: short"},
		{name: "mcp", summary: "🔌 mcp_call: server", want: "  mcp_call: server"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newModelWithViewport(&stubAgent{statusLine: "ready"})
			m.appendToolResult(ToolResult{
				Name:      "tool",
				Summary:   tt.summary,
				Detail:    "line1",
				Collapsed: true,
			})

			if got := m.toolBlockSummaryLine(0); got != tt.want {
				t.Fatalf("summary = %q, want %q", got, tt.want)
			}
		})
	}
}
