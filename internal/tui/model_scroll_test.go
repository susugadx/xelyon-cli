package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestModel_KeyDownScrollsViewport は Alternate Scroll Mode (1007) で
// ホイールがカーソルキーに変換された場合に viewport がスクロールすることを検証。
func TestModel_KeyDownScrollsViewport(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.vp = lightViewport{width: 10, height: 5}
	m.vp.setContent(strings.Repeat("line\n", 20))

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyDown})
	got := updated.(Model)

	if got.vp.yOffset != 1 {
		t.Fatalf("yOffset = %d, want 1", got.vp.yOffset)
	}
}

func TestModel_MouseWheelScrollsViewport(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.vp = lightViewport{width: 10, height: 5}
	setModelRawLines(&m, 20)
	m.navigationMode = true
	m.cursorLine = 0

	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	got := updated.(Model)

	if got.vp.yOffset != 3 {
		t.Fatalf("yOffset = %d, want 3", got.vp.yOffset)
	}
	if got.cursorLine != 3 {
		t.Fatalf("cursorLine = %d, want 3", got.cursorLine)
	}
	if got.cursorLine < got.vp.yOffset || got.cursorLine >= got.vp.yOffset+got.vp.height {
		t.Fatalf("cursorLine = %d should stay visible in viewport [%d, %d)", got.cursorLine, got.vp.yOffset, got.vp.yOffset+got.vp.height)
	}
}

func TestModel_MouseWheelToBottomClearsNewOutputBadge(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	for i := 0; i < 40; i++ {
		m.appendContentLines(fmt.Sprintf("line %d", i))
	}

	m.vp.gotoTop()
	m.appendContentLines("new line")
	m.chromeDirty = true
	m.rebuildChrome()
	if !strings.Contains(m.chromeCache, "New output") {
		t.Fatalf("chromeCache should include new output badge, got %q", m.chromeCache)
	}

	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	m = updated.(Model)
	for !m.vp.atBottom() {
		updated, _ = m.Update(tea.MouseMsg{
			Button: tea.MouseButtonWheelDown,
			Action: tea.MouseActionPress,
		})
		m = updated.(Model)
	}

	if m.newOutput {
		t.Fatal("newOutput should be cleared after mouse scrolling to bottom")
	}
	if strings.Contains(m.chromeCache, "New output") {
		t.Fatalf("chromeCache should clear new output badge, got %q", m.chromeCache)
	}
}

func TestModel_ScrollingToBottomClearsNewOutputBadge(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	for i := 0; i < 40; i++ {
		m.appendContentLines(fmt.Sprintf("line %d", i))
	}

	m.vp.gotoTop()
	m.appendContentLines("new line")
	m.chromeDirty = true
	m.rebuildChrome()
	if !strings.Contains(m.chromeCache, "New output") {
		t.Fatalf("chromeCache should include new output badge, got %q", m.chromeCache)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = updated.(Model)
	for !m.vp.atBottom() {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
		m = updated.(Model)
	}

	if m.newOutput {
		t.Fatal("newOutput should be cleared after scrolling to bottom")
	}
	if strings.Contains(m.chromeCache, "New output") {
		t.Fatalf("chromeCache should clear new output badge, got %q", m.chromeCache)
	}
}
