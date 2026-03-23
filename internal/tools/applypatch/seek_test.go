package applypatch

import "testing"

func TestSeekSequence_ExactMatch(t *testing.T) {
	lines := []string{"foo", "bar", "baz"}
	pattern := []string{"bar", "baz"}

	got, ok := SeekSequence(lines, pattern, 0, false)
	if !ok || got != 1 {
		t.Fatalf("SeekSequence() = (%d, %v), want (1, true)", got, ok)
	}
}

func TestSeekSequence_TrimEndMatch(t *testing.T) {
	lines := []string{"foo   ", "bar\t\t"}
	pattern := []string{"foo", "bar"}

	got, ok := SeekSequence(lines, pattern, 0, false)
	if !ok || got != 0 {
		t.Fatalf("SeekSequence() = (%d, %v), want (0, true)", got, ok)
	}
}

func TestSeekSequence_TrimMatch(t *testing.T) {
	lines := []string{"    foo   ", "   bar\t"}
	pattern := []string{"foo", "bar"}

	got, ok := SeekSequence(lines, pattern, 0, false)
	if !ok || got != 0 {
		t.Fatalf("SeekSequence() = (%d, %v), want (0, true)", got, ok)
	}
}

func TestSeekSequence_PatternLongerThanInput(t *testing.T) {
	lines := []string{"just one line"}
	pattern := []string{"too", "many", "lines"}

	if _, ok := SeekSequence(lines, pattern, 0, false); ok {
		t.Fatal("SeekSequence() should not match when pattern is longer than input")
	}
}

func TestSeekSequence_EmptyPattern(t *testing.T) {
	got, ok := SeekSequence([]string{"foo"}, nil, 3, false)
	if !ok || got != 3 {
		t.Fatalf("SeekSequence() = (%d, %v), want (3, true)", got, ok)
	}
}

func TestSeekSequence_EOFMode(t *testing.T) {
	lines := []string{"foo", "bar", "baz"}
	pattern := []string{"bar", "baz"}

	got, ok := SeekSequence(lines, pattern, 0, true)
	if !ok || got != 1 {
		t.Fatalf("SeekSequence() = (%d, %v), want (1, true)", got, ok)
	}
}

func TestSeekSequence_UnicodeNormalization(t *testing.T) {
	lines := []string{"import asyncio  # local import \u2013 avoids top\u2011level dep"}
	pattern := []string{"import asyncio  # local import - avoids top-level dep"}

	got, ok := SeekSequence(lines, pattern, 0, false)
	if !ok || got != 0 {
		t.Fatalf("SeekSequence() = (%d, %v), want (0, true)", got, ok)
	}
}

func TestFindLineNumber(t *testing.T) {
	content := "first\nfunc target() {\n}\n"

	if got := FindLineNumber(content, "func target() {"); got != 2 {
		t.Fatalf("FindLineNumber() = %d, want 2", got)
	}
	if got := FindLineNumber(content, "missing"); got != -1 {
		t.Fatalf("FindLineNumber() = %d, want -1", got)
	}
}
