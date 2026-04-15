package agent

import "testing"

func TestExtractExplanationAndTool_NoToolCall(t *testing.T) {
	response := "This is just a plain text response with no tool calls."

	explanation, toolJSON := extractExplanationAndTool(response)
	if explanation != response {
		t.Errorf("extractExplanationAndTool() explanation = %q, want original response", explanation)
	}
	if toolJSON != "" {
		t.Errorf("extractExplanationAndTool() toolJSON = %q, want empty string", toolJSON)
	}
}

func TestExtractExplanationAndTool_OnlyToolCall(t *testing.T) {
	response := `{"tool": "read_file", "args": {"paths": ["/test.txt"]}}`

	explanation, toolJSON := extractExplanationAndTool(response)
	if explanation != "" {
		t.Errorf("extractExplanationAndTool() explanation = %q, want empty string", explanation)
	}
	if toolJSON != response {
		t.Errorf("extractExplanationAndTool() toolJSON = %q, want %q", toolJSON, response)
	}
}

func TestExtractExplanationAndTool_BothParts(t *testing.T) {
	response := `I'll read the file for you.

{"tool": "read_file", "args": {"paths": ["/test.txt"]}}`

	explanation, toolJSON := extractExplanationAndTool(response)
	expectedExplanation := "I'll read the file for you."
	if explanation != expectedExplanation {
		t.Errorf("extractExplanationAndTool() explanation = %q, want %q", explanation, expectedExplanation)
	}

	expectedToolJSON := `{"tool": "read_file", "args": {"paths": ["/test.txt"]}}`
	if toolJSON != expectedToolJSON {
		t.Errorf("extractExplanationAndTool() toolJSON = %q, want %q", toolJSON, expectedToolJSON)
	}
}

func TestExtractExplanationAndTool_NestedJSON(t *testing.T) {
	response := `Here's the file operation:

{"tool": "write_file", "args": {"path": "/config.json", "content": "{\"key\": \"value\"}"}}`

	explanation, toolJSON := extractExplanationAndTool(response)
	if explanation != "Here's the file operation:" {
		t.Errorf("extractExplanationAndTool() explanation = %q, want 'Here's the file operation:'", explanation)
	}

	expectedToolJSON := `{"tool": "write_file", "args": {"path": "/config.json", "content": "{\"key\": \"value\"}"}}`
	if toolJSON != expectedToolJSON {
		t.Errorf("extractExplanationAndTool() toolJSON = %q, want %q", toolJSON, expectedToolJSON)
	}
}

func TestExtractExplanationAndTool_SpacedToolPattern(t *testing.T) {
	response := `Explanation text

{ "tool": "bash", "args": {"command": "ls -la"}}`

	explanation, toolJSON := extractExplanationAndTool(response)
	if explanation != "Explanation text" {
		t.Errorf("extractExplanationAndTool() explanation = %q, want 'Explanation text'", explanation)
	}
	if toolJSON == "" {
		t.Error("extractExplanationAndTool() should detect spaced tool pattern")
	}
}

func TestExtractExplanationAndTool_EscapedQuotes(t *testing.T) {
	response := `{"tool": "str_replace", "args": {"path": "/test.txt", "old_str": "say \"hello\"", "new_str": "say \"goodbye\""}}`

	explanation, toolJSON := extractExplanationAndTool(response)
	if explanation != "" {
		t.Errorf("extractExplanationAndTool() explanation = %q, want empty string", explanation)
	}
	if toolJSON != response {
		t.Errorf("extractExplanationAndTool() toolJSON = %q, want %q", toolJSON, response)
	}
}

func TestExtractExplanationAndTool_MultipleJSONObjects(t *testing.T) {
	response := `{"tool": "read_file", "args": {"paths": ["/a.txt"]}}
{"tool": "read_file", "args": {"paths": ["/b.txt"]}}`

	_, toolJSON := extractExplanationAndTool(response)
	expectedFirst := `{"tool": "read_file", "args": {"paths": ["/a.txt"]}}`
	if toolJSON != expectedFirst {
		t.Errorf("extractExplanationAndTool() should extract first tool call only, got %q", toolJSON)
	}
}

func TestExtractExplanationAndTool_UnclosedBrace(t *testing.T) {
	response := `Explanation

{"tool": "bash", "args": {"command": "echo 'test'"`

	explanation, toolJSON := extractExplanationAndTool(response)
	if explanation != "Explanation" {
		t.Errorf("extractExplanationAndTool() explanation = %q, want 'Explanation'", explanation)
	}
	if toolJSON == "" {
		t.Error("extractExplanationAndTool() should return partial JSON when unclosed")
	}
}
