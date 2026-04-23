package tools

import "testing"

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
		{
			name:     "ast_grep with struct pattern",
			input:    `{"id":"call_ast","tool":"ast_grep","args":{"pattern":"type $NAME struct { $$$FIELDS }","lang":"go","path":"."}}`,
			wantLen:  1,
			wantTool: "ast_grep",
		},
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
