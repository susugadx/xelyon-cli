package tools

import "testing"

func TestParseToolCalls_NoToolCalls(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "plain text", input: "This is just a regular response without any tool calls."},
		{name: "json but not tool call", input: `{"message": "hello", "status": "ok"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseToolCalls(tt.input)
			if len(result) != 0 {
				t.Errorf("ParseToolCalls() returned %d calls, want 0", len(result))
			}
		})
	}
}
