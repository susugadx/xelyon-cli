package tui

import (
	"fmt"
	"testing"
)

func TestModel_AutoFollow_DoesNotJumpWhenScrolledUp(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	for i := 0; i < 40; i++ {
		m.appendContentLines(fmt.Sprintf("line %d", i))
	}

	m.vp.gotoTop()
	savedOffset := m.vp.yOffset
	m.appendContentLines("new content while scrolled up")

	if m.vp.yOffset != savedOffset {
		t.Fatalf("yOffset = %d, want %d (should not jump to bottom)", m.vp.yOffset, savedOffset)
	}
	if !m.newOutput {
		t.Fatal("newOutput should be true when content added while scrolled up")
	}
}

func TestModel_AutoFollow_StreamDoesNotJumpWhenScrolledUp(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	for i := 0; i < 40; i++ {
		m.appendContentLines(fmt.Sprintf("line %d", i))
	}

	m.vp.gotoTop()
	savedOffset := m.vp.yOffset
	m.appendStreamText("streamed text")

	if m.vp.yOffset != savedOffset {
		t.Fatalf("yOffset = %d, want %d (stream should not jump to bottom)", m.vp.yOffset, savedOffset)
	}
	if !m.newOutput {
		t.Fatal("newOutput should be true when stream added while scrolled up")
	}
}

func TestModel_AutoFollow_StaysAtBottomWhenNewContentArrives(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	for i := 0; i < 40; i++ {
		m.appendContentLines(fmt.Sprintf("line %d", i))
	}

	if !m.vp.atBottom() {
		t.Fatal("viewport should be at bottom after initial content")
	}

	m.appendContentLines("another line")

	if !m.vp.atBottom() {
		t.Fatal("viewport should stay at bottom after appending while at bottom")
	}
	if m.newOutput {
		t.Fatal("newOutput should be false when at bottom")
	}
}

func TestModel_SyncViewportContent_NotReadyIsNoop(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.syncViewportContent()
	if m.newOutput {
		t.Fatal("newOutput should not be set when not ready")
	}
}
