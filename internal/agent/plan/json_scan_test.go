package plan

import "testing"

func TestFindClosingBrace(t *testing.T) {
	tests := []struct {
		name     string
		response string
		start    int
		expected int
	}{
		{
			name:     "simple object",
			response: `{"key": "value"}`,
			start:    0,
			expected: 16,
		},
		{
			name:     "nested object",
			response: `{"outer": {"inner": "value"}}`,
			start:    0,
			expected: 29,
		},
		{
			name:     "with escaped quotes",
			response: `{"key": "value with \"quotes\""}`,
			start:    0,
			expected: 32,
		},
		{
			name:     "unclosed",
			response: `{"key": "value"`,
			start:    0,
			expected: -1,
		},
		{
			name:     "brace in string",
			response: `{"key": "{}"}`,
			start:    0,
			expected: 13,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findClosingBrace(tt.response, tt.start)
			if result != tt.expected {
				t.Errorf("findClosingBrace(%q, %d) = %d, want %d", tt.response, tt.start, result, tt.expected)
			}
		})
	}
}
