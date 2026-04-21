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
		// write_file with code content
		{
			name:     "write_file with Go code",
			input:    `{"id":"call_write","tool":"write_file","args":{"path":"main.go","content":"package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}"}}`,
			wantLen:  1,
			wantTool: "write_file",
		},
		{
			name:     "write_file with nested braces",
			input:    `{"id":"call_w2","tool":"write_file","args":{"path":"test.go","content":"type Config struct {\n\tData map[string]interface{}\n}"}}`,
			wantLen:  1,
			wantTool: "write_file",
		},
		// bash with complex commands
		{
			name:     "bash with echo containing braces",
			input:    `{"id":"call_bash","tool":"bash","args":{"command":"echo \"func foo() { return 1 }\""}}`,
			wantLen:  1,
			wantTool: "bash",
		},
		{
			name:     "bash with JSON output",
			input:    `{"id":"call_b2","tool":"bash","args":{"command":"curl -s api.example.com | jq '{\"key\": .value}'"}}`,
			wantLen:  1,
			wantTool: "bash",
		},
		// ast_grep with code pattern
		{
			name:     "ast_grep with struct pattern",
			input:    `{"id":"call_ast","tool":"ast_grep","args":{"pattern":"type $NAME struct { $$$FIELDS }","lang":"go","path":"."}}`,
			wantLen:  1,
			wantTool: "ast_grep",
		},
		// http_request with JSON body
		{
			name:     "http_request with JSON body",
			input:    `{"id":"call_http","tool":"http_request","args":{"method":"POST","url":"https://api.example.com","body":"{\"nested\":{\"key\":\"value\"}}"}}`,
			wantLen:  1,
			wantTool: "http_request",
		},
		{
			name:     "http_request with headers JSON",
			input:    `{"id":"call_h2","tool":"http_request","args":{"method":"GET","url":"https://api.example.com","headers":"{\"Authorization\":\"Bearer token\"}"}}`,
			wantLen:  1,
			wantTool: "http_request",
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
{"tool": "bash", "args": {"command": "grep -rn 'func main' ."}}

Finally done.`

	result := ParseToolCalls(input)
	if len(result) != 2 {
		t.Errorf("ParseToolCalls() returned %d calls, want 2", len(result))
		return
	}
	if result[0].Tool != "read_file" {
		t.Errorf("First tool = %q, want 'read_file'", result[0].Tool)
	}
	if result[1].Tool != "bash" {
		t.Errorf("Second tool = %q, want 'bash'", result[1].Tool)
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

func TestParseToolCalls_CodeBlockWithRealToolCall(t *testing.T) {
	// Geminiパターン: コードブロック内のJSONは無視し、実際のツール呼び出しのみ抽出
	input := "I'll run the command:\n```json\n{\"tool\": \"bash\", \"args\": {\"command\": \"make ci-check\"}}\n```\n\n{\"tool\": \"bash\", \"args\": {\"command\": \"make ci-check\"}}"

	result := ParseToolCalls(input)
	if len(result) != 1 {
		t.Errorf("ParseToolCalls() returned %d calls, want 1 (only the real one)", len(result))
	}
	if len(result) > 0 && result[0].Tool != "bash" {
		t.Errorf("Tool = %q, want 'bash'", result[0].Tool)
	}
}

func TestParseToolCalls_OnlyCodeBlock(t *testing.T) {
	// ツール呼び出しがコードブロック内にのみある場合 → 0件
	input := "Here's what I would run:\n```\n{\"tool\": \"bash\", \"args\": {\"command\": \"go test ./...\"}}\n```\nTask completed!"

	result := ParseToolCalls(input)
	if len(result) != 0 {
		t.Errorf("ParseToolCalls() returned %d calls, want 0 (all in code block)", len(result))
	}
}

func TestParseToolCalls_MultipleCodeBlocks(t *testing.T) {
	// 複数のコードブロックがあり、間に実際のツール呼び出しがある
	input := "Example 1:\n```json\n{\"tool\": \"read_file\", \"args\": {\"path\": \"a.go\"}}\n```\n\n{\"tool\": \"bash\", \"args\": {\"command\": \"ls\"}}\n\nExample 2:\n```\n{\"tool\": \"write_file\", \"args\": {\"path\": \"b.go\", \"content\": \"test\"}}\n```"

	result := ParseToolCalls(input)
	if len(result) != 1 {
		t.Errorf("ParseToolCalls() returned %d calls, want 1", len(result))
	}
	if len(result) > 0 && result[0].Tool != "bash" {
		t.Errorf("Tool = %q, want 'bash'", result[0].Tool)
	}
}

func TestParseToolCalls_IncompleteJSONInCodeBlock_DoesNotBlockOutsideJSON(t *testing.T) {
	// コードブロック内の不完全 JSON は除外され、外側の有効 JSON は抽出される
	input := "```json\n{\"tool\": \"write_file\", \"args\": {\"path\": \"a.go\"\n```\n\n{\"tool\": \"bash\", \"args\": {\"command\": \"echo ok\"}}"

	result := ParseToolCalls(input)
	if len(result) != 1 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 1", len(result))
	}
	if result[0].Tool != "bash" {
		t.Errorf("Tool = %q, want 'bash'", result[0].Tool)
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
