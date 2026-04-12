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
		{"gather_context", "🧭"},
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

func TestToolConfirmBoxToWriter_TruncatesLongUnknownToolName(t *testing.T) {
	var buf bytes.Buffer

	ToolConfirmBoxToWriter(&buf, strings.Repeat("unknown_tool_name_", 4), []string{"path: test.txt"})

	output := stripANSI(buf.String())
	if !strings.Contains(output, "🔧") {
		t.Fatalf("expected output to contain default icon, got %q", output)
	}
	if !strings.Contains(output, "...") {
		t.Fatalf("expected long title to be truncated, got %q", output)
	}
}

func TestToolConfirmBoxToWriter_EmptyDetailsAndNilWriter(t *testing.T) {
	ToolConfirmBoxToWriter(nil, "write_file", nil)

	var buf bytes.Buffer
	ToolConfirmBoxToWriter(&buf, "write_file", nil)

	output := stripANSI(buf.String())
	if !strings.Contains(output, "write_file") {
		t.Fatalf("expected output to contain tool name, got %q", output)
	}
}

func TestToolConfirmBoxWithRuntime_UsesRuntimeWriter(t *testing.T) {
	var buf bytes.Buffer
	runtime := NewRuntime(strings.NewReader(""), &buf, &buf)

	ToolConfirmBoxWithRuntime(runtime, "write_file", []string{"path: test.txt"})

	output := stripANSI(buf.String())
	if !strings.Contains(output, "write_file") {
		t.Fatalf("expected runtime output to contain tool name, got %q", output)
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

func TestConfirmPromptBoxToWriter_TruncatesLongMessageAndNilWriter(t *testing.T) {
	ConfirmPromptBoxToWriter(nil, "ignored")

	var buf bytes.Buffer
	ConfirmPromptBoxToWriter(&buf, strings.Repeat("a", 80))

	output := stripANSI(buf.String())
	if !strings.Contains(output, "...") {
		t.Fatalf("expected truncated prompt message, got %q", output)
	}
}

func TestConfirmPromptBoxWithRuntime_UsesRuntimeWriter(t *testing.T) {
	var buf bytes.Buffer
	runtime := NewRuntime(strings.NewReader(""), &buf, &buf)

	ConfirmPromptBoxWithRuntime(runtime, "Apply changes?")

	output := stripANSI(buf.String())
	if !strings.Contains(output, "Apply changes?") {
		t.Fatalf("expected runtime output to contain message, got %q", output)
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

func TestSimpleDividerToWriter_NilWriter(t *testing.T) {
	SimpleDividerToWriter(nil, 5)
}

func TestSimpleDividerWithRuntime_UsesRuntimeWriter(t *testing.T) {
	var buf bytes.Buffer
	runtime := NewRuntime(strings.NewReader(""), &buf, &buf)

	SimpleDividerWithRuntime(runtime, 3)

	output := stripANSI(buf.String())
	if output != "───\n" {
		t.Fatalf("SimpleDividerWithRuntime() = %q, want %q", output, "───\n")
	}
}

func TestPrintBoxLineToWriter_TruncatesLongText(t *testing.T) {
	var buf bytes.Buffer

	printBoxLineToWriter(&buf, strings.Repeat("a", 80), 20)

	output := stripANSI(buf.String())
	if !strings.Contains(output, "...") {
		t.Fatalf("expected truncated box line, got %q", output)
	}
}

func TestPrintBoxLineToWriter_ShortTextAndNilWriter(t *testing.T) {
	printBoxLineToWriter(nil, "ignored", 20)

	var buf bytes.Buffer
	printBoxLineToWriter(&buf, "short", 20)

	output := stripANSI(buf.String())
	if !strings.Contains(output, "short") {
		t.Fatalf("expected short box line, got %q", output)
	}
}

func TestTruncateBoxText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		maxLen int
		want   string
	}{
		{name: "empty max", text: "abcdef", maxLen: 0, want: ""},
		{name: "small max", text: "abcdef", maxLen: 3, want: "abc"},
		{name: "truncate", text: "abcdef", maxLen: 5, want: "ab..."},
		{name: "keep", text: "abc", maxLen: 5, want: "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncateBoxText(tt.text, tt.maxLen); got != tt.want {
				t.Fatalf("truncateBoxText(%q, %d) = %q, want %q", tt.text, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestBoxLinePadding(t *testing.T) {
	if got := boxLinePadding(10, 3); got != 5 {
		t.Fatalf("boxLinePadding(10, 3) = %d, want 5", got)
	}
	if got := boxLinePadding(5, 10); got != 0 {
		t.Fatalf("boxLinePadding(5, 10) = %d, want 0", got)
	}
}
