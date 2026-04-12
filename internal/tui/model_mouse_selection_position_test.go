package tui

import "testing"

func TestScreenToRawPosition_Basic(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 20)
	m.vp.gotoTop()

	rawLine, rawCol, ok := m.screenToRawPosition(3, 0)
	if !ok {
		t.Fatal("expected ok")
	}
	if rawLine != 0 {
		t.Fatalf("rawLine = %d, want 0", rawLine)
	}
	if rawCol != 3 {
		t.Fatalf("rawCol = %d, want 3", rawCol)
	}
}

func TestScreenToRawPosition_WithOffset(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 50)
	m.vp.yOffset = 10

	rawLine, rawCol, ok := m.screenToRawPosition(3, 2)
	if !ok {
		t.Fatal("expected ok")
	}
	if rawLine != 12 {
		t.Fatalf("rawLine = %d, want 12", rawLine)
	}
	if rawCol != 3 {
		t.Fatalf("rawCol = %d, want 3", rawCol)
	}
}

func TestScreenToRawPosition_ClampsToContent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 5)

	rawLine, rawCol, ok := m.screenToRawPosition(99, 0)
	if !ok {
		t.Fatal("expected ok")
	}
	if rawLine != 0 {
		t.Fatalf("rawLine = %d, want 0", rawLine)
	}
	if rawCol > 4 {
		t.Fatalf("rawCol = %d, want <= 4", rawCol)
	}
}

func TestScreenToRawPosition_EmptyLayout(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	_, _, ok := m.screenToRawPosition(0, 0)
	if ok {
		t.Fatal("expected !ok for empty layout")
	}
}

func TestScreenToRawPosition_NegativeY(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.vp.yOffset = 5

	rawLine, _, ok := m.screenToRawPosition(0, -3)
	if !ok {
		t.Fatal("expected ok")
	}
	if rawLine != 2 {
		t.Fatalf("rawLine = %d, want 2", rawLine)
	}
}
