package tui

import "testing"

func TestNormalizedMouseSelection_NoSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	_, _, ok := m.normalizedMouseSelection()
	if ok {
		t.Fatal("expected !ok when no selection")
	}
}

func TestNormalizedMouseSelection_Reversed(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.mouseSelAnchor = visualPosition{line: 5, col: 3}
	m.mouseSelEnd = visualPosition{line: 2, col: 1}

	start, end, ok := m.normalizedMouseSelection()
	if !ok {
		t.Fatal("expected ok")
	}
	if start.line != 2 || start.col != 1 {
		t.Fatalf("start = %v, want {2, 1}", start)
	}
	if end.line != 5 || end.col != 3 {
		t.Fatalf("end = %v, want {5, 3}", end)
	}
}

func TestHasActiveMouseSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	if m.hasActiveMouseSelection() {
		t.Fatal("expected false for initial state")
	}

	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 0}
	if m.hasActiveMouseSelection() {
		t.Fatal("expected false when anchor == end")
	}

	m.mouseSelEnd = visualPosition{line: 0, col: 5}
	if !m.hasActiveMouseSelection() {
		t.Fatal("expected true when anchor != end")
	}
}

func TestClearMouseSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.mouseSelAnchor = visualPosition{line: 1, col: 2}
	m.mouseSelEnd = visualPosition{line: 3, col: 4}
	m.mouseDragging = true
	m.mouseAutoScrolling = true

	m.clearMouseSelection()

	if m.mouseSelAnchor.line != -1 {
		t.Fatal("expected anchor cleared")
	}
	if m.mouseSelEnd.line != -1 {
		t.Fatal("expected end cleared")
	}
	if m.mouseDragging {
		t.Fatal("expected dragging cleared")
	}
	if m.mouseAutoScrolling {
		t.Fatal("expected auto-scrolling cleared")
	}
}
