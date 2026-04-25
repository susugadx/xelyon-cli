package termtext

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestPlainTextDisplayWidthMatchesLipglossForCombiningAndEmoji(t *testing.T) {
	testCases := []string{
		"e\u0301",
		"👨‍👩‍👧‍👦",
		"a\te\u0301",
	}

	for _, tc := range testCases {
		want := lipgloss.Width(strings.ReplaceAll(tc, "\t", strings.Repeat(" ", VisualTabWidth)))
		if got := PlainTextDisplayWidth(tc); got != want {
			t.Fatalf("PlainTextDisplayWidth(%q) = %d, want %d", tc, got, want)
		}
	}
}

func TestSanitizeSingleLineANSI_CollapsesWhitespaceAcrossANSIBoundaries(t *testing.T) {
	testCases := []struct {
		input string
		want  string
	}{
		{input: "a\n\033[31m\nb", want: "a \033[31mb"},
		{input: "a\t\033[0m\tb", want: "a \033[0mb"},
	}

	for _, tc := range testCases {
		if got := SanitizeSingleLineANSI(tc.input); got != tc.want {
			t.Fatalf("SanitizeSingleLineANSI(%q) = %q, want %q", tc.input, got, tc.want)
		}
		if got := StripANSI(SanitizeSingleLineANSI(tc.input)); got != "a b" {
			t.Fatalf("stripped SanitizeSingleLineANSI(%q) = %q, want %q", tc.input, got, "a b")
		}
	}
}

func TestTruncateWithANSI_AppendsResetWhenTruncated(t *testing.T) {
	got := TruncateWithANSI("\033[31mabcdef", 3)

	if !strings.HasSuffix(got, "\033[0m") {
		t.Fatalf("truncated line should end with reset, got %q", got)
	}
	if width := lipgloss.Width(got); width != 3 {
		t.Fatalf("rendered width = %d, want 3", width)
	}
}

func TestFillANSITextWidth_ReappliesBackgroundAfterBgResetCodes(t *testing.T) {
	bg := "\033[48;5;236m"
	line := "\033[32mok\033[0m\033[49m"
	got := FillANSITextWidth(line, 8, bg)

	if width := lipgloss.Width(got); width != 8 {
		t.Fatalf("rendered width = %d, want 8; line=%q", width, got)
	}
	plain := StripANSI(got)
	if plain != "ok      " {
		t.Fatalf("plain text = %q, want %q", plain, "ok      ")
	}
	if !strings.Contains(got, "\033[0m"+bg+" ") {
		t.Fatalf("padding should be preceded by background reapply, got %q", got)
	}
}
