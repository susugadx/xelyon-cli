package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

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
