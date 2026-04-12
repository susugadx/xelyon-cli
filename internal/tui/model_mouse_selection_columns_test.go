package tui

import "testing"

func TestMouseSelectionColumnsForLine_SingleLine(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.mouseSelAnchor = visualPosition{line: 3, col: 2}
	m.mouseSelEnd = visualPosition{line: 3, col: 4}

	start, end, ok := m.mouseSelectionColumnsForLine(3)
	if !ok {
		t.Fatal("expected ok")
	}
	if start != 2 {
		t.Fatalf("startCol = %d, want 2", start)
	}
	if end != 5 {
		t.Fatalf("endCol = %d, want 5", end)
	}
}

func TestMouseSelectionColumnsForLine_MultiLine(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.mouseSelAnchor = visualPosition{line: 2, col: 1}
	m.mouseSelEnd = visualPosition{line: 5, col: 3}

	start, _, ok := m.mouseSelectionColumnsForLine(2)
	if !ok {
		t.Fatal("expected ok for first line")
	}
	if start != 1 {
		t.Fatalf("first line startCol = %d, want 1", start)
	}

	start, end, ok := m.mouseSelectionColumnsForLine(3)
	if !ok {
		t.Fatal("expected ok for intermediate line")
	}
	if start != 0 {
		t.Fatalf("intermediate startCol = %d, want 0", start)
	}
	if end != 9999 {
		t.Fatalf("intermediate endCol = %d, want 9999", end)
	}

	_, end, ok = m.mouseSelectionColumnsForLine(5)
	if !ok {
		t.Fatal("expected ok for last line")
	}
	if end != 4 {
		t.Fatalf("last line endCol = %d, want 4", end)
	}

	_, _, ok = m.mouseSelectionColumnsForLine(1)
	if ok {
		t.Fatal("expected !ok for line before selection")
	}
	_, _, ok = m.mouseSelectionColumnsForLine(6)
	if ok {
		t.Fatal("expected !ok for line after selection")
	}
}

func TestMouseSelectionColumnsForLine_ReversedSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.mouseSelAnchor = visualPosition{line: 5, col: 3}
	m.mouseSelEnd = visualPosition{line: 2, col: 1}

	start, _, ok := m.mouseSelectionColumnsForLine(2)
	if !ok {
		t.Fatal("expected ok")
	}
	if start != 1 {
		t.Fatalf("startCol = %d, want 1", start)
	}
}

func TestMouseSelectionColumnsForVisualRow_Basic(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"short"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.mouseSelAnchor = visualPosition{line: 0, col: 1}
	m.mouseSelEnd = visualPosition{line: 0, col: 3}

	startCol, endCol, ok := m.mouseSelectionColumnsForLine(0)
	if !ok {
		t.Fatal("expected ok")
	}

	localStart, localEnd := m.mouseSelectionColumnsForVisualRow(0, 0, startCol, endCol)
	if localStart != 1 {
		t.Fatalf("localStart = %d, want 1", localStart)
	}
	if localEnd != 4 {
		t.Fatalf("localEnd = %d, want 4", localEnd)
	}
}
