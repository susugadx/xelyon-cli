package tools

import (
	"testing"
)

func TestDecodeToolCallJSON_EmptyToolSkipped(t *testing.T) {
	got, ok := decodeToolCallJSON(`{"tool":"","args":{"path":"main.go"}}`, newParseDebugLogger(false, nil))
	if ok {
		t.Fatalf("decodeToolCallJSON() ok = true, want false (got=%+v)", got)
	}
}

func TestDecodeToolCallJSON_RepairRawNewline(t *testing.T) {
	input := "{\"tool\":\"str_replace\",\"args\":{\"path\":\"main.go\",\"old_str\":\"line1\nline2\",\"new_str\":\"line1\nline3\"}}"
	got, ok := decodeToolCallJSON(input, newParseDebugLogger(false, nil))
	if !ok {
		t.Fatal("decodeToolCallJSON() ok = false, want true")
	}
	if got.Tool != "str_replace" {
		t.Fatalf("Tool = %q, want 'str_replace'", got.Tool)
	}
	if got.Args["path"] != "main.go" {
		t.Fatalf("Args[path] = %q, want 'main.go'", got.Args["path"])
	}
}
