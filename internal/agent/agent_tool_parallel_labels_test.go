package agent

import "testing"

func TestNormalizeParallelToolFamily(t *testing.T) {
	tests := []struct {
		name string
		tool string
		want string
	}{
		{name: "read_file", tool: "read_file", want: "reads"},
		{name: "read_files", tool: "read_files", want: "reads"},
		{name: "list_dir", tool: "list_dir", want: "reads"},
		{name: "search_code", tool: "search_code", want: "searches"},
		{name: "web_search", tool: "web_search", want: "web"},
		{name: "bash passthrough", tool: "bash", want: "bash"},
		{name: "spawn passthrough", tool: "spawn_agent", want: "spawn_agent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeParallelToolFamily(tt.tool); got != tt.want {
				t.Fatalf("normalizeParallelToolFamily(%q) = %q, want %q", tt.tool, got, tt.want)
			}
		})
	}
}

func TestParallelGroupSummaryLabel(t *testing.T) {
	tests := []struct {
		name string
		tool string
		want string
	}{
		{name: "reads", tool: "read_file", want: "reads"},
		{name: "searches", tool: "search_code", want: "searches"},
		{name: "web", tool: "web_search", want: "web"},
		{name: "bash special", tool: "bash", want: "bash"},
		{name: "spawn passthrough", tool: "spawn_agent", want: "spawn_agent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parallelGroupSummaryLabel(tt.tool); got != tt.want {
				t.Fatalf("parallelGroupSummaryLabel(%q) = %q, want %q", tt.tool, got, tt.want)
			}
		})
	}
}

func TestParallelSpinnerBucket(t *testing.T) {
	tests := []struct {
		name string
		tool string
		want string
	}{
		{name: "reads", tool: "read_file", want: "reads"},
		{name: "searches", tool: "search_code", want: "searches"},
		{name: "web", tool: "web_search", want: "web"},
		{name: "spawn", tool: "spawn_agent", want: "spawn"},
		{name: "wait", tool: "wait_agent", want: "wait"},
		{name: "bash grouped as tools", tool: "bash", want: "tools"},
		{name: "unknown grouped as tools", tool: "custom_tool", want: "tools"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parallelSpinnerBucket(tt.tool); got != tt.want {
				t.Fatalf("parallelSpinnerBucket(%q) = %q, want %q", tt.tool, got, tt.want)
			}
		})
	}
}
