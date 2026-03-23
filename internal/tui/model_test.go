package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type stubAgent struct {
	processing   bool
	cancelCalls  int
	cleanupCalls int
	statusLine   string
}

func (s *stubAgent) Chat(input string) {}

func (s *stubAgent) HandleCommand(cmd string) bool { return false }

func (s *stubAgent) GetStatusLine() string { return s.statusLine }

func (s *stubAgent) Cancel() { s.cancelCalls++ }

func (s *stubAgent) Cleanup() { s.cleanupCalls++ }

func (s *stubAgent) IsProcessing() bool { return s.processing }

func TestModel_Update_MouseMsgScrollsOnce(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent)
	m.ready = true
	m.viewport = viewport.New(10, 5)
	m.viewport.SetContent(strings.Repeat("line\n", 20))

	updated, _ := m.Update(tea.MouseMsg{Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown})
	got := updated.(Model)

	if got.viewport.YOffset != 3 {
		t.Fatalf("YOffset = %d, want 3", got.viewport.YOffset)
	}
}

func TestModel_HandleKeyMsg_CtrlCRequiresTwoPresses(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent)

	updated, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyCtrlC})
	m1 := updated.(Model)
	if cmd != nil {
		t.Fatalf("first ctrl+c returned unexpected command")
	}
	if agent.cancelCalls != 0 {
		t.Fatalf("cancelCalls = %d, want 0", agent.cancelCalls)
	}
	if agent.cleanupCalls != 0 {
		t.Fatalf("cleanupCalls = %d, want 0", agent.cleanupCalls)
	}
	if m1.lastInterrupt.IsZero() {
		t.Fatal("expected lastInterrupt to be recorded")
	}

	updated, cmd = m1.handleKeyMsg(tea.KeyMsg{Type: tea.KeyCtrlC})
	m2 := updated.(Model)
	if cmd == nil {
		t.Fatal("second ctrl+c returned nil command, want tea.Quit")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("second ctrl+c returned %T, want tea.QuitMsg", msg)
	}
	if agent.cleanupCalls != 1 {
		t.Fatalf("cleanupCalls = %d, want 1", agent.cleanupCalls)
	}
	if !m2.quitting {
		t.Fatal("expected model to enter quitting state")
	}
}

func TestModel_HandleKeyMsg_CtrlCRestartsWindow(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyCtrlC})
	m1 := updated.(Model)
	m1.lastInterrupt = time.Now().Add(-4 * time.Second)

	_, cmd := m1.handleKeyMsg(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd != nil {
		t.Fatalf("second ctrl+c after timeout returned unexpected command")
	}
	if agent.cleanupCalls != 0 {
		t.Fatalf("cleanupCalls = %d, want 0", agent.cleanupCalls)
	}
}
