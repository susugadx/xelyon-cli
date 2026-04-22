package tools

import "testing"

func TestFindBalancedJSONObjectEnd(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		start    int
		wantEnd  int
		wantFail bool
	}{
		{
			name:    "simple object",
			input:   `{"tool":"bash"} tail`,
			start:   0,
			wantEnd: 15,
		},
		{
			name:    "nested object with braces in string",
			input:   `prefix {"args":{"body":"{\"k\":\"v\"}"}} suffix`,
			start:   7,
			wantEnd: 40,
		},
		{
			name:     "incomplete object",
			input:    `{"tool":"bash"`,
			start:    0,
			wantFail: true,
		},
		{
			name:    "escaped quote inside string",
			input:   `{"text":"a\"b"}x`,
			start:   0,
			wantEnd: 15,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findBalancedJSONObjectEnd(tt.input, tt.start)
			if tt.wantFail {
				if got != -1 {
					t.Fatalf("findBalancedJSONObjectEnd() = %d, want -1", got)
				}
				return
			}
			if got != tt.wantEnd {
				t.Fatalf("findBalancedJSONObjectEnd() = %d, want %d", got, tt.wantEnd)
			}
		})
	}
}
