package tui

import (
	"path/filepath"
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

func TestResumeAllPickerResumesDifferentWorkingDir(t *testing.T) {
	currentDir := filepath.Join(t.TempDir(), "current")
	otherDir := filepath.Join(t.TempDir(), "other")
	agent := &stubAgent{
		statusLine: "ready",
		sessionCandidates: []SessionCandidate{
			{ID: "other-session", Preview: "other", WorkingDir: otherDir, LastModified: time.Now()},
		},
	}
	m := newModelWithViewport(agent)
	m.workingDir = currentDir

	var cmd tea.Cmd
	m, cmd = m.openSessionPicker(true, false)
	if cmd != nil {
		t.Fatalf("openSessionPicker cmd = %T, want nil", cmd)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd != nil {
		t.Fatalf("resume selection cmd = %T, want nil", cmd)
	}
	if len(agent.resumedSessionIDs) != 1 || agent.resumedSessionIDs[0] != "other-session" {
		t.Fatalf("resumedSessionIDs = %#v, want other-session", agent.resumedSessionIDs)
	}
	if m.sessionPicker != nil {
		t.Fatal("session picker should close after --all resume")
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

func TestSessionLifecycleCommandsRejectWhileAgentBusy(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "new", input: "/new"},
		{name: "new with args", input: "/new typo"},
		{name: "clear", input: "/clear"},
		{name: "clear with args", input: "/clear typo"},
		{name: "resume picker", input: "/resume"},
		{name: "resume last", input: "/resume --last"},
		{name: "resume id", input: "/resume session-42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &stubAgent{
				statusLine:           "ready",
				lastSessionCandidate: SessionCandidate{ID: "last-session"},
				sessionCandidates: []SessionCandidate{
					{ID: "session-1", Preview: "first", Model: "gpt-test", ProviderName: "openai", LastModified: time.Now()},
				},
			}
			m := newModelWithViewport(agent)
			m.beginAgentActivity()
			firstBlock := m.agentActivity.block

			updated, cmd := m.handleCommandSubmission(composerSubmission{
				kind:         composerSubmissionCommand,
				commandInput: tt.input,
				payload:      tt.input,
			})
			m = updated.(Model)

			if cmd != nil {
				t.Fatalf("cmd = %T, want nil", cmd)
			}
			if m.transientStatus != agentTurnBusyStatus {
				t.Fatalf("transientStatus = %q, want %q", m.transientStatus, agentTurnBusyStatus)
			}
			if m.agentActivity.block != firstBlock {
				t.Fatalf("agent activity block changed from %#v to %#v", firstBlock, m.agentActivity.block)
			}
			if len(agent.startedSessionIDs) != 0 {
				t.Fatalf("startedSessionIDs = %#v, want none", agent.startedSessionIDs)
			}
			if agent.resumeLastCalls != 0 {
				t.Fatalf("resumeLastCalls = %d, want 0", agent.resumeLastCalls)
			}
			if len(agent.resumedSessionIDs) != 0 {
				t.Fatalf("resumedSessionIDs = %#v, want none", agent.resumedSessionIDs)
			}
			if len(agent.handledInputs) != 0 {
				t.Fatalf("handledInputs = %#v, want no agent command dispatch", agent.handledInputs)
			}
			if m.sessionPicker != nil {
				t.Fatal("session picker should not open while agent is busy")
			}
		})
	}
}

func TestNewAndClearCommandsRejectArgsWhileIdle(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantStatus string
	}{
		{name: "new", input: "/new typo", wantStatus: "Usage: /new"},
		{name: "clear", input: "/clear typo", wantStatus: "Usage: /clear"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := &stubAgent{statusLine: "ready"}
			m := newModelWithViewport(agent)
			m.textInput.SetValue(tt.input)

			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = updated.(Model)
			if cmd != nil {
				t.Fatalf("cmd = %T, want nil", cmd)
			}
			if len(agent.startedSessionIDs) != 0 {
				t.Fatalf("startedSessionIDs = %#v, want none", agent.startedSessionIDs)
			}
			if len(agent.handledInputs) != 0 {
				t.Fatalf("handledInputs = %#v, want no agent dispatch", agent.handledInputs)
			}
			if m.transientStatus != tt.wantStatus {
				t.Fatalf("transientStatus = %q, want %q", m.transientStatus, tt.wantStatus)
			}
		})
	}
}

func TestStartupSessionPickerSelectionUsesStartupResume(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
		sessionCandidates: []SessionCandidate{
			{ID: "session-startup", Preview: "first", Model: "gpt-test", ProviderName: "openai", LastModified: time.Now()},
		},
	}
	m := newModelWithViewport(agent)
	m.appendSystemNotice("bootstrap transcript")

	var cmd tea.Cmd
	m, cmd = m.openSessionPicker(false, true)
	if cmd != nil {
		t.Fatalf("openSessionPicker cmd = %T, want nil", cmd)
	}
	if m.sessionPicker == nil || !m.sessionPicker.startup {
		t.Fatalf("sessionPicker = %#v, want startup picker", m.sessionPicker)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd != nil {
		t.Fatalf("resume selection cmd = %T, want nil", cmd)
	}
	if len(agent.startupResumedSessionIDs) != 1 || agent.startupResumedSessionIDs[0] != "session-startup" {
		t.Fatalf("startupResumedSessionIDs = %#v, want session-startup", agent.startupResumedSessionIDs)
	}
	if len(agent.resumedSessionIDs) != 0 {
		t.Fatalf("resumedSessionIDs = %#v, want normal resume unused", agent.resumedSessionIDs)
	}
	if containsMessage(m.messages, "bootstrap transcript") {
		t.Fatalf("startup resume should reset visible transcript, got %#v", m.messages)
	}
}

func TestStartupResumeAllPickerResumesDifferentWorkingDir(t *testing.T) {
	currentDir := filepath.Join(t.TempDir(), "current")
	otherDir := filepath.Join(t.TempDir(), "other")
	agent := &stubAgent{
		statusLine: "ready",
		sessionCandidates: []SessionCandidate{
			{ID: "session-startup", Preview: "first", WorkingDir: otherDir, LastModified: time.Now()},
		},
	}
	m := newModelWithViewport(agent)
	m.workingDir = currentDir

	var cmd tea.Cmd
	m, cmd = m.openSessionPicker(true, true)
	if cmd != nil {
		t.Fatalf("openSessionPicker cmd = %T, want nil", cmd)
	}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd != nil {
		t.Fatalf("resume selection cmd = %T, want nil", cmd)
	}
	if len(agent.startupResumedSessionIDs) != 1 || agent.startupResumedSessionIDs[0] != "session-startup" {
		t.Fatalf("startupResumedSessionIDs = %#v, want session-startup", agent.startupResumedSessionIDs)
	}
	if len(agent.resumedSessionIDs) != 0 {
		t.Fatalf("resumedSessionIDs = %#v, want none", agent.resumedSessionIDs)
	}
	if m.sessionPicker != nil {
		t.Fatal("startup session picker should close after --all resume")
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
