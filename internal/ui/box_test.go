package ui

import (
	"testing"
)

func TestGetToolIcon(t *testing.T) {
	tests := []struct {
		toolName string
		want     string
	}{
		// File operations
		{"read_file", "📖"},
		{"write_file", "📝"},
		{"str_replace", "✏️"},
		{"delete_file", "🗑️"},
		{"delete_lines", "✂️"},
		{"move_file", "📦"},
		{"copy_file", "📋"},
		{"list_dir", "📁"},

		// Search
		{"search_code", "🔍"},
		{"search_file", "🔎"},

		// Shell
		{"bash", "💻"},

		// Git operations
		{"git_add", "➕"},
		{"git_commit", "💾"},
		{"git_push", "🚀"},
		{"git_status", "📊"},
		{"git_diff", "📄"},
		{"git_log", "📜"},
		{"git_branch", "🌿"},
		{"git_checkout", "🔀"},
		{"git_stash", "📥"},

		// Code quality
		{"lint", "🔬"},
		{"test", "🧪"},

		// Web
		{"web_search", "🌐"},

		// Insert operations
		{"insert_after", "⬇️"},
		{"insert_before", "⬆️"},

		// Unknown tool should return default icon
		{"unknown_tool", "🔧"},
		{"", "🔧"},
		{"some_random_tool", "🔧"},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			got := getToolIcon(tt.toolName)
			if got != tt.want {
				t.Errorf("getToolIcon(%q) = %q, want %q", tt.toolName, got, tt.want)
			}
		})
	}
}
