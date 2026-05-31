package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/stdio"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newLedgerCommandTestAgent(store *taskstate.Store) (*Agent, *bytes.Buffer) {
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

func newLedgerCommandStore(t *testing.T) *taskstate.Store {
	t.Helper()
	root := t.TempDir()
	return taskstate.NewStoreWithWorkspace(root, root)
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

func TestLedgerCommand_WithArgumentsPrintsUsage(t *testing.T) {
	for _, surface := range []commandcatalog.CommandSurface{
		commandcatalog.CommandSurfaceClassic,
		commandcatalog.CommandSurfaceTUI,
	} {
		t.Run(string(surface), func(t *testing.T) {
			agent, out := newLedgerCommandTestAgent(newLedgerCommandStore(t))
			if !handleSpecialCommandForSurface("/ledger foo", agent, surface) {
				t.Fatalf("/ledger foo was not handled for surface=%s", surface)
			}

			output := out.String()
			assertLedgerOutputContains(t, output, "Usage: /ledger")
			assertLedgerOutputOmits(t, output, "Runtime task ledger")
		})
	}
}

func TestLedgerCommand_NilLedgerPrintsEmptyLedger(t *testing.T) {
	agent, out := newLedgerCommandTestAgent(nil)
	if !handleLedgerCommand(agent, nil) {
		t.Fatal("handleLedgerCommand returned false")
	}
	assertLedgerOutputContains(t, out.String(), "No evidence recorded.")
}

func TestLedgerCommand_NilRuntimePrintsEmptyLedger(t *testing.T) {
	var out bytes.Buffer
	stdio.SetDefaults(strings.NewReader(""), &out, &out)
	t.Cleanup(func() { stdio.SetDefaults(nil, nil, nil) })

	if !handleLedgerCommand(&Agent{}, nil) {
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
		"internal/taskstate/taskstate.go:L20-L22 | source: read_file | tool_call_id: call_123",
		"excerpt: type RuntimeTaskState struct { ChangedFiles ChangedFiles }",
		"internal/agent/runtime.go | reason: owner boundary | source: read_file | tool_call_id: call_123",
		"command: go test ./internal/agent | status: failed | exit code: 1 | excerpt: FAIL internal/agent panic trace",
		"command: go test ./internal/taskstate | status: passed | exit code: 0 | excerpt: ok internal/taskstate",
	)
	assertLedgerOutputOmits(t, output, "RuntimeTaskState struct {\nChangedFiles")
}

func TestLedgerCommand_RendersProviderHistoryRehydrateCandidates(t *testing.T) {
	root := t.TempDir()
	path := "internal/agent/provider_history.go"
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("mkdir rehydrate candidate parent: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("ACTUAL_REHYDRATED_CONTENT\n"), 0o644); err != nil {
		t.Fatalf("write rehydrate candidate file: %v", err)
	}
	store := taskstate.NewStoreWithRoot(root)
	agent, out := newLedgerCommandTestAgent(store)
	agent.Runtime.LastProviderHistoryProjectionReport = recordProviderHistoryRehydratePlanFixture(store, path, 7, 18)

	if !handleSpecialCommandForSurface("/ledger", agent, commandcatalog.CommandSurfaceClassic) {
		t.Fatal("/ledger was not handled")
	}

	output := out.String()
	assertLedgerOutputContains(t, output,
		"Rehydrate candidates:",
		"internal/agent/provider_history.go:L7-L18",
		"source: read_file | reason: edit_target_missing_recent_evidence | stale: false",
	)
	assertLedgerOutputOmits(t, output, "ACTUAL_REHYDRATED_CONTENT")
}

func populateLedgerCommandStore(store *taskstate.Store) {
	recorder := store.Recorder()
	recorder.RecordChangedFile("src/main.go")
	recorder.RecordTouchedFile("docs/commands.md")
	recorder.RecordToolObservation(taskstate.ToolObservation{
		ToolName:   "read_file",
		ToolCallID: "call_123",
		Structured: &tools.RuntimeObservation{
			Evidence: []tools.ObservationEvidence{{
				Path:      "internal/taskstate/taskstate.go",
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
	recorder.SetLastFailedTests([]taskstate.TestResult{
		taskstate.NewTestResultWithExitCode("go test ./internal/agent", 1, "failed", "FAIL internal/agent\npanic trace"),
	})
	recorder.SetLastPassedTests([]taskstate.TestResult{
		taskstate.NewTestResultWithExitCode("go test ./internal/taskstate", 0, "passed", "ok internal/taskstate"),
	})
}

func TestLedgerCommand_ShortensLongExcerpts(t *testing.T) {
	const hiddenTail = "SHOULD_NOT_APPEAR_IN_LEDGER_OUTPUT"
	store := newLedgerCommandStore(t)
	longEvidence := strings.Repeat("evidence ", 80) + hiddenTail
	longTest := strings.Repeat("failure ", 80) + hiddenTail
	recorder := store.Recorder()
	recorder.RecordToolObservation(taskstate.ToolObservation{
		ToolName:   "read_file",
		ToolCallID: "call_long",
		Structured: &tools.RuntimeObservation{
			Evidence: []tools.ObservationEvidence{{
				Path:      "internal/taskstate/taskstate.go",
				StartLine: 30,
				EndLine:   31,
				Excerpt:   longEvidence,
			}},
		},
	})
	recorder.SetLastFailedTests([]taskstate.TestResult{
		taskstate.NewTestResultWithExitCode("go test ./...", 1, "failed", longTest),
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
	for _, input := range []string{"/ledger", "/ledger foo"} {
		t.Run(input, func(t *testing.T) {
			store := newLedgerCommandStore(t)
			store.Recorder().RecordChangedFile("src/main.go")
			agent, _ := newLedgerCommandTestAgent(store)
			agent.Runtime.LastProviderHistoryProjectionReport = recordProviderHistoryRehydratePlanFixture(store, "src/main.go", 1, 2)
			agent.History = []api.Message{{Role: "user", Content: "hello"}}
			agent.session.Messages = []history.MessageEntry{{Role: "assistant", Content: "persisted"}}
			beforeHistory := append([]api.Message(nil), agent.History...)
			beforeMessages := append([]history.MessageEntry(nil), agent.session.Messages...)

			if !handleSpecialCommandForSurface(input, agent, commandcatalog.CommandSurfaceClassic) {
				t.Fatalf("%s was not handled", input)
			}

			if !reflect.DeepEqual(agent.History, beforeHistory) {
				t.Fatalf("History mutated: got %#v, want %#v", agent.History, beforeHistory)
			}
			if !reflect.DeepEqual(agent.session.Messages, beforeMessages) {
				t.Fatalf("session.Messages mutated: got %#v, want %#v", agent.session.Messages, beforeMessages)
			}
		})
	}
}
