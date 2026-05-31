package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

func TestAgent_RehydrateEvidencePointer_DoesNotMutateHistoryOrSession(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "evidence.go"), []byte("line 1\nline 2\n"), 0o644); err != nil {
		t.Fatalf("write evidence file: %v", err)
	}
	store := taskstate.NewStoreWithRoot(root)
	session := history.NewSession("gpt-5.4")
	session.AddMessage("user", "persisted", "gpt-5.4")
	agent := &Agent{
		Runtime: &AgentRuntime{
			TaskLedger: store,
		},
		History: []api.Message{{Role: "user", Content: "live"}},
		agentConversationState: agentConversationState{
			session: session,
		},
	}
	beforeHistory := append([]api.Message(nil), agent.History...)
	beforeSessionMessages := append([]history.MessageEntry(nil), session.Messages...)

	result, err := agent.RehydrateEvidencePointer(context.Background(), taskstate.EvidencePointer{
		Path:      "evidence.go",
		StartLine: 2,
		EndLine:   2,
	})
	if err != nil {
		t.Fatalf("RehydrateEvidencePointer() error = %v", err)
	}
	if result.Content != "line 2" {
		t.Fatalf("rehydrated content = %q, want line 2", result.Content)
	}
	if !reflect.DeepEqual(agent.History, beforeHistory) {
		t.Fatalf("History mutated:\ngot:  %#v\nwant: %#v", agent.History, beforeHistory)
	}
	if !reflect.DeepEqual(session.Messages, beforeSessionMessages) {
		t.Fatalf("session.Messages mutated:\ngot:  %#v\nwant: %#v", session.Messages, beforeSessionMessages)
	}
}

func TestAgent_RehydrateEvidencePointer_LedgerPointerDoesNotRequireInvocationCWD(t *testing.T) {
	root := t.TempDir()
	invocationCWD := filepath.Join(root, "pkg")
	if err := os.MkdirAll(invocationCWD, 0o755); err != nil {
		t.Fatalf("mkdir invocation cwd: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "evidence.go"), []byte("root line\n"), 0o644); err != nil {
		t.Fatalf("write evidence file: %v", err)
	}
	store := taskstate.NewStoreWithWorkspace(root, invocationCWD)
	store.Recorder().RecordToolObservation(taskstate.ToolObservation{
		ToolName: "read_file",
		Result:   "📄 File: evidence.go\n1: root line",
	})
	pointers := taskstate.EvidencePointersFromState(store.Snapshot())
	if len(pointers) != 1 ||
		pointers[0].Path != "evidence.go" ||
		pointers[0].PathBase != taskstate.EvidencePointerPathBaseRepoRoot {
		t.Fatalf("EvidencePointersFromState() = %#v, want one repo-relative pointer", pointers)
	}
	if err := os.RemoveAll(invocationCWD); err != nil {
		t.Fatalf("remove invocation cwd: %v", err)
	}
	agent := &Agent{
		Runtime: &AgentRuntime{
			TaskLedger: store,
		},
	}

	result, err := agent.RehydrateEvidencePointer(context.Background(), pointers[0])
	if err != nil {
		t.Fatalf("RehydrateEvidencePointer() error = %v", err)
	}
	if result.Path != "evidence.go" || result.Content != "root line" {
		t.Fatalf("rehydrated result = %#v, want root evidence.go content", result)
	}
}

func TestAgent_RehydrateEvidencePointer_NilRuntimeOrLedgerReturnsStructuredError(t *testing.T) {
	tests := []struct {
		name  string
		agent *Agent
	}{
		{name: "nil agent"},
		{name: "nil runtime", agent: &Agent{}},
		{name: "nil ledger", agent: &Agent{Runtime: &AgentRuntime{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.agent.RehydrateEvidencePointer(context.Background(), taskstate.EvidencePointer{
				Path:      "evidence.go",
				StartLine: 1,
				EndLine:   1,
			})
			if result.Reason != taskstate.EvidenceRehydrateReasonWorkspaceUnavailable {
				t.Fatalf("result reason = %q, want workspace_unavailable", result.Reason)
			}
			var rehydrateErr *taskstate.EvidenceRehydrateError
			if !errors.As(err, &rehydrateErr) {
				t.Fatalf("error = %T %v, want *EvidenceRehydrateError", err, err)
			}
			if rehydrateErr.Reason != taskstate.EvidenceRehydrateReasonWorkspaceUnavailable {
				t.Fatalf("error reason = %q, want workspace_unavailable", rehydrateErr.Reason)
			}
		})
	}
}
