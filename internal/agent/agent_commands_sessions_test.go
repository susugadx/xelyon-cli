package agent

import (
	"bytes"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestHandleSaveAndSessionsCommand_UseRuntimeOutput(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	storage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	session := history.NewSession("test-model")
	session.AddMessage("user", "hello", "test-model")

	var out bytes.Buffer
	agent := &Agent{
		session: session,
		storage: storage,
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	if !handleSaveCommand(agent) {
		t.Fatal("handleSaveCommand() = false, want true")
	}
	if !handleSessionsCommand(agent) {
		t.Fatal("handleSessionsCommand() = false, want true")
	}

	output := out.String()
	if !strings.Contains(output, "Session saved") {
		t.Fatalf("expected runtime output to contain save message, got %q", output)
	}
	if !strings.Contains(output, "Recent Sessions") {
		t.Fatalf("expected runtime output to contain sessions header, got %q", output)
	}
	if !strings.Contains(output, session.ID) {
		t.Fatalf("expected runtime output to contain session ID, got %q", output)
	}
}

func TestHandleLoadCommand_UsesRuntimeOutput(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	storage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	session := history.NewSession("test-model")
	session.AddMessage("user", "hello", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var out bytes.Buffer
	agent := &Agent{
		storage:         storage,
		CurrentProvider: &mockProvider{name: "test"},
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
	}

	if !handleLoadCommand(agent, []string{session.ID}) {
		t.Fatal("handleLoadCommand() = false, want true")
	}

	if agent.session == nil || agent.session.ID != session.ID {
		t.Fatalf("expected loaded session ID %q, got %#v", session.ID, agent.session)
	}
	if !strings.Contains(out.String(), "Loaded session") {
		t.Fatalf("expected runtime output to contain load message, got %q", out.String())
	}
}
