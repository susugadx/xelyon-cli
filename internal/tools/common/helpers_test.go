package common

import (
	"testing"
)

func TestTruncate(t *testing.T) {
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
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "truncate needed",
			input:  "hello world",
			maxLen: 8,
			want:   "hello wo...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 5,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("Truncate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeLeadingWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "tabs to spaces",
			input: "\tfunc main() {\n\t\treturn\n\t}",
			want:  "func main() {\nreturn\n}",
		},
		{
			name:  "leading spaces removed",
			input: "    line1\n        line2",
			want:  "line1\nline2",
		},
		{
			name:  "preserve internal spaces",
			input: "hello  world",
			want:  "hello  world",
		},
		{
			name:  "mixed tabs and spaces",
			input: "\t    hello\n\tworld",
			want:  "hello\nworld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeLeadingWhitespace(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeLeadingWhitespace() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMinMax(t *testing.T) {
	tests := []struct {
		a       int
		b       int
		wantMin int
		wantMax int
	}{
		{a: 5, b: 10, wantMin: 5, wantMax: 10},
		{a: 10, b: 5, wantMin: 5, wantMax: 10},
		{a: 0, b: 0, wantMin: 0, wantMax: 0},
		{a: -5, b: 5, wantMin: -5, wantMax: 5},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			gotMin := Min(tt.a, tt.b)
			gotMax := Max(tt.a, tt.b)

			if gotMin != tt.wantMin {
				t.Errorf("Min(%d, %d) = %d, want %d", tt.a, tt.b, gotMin, tt.wantMin)
			}
			if gotMax != tt.wantMax {
				t.Errorf("Max(%d, %d) = %d, want %d", tt.a, tt.b, gotMax, tt.wantMax)
			}
		})
	}
}
