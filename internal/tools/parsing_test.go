package tools

import (
	"testing"
)

func TestTruncateDebug(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length",
			input:  "0123456789",
			maxLen: 10,
			want:   "0123456789",
		},
		{
			name:   "long string truncated",
			input:  "this is a very long string that should be truncated",
			maxLen: 20,
			want:   "this is a very long ...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "single character",
			input:  "a",
			maxLen: 10,
			want:   "a",
		},
		{
			name:   "maxLen exactly at string length",
			input:  "exact",
			maxLen: 5,
			want:   "exact",
		},
		{
			name:   "long string short maxLen",
			input:  "abcdefghijklmnop",
			maxLen: 5,
			want:   "abcde...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateDebug(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateDebug(%q, %d) = %q, want %q",
					tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestParseToolCalls_SingleTool(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
		wantTool string
	}{
		{
			name:     "simple tool call",
			input:    `{"tool": "read_file", "args": {"path": "main.go"}}`,
			wantLen:  1,
			wantTool: "read_file",
		},
		{
			name:     "tool call with space",
			input:    `{ "tool": "write_file", "args": {"path": "test.txt", "content": "hello"}}`,
			wantLen:  1,
			wantTool: "write_file",
		},
		{
			name:     "tool call with surrounding text",
			input:    `Let me read the file. {"tool": "read_file", "args": {"path": "main.go"}} Done.`,
			wantLen:  1,
			wantTool: "read_file",
		},
		{
			name:     "str_replace with braces in content",
			input:    `{"id":"call_abc","tool":"str_replace","args":{"path":"main.go","old_str":"func foo() {","new_str":"func bar() {"}}`,
			wantLen:  1,
			wantTool: "str_replace",
		},
		{
			name:     "str_replace with complex code",
			input:    `{"id":"call_xyz","tool":"str_replace","args":{"path":"test.go","old_str":"if err != nil {\n\treturn err\n}","new_str":"if err != nil {\n\treturn fmt.Errorf(\"failed: %w\", err)\n}"}}`,
			wantLen:  1,
			wantTool: "str_replace",
		},
		{
			name:     "str_replace with escaped quotes",
			input:    `{"id":"call_123","tool":"str_replace","args":{"path":"main.go","old_str":"fmt.Println(\"hello\")","new_str":"fmt.Println(\"world\")"}}`,
			wantLen:  1,
			wantTool: "str_replace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseToolCalls(tt.input)
			if len(result) != tt.wantLen {
				t.Errorf("ParseToolCalls() returned %d calls, want %d", len(result), tt.wantLen)
				return
			}
			if tt.wantLen > 0 && result[0].Tool != tt.wantTool {
				t.Errorf("ParseToolCalls()[0].Tool = %q, want %q", result[0].Tool, tt.wantTool)
			}
		})
	}
}

func TestParseToolCalls_MultipleTool(t *testing.T) {
	input := `First I'll read the file:
{"tool": "read_file", "args": {"path": "main.go"}}

Then I'll search for a pattern:
{"tool": "search_code", "args": {"pattern": "func main", "path": "."}}

Finally done.`

	result := ParseToolCalls(input)
	if len(result) != 2 {
		t.Errorf("ParseToolCalls() returned %d calls, want 2", len(result))
		return
	}
	if result[0].Tool != "read_file" {
		t.Errorf("First tool = %q, want 'read_file'", result[0].Tool)
	}
	if result[1].Tool != "search_code" {
		t.Errorf("Second tool = %q, want 'search_code'", result[1].Tool)
	}
}

func TestParseToolCalls_NoToolCalls(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "plain text",
			input: "This is just a regular response without any tool calls.",
		},
		{
			name:  "json but not tool call",
			input: `{"message": "hello", "status": "ok"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseToolCalls(tt.input)
			if len(result) != 0 {
				t.Errorf("ParseToolCalls() returned %d calls, want 0", len(result))
			}
		})
	}
}

func TestParseToolCalls_InMarkdownCodeBlock(t *testing.T) {
	// Tool calls inside markdown code blocks should be ignored
	input := "Here is an example:\n```json\n{\"tool\": \"read_file\", \"args\": {\"path\": \"test.go\"}}\n```\nThis is just an example."

	result := ParseToolCalls(input)
	if len(result) != 0 {
		t.Errorf("ParseToolCalls() returned %d calls from markdown block, want 0", len(result))
	}
}

func TestParseToolCall_SingleCall(t *testing.T) {
	input := `{"tool": "bash", "args": {"command": "ls -la"}}`
	result := ParseToolCall(input)

	if result == nil {
		t.Fatal("ParseToolCall() returned nil")
	}
	if result.Tool != "bash" {
		t.Errorf("Tool = %q, want 'bash'", result.Tool)
	}
	if result.Args["command"] != "ls -la" {
		t.Errorf("Args[command] = %q, want 'ls -la'", result.Args["command"])
	}
}

func TestParseToolCall_NoCall(t *testing.T) {
	input := "Just a regular message without any tool calls"
	result := ParseToolCall(input)

	if result != nil {
		t.Errorf("ParseToolCall() = %v, want nil", result)
	}
}
