package tools

import (
	"strings"
	"testing"
)

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
