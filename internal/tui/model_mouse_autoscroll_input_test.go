package tui

import "testing"

func TestAutoScrollAmount(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.vp.height = 20

	tests := []struct {
		name     string
		y        int
		wantSign int
	}{
		{"top edge y=0", 0, -1},
		{"top edge y=1", 1, -1},
		{"top edge y=2", 2, -1},
		{"middle y=5", 5, 0},
		{"middle y=10", 10, 0},
		{"bottom edge y=17", 17, 1},
		{"bottom edge y=18", 18, 1},
		{"bottom edge y=19", 19, 1},
		{"below viewport y=20", 20, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.mouseLastScreenY = tt.y
			amount := m.autoScrollAmount()
			switch {
			case tt.wantSign < 0 && amount >= 0:
				t.Fatalf("amount = %d, want negative", amount)
			case tt.wantSign == 0 && amount != 0:
				t.Fatalf("amount = %d, want 0", amount)
			case tt.wantSign > 0 && amount <= 0:
				t.Fatalf("amount = %d, want positive", amount)
			}
		})
	}
}

func TestAutoScroll_SpeedIncreases(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.vp.height = 20

	m.mouseLastScreenY = 2
	slow := m.autoScrollAmount()
	m.mouseLastScreenY = 0
	fast := m.autoScrollAmount()

	if fast >= slow {
		t.Fatalf("speed at y=0 (%d) should be faster than y=2 (%d)", fast, slow)
	}
}

func TestHandleAutoScroll_StopsWhenNotDragging(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 50)
	m.mouseDragging = false
	m.mouseAutoScrolling = true

	cmd := m.handleAutoScroll()
	if cmd != nil {
		t.Fatal("expected nil cmd when not dragging")
	}
	if m.mouseAutoScrolling {
		t.Fatal("expected mouseAutoScrolling to be cleared")
	}
}

func TestHandleAutoScroll_ScrollsAndUpdates(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 50)
	m.vp.gotoTop()
	m.mouseDragging = true
	m.mouseAutoScrolling = true
	m.mouseLastScreenY = m.vp.height
	m.mouseLastScreenX = 5
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 5}

	oldOffset := m.vp.yOffset
	cmd := m.handleAutoScroll()

	if cmd == nil {
		t.Fatal("expected non-nil cmd for continued scrolling")
	}
	if m.vp.yOffset <= oldOffset {
		t.Fatalf("expected viewport to scroll down: offset %d -> %d", oldOffset, m.vp.yOffset)
	}
	if m.mouseSelEnd.line == 0 && m.mouseSelEnd.col == 5 {
		t.Fatal("expected selection end to update after scroll")
	}
}
