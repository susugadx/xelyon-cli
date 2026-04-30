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

func TestModel_StartupSubmissionAppendsUserMessageBeforeRunning(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	var ran bool
	startup := &StartupSubmission{
		UserMessage: "describe this image",
		Cmd: func() tea.Msg {
			ran = true
			return AgentDoneMsg{}
		},
	}
	m := NewModelWithStartupSubmission(agent, "", startup)

	updated, cmd := m.Update(startupSubmissionMsg{submission: *startup})
	m = updated.(Model)

	if len(m.messages) != 1 {
		t.Fatalf("messages len = %d, want 1", len(m.messages))
	}
	if got := m.messages[0].Role; got != "user" {
		t.Fatalf("message role = %q, want user", got)
	}
	if got := m.messages[0].Content; got != startup.UserMessage {
		t.Fatalf("message content = %q, want %q", got, startup.UserMessage)
	}
	if len(m.rawLines) == 0 {
		t.Fatal("expected startup user message to be rendered into rawLines")
	}
	if cmd == nil {
		t.Fatal("expected startup command to be returned after rendering user message")
	}
	cmd()
	if !ran {
		t.Fatal("expected startup command to run")
	}
}
