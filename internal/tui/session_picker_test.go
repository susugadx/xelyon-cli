package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestResumeCommandOpensPickerAndResumesSelection(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		sessionCandidates: []SessionCandidate{
			{ID: "session-1", Preview: "first", Model: "gpt-test", ProviderName: "openai", LastModified: time.Now()},
		},
	}
	m := newModelWithViewport(agent)
	m.textInput.SetValue("/resume")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd != nil {
		t.Fatalf("cmd = %T, want nil", cmd)
	}
	if m.sessionPicker == nil {
		t.Fatal("session picker should be open")
	}

	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd != nil {
		t.Fatalf("resume selection cmd = %T, want nil", cmd)
	}
	if len(agent.resumedSessionIDs) != 1 || agent.resumedSessionIDs[0] != "session-1" {
		t.Fatalf("resumedSessionIDs = %#v, want session-1", agent.resumedSessionIDs)
	}
	if m.sessionPicker != nil {
		t.Fatal("session picker should close after resume")
	}
	if len(m.messages) == 0 || !strings.Contains(m.messages[len(m.messages)-1].Content, "Resumed session") {
		t.Fatalf("messages = %#v, want resume notice", m.messages)
	}
}

func TestResumeCommandWithIDResumesDirectly(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.textInput.SetValue("/resume session-42")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd != nil {
		t.Fatalf("cmd = %T, want nil", cmd)
	}
	if len(agent.resumedSessionIDs) != 1 || agent.resumedSessionIDs[0] != "session-42" {
		t.Fatalf("resumedSessionIDs = %#v, want session-42", agent.resumedSessionIDs)
	}
	if m.sessionPicker != nil {
		t.Fatal("session picker should not open for direct ID")
	}
}

func TestResumeCommandLastResetsVisibleTranscript(t *testing.T) {
	agent := &stubAgent{
		statusLine:           "ready",
		lastSessionCandidate: SessionCandidate{ID: "last-session"},
		handledCommands:      map[string]bool{"/resume --last": true},
		sessionCandidates:    []SessionCandidate{{ID: "should-not-open-picker"}},
	}
	m := newModelWithViewport(agent)
	m.appendSystemNotice("old visible transcript")
	m.textInput.SetValue("/resume --last")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd != nil {
		t.Fatalf("cmd = %T, want nil", cmd)
	}
	if agent.resumeLastCalls != 1 {
		t.Fatalf("resumeLastCalls = %d, want 1", agent.resumeLastCalls)
	}
	if len(agent.handledInputs) != 0 {
		t.Fatalf("handledInputs = %#v, want no generic command dispatch", agent.handledInputs)
	}
	if containsMessage(m.messages, "old visible transcript") {
		t.Fatalf("/resume --last should reset visible transcript, got %#v", m.messages)
	}
	if len(m.messages) == 0 || !strings.Contains(m.messages[len(m.messages)-1].Content, "Resumed session last-ses") {
		t.Fatalf("messages = %#v, want last resume notice", m.messages)
	}
}

func TestResumeCommandRejectsConflictingLastArgs(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.textInput.SetValue("/resume --last --all")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd != nil {
		t.Fatalf("cmd = %T, want nil", cmd)
	}
	if agent.resumeLastCalls != 0 {
		t.Fatalf("resumeLastCalls = %d, want 0", agent.resumeLastCalls)
	}
	if len(agent.resumedSessionIDs) != 0 {
		t.Fatalf("resumedSessionIDs = %#v, want none", agent.resumedSessionIDs)
	}
	if !strings.Contains(m.transientStatus, "--last cannot be used with --all") {
		t.Fatalf("transientStatus = %q, want conflict message", m.transientStatus)
	}
}

func TestNewAndClearSessionVisibility(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}

	m := newModelWithViewport(agent)
	m.appendSystemNotice("existing")
	m.textInput.SetValue("/new")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if len(agent.startedSessionIDs) != 1 {
		t.Fatalf("startedSessionIDs len = %d, want 1", len(agent.startedSessionIDs))
	}
	if !containsMessage(m.messages, "existing") {
		t.Fatalf("/new should preserve visible transcript, got %#v", m.messages)
	}

	m.textInput.SetValue("/clear")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if len(agent.startedSessionIDs) != 2 {
		t.Fatalf("startedSessionIDs len = %d, want 2", len(agent.startedSessionIDs))
	}
	if containsMessage(m.messages, "existing") {
		t.Fatalf("/clear should clear visible transcript, got %#v", m.messages)
	}
}

func containsMessage(messages []ChatMessage, text string) bool {
	for _, message := range messages {
		if strings.Contains(message.Content, text) {
			return true
		}
	}
	return false
}
