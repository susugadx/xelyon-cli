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
		{name: "compress", in: "🗜️ compress: context", want: "compress: context"},
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

func TestCompressionToolTarget(t *testing.T) {
	info := ToolDisplayInfo{
		ToolName: ToolNameCompress,
		Args: map[string]string{
			ToolArgCompressionMode:         ToolCompressionModeHistory,
			ToolArgCompressionReason:       ToolCompressionReasonAuto,
			ToolArgCompressionBeforeTokens: "12,000",
			ToolArgCompressionAfterTokens:  "4,000",
		},
	}

	if got, want := ToolTarget(info), "context · auto · 12,000 -> 4,000 tok"; got != want {
		t.Fatalf("ToolTarget() = %q, want %q", got, want)
	}
}

func TestCompressionToolTargetCompactSkipped(t *testing.T) {
	info := ToolDisplayInfo{
		ToolName: ToolNameCompress,
		Args: map[string]string{
			ToolArgCompressionMode:         ToolCompressionModeCompactAPI,
			ToolArgCompressionReason:       ToolCompressionReasonAuto,
			ToolArgCompressionOutcome:      "history too short",
			ToolArgCompressionBeforeTokens: "2,000",
		},
	}

	if got, want := ToolTarget(info), "compact API · auto · history too short · 2,000 tok"; got != want {
		t.Fatalf("ToolTarget() = %q, want %q", got, want)
	}
}
