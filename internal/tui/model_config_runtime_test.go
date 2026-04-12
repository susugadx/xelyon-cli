package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestConfigScreen_CtrlC_CancelsWhileProcessing(t *testing.T) {
	agent := &stubAgent{}
	agent.setProcessing(true)

	m := newModelWithViewport(agent)
	updated, _ := m.openConfigScreen()
	m = updated.(Model)

	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig", m.screen)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if cmd != nil {
		t.Fatal("ctrl+c while processing should not quit")
	}
	agent.mu.RLock()
	cancelCalls := agent.cancelCalls
	agent.mu.RUnlock()
	if cancelCalls != 1 {
		t.Fatalf("cancelCalls = %d, want 1", cancelCalls)
	}
	if m.screen != screenConfig {
		t.Fatalf("screen = %d after ctrl+c, want screenConfig", m.screen)
	}
	if m.configScreen == nil {
		t.Fatal("configScreen should remain available after ctrl+c")
	}
}

func TestConfigScreen_CtrlC_NoUnexpectedEffectWhenIdle(t *testing.T) {
	agent := &stubAgent{}
	m := newModelWithViewport(agent)
	updated, _ := m.openConfigScreen()
	m = updated.(Model)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if cmd != nil {
		t.Fatal("first ctrl+c while idle should not quit")
	}
	agent.mu.RLock()
	cancelCalls := agent.cancelCalls
	agent.mu.RUnlock()
	if cancelCalls != 0 {
		t.Fatalf("cancelCalls = %d, want 0", cancelCalls)
	}
	if m.lastInterrupt.IsZero() {
		t.Fatal("lastInterrupt should be recorded on idle ctrl+c")
	}
	if m.screen != screenConfig {
		t.Fatalf("screen = %d after idle ctrl+c, want screenConfig", m.screen)
	}
	if m.configScreen == nil {
		t.Fatal("configScreen should remain available after idle ctrl+c")
	}
}

func TestConfigScreen_MessagesNotLostDuringConfigScreen(t *testing.T) {
	m := newConfigTestModel()

	msgsBefore := len(m.messages)
	updated, _ := m.Update(AppendMessageMsg{
		Message: ChatMessage{Role: "assistant", Content: "background output"},
	})
	m = updated.(Model)

	if len(m.messages) <= msgsBefore {
		t.Fatal("AppendMessageMsg should be buffered during config screen")
	}
	if m.screen != screenConfig {
		t.Fatalf("screen = %d, want screenConfig", m.screen)
	}

	updated, _ = m.Update(AgentDoneMsg{})
	m = updated.(Model)
	if m.screen != screenConfig {
		t.Fatalf("screen = %d after AgentDoneMsg, want screenConfig", m.screen)
	}
}
