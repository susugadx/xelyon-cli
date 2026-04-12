package tui

import "testing"

func TestMouseSelectionText_SingleLine(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"Hello, World!"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 4}

	text, lines := m.mouseSelectionText()
	if text != "Hello" {
		t.Fatalf("text = %q, want %q", text, "Hello")
	}
	if lines != 1 {
		t.Fatalf("lines = %d, want 1", lines)
	}
}

func TestMouseSelectionText_MultiLine(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"first line", "second line", "third line"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.mouseSelAnchor = visualPosition{line: 0, col: 6}
	m.mouseSelEnd = visualPosition{line: 2, col: 4}

	text, lines := m.mouseSelectionText()
	expected := "line\nsecond line\nthird"
	if text != expected {
		t.Fatalf("text = %q, want %q", text, expected)
	}
	if lines != 3 {
		t.Fatalf("lines = %d, want 3", lines)
	}
}

func TestMouseSelectionText_WithANSI(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"\033[31mred text\033[0m"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 2}

	text, _ := m.mouseSelectionText()
	if text != "red" {
		t.Fatalf("text = %q, want %q (ANSI should be stripped)", text, "red")
	}
}
