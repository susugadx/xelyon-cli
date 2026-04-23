package tools

import "testing"

func TestParseToolCall_SingleCall(t *testing.T) {
	input := `{"tool": "bash", "args": {"command": "ls -la"}}`
	result := ParseToolCall(input)

	if result == nil {
		t.Fatal("ParseToolCall() returned nil")
	}
	if result.Tool != "bash" {
		t.Errorf("Tool = %q, want 'bash'", result.Tool)
	}
	if result.Args["command"] != "ls -la" {
		t.Errorf("Args[command] = %q, want 'ls -la'", result.Args["command"])
	}
}

func TestParseToolCall_NoCall(t *testing.T) {
	input := "Just a regular message without any tool calls"
	result := ParseToolCall(input)

	if result != nil {
		t.Errorf("ParseToolCall() = %v, want nil", result)
	}
}
