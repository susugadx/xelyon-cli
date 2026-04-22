package tools

import "testing"

func TestParseToolCalls_IncompleteJSON_UsesXMLRescueWhenNoJSONParsed(t *testing.T) {
	// JSON が不完全で抽出できない場合でも、JSON 0件なら XML rescue を試す。
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
	// 先頭 JSON を1件取れた時点で XML rescue は発動しない（既存契約）。
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

func TestParseToolCalls_XMLRescue_JSONTakesPriority(t *testing.T) {
	// JSON が見つかれば XML rescue は発動しない。
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
