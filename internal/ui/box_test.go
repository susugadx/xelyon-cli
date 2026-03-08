package ui

import (
	"bytes"
	"strings"
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
		{"copy_file", "📋"},
		{"list_dir", "📁"},

		// Shell
		{"bash", "💻"},

		// Git operations
		{"git_add", "➕"},
		{"git_push", "🚀"},
		{"git_status", "📊"},
		{"git_diff", "📄"},
		{"git_log", "📜"},
		{"git_branch", "🌿"},
		{"git_stash", "📥"},

		// Web
		{"web_search", "🌐"},

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

func TestToolConfirmBoxToWriter_UsesInjectedWriter(t *testing.T) {
	var buf bytes.Buffer

	ToolConfirmBoxToWriter(&buf, "write_file", []string{"path: test.txt"})

	output := stripANSI(buf.String())
	if !strings.Contains(output, "write_file") {
		t.Fatalf("expected output to contain tool name, got %q", output)
	}
	if !strings.Contains(output, "path: test.txt") {
		t.Fatalf("expected output to contain detail line, got %q", output)
	}
	if !strings.Contains(output, "[y] Approve") {
		t.Fatalf("expected output to contain approval choices, got %q", output)
	}
}

func TestConfirmPromptBoxToWriter_UsesInjectedWriter(t *testing.T) {
	var buf bytes.Buffer

	ConfirmPromptBoxToWriter(&buf, "Apply changes?")

	output := stripANSI(buf.String())
	if !strings.Contains(output, "Apply changes?") {
		t.Fatalf("expected output to contain message, got %q", output)
	}
	if !strings.Contains(output, "[c] Comment") {
		t.Fatalf("expected output to contain comment choice, got %q", output)
	}
}

func TestSimpleDividerToWriter_UsesInjectedWriter(t *testing.T) {
	var buf bytes.Buffer

	SimpleDividerToWriter(&buf, 5)

	output := stripANSI(buf.String())
	if output != "─────\n" {
		t.Fatalf("SimpleDividerToWriter() = %q, want %q", output, "─────\n")
	}
}
