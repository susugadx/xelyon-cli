package commandruntime

import (
	"testing"
)

func TestShouldStartQuoteSegmentInsideToken(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		quote rune
		want  bool
	}{
		{
			name:  "contraction stays literal",
			text:  "/note don't",
			quote: '\'',
			want:  false,
		},
		{
			name:  "quoted segment after token prefix",
			text:  "/note foo'bar baz'qux",
			quote: '\'',
			want:  true,
		},
		{
			name:  "quoted segment short first word",
			text:  "/note foo'a b'qux",
			quote: '\'',
			want:  true,
		},
		{
			name:  "double quote inside token with close",
			text:  `/note foo"bar baz"qux`,
			quote: '"',
			want:  true,
		},
		{
			name:  "no closing quote inside token",
			text:  "/note foo'bar",
			quote: '\'',
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runes := []rune(tt.text)
			openIndex := firstRuneIndex(runes, tt.quote)
			if openIndex < 0 {
				t.Fatalf("no quote in %q", tt.text)
			}
			got := shouldStartQuoteSegmentInsideToken(runes, openIndex, tt.quote)
			if got != tt.want {
				t.Fatalf("shouldStartQuoteSegmentInsideToken(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

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
			name: "contraction followed by possessive t",
			text: "/note can't's",
			want: true,
		},
		{
			name: "contraction followed by possessive re",
			text: "/note we're's",
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
			openIndex := firstRuneIndex(runes, '\'')
			if openIndex < 0 {
				t.Fatalf("no apostrophe in %q", tt.text)
			}
			closeIndex := findClosingQuoteIndex(runes, openIndex+1, '\'')
			got := shouldTreatApostropheAsLiteralInCurrentToken(runes, openIndex, closeIndex)
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

func TestTokenFragmentAfterApostropheBoundary(t *testing.T) {
	runes := []rune("can't's stop")
	first := tokenFragmentAfterApostropheBoundary(runes, 3)
	if got, want := string(first), "t"; got != want {
		t.Fatalf("tokenFragmentAfterApostropheBoundary(first) = %q, want %q", got, want)
	}
	second := tokenFragmentAfterApostropheBoundary(runes, 5)
	if got, want := string(second), "s"; got != want {
		t.Fatalf("tokenFragmentAfterApostropheBoundary(second) = %q, want %q", got, want)
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

func TestIsContractionSuffix(t *testing.T) {
	if !isContractionSuffix([]rune("t")) {
		t.Fatal("isContractionSuffix(\"t\") = false, want true")
	}
	if !isContractionSuffix([]rune("re")) {
		t.Fatal("isContractionSuffix(\"re\") = false, want true")
	}
	if isContractionSuffix([]rune("a")) {
		t.Fatal("isContractionSuffix(\"a\") = true, want false")
	}
	if isContractionSuffix([]rune("to")) {
		t.Fatal("isContractionSuffix(\"to\") = true, want false")
	}
}

func firstRuneIndex(runes []rune, target rune) int {
	for i, r := range runes {
		if r == target {
			return i
		}
	}
	return -1
}
