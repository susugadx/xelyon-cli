package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

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
