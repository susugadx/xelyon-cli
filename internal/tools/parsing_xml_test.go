package tools

import (
	"io"
	"testing"
)

func TestParseToolCalls_IncompleteJSON_UsesXMLRescueWhenNoJSONParsed(t *testing.T) {
	// JSON が不完全で抽出できない場合でも、JSON 0件なら XML rescue を試す
	input := `{"tool": "read_file", "args": {"path": "main.go"}

<list_dir>
  <path>.</path>
</list_dir>`

	result := parseToolCallsForXMLTest(t, input)
	if len(result) != 1 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 1 (XML rescue)", len(result))
	}
	if result[0].Tool != "list_dir" {
		t.Errorf("Tool = %q, want 'list_dir'", result[0].Tool)
	}
	if result[0].Args["path"] != "." {
		t.Errorf("Args[path] = %q, want '.'", result[0].Args["path"])
	}
}

func TestParseToolCalls_IncompleteJSON_AfterValidJSONDoesNotRunXMLRescue(t *testing.T) {
	// 先頭 JSON を1件取れた時点で XML rescue は発動しない（既存契約）
	input := `{"tool": "read_file", "args": {"path": "main.go"}}
{"tool": "str_replace", "args": {"path": "main.go", "old_str": "old", "new_str": "new"

<list_dir>
  <path>.</path>
</list_dir>`

	result := parseToolCallsForXMLTest(t, input)
	if len(result) != 1 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 1 (JSON only)", len(result))
	}
	if result[0].Tool != "read_file" {
		t.Errorf("Tool = %q, want 'read_file'", result[0].Tool)
	}
}

// xmlTestTool はXML rescueテスト用のダミーツール
// Run を実行するテストではないため、実行結果は空で返す。
type xmlTestTool struct {
	name string
}

func (t *xmlTestTool) Name() string                       { return t.name }
func (t *xmlTestTool) Description() string                { return "test tool" }
func (t *xmlTestTool) Parameters() map[string]interface{} { return nil }
func (t *xmlTestTool) Run(_ ExecutionContext, args map[string]string) (string, *FileChange, error) {
	return "", nil, nil
}

func parseToolCallsForXMLTest(t *testing.T, input string) []*ToolCall {
	t.Helper()
	return ParseToolCallsWithRegistry(input, newXMLTestRegistry(t), io.Discard)
}

// newXMLTestRegistry はXML rescueテスト専用レジストリを返す。
// DefaultRegistry への副作用を避けるため clone を使う。
func newXMLTestRegistry(t *testing.T) *Registry {
	t.Helper()
	registry := DefaultRegistry.Clone()
	for _, name := range []string{"read_file", "list_dir", "bash", "write_file", "str_replace"} {
		if !registry.HasTool(name) {
			registry.Register(&xmlTestTool{name: name})
		}
	}
	return registry
}

func TestParseToolCalls_XMLRescue_WithArgsWrapper(t *testing.T) {
	input := `Let me read the file for you.

<read_file>
<args>
  <path>main.go</path>
</args>
</read_file>

Done.`

	result := parseToolCallsForXMLTest(t, input)
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
	input := `I'll list the directory.

<list_dir>
  <path>.</path>
</list_dir>`

	result := parseToolCallsForXMLTest(t, input)
	if len(result) != 1 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 1", len(result))
	}
	if result[0].Tool != "list_dir" {
		t.Errorf("Tool = %q, want 'list_dir'", result[0].Tool)
	}
	if result[0].Args["path"] != "." {
		t.Errorf("Args[path] = %q, want '.'", result[0].Args["path"])
	}
}

func TestParseToolCalls_XMLRescue_MultipleToolCalls(t *testing.T) {
	input := `First read, then list.

<read_file>
<path>main.go</path>
</read_file>

<list_dir>
<path>.</path>
</list_dir>`

	result := parseToolCallsForXMLTest(t, input)
	if len(result) != 2 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 2", len(result))
	}
	if result[0].Tool != "read_file" {
		t.Errorf("First tool = %q, want 'read_file'", result[0].Tool)
	}
	if result[1].Tool != "list_dir" {
		t.Errorf("Second tool = %q, want 'list_dir'", result[1].Tool)
	}
}

func TestParseToolCalls_XMLRescue_UnknownToolIgnored(t *testing.T) {
	// "unknown_tool" は registry に未登録なのでスキップされる
	input := `<unknown_tool>
<param1>value1</param1>
</unknown_tool>`

	result := parseToolCallsForXMLTest(t, input)
	if len(result) != 0 {
		t.Errorf("ParseToolCalls() returned %d calls, want 0 (unknown tool)", len(result))
	}
}

func TestParseToolCalls_XMLRescue_InCodeBlockIgnored(t *testing.T) {
	input := "Example:\n```\n<read_file>\n<path>test.go</path>\n</read_file>\n```"

	result := parseToolCallsForXMLTest(t, input)
	if len(result) != 0 {
		t.Errorf("ParseToolCalls() returned %d calls, want 0 (in code block)", len(result))
	}
}

func TestParseToolCalls_XMLRescue_JSONTakesPriority(t *testing.T) {
	// JSONが見つかればXML rescueは発動しない
	input := `{"tool": "read_file", "args": {"path": "main.go"}}

<bash>
<command>ls</command>
</bash>`

	result := parseToolCallsForXMLTest(t, input)
	if len(result) != 1 {
		t.Fatalf("ParseToolCalls() returned %d calls, want 1 (JSON only)", len(result))
	}
	if result[0].Tool != "read_file" {
		t.Errorf("Tool = %q, want 'read_file' (JSON takes priority)", result[0].Tool)
	}
}

func TestParseToolCalls_XMLRescue_BashCommand(t *testing.T) {
	input := `<bash>
<args>
  <command>go test ./...</command>
</args>
</bash>`

	result := parseToolCallsForXMLTest(t, input)
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
	input := `<read_file>
{"path": "main.go"}
</read_file>`

	result := parseToolCallsForXMLTest(t, input)
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
	input := `<bash>
{"command": "cat main.go"}
</bash>`

	result := parseToolCallsForXMLTest(t, input)
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
