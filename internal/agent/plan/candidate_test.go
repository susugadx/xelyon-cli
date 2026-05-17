package plan

import "testing"

func TestIsToolCallJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected bool
	}{
		{
			name:     "tool call",
			json:     `{"tool": "read_file", "path": "main.go"}`,
			expected: true,
		},
		{
			name:     "tool with space",
			json:     `{ "tool": "write_file" }`,
			expected: true,
		},
		{
			name:     "malformed tool call",
			json:     `{"tool": invalid}`,
			expected: true,
		},
		{
			name:     "plan JSON",
			json:     `{"summary": "test", "steps": []}`,
			expected: false,
		},
		{
			name:     "tool string value is not tool call",
			json:     `{"summary": "tool", "steps": []}`,
			expected: false,
		},
		{
			name:     "nested tool key is not tool call",
			json:     `{"plan":{"summary":"test","steps":[{"id":1,"description":"Step","tool":"read_file"}]}}`,
			expected: false,
		},
		{
			name:     "toolbox key is not tool call",
			json:     `{"toolbox": "read_file"}`,
			expected: false,
		},
		{
			name:     "empty object",
			json:     `{}`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isToolCallJSON(tt.json)
			if result != tt.expected {
				t.Errorf("isToolCallJSON(%q) = %v, want %v", tt.json, result, tt.expected)
			}
		})
	}
}
