package tui

import (
	"strings"
	"testing"
)

func TestModel_StreamTextCarriageReturnOverwritesLine(t *testing.T) {
	m, _ := newStreamTestModel(30, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "progress 10%", Done: false})
	m = updated.(Model)
	updated, _ = m.Update(StreamTextMsg{Text: "\rprogress 20%", Done: true})
	m = updated.(Model)

	if len(m.rawLines) != 1 {
		t.Fatalf("rawLines len = %d, want 1", len(m.rawLines))
	}
	if m.rawLines[0] != "progress 20%" {
		t.Fatalf("rawLines[0] = %q, want %q", m.rawLines[0], "progress 20%")
	}
	view := m.viewportView()
	if strings.Contains(view, "\r") {
		t.Fatalf("viewportView should not contain carriage return, got %q", view)
	}
	if !strings.Contains(stripANSI(view), "progress 20%") {
		t.Fatalf("viewportView should render overwritten stream text, got %q", view)
	}
}

func TestModel_StreamTextCarriageReturnOverwritesLineAcrossChunks(t *testing.T) {
	m, _ := newStreamTestModel(30, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "progress 10%\r", Done: false})
	m = updated.(Model)
	updated, _ = m.Update(StreamTextMsg{Text: "progress 20%", Done: true})
	m = updated.(Model)

	if len(m.rawLines) != 1 {
		t.Fatalf("rawLines len = %d, want 1", len(m.rawLines))
	}
	if m.rawLines[0] != "progress 20%" {
		t.Fatalf("rawLines[0] = %q, want %q", m.rawLines[0], "progress 20%")
	}
	view := m.viewportView()
	if strings.Contains(view, "\r") {
		t.Fatalf("viewportView should not contain carriage return, got %q", view)
	}
	if !strings.Contains(stripANSI(view), "progress 20%") {
		t.Fatalf("viewportView should render overwritten stream text, got %q", view)
	}
}

func TestModel_StreamTextCarriageReturnShortRewriteKeepsTrailingText(t *testing.T) {
	m, _ := newStreamTestModel(30, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "abcdef", Done: false})
	m = updated.(Model)
	updated, _ = m.Update(StreamTextMsg{Text: "\rxy", Done: true})
	m = updated.(Model)

	if len(m.rawLines) != 1 {
		t.Fatalf("rawLines len = %d, want 1", len(m.rawLines))
	}
	if m.rawLines[0] != "xycdef" {
		t.Fatalf("rawLines[0] = %q, want %q", m.rawLines[0], "xycdef")
	}
}

func TestModel_StreamTextCarriageReturnShortRewriteKeepsTrailingTextAcrossChunks(t *testing.T) {
	m, _ := newStreamTestModel(30, 8)

	updated, _ := m.Update(StreamTextMsg{Text: "abcdef\r", Done: false})
	m = updated.(Model)
	updated, _ = m.Update(StreamTextMsg{Text: "xy", Done: true})
	m = updated.(Model)

	if len(m.rawLines) != 1 {
		t.Fatalf("rawLines len = %d, want 1", len(m.rawLines))
	}
	if m.rawLines[0] != "xycdef" {
		t.Fatalf("rawLines[0] = %q, want %q", m.rawLines[0], "xycdef")
	}
}
