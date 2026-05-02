package commandruntime

import (
	"testing"
)

func TestShouldTreatApostropheAsLiteralInCurrentToken(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{
			name: "contractions across words",
			text: "/note don't it's",
			want: true,
		},
		{
			name: "shell style quoted segment with suffix",
			text: "/note foo'bar baz'qux",
			want: false,
		},
		{
			name: "quoted segment prefix without suffix",
			text: "/note foo'bar baz'",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runes := []rune(tt.text)
			openIndex := -1
			for i, r := range runes {
				if r == '\'' {
					openIndex = i
					break
				}
			}
			if openIndex < 0 {
				t.Fatalf("no apostrophe in %q", tt.text)
			}
			got := shouldTreatApostropheAsLiteralInCurrentToken(runes, openIndex)
			if got != tt.want {
				t.Fatalf("shouldTreatApostropheAsLiteralInCurrentToken(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

func TestNextWhitespaceIndex(t *testing.T) {
	runes := []rune("ab cd")
	if got, want := nextWhitespaceIndex(runes, 0), 2; got != want {
		t.Fatalf("nextWhitespaceIndex() = %d, want %d", got, want)
	}
	if got, want := nextWhitespaceIndex(runes, 3), len(runes); got != want {
		t.Fatalf("nextWhitespaceIndex() = %d, want %d", got, want)
	}
}

func TestTokenFragmentAfterIndex(t *testing.T) {
	runes := []rune("don't stop")
	fragment := tokenFragmentAfterIndex(runes, 3) // apostrophe index
	if got, want := string(fragment), "t"; got != want {
		t.Fatalf("tokenFragmentAfterIndex() = %q, want %q", got, want)
	}
}

func TestIsShortAlphabeticFragment(t *testing.T) {
	if !isShortAlphabeticFragment([]rune("s"), 2) {
		t.Fatal("isShortAlphabeticFragment(\"s\") = false, want true")
	}
	if isShortAlphabeticFragment([]rune("ing"), 2) {
		t.Fatal("isShortAlphabeticFragment(\"ing\") = true, want false")
	}
	if isShortAlphabeticFragment([]rune("3"), 2) {
		t.Fatal("isShortAlphabeticFragment(\"3\") = true, want false")
	}
}
