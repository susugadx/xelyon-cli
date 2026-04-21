package tools

import "testing"

func TestFindCodeBlockRanges(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		wantRanges int
	}{
		{
			name:       "no code blocks",
			text:       "plain text without code blocks",
			wantRanges: 0,
		},
		{
			name:       "single code block",
			text:       "text ```code``` more",
			wantRanges: 1,
		},
		{
			name:       "two code blocks",
			text:       "```a``` middle ```b```",
			wantRanges: 2,
		},
		{
			name:       "unclosed code block",
			text:       "```start of code block",
			wantRanges: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findCodeBlockRanges(tt.text)
			if len(got) != tt.wantRanges {
				t.Errorf("findCodeBlockRanges() returned %d ranges, want %d", len(got), tt.wantRanges)
			}
		})
	}
}

func TestIsInCodeBlock(t *testing.T) {
	ranges := [][2]int{
		{5, 10},
		{20, 30},
	}

	tests := []struct {
		name string
		pos  int
		want bool
	}{
		{name: "before first range", pos: 4, want: false},
		{name: "at first range start", pos: 5, want: true},
		{name: "inside first range", pos: 8, want: true},
		{name: "at first range end", pos: 10, want: false},
		{name: "between ranges", pos: 15, want: false},
		{name: "inside second range", pos: 25, want: true},
		{name: "after second range", pos: 31, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInCodeBlock(tt.pos, ranges)
			if got != tt.want {
				t.Errorf("isInCodeBlock(%d, ranges) = %v, want %v", tt.pos, got, tt.want)
			}
		})
	}
}
