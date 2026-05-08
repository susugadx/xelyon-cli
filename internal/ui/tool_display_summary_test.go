package ui

import "testing"

func TestStripToolDisplayIconPrefix(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "read file", in: "📄 read_file: a.go", want: "read_file: a.go"},
		{name: "gather context", in: "🧭 gather_context: query", want: "gather_context: query"},
		{name: "str replace", in: "✏️ str_replace: a.go:30", want: "str_replace: a.go:30"},
		{name: "spawn agent", in: "🚀 spawn_agent: inspect files", want: "spawn_agent: inspect files"},
		{name: "wait agent", in: "⏳ wait_agent: 2 agents", want: "wait_agent: 2 agents"},
		{name: "git", in: "📦 git_status: short", want: "git_status: short"},
		{name: "mcp", in: "🔌 mcp_call: server", want: "mcp_call: server"},
		{name: "unknown", in: "🔧 custom_tool: target", want: "custom_tool: target"},
		{name: "no icon", in: "read_file: a.go", want: "read_file: a.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripToolDisplayIconPrefix(tt.in); got != tt.want {
				t.Fatalf("StripToolDisplayIconPrefix(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
