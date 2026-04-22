package tools

import "testing"

func TestTruncateDebug(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "exact length",
			input:  "0123456789",
			maxLen: 10,
			want:   "0123456789",
		},
		{
			name:   "long string truncated",
			input:  "this is a very long string that should be truncated",
			maxLen: 20,
			want:   "this is a very long ...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "single character",
			input:  "a",
			maxLen: 10,
			want:   "a",
		},
		{
			name:   "maxLen exactly at string length",
			input:  "exact",
			maxLen: 5,
			want:   "exact",
		},
		{
			name:   "long string short maxLen",
			input:  "abcdefghijklmnop",
			maxLen: 5,
			want:   "abcde...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateDebug(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateDebug(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}
