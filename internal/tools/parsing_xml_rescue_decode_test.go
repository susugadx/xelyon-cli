package tools

import "testing"

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
