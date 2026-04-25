package termtext

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestStyleWithCursor_CursorLine_RangeBasedANSI(t *testing.T) {
	const cursorLineBg = "\033[48;5;236m"
	const cursorCharBg = "\033[48;5;255;38;5;16m"

	styled := StylePlainTextRangeWithCursor("ABCDE", 0, 0, "", 2, cursorCharBg, cursorLineBg)

	lineBgCount := strings.Count(styled, cursorLineBg)
	if lineBgCount > 3 {
		t.Errorf("lineBg emitted %d times (per-char flicker); want <=3 transitions (styled=%q)", lineBgCount, styled)
	}

	resetCount := strings.Count(styled, "\033[0m")
	if resetCount > 3 {
		t.Errorf("reset count = %d, want <=3 (styled=%q)", resetCount, styled)
	}

	if w := lipgloss.Width(styled); w != 5 {
		t.Errorf("width = %d, want 5", w)
	}

	if p := StripANSI(styled); p != "ABCDE" {
		t.Errorf("plain = %q, want ABCDE", p)
	}
}

func TestStyleWithCursor_VisualCharWithCursor_RangeBased(t *testing.T) {
	const visualBg = "\033[48;5;240m"
	const visualCursorBg = "\033[48;5;255;38;5;16m"

	styled := StylePlainTextRangeWithCursor("ABCDE", 1, 4, visualBg, 3, visualCursorBg, "")

	resetCount := strings.Count(styled, "\033[0m")
	if resetCount > 4 {
		t.Errorf("reset count = %d, want <=4 (styled=%q)", resetCount, styled)
	}
	if w := lipgloss.Width(styled); w != 5 {
		t.Errorf("width = %d, want 5", w)
	}
	if p := StripANSI(styled); p != "ABCDE" {
		t.Errorf("plain = %q, want ABCDE", p)
	}
}

func TestStyleWithCursor_VisualLineWithCursor_RangeBased(t *testing.T) {
	const visualBg = "\033[48;5;240m"
	const cursorBg = "\033[48;5;255;38;5;16m"

	styled := StylePlainTextRangeWithCursor("ABCDE", 0, 5, visualBg, 2, cursorBg, "")

	resetCount := strings.Count(styled, "\033[0m")
	if resetCount > 3 {
		t.Errorf("reset count = %d, want <=3 (styled=%q)", resetCount, styled)
	}
	if w := lipgloss.Width(styled); w != 5 {
		t.Errorf("width = %d, want 5", w)
	}
}

func TestStyleWithCursor_EmptyString(t *testing.T) {
	const cursorLineBg = "\033[48;5;236m"
	const cursorCharBg = "\033[48;5;255;38;5;16m"

	styled := StylePlainTextRangeWithCursor("", 0, 0, "", 0, cursorCharBg, cursorLineBg)
	if !strings.Contains(styled, cursorCharBg) {
		t.Errorf("empty line cursor should show cursorBg, got %q", styled)
	}

	styled2 := StylePlainTextRangeWithCursor("", 0, 0, "", 5, cursorCharBg, cursorLineBg)
	if !strings.Contains(styled2, cursorLineBg) {
		t.Errorf("empty line with lineBg should show lineBg, got %q", styled2)
	}
}

func TestStyleWithCursor_CJKCursorPosition(t *testing.T) {
	const cursorBg = "\033[48;5;255;38;5;16m"
	const lineBg = "\033[48;5;236m"

	styled := StylePlainTextRangeWithCursor("abc日def", 0, 0, "", 3, cursorBg, lineBg)

	if !strings.Contains(styled, cursorBg) {
		t.Fatalf("CJK cursor should show cursorBg, got %q", styled)
	}
	if w := lipgloss.Width(styled); w != 8 {
		t.Errorf("width = %d, want 8", w)
	}
	if p := StripANSI(styled); p != "abc日def" {
		t.Errorf("plain = %q, want abc日def", p)
	}
}
