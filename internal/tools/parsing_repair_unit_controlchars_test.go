package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

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
