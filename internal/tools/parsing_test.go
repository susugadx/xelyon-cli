package tools

import (
	"encoding/json"
	"strings"
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
		// grep_replace with regex patterns
		{
			name:     "grep_replace with brace pattern",
			input:    `{"id":"call_grep","tool":"grep_replace","args":{"pattern":"func\\s+\\w+\\s*\\(","replacement":"func newName(","path":"."}}`,
			wantLen:  1,
			wantTool: "grep_replace",
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

// xmlTestTool はXML rescueテスト用のダミーツール
type xmlTestTool struct {
	name string
}

func (t *xmlTestTool) Name() string                       { return t.name }
func (t *xmlTestTool) Description() string                { return "test tool" }
func (t *xmlTestTool) Parameters() map[string]interface{} { return nil }
func (t *xmlTestTool) Run(args map[string]string) (string, *FileChange, error) {
	return "", nil, nil
}

// registerXMLTestTools はXML rescueテスト用にDefaultRegistryにダミーツールを登録する
func registerXMLTestTools(t *testing.T) {
	t.Helper()
	for _, name := range []string{"read_file", "search_code", "bash", "write_file", "str_replace"} {
		if !DefaultRegistry.HasTool(name) {
			DefaultRegistry.Register(&xmlTestTool{name: name})
		}
	}
}

func TestParseToolCalls_XMLRescue_WithArgsWrapper(t *testing.T) {
	registerXMLTestTools(t)

	input := `Let me read the file for you.

<read_file>
<args>
  <path>main.go</path>
</args>
</read_file>

Done.`

	result := ParseToolCalls(input)
	if len(result) != 1 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 1", len(result))
	}
	if result[0].Tool != "read_file" {
		t.Errorf("Tool = %q, want 'read_file'", result[0].Tool)
	}
	if result[0].Args["path"] != "main.go" {
		t.Errorf("Args[path] = %q, want 'main.go'", result[0].Args["path"])
	}
}

func TestParseToolCalls_XMLRescue_WithoutArgsWrapper(t *testing.T) {
	registerXMLTestTools(t)

	input := `I'll search for the pattern.

<search_code>
  <pattern>func main</pattern>
  <path>.</path>
</search_code>`

	result := ParseToolCalls(input)
	if len(result) != 1 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 1", len(result))
	}
	if result[0].Tool != "search_code" {
		t.Errorf("Tool = %q, want 'search_code'", result[0].Tool)
	}
	if result[0].Args["pattern"] != "func main" {
		t.Errorf("Args[pattern] = %q, want 'func main'", result[0].Args["pattern"])
	}
	if result[0].Args["path"] != "." {
		t.Errorf("Args[path] = %q, want '.'", result[0].Args["path"])
	}
}

func TestParseToolCalls_XMLRescue_MultipleToolCalls(t *testing.T) {
	registerXMLTestTools(t)

	input := `First read, then search.

<read_file>
<path>main.go</path>
</read_file>

<search_code>
<pattern>TODO</pattern>
<path>.</path>
</search_code>`

	result := ParseToolCalls(input)
	if len(result) != 2 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 2", len(result))
	}
	if result[0].Tool != "read_file" {
		t.Errorf("First tool = %q, want 'read_file'", result[0].Tool)
	}
	if result[1].Tool != "search_code" {
		t.Errorf("Second tool = %q, want 'search_code'", result[1].Tool)
	}
}

func TestParseToolCalls_XMLRescue_UnknownToolIgnored(t *testing.T) {
	registerXMLTestTools(t)

	// "unknown_tool" はDefaultRegistryに未登録なのでスキップされる
	input := `<unknown_tool>
<param1>value1</param1>
</unknown_tool>`

	result := ParseToolCalls(input)
	if len(result) != 0 {
		t.Errorf("ParseToolCalls() returned %d calls, want 0 (unknown tool)", len(result))
	}
}

func TestParseToolCalls_XMLRescue_InCodeBlockIgnored(t *testing.T) {
	registerXMLTestTools(t)

	input := "Example:\n```\n<read_file>\n<path>test.go</path>\n</read_file>\n```"

	result := ParseToolCalls(input)
	if len(result) != 0 {
		t.Errorf("ParseToolCalls() returned %d calls, want 0 (in code block)", len(result))
	}
}

func TestParseToolCalls_XMLRescue_JSONTakesPriority(t *testing.T) {
	registerXMLTestTools(t)

	// JSONが見つかればXML rescueは発動しない
	input := `{"tool": "read_file", "args": {"path": "main.go"}}

<bash>
<command>ls</command>
</bash>`

	result := ParseToolCalls(input)
	if len(result) != 1 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 1 (JSON only)", len(result))
	}
	if result[0].Tool != "read_file" {
		t.Errorf("Tool = %q, want 'read_file' (JSON takes priority)", result[0].Tool)
	}
}

func TestParseToolCalls_XMLRescue_BashCommand(t *testing.T) {
	registerXMLTestTools(t)

	input := `<bash>
<args>
  <command>go test ./...</command>
</args>
</bash>`

	result := ParseToolCalls(input)
	if len(result) != 1 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 1", len(result))
	}
	if result[0].Tool != "bash" {
		t.Errorf("Tool = %q, want 'bash'", result[0].Tool)
	}
	if result[0].Args["command"] != "go test ./..." {
		t.Errorf("Args[command] = %q, want 'go test ./...'", result[0].Args["command"])
	}
}

func TestParseToolCalls_XMLRescue_JSONInsideXMLTags(t *testing.T) {
	registerXMLTestTools(t)

	input := `<read_file>
{"path": "main.go"}
</read_file>`

	result := ParseToolCalls(input)
	if len(result) != 1 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 1", len(result))
	}
	if result[0].Tool != "read_file" {
		t.Errorf("Tool = %q, want 'read_file'", result[0].Tool)
	}
	if result[0].Args["path"] != "main.go" {
		t.Errorf("Args[path] = %q, want 'main.go'", result[0].Args["path"])
	}
}

func TestParseToolCalls_XMLRescue_BashJSONInsideXMLTags(t *testing.T) {
	registerXMLTestTools(t)

	input := `<bash>
{"command": "cat main.go"}
</bash>`

	result := ParseToolCalls(input)
	if len(result) != 1 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 1", len(result))
	}
	if result[0].Tool != "bash" {
		t.Errorf("Tool = %q, want 'bash'", result[0].Tool)
	}
	if result[0].Args["command"] != "cat main.go" {
		t.Errorf("Args[command] = %q, want 'cat main.go'", result[0].Args["command"])
	}
}

func TestParseXMLParams(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    map[string]string
	}{
		{
			name:    "with args wrapper",
			content: "<args>\n  <path>main.go</path>\n</args>",
			want:    map[string]string{"path": "main.go"},
		},
		{
			name:    "without args wrapper",
			content: "<path>main.go</path>\n<pattern>func main</pattern>",
			want:    map[string]string{"path": "main.go", "pattern": "func main"},
		},
		{
			name:    "single param",
			content: "<command>ls -la</command>",
			want:    map[string]string{"command": "ls -la"},
		},
		{
			name:    "empty content",
			content: "",
			want:    map[string]string{},
		},
		{
			name:    "JSON inside XML",
			content: `{"path": "main.go"}`,
			want:    map[string]string{"path": "main.go"},
		},
		{
			name:    "JSON with multiple keys",
			content: `{"pattern": "func main", "path": "."}`,
			want:    map[string]string{"pattern": "func main", "path": "."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseXMLParams(tt.content)
			if len(got) != len(tt.want) {
				t.Errorf("parseXMLParams() returned %d params, want %d: got=%v", len(got), len(tt.want), got)
				return
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("parseXMLParams()[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

// --- repairJSONStringValues tests ---

func TestRepairJSONStringValues_NormalJSON(t *testing.T) {
	// 正常な JSON はそのまま通る
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "read_file",
			input: `{"tool": "read_file", "args": {"path": "main.go"}}`,
		},
		{
			name:  "bash",
			input: `{"tool": "bash", "args": {"command": "go build"}}`,
		},
		{
			name:  "str_replace with escaped newlines",
			input: `{"tool": "str_replace", "args": {"path": "main.go", "old_str": "line1\nline2", "new_str": "line1\nline2\nline3"}}`,
		},
		{
			name:  "str_replace with escaped tabs and quotes",
			input: `{"tool": "str_replace", "args": {"path": "main.go", "old_str": "\tfmt.Println(\"hello\")", "new_str": "\tfmt.Println(\"world\")"}}`,
		},
		{
			name:  "empty string values",
			input: `{"tool": "str_replace", "args": {"path": "a.go", "old_str": "", "new_str": ""}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := repairJSONStringValues(tt.input)
			if result != tt.input {
				t.Errorf("repairJSONStringValues() modified normal JSON:\n  input:  %q\n  result: %q", tt.input, result)
			}
		})
	}
}

func TestRepairJSONStringValues_RawNewlines(t *testing.T) {
	// old_str に生改行を含む str_replace JSON → 修復されてパース成功
	input := "{\"tool\": \"str_replace\", \"args\": {\"path\": \"main.go\", \"old_str\": \"func main() {\n\treturn\n}\", \"new_str\": \"func main() {\n\treturn nil\n}\"}}"
	// ↑ 実際の改行文字 (0x0A) と実際のタブ文字 (0x09) が含まれる

	repaired := repairJSONStringValues(input)

	// 修復後は json.Unmarshal が成功するはず
	var tc ToolCall
	if err := json.Unmarshal([]byte(repaired), &tc); err != nil {
		t.Fatalf("json.Unmarshal failed after repair: %v\nrepaired: %s", err, repaired)
	}

	if tc.Tool != "str_replace" {
		t.Errorf("Tool = %q, want 'str_replace'", tc.Tool)
	}

	tc.NormalizeArgs()
	if tc.Args["path"] != "main.go" {
		t.Errorf("Args[path] = %q, want 'main.go'", tc.Args["path"])
	}

	// old_str は修復後に改行とタブを含むはず
	oldStr := tc.Args["old_str"]
	if !strings.Contains(oldStr, "\n") {
		t.Errorf("Args[old_str] should contain newlines, got: %q", oldStr)
	}
	if !strings.Contains(oldStr, "\t") {
		t.Errorf("Args[old_str] should contain tabs, got: %q", oldStr)
	}
}

func TestRepairJSONStringValues_GoCodeRealistic(t *testing.T) {
	// 実際の Go コードを含む str_replace JSON（Gemini FC rescue の典型パターン）
	// 生改行 + 生タブ + エスケープ済み引用符
	input := "{\"tool\": \"str_replace\", \"args\": {\"path\": \"handler.go\", \"old_str\": \"func handleRequest(w http.ResponseWriter, r *http.Request) {\n\tif r.Method != \\\"GET\\\" {\n\t\thttp.Error(w, \\\"Method not allowed\\\", 405)\n\t\treturn\n\t}\n}\", \"new_str\": \"func handleRequest(w http.ResponseWriter, r *http.Request) {\n\tswitch r.Method {\n\tcase \\\"GET\\\":\n\t\thandleGet(w, r)\n\tdefault:\n\t\thttp.Error(w, \\\"Method not allowed\\\", 405)\n\t}\n}\"}}"

	repaired := repairJSONStringValues(input)

	var tc ToolCall
	if err := json.Unmarshal([]byte(repaired), &tc); err != nil {
		t.Fatalf("json.Unmarshal failed after repair: %v\nrepaired: %s", err, repaired)
	}

	if tc.Tool != "str_replace" {
		t.Errorf("Tool = %q, want 'str_replace'", tc.Tool)
	}

	tc.NormalizeArgs()
	if tc.Args["path"] != "handler.go" {
		t.Errorf("Args[path] = %q, want 'handler.go'", tc.Args["path"])
	}

	// old_str に switch/case パターンのコードが含まれるはず
	oldStr := tc.Args["old_str"]
	if !strings.Contains(oldStr, "handleRequest") {
		t.Errorf("Args[old_str] should contain 'handleRequest', got: %q", oldStr)
	}
	if !strings.Contains(oldStr, `"GET"`) {
		t.Errorf("Args[old_str] should contain unescaped quotes from code, got: %q", oldStr)
	}
}

func TestRepairJSONStringValues_NoDoubleEscape(t *testing.T) {
	// 既にエスケープ済みの JSON → 二重エスケープされない
	input := `{"tool": "str_replace", "args": {"path": "a.go", "old_str": "line1\nline2\ttab\r\nwindows", "new_str": "replaced"}}`

	repaired := repairJSONStringValues(input)

	if repaired != input {
		t.Errorf("repairJSONStringValues() should not modify already-escaped JSON:\n  input:  %q\n  result: %q", input, repaired)
	}

	// パースも成功するはず
	var tc ToolCall
	if err := json.Unmarshal([]byte(repaired), &tc); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	tc.NormalizeArgs()
	if !strings.Contains(tc.Args["old_str"], "\n") {
		t.Error("Args[old_str] should contain newline after parse")
	}
}

func TestRepairJSONStringValues_CarriageReturn(t *testing.T) {
	// \r\n (Windows 改行) が生で含まれるケース
	input := "{\"tool\": \"bash\", \"args\": {\"command\": \"echo hello\r\necho world\"}}"

	repaired := repairJSONStringValues(input)

	var tc ToolCall
	if err := json.Unmarshal([]byte(repaired), &tc); err != nil {
		t.Fatalf("json.Unmarshal failed after repair: %v\nrepaired: %s", err, repaired)
	}
	tc.NormalizeArgs()
	if tc.Args["command"] != "echo hello\r\necho world" {
		t.Errorf("Args[command] = %q, want 'echo hello\\r\\necho world'", tc.Args["command"])
	}
}

func TestRepairJSONStringValues_OtherControlChars(t *testing.T) {
	// その他の制御文字 (0x01 等) → \uXXXX にエスケープ
	input := "{\"tool\": \"bash\", \"args\": {\"command\": \"test\x01value\"}}"

	repaired := repairJSONStringValues(input)

	var tc ToolCall
	if err := json.Unmarshal([]byte(repaired), &tc); err != nil {
		t.Fatalf("json.Unmarshal failed after repair: %v\nrepaired: %s", err, repaired)
	}
	tc.NormalizeArgs()
	if tc.Args["command"] != "test\x01value" {
		t.Errorf("Args[command] = %q, want 'test\\x01value'", tc.Args["command"])
	}
}

// --- ParseToolCalls with JSON repair integration tests ---

func TestParseToolCalls_RepairStrReplace_RawNewlines(t *testing.T) {
	// FC rescue パターン: str_replace の old_str/new_str に生改行が含まれる
	input := "I'll fix the code.\n{\"tool\": \"str_replace\", \"args\": {\"path\": \"main.go\", \"old_str\": \"func main() {\n\tfmt.Println(\\\"hello\\\")\n}\", \"new_str\": \"func main() {\n\tfmt.Println(\\\"world\\\")\n}\"}}"

	result := ParseToolCalls(input)
	if len(result) != 1 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 1", len(result))
	}
	if result[0].Tool != "str_replace" {
		t.Errorf("Tool = %q, want 'str_replace'", result[0].Tool)
	}
	if result[0].Args["path"] != "main.go" {
		t.Errorf("Args[path] = %q, want 'main.go'", result[0].Args["path"])
	}
	// old_str にコード内容が含まれるか確認
	if !strings.Contains(result[0].Args["old_str"], "Println") {
		t.Errorf("Args[old_str] should contain 'Println', got: %q", result[0].Args["old_str"])
	}
}

func TestParseToolCalls_RepairStrReplace_WithReadFile(t *testing.T) {
	// FC rescue パターン: read_file（正常JSON）+ str_replace（生改行あり）の組み合わせ
	input := "{\"tool\": \"read_file\", \"args\": {\"path\": \"main.go\"}}\n{\"tool\": \"str_replace\", \"args\": {\"path\": \"main.go\", \"old_str\": \"old\ncode\", \"new_str\": \"new\ncode\"}}"

	result := ParseToolCalls(input)
	if len(result) != 2 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 2", len(result))
	}
	if result[0].Tool != "read_file" {
		t.Errorf("First tool = %q, want 'read_file'", result[0].Tool)
	}
	if result[1].Tool != "str_replace" {
		t.Errorf("Second tool = %q, want 'str_replace'", result[1].Tool)
	}
}

func TestParseToolCalls_RepairNormalJSONUnchanged(t *testing.T) {
	// 正常な JSON は修復なしでそのまま通る
	input := `{"id": "call_001", "tool": "str_replace", "args": {"path": "main.go", "old_str": "line1\nline2", "new_str": "replaced"}}`

	result := ParseToolCalls(input)
	if len(result) != 1 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 1", len(result))
	}
	if result[0].Tool != "str_replace" {
		t.Errorf("Tool = %q, want 'str_replace'", result[0].Tool)
	}
	if result[0].ID != "call_001" {
		t.Errorf("ID = %q, want 'call_001'", result[0].ID)
	}
}
