package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModel_ClearsTextInputPrompt(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	if m.textInput.Prompt != "" {
		t.Fatalf("textInput.Prompt = %q, want empty string", m.textInput.Prompt)
	}
}

func TestModel_FooterHeight(t *testing.T) {
	m := NewModel(&stubAgent{statusLine: "ready"}, "")
	got := m.footerHeight()
	want := statusBarHeight + inputHeight
	if got != want {
		t.Fatalf("footerHeight() = %d, want %d", got, want)
	}
}

func TestModel_FooterHeight_UsedInWindowSizeMsg(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m = updated.(Model)

	wantVPHeight := 30 - m.footerHeight()
	if m.vp.height != wantVPHeight {
		t.Fatalf("vp.height = %d, want %d (height %d - footerHeight %d)", m.vp.height, wantVPHeight, 30, m.footerHeight())
	}
}
