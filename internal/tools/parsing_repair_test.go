package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

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
