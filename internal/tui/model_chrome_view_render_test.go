package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModel_View_RendersSinglePromptAndContainsInput(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 80
	m.height = 3
	m.vp = lightViewport{width: m.width, height: 1}
	m.vp.setContent("body")
	m.textInput.SetValue("hello")
	m.padLineCache = strings.Repeat(" ", m.width)
	m.rebuildChrome()

	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) < 3 {
		t.Fatalf("view should have at least 3 lines, got %d: %q", len(lines), view)
	}

	if strings.Count(view, inputPrompt) != 1 {
		t.Fatalf("prompt count = %d, want 1; view=%q", strings.Count(view, inputPrompt), view)
	}
	if !strings.Contains(view, "ready") {
		t.Fatalf("view should contain status line, got %q", view)
	}
	if !strings.Contains(view, "hello") {
		t.Fatalf("view should contain input value, got %q", view)
	}
	if !strings.Contains(view, "/copy") {
		t.Fatalf("view should contain copy hint, got %q", view)
	}
}

func TestModel_UpdateKeyMsgRebuildsChrome(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rebuildChrome()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)

	if !strings.Contains(m.chromeCache, "a") {
		t.Fatalf("chromeCache should include typed input, got %q", m.chromeCache)
	}
}
