package tui

import (
	"strings"
	"testing"
)

func TestModel_StreamTextMergesChunksAcrossMessages(t *testing.T) {
	m, _ := newStreamTestModel(20, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "hello", Done: false})
	m = updated.(Model)
	updated, _ = m.Update(StreamTextMsg{Text: "\nworld", Done: true})
	m = updated.(Model)

	if len(m.rawLines) != 2 {
		t.Fatalf("rawLines len = %d, want 2", len(m.rawLines))
	}
	if m.rawLines[0] != "hello" || m.rawLines[1] != "world" {
		t.Fatalf("rawLines = %#v, want [hello world]", m.rawLines)
	}
	if m.streamingActive {
		t.Fatal("streamingActive should be reset after done")
	}
}

func TestModel_StreamTextInitialMultiLineChunkStartsEachLineAtColumnZero(t *testing.T) {
	m, _ := newStreamTestModel(20, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "foo\nbar", Done: true})
	m = updated.(Model)

	if len(m.rawLines) != 2 {
		t.Fatalf("rawLines len = %d, want 2", len(m.rawLines))
	}
	if m.rawLines[0] != "foo" || m.rawLines[1] != "bar" {
		t.Fatalf("rawLines = %#v, want [foo bar]", m.rawLines)
	}
}

func TestModel_StreamTextPreservesSplitANSISequencesAcrossChunks(t *testing.T) {
	m, _ := newStreamTestModel(40, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "\033[31", Done: false})
	m = updated.(Model)
	if len(m.rawLines) != 1 {
		t.Fatalf("rawLines len after first chunk = %d, want 1", len(m.rawLines))
	}
	if m.rawLines[0] != "" {
		t.Fatalf("rawLines[0] after partial ANSI = %q, want empty", m.rawLines[0])
	}

	updated, _ = m.Update(StreamTextMsg{Text: "mred", Done: true})
	m = updated.(Model)

	if len(m.rawLines) != 1 {
		t.Fatalf("rawLines len after second chunk = %d, want 1", len(m.rawLines))
	}
	if got := stripANSI(m.rawLines[0]); got != "red" {
		t.Fatalf("stripped raw line = %q, want %q", got, "red")
	}
	if !strings.Contains(m.rawLines[0], "\033[31m") {
		t.Fatalf("rawLines[0] should contain combined ANSI sequence, got %q", m.rawLines[0])
	}
}

func TestModel_StreamTextPreservesTabsInRawLines(t *testing.T) {
	m, agent := newStreamTestModel(24, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "a\tb", Done: true})
	m = updated.(Model)

	if len(m.rawLines) != 1 {
		t.Fatalf("rawLines len = %d, want 1", len(m.rawLines))
	}
	if m.rawLines[0] != "a\tb" {
		t.Fatalf("rawLines[0] = %q, want %q", m.rawLines[0], "a\\tb")
	}
	if err := m.copyRawRangePlain(0, 0); err != nil {
		t.Fatalf("copyRawRangePlain() error = %v", err)
	}
	if len(agent.copyTexts) != 1 || agent.copyTexts[0] != "a\tb" {
		t.Fatalf("copyTexts = %#v, want [a\\tb]", agent.copyTexts)
	}
}
