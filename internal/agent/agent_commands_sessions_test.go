package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
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
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
		agentConversationState: agentConversationState{
			session: session,
			storage: storage,
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
		CurrentProvider: &mockProvider{name: "test"},
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
		},
		agentConversationState: agentConversationState{
			storage: storage,
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

func TestHandleLoadCommand_DoesNotInjectLegacyPlanMarker(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	storage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	session := history.NewSession("test-model")
	session.AddMessage("user", "loaded session", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var out bytes.Buffer
	var capturedHistories [][]api.Message
	provider := &scriptedChatProvider{
		name:            "test",
		functionCalling: true,
		chatWithToolsFn: func(call int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			snapshot := append([]api.Message(nil), history...)
			capturedHistories = append(capturedHistories, snapshot)
			return "done", nil
		},
	}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.storage = storage

	if !handleLoadCommand(agent, []string{session.ID}) {
		t.Fatal("handleLoadCommand() = false, want true")
	}

	req := &chatRequest{input: "continue"}
	if err := agent.executeChatRequest(context.Background(), req); err != nil {
		t.Fatalf("executeChatRequest() error = %v", err)
	}
	if len(capturedHistories) != 1 {
		t.Fatalf("capturedHistories = %d, want 1", len(capturedHistories))
	}
	lastUserMessage := capturedHistories[0][len(capturedHistories[0])-1].Content
	if strings.Contains(lastUserMessage, "[APPROVED PLAN HANDOFF]") {
		t.Fatalf("normal mode request should not inject legacy plan marker after /load, got %#v", capturedHistories[0])
	}
}

func TestParseResumeCommandArgs_RejectsConflictingOptions(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "last all",
			args:    []string{"--last", "--all"},
			wantErr: "--last cannot be used with --all",
		},
		{
			name:    "last id",
			args:    []string{"--last", "session-1"},
			wantErr: "--last cannot be used with a session ID",
		},
		{
			name:    "all id",
			args:    []string{"--all", "session-1"},
			wantErr: "--all cannot be used with a session ID",
		},
		{
			name:    "duplicate ids",
			args:    []string{"session-1", "session-2"},
			wantErr: "usage: /resume [--last|--all|session-id]",
		},
		{
			name:    "unknown option",
			args:    []string{"--unknown"},
			wantErr: "usage: /resume [--last|--all|session-id]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseResumeCommandArgs(tt.args)
			if err == nil {
				t.Fatal("parseResumeCommandArgs() error = nil, want error")
			}
			if got := err.Error(); got != tt.wantErr {
				t.Fatalf("parseResumeCommandArgs() error = %q, want %q", got, tt.wantErr)
			}
		})
	}
}

func TestParseResumeCommandArgs_ValidForms(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantAll       bool
		wantSessionID string
	}{
		{name: "empty"},
		{name: "last", args: []string{"--last"}},
		{name: "all", args: []string{"--all"}, wantAll: true},
		{name: "id", args: []string{"session-1"}, wantSessionID: "session-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, sessionID, err := parseResumeCommandArgs(tt.args)
			if err != nil {
				t.Fatalf("parseResumeCommandArgs() error = %v", err)
			}
			if opts.All != tt.wantAll {
				t.Fatalf("opts.All = %v, want %v", opts.All, tt.wantAll)
			}
			if sessionID != tt.wantSessionID {
				t.Fatalf("sessionID = %q, want %q", sessionID, tt.wantSessionID)
			}
		})
	}
}

func TestHandleNewAndClearCommandsRejectArgs(t *testing.T) {
	disableColors(t)

	tests := []struct {
		name      string
		input     string
		wantUsage string
	}{
		{name: "new", input: "/new typo", wantUsage: "usage: /new"},
		{name: "clear", input: "/clear typo", wantUsage: "usage: /clear"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			agent := newChatRequestTestAgent(t, &mockProvider{name: "openai"}, &out)
			sessionID := agent.session.ID

			if !handleSpecialCommandForSurface(tt.input, agent, commandcatalog.CommandSurfaceTUI) {
				t.Fatalf("handleSpecialCommandForSurface(%q) = false, want true", tt.input)
			}
			if agent.session.ID != sessionID {
				t.Fatalf("agent.session.ID = %q, want unchanged %q", agent.session.ID, sessionID)
			}
			if !strings.Contains(out.String(), tt.wantUsage) {
				t.Fatalf("output = %q, want %q", out.String(), tt.wantUsage)
			}
		})
	}
}
