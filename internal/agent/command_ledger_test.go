package agent

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/ledger"
	"github.com/susugadx/xelyon-cli/internal/stdio"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newLedgerCommandTestAgent(store *ledger.Store) (*Agent, *bytes.Buffer) {
	out := &bytes.Buffer{}
	return &Agent{
		Runtime: &AgentRuntime{
			TaskLedger: store,
			UI:         ui.NewRuntime(strings.NewReader(""), out, out),
		},
		agentConversationState: agentConversationState{
			session: history.NewSession("gpt-test"),
		},
	}, out
}

func newLedgerCommandStore(t *testing.T) *ledger.Store {
	t.Helper()
	root := t.TempDir()
	return ledger.NewStoreWithWorkspace(root, root)
}

func assertLedgerOutputContains(t *testing.T, output string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(output, fragment) {
			t.Fatalf("/ledger output missing %q:\n%s", fragment, output)
		}
	}
}

func assertLedgerOutputOmits(t *testing.T, output string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if strings.Contains(output, fragment) {
			t.Fatalf("/ledger output should not contain %q:\n%s", fragment, output)
		}
	}
}

func TestLedgerCommand_EmptyLedgerPrintsAllSections(t *testing.T) {
	for _, surface := range []commandcatalog.CommandSurface{
		commandcatalog.CommandSurfaceClassic,
		commandcatalog.CommandSurfaceTUI,
	} {
		t.Run(string(surface), func(t *testing.T) {
			agent, out := newLedgerCommandTestAgent(newLedgerCommandStore(t))
			if !handleSpecialCommandForSurface("/ledger", agent, surface) {
				t.Fatalf("/ledger was not handled for surface=%s", surface)
			}

			assertLedgerOutputContains(t, out.String(),
				"Runtime task ledger",
				"Changed files:",
				"No changed files recorded.",
				"Touched files:",
				"No touched files recorded.",
				"Evidence:",
				"No evidence recorded.",
				"Recommended reads:",
				"No recommended reads recorded.",
				"Last failed tests:",
				"No failed tests recorded.",
				"Last passed tests:",
				"No passed tests recorded.",
			)
		})
	}
}

func TestLedgerCommand_NilLedgerPrintsEmptyLedger(t *testing.T) {
	agent, out := newLedgerCommandTestAgent(nil)
	if !handleLedgerCommand(agent) {
		t.Fatal("handleLedgerCommand returned false")
	}
	assertLedgerOutputContains(t, out.String(), "No evidence recorded.")
}

func TestLedgerCommand_NilRuntimePrintsEmptyLedger(t *testing.T) {
	var out bytes.Buffer
	stdio.SetDefaults(strings.NewReader(""), &out, &out)
	t.Cleanup(func() { stdio.SetDefaults(nil, nil, nil) })

	if !handleLedgerCommand(&Agent{}) {
		t.Fatal("handleLedgerCommand returned false")
	}
	assertLedgerOutputContains(t, out.String(), "No changed files recorded.")
}

func TestLedgerCommand_PopulatedLedgerRendersSnapshot(t *testing.T) {
	store := newLedgerCommandStore(t)
	populateLedgerCommandStore(store)

	agent, out := newLedgerCommandTestAgent(store)
	if !handleSpecialCommandForSurface("/ledger", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/ledger was not handled")
	}

	output := out.String()
	assertLedgerOutputContains(t, output,
		"  - src/main.go",
		"  - docs/commands.md",
		"internal/ledger/ledger.go:L20-L22 | source: read_file | tool_call_id: call_123",
		"excerpt: type RuntimeTaskState struct { ChangedFiles ChangedFiles }",
		"internal/agent/runtime.go | reason: owner boundary | source: read_file | tool_call_id: call_123",
		"command: go test ./internal/agent | status: failed | exit code: 1 | excerpt: FAIL internal/agent panic trace",
		"command: go test ./internal/ledger | status: passed | exit code: 0 | excerpt: ok internal/ledger",
	)
	assertLedgerOutputOmits(t, output, "RuntimeTaskState struct {\nChangedFiles")
}

func populateLedgerCommandStore(store *ledger.Store) {
	recorder := store.Recorder()
	recorder.RecordChangedFile("src/main.go")
	recorder.RecordTouchedFile("docs/commands.md")
	recorder.RecordToolObservation(ledger.ToolObservation{
		ToolName:   "read_file",
		ToolCallID: "call_123",
		Structured: &tools.RuntimeObservation{
			Evidence: []tools.ObservationEvidence{{
				Path:      "internal/ledger/ledger.go",
				StartLine: 20,
				EndLine:   22,
				Excerpt:   "type RuntimeTaskState struct {\nChangedFiles ChangedFiles\n}",
			}},
			RecommendedReads: []tools.ObservationRecommendedRead{{
				Path:   "internal/agent/runtime.go",
				Reason: "owner boundary",
			}},
		},
	})
	recorder.SetLastFailedTests([]ledger.TestResult{
		ledger.NewTestResultWithExitCode("go test ./internal/agent", 1, "failed", "FAIL internal/agent\npanic trace"),
	})
	recorder.SetLastPassedTests([]ledger.TestResult{
		ledger.NewTestResultWithExitCode("go test ./internal/ledger", 0, "passed", "ok internal/ledger"),
	})
}

func TestLedgerCommand_ShortensLongExcerpts(t *testing.T) {
	const hiddenTail = "SHOULD_NOT_APPEAR_IN_LEDGER_OUTPUT"
	store := newLedgerCommandStore(t)
	longEvidence := strings.Repeat("evidence ", 80) + hiddenTail
	longTest := strings.Repeat("failure ", 80) + hiddenTail
	recorder := store.Recorder()
	recorder.RecordToolObservation(ledger.ToolObservation{
		ToolName:   "read_file",
		ToolCallID: "call_long",
		Structured: &tools.RuntimeObservation{
			Evidence: []tools.ObservationEvidence{{
				Path:      "internal/ledger/ledger.go",
				StartLine: 30,
				EndLine:   31,
				Excerpt:   longEvidence,
			}},
		},
	})
	recorder.SetLastFailedTests([]ledger.TestResult{
		ledger.NewTestResultWithExitCode("go test ./...", 1, "failed", longTest),
	})

	agent, out := newLedgerCommandTestAgent(store)
	if !handleSpecialCommandForSurface("/ledger", agent, commandcatalog.CommandSurfaceTUI) {
		t.Fatal("/ledger was not handled")
	}

	output := out.String()
	assertLedgerOutputOmits(t, output, hiddenTail)
	assertLedgerOutputContains(t, output, "excerpt: evidence evidence", "excerpt: failure failure", "...")
}

func TestLedgerCommand_DoesNotMutateConversationState(t *testing.T) {
	store := newLedgerCommandStore(t)
	store.Recorder().RecordChangedFile("src/main.go")
	agent, _ := newLedgerCommandTestAgent(store)
	agent.History = []api.Message{{Role: "user", Content: "hello"}}
	agent.session.Messages = []history.MessageEntry{{Role: "assistant", Content: "persisted"}}
	beforeHistory := append([]api.Message(nil), agent.History...)
	beforeMessages := append([]history.MessageEntry(nil), agent.session.Messages...)

	if !handleSpecialCommandForSurface("/ledger", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/ledger was not handled")
	}

	if !reflect.DeepEqual(agent.History, beforeHistory) {
		t.Fatalf("History mutated: got %#v, want %#v", agent.History, beforeHistory)
	}
	if !reflect.DeepEqual(agent.session.Messages, beforeMessages) {
		t.Fatalf("session.Messages mutated: got %#v, want %#v", agent.session.Messages, beforeMessages)
	}
}
