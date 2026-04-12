package tui

import (
	"strings"
	"testing"
)

func TestModel_StreamTextCarriageReturnRewritePreservesANSISequences(t *testing.T) {
	m, _ := newStreamTestModel(40, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "\033[31mabcdef\033[0m", Done: false})
	m = updated.(Model)
	updated, _ = m.Update(StreamTextMsg{Text: "\rxy", Done: true})
	m = updated.(Model)

	if len(m.rawLines) != 1 {
		t.Fatalf("rawLines len = %d, want 1", len(m.rawLines))
	}
	if !strings.Contains(m.rawLines[0], "\033[31m") || !strings.Contains(m.rawLines[0], "\033[0m") {
		t.Fatalf("rawLines[0] should preserve ANSI sequences, got %q", m.rawLines[0])
	}
	if got := stripANSI(m.rawLines[0]); got != "xycdef" {
		t.Fatalf("stripped raw line = %q, want %q", got, "xycdef")
	}
}

func TestModel_StreamTextCarriageReturnRewriteRestoresSuffixANSIStyle(t *testing.T) {
	m, _ := newStreamTestModel(40, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "\033[31mabcdef\033[0m", Done: false})
	m = updated.(Model)
	updated, _ = m.Update(StreamTextMsg{Text: "\r\033[32mxy", Done: true})
	m = updated.(Model)

	if got := stripANSI(m.rawLines[0]); got != "xycdef" {
		t.Fatalf("stripped raw line = %q, want %q", got, "xycdef")
	}
	if !strings.Contains(m.rawLines[0], "\033[32mx") {
		t.Fatalf("overwritten prefix should use new ANSI style, got %q", m.rawLines[0])
	}
	if !strings.Contains(m.rawLines[0], "\033[31mc") {
		t.Fatalf("untouched suffix should restore original ANSI style, got %q", m.rawLines[0])
	}
}

func TestModel_StreamTextPreservesActiveANSIStyleAcrossChunks(t *testing.T) {
	m, _ := newStreamTestModel(40, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "\033[31mred", Done: false})
	m = updated.(Model)
	updated, _ = m.Update(StreamTextMsg{Text: "more", Done: true})
	m = updated.(Model)

	if got := stripANSI(m.rawLines[0]); got != "redmore" {
		t.Fatalf("stripped raw line = %q, want %q", got, "redmore")
	}
	if !strings.Contains(m.rawLines[0], "\033[31mredmore") {
		t.Fatalf("active ANSI style should continue across chunks, got %q", m.rawLines[0])
	}
}

func TestModel_StreamTextPreservesActiveANSIStyleAcrossNewlines(t *testing.T) {
	m, _ := newStreamTestModel(40, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "\033[31mfoo\nbar", Done: true})
	m = updated.(Model)

	if len(m.rawLines) != 2 {
		t.Fatalf("rawLines len = %d, want 2", len(m.rawLines))
	}
	if got := stripANSI(m.rawLines[1]); got != "bar" {
		t.Fatalf("stripped second line = %q, want %q", got, "bar")
	}
	if !strings.Contains(m.rawLines[1], "\033[31mbar") {
		t.Fatalf("second line should keep active ANSI style, got %q", m.rawLines[1])
	}
}

func TestModel_StreamTextDoesNotPersistNonSGRANSIState(t *testing.T) {
	m, _ := newStreamTestModel(40, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "\033[31mfoo\033[K", Done: false})
	m = updated.(Model)
	updated, _ = m.Update(StreamTextMsg{Text: "bar", Done: true})
	m = updated.(Model)

	if got := stripANSI(m.rawLines[0]); got != "foobar" {
		t.Fatalf("stripped raw line = %q, want %q", got, "foobar")
	}
	if strings.Contains(m.rawLines[0], "\033[K") {
		t.Fatalf("non-SGR ANSI should not be persisted into active style replay, got %q", m.rawLines[0])
	}
	if !strings.Contains(m.rawLines[0], "\033[31mfoobar") {
		t.Fatalf("SGR style should continue while non-SGR control is ignored, got %q", m.rawLines[0])
	}
}
