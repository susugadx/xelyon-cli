package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
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

func TestHandleLoadCommand_RestoresPendingApprovedPlanForNextTurn(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	storage, err := history.NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	session := history.NewSession("test-model")
	session.AddMessage("user", "loaded session", "test-model")
	session.PendingApprovedPlan = "Implementation Plan\n1. Restore the approved plan"
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
	agent.PendingApprovedPlan = "Implementation Plan\n1. Stale local plan"
	agent.storage = storage

	if !handleLoadCommand(agent, []string{session.ID}) {
		t.Fatal("handleLoadCommand() = false, want true")
	}
	if agent.PendingApprovedPlan != session.PendingApprovedPlan {
		t.Fatalf("PendingApprovedPlan = %q, want loaded session plan %q", agent.PendingApprovedPlan, session.PendingApprovedPlan)
	}

	req := &chatRequest{input: "new task"}
	if err := agent.executeChatRequest(context.Background(), req); err != nil {
		t.Fatalf("executeChatRequest() error = %v", err)
	}
	if req.approvedPlanHandoff != session.PendingApprovedPlan {
		t.Fatalf("approvedPlanHandoff = %q, want restored plan after /load", req.approvedPlanHandoff)
	}
	if len(capturedHistories) != 1 {
		t.Fatalf("capturedHistories = %d, want 1", len(capturedHistories))
	}
	lastUserMessage := capturedHistories[0][len(capturedHistories[0])-1].Content
	if !strings.Contains(lastUserMessage, "[APPROVED PLAN HANDOFF]") {
		t.Fatalf("approved plan handoff should be injected after /load, got %#v", capturedHistories[0])
	}
	if !strings.Contains(lastUserMessage, "Restore the approved plan") {
		t.Fatalf("expected loaded plan in handoff guidance, got %q", lastUserMessage)
	}
	if strings.Contains(lastUserMessage, "Stale local plan") {
		t.Fatalf("stale local pending plan should be replaced by loaded session state, got %q", lastUserMessage)
	}
}
