package prompt

import "testing"

func TestIsAlphaNum(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"lowercase", "abc", true},
		{"uppercase", "ABC", true},
		{"digits", "123", true},
		{"mixed", "aBc123", true},
		{"single letter", "a", true},
		{"single digit", "0", true},
		{"empty", "", false},
		{"space", "a b", false},
		{"hyphen", "a-b", false},
		{"underscore", "a_b", false},
		{"dot", "a.b", false},
		{"special char", "abc!", false},
		{"unicode", "あ", false},
		{"leading space", " abc", false},
		{"trailing space", "abc ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAlphaNum(tt.in)
			if got != tt.want {
				t.Errorf("isAlphaNum(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
