package tools

import "testing"

func TestParseToolCalls_MultipleTool(t *testing.T) {
	input := `First I'll read the file:
{"tool": "read_file", "args": {"path": "main.go"}}

Then I'll search for a pattern:
{"tool": "bash", "args": {"command": "grep -rn 'func main' ."}}

Finally done.`

	result := ParseToolCalls(input)
	if len(result) != 2 {
		t.Errorf("ParseToolCalls() returned %d calls, want 2", len(result))
		return
	}
	if result[0].Tool != "read_file" {
		t.Errorf("First tool = %q, want 'read_file'", result[0].Tool)
	}
	if result[1].Tool != "bash" {
		t.Errorf("Second tool = %q, want 'bash'", result[1].Tool)
	}
}
