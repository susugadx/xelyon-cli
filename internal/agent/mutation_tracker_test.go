package agent

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/turnsupport"
)

type ledgerMutationTrackerFixture struct {
	root       string
	taskLedger *taskstate.Store
	agent      *Agent
	tracker    *MutationTracker
	state      turnsupport.MutationState
}

func newLedgerMutationTrackerFixture(t *testing.T) *ledgerMutationTrackerFixture {
	t.Helper()

	root := t.TempDir()
	taskLedger := taskstate.NewStoreWithRoot(root)
	agent := &Agent{
		Runtime: &AgentRuntime{TaskLedger: taskLedger},
		agentConversationState: agentConversationState{
			session: history.NewSession("test-model"),
		},
		agentWorkspaceState: agentWorkspaceState{
			changeStack: []tools.FileChange{},
		},
	}

	return &ledgerMutationTrackerFixture{
		root:       root,
		taskLedger: taskLedger,
		agent:      agent,
		tracker:    agent.mutationTracker(),
		state:      turnsupport.NewMutationState(),
	}
}

func assertAgentHistoryUnchanged(t *testing.T, agent *Agent) {
	t.Helper()

	if len(agent.History) != 0 {
		t.Fatalf("History len = %d, want 0", len(agent.History))
	}
	if got := len(agent.session.Messages); got != 0 {
		t.Fatalf("session messages len = %d, want 0", got)
	}
}

func TestAddPendingLSPFile(t *testing.T) {
	a := &Agent{}

	a.addPendingLSPFile("")
	if len(a.pendingLSPFiles) != 0 {
		t.Errorf("expected 0 files after adding empty path, got %d", len(a.pendingLSPFiles))
	}

	a.addPendingLSPFile("/src/main.go")
	if len(a.pendingLSPFiles) != 1 || a.pendingLSPFiles[0] != "/src/main.go" {
		t.Errorf("expected [/src/main.go], got %v", a.pendingLSPFiles)
	}

	a.addPendingLSPFile("/src/main.go")
	if len(a.pendingLSPFiles) != 1 {
		t.Errorf("expected 1 file after duplicate add, got %d", len(a.pendingLSPFiles))
	}

	a.addPendingLSPFile("/src/util.go")
	if len(a.pendingLSPFiles) != 2 {
		t.Errorf("expected 2 files, got %d", len(a.pendingLSPFiles))
	}
}

func TestAddPendingLSPFilesFromChange(t *testing.T) {
	t.Run("nil change is safe", func(t *testing.T) {
		a := &Agent{}
		a.addPendingLSPFilesFromChange(nil)
		if len(a.pendingLSPFiles) != 0 {
			t.Errorf("expected 0 files, got %d", len(a.pendingLSPFiles))
		}
	})

	t.Run("empty details is safe", func(t *testing.T) {
		a := &Agent{}
		a.addPendingLSPFilesFromChange(&tools.FileChange{})
		if len(a.pendingLSPFiles) != 0 {
			t.Errorf("expected 0 files, got %d", len(a.pendingLSPFiles))
		}
	})

	t.Run("single file from apply_patch", func(t *testing.T) {
		a := &Agent{}
		a.addPendingLSPFilesFromChange(&tools.FileChange{
			Tool: "apply_patch",
			Details: []tools.FileChangeDetail{
				{FilePath: "/src/main.go", Action: "modified"},
			},
		})
		if len(a.pendingLSPFiles) != 1 || a.pendingLSPFiles[0] != "/src/main.go" {
			t.Errorf("expected [/src/main.go], got %v", a.pendingLSPFiles)
		}
	})

	t.Run("multiple files from apply_patch", func(t *testing.T) {
		a := &Agent{}
		a.addPendingLSPFilesFromChange(&tools.FileChange{
			Tool: "apply_patch",
			Details: []tools.FileChangeDetail{
				{FilePath: "/src/main.go", Action: "modified"},
				{FilePath: "/src/util.go", Action: "created"},
				{FilePath: "/src/old.go", Action: "deleted"},
			},
		})
		if len(a.pendingLSPFiles) != 3 {
			t.Errorf("expected 3 files, got %d", len(a.pendingLSPFiles))
		}
	})

	t.Run("dedup with existing pending files", func(t *testing.T) {
		a := &Agent{}
		a.addPendingLSPFile("/src/main.go")
		a.addPendingLSPFilesFromChange(&tools.FileChange{
			Tool: "apply_patch",
			Details: []tools.FileChangeDetail{
				{FilePath: "/src/main.go", Action: "modified"},
				{FilePath: "/src/new.go", Action: "created"},
			},
		})
		if len(a.pendingLSPFiles) != 2 {
			t.Errorf("expected 2 files (dedup), got %d: %v", len(a.pendingLSPFiles), a.pendingLSPFiles)
		}
	})
}

func TestMutationTracker_RecordToolResult_UpdatesTurnMutationState(t *testing.T) {
	a := &Agent{
		agentWorkspaceState: agentWorkspaceState{
			changeStack: []tools.FileChange{},
		},
	}
	tracker := a.mutationTracker()
	state := turnsupport.NewMutationState()

	change := &tools.FileChange{
		FilePath: "/src/main.go",
		Tool:     "write_file",
		Details: []tools.FileChangeDetail{
			{FilePath: "/src/main.go", Action: "modified"},
			{FilePath: "/src/util.go", Action: "modified"},
		},
	}

	tracker.RecordToolResult(&tools.ToolCall{Tool: "write_file"}, "ok", change, &state)

	if !state.HasMutations() {
		t.Fatal("expected turn-local mutation state to be updated by file change event")
	}
	snapshot := state.Snapshot()
	if snapshot.MutationCount != 1 {
		t.Fatalf("MutationCount = %d, want 1", snapshot.MutationCount)
	}
	if len(snapshot.Files) != 2 {
		t.Fatalf("len(Files) = %d, want 2", len(snapshot.Files))
	}
	if snapshot.ProgressFingerprint == "" {
		t.Fatal("expected non-empty progress fingerprint")
	}
	if len(a.changeStack) != 1 {
		t.Fatalf("changeStack len = %d, want 1", len(a.changeStack))
	}
}

func TestMutationTracker_RecordToolResult_RecordsTaskLedgerWithoutChangingHistory(t *testing.T) {
	fixture := newLedgerMutationTrackerFixture(t)

	change := &tools.FileChange{
		FilePath: filepath.Join(fixture.root, "src/main.go"),
		Tool:     "apply_patch",
		Details: []tools.FileChangeDetail{
			{FilePath: filepath.Join(fixture.root, "src/main.go"), Action: "modified"},
			{FilePath: filepath.Join(fixture.root, "src/util.go"), Action: "created"},
			{FilePath: filepath.Join(fixture.root, "src/main.go"), Action: "modified"},
		},
	}

	fixture.tracker.RecordToolResult(&tools.ToolCall{Tool: "apply_patch"}, "ok", change, &fixture.state)

	if len(fixture.agent.changeStack) != 1 {
		t.Fatalf("changeStack len = %d, want 1", len(fixture.agent.changeStack))
	}
	if !fixture.state.HasMutations() {
		t.Fatal("expected turn-local mutation state to keep recording file changes")
	}
	assertAgentHistoryUnchanged(t, fixture.agent)

	paths := fixture.taskLedger.Snapshot().ChangedFiles.Paths()
	want := []string{"src/main.go", "src/util.go"}
	if len(paths) != len(want) {
		t.Fatalf("ledger changed paths = %v, want %v", paths, want)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("ledger changed paths = %v, want %v", paths, want)
		}
	}
}

func TestMutationTracker_RecordToolResultHelper_TracksLegacyPathWithoutChangingHistory(t *testing.T) {
	fixture := newLedgerMutationTrackerFixture(t)

	change := &tools.FileChange{
		FilePath: filepath.Join(fixture.root, "src/main.go"),
		Tool:     "write_file",
	}

	fixture.tracker.recordToolResult(toolResultRecord{
		toolCall:                &tools.ToolCall{Tool: "write_file"},
		result:                  "ok",
		change:                  change,
		turnMutations:           &fixture.state,
		trackProjectMapMutation: false,
	})

	if len(fixture.agent.changeStack) != 1 {
		t.Fatalf("changeStack len = %d, want 1", len(fixture.agent.changeStack))
	}
	if !fixture.state.HasMutations() {
		t.Fatal("expected turn-local mutation state to keep recording file changes")
	}
	assertAgentHistoryUnchanged(t, fixture.agent)

	paths := fixture.taskLedger.Snapshot().ChangedFiles.Paths()
	if !reflect.DeepEqual(paths, []string{"src/main.go"}) {
		t.Fatalf("ledger changed paths = %v, want [src/main.go]", paths)
	}
}

func TestMutationTracker_RecordToolResult_RecordsReadSearchBashFactsWithoutHistory(t *testing.T) {
	fixture := newLedgerMutationTrackerFixture(t)

	fixture.tracker.RecordToolResult(&tools.ToolCall{
		ID:   "read-1",
		Tool: "read_file",
	}, "📄 File: internal/taskstate/taskstate.go\n7: type RuntimeTaskState struct {}", nil, &fixture.state)
	fixture.tracker.RecordToolResult(&tools.ToolCall{
		ID:   "search-1",
		Tool: "search_code",
	}, strings.Join([]string{
		"Found 1 match(es) in 1 file(s)",
		"",
		"📄 internal/agent/agent.go (1 match(es)) [L1]",
		"  [ref]     >   12 │ RuntimeTaskState",
		"",
		"Recommended reads:",
		"  - internal/agent/runtime.go:26 | runtime owner",
	}, "\n"), nil, &fixture.state)
	fixture.tracker.RecordToolResult(&tools.ToolCall{
		Tool: "bash",
		Args: map[string]string{"command": "go test ./internal/taskstate"},
	}, "ok\ninternal/taskstate/ledger_test.go:10: pass", nil, &fixture.state)

	if fixture.state.HasMutations() {
		t.Fatal("read/search/bash observations without FileChange must not update turn mutation state")
	}
	if len(fixture.agent.changeStack) != 0 {
		t.Fatalf("changeStack len = %d, want 0", len(fixture.agent.changeStack))
	}
	assertAgentHistoryUnchanged(t, fixture.agent)

	snapshot := fixture.taskLedger.Snapshot()
	wantTouched := []string{"internal/taskstate/taskstate.go", "internal/agent/agent.go", "internal/taskstate/ledger_test.go"}
	if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, wantTouched) {
		t.Fatalf("ledger touched paths = %v, want %v", got, wantTouched)
	}
	if got := snapshot.Evidence.Items(); len(got) != 2 || got[0].ToolCallID() != "read-1" || got[1].Source() != "search_code" {
		t.Fatalf("ledger evidence = %#v", got)
	}
	if got := snapshot.RecommendedReads.Items(); len(got) != 1 || got[0].Path() != "internal/agent/runtime.go" {
		t.Fatalf("ledger recommended reads = %#v", got)
	}
	if got := snapshot.LastPassedTests.Results(); len(got) != 1 || got[0].Command() != "go test ./internal/taskstate" {
		t.Fatalf("ledger passed tests = %#v", got)
	}
}

func TestMutationTracker_RecordToolExecutionResult_RecordsStructuredObservationWithoutHistory(t *testing.T) {
	fixture := newLedgerMutationTrackerFixture(t)

	fixture.tracker.RecordToolExecutionResult(&tools.ToolCall{
		ID:   "structured-search",
		Tool: "search_code",
	}, toolruntime.Result{
		Result: "rendered output is not history",
		Observation: &tools.RuntimeObservation{
			TouchedFiles: []tools.ObservationPath{{
				Path: "internal/taskstate/taskstate.go",
			}},
			Evidence: []tools.ObservationEvidence{{
				Path:      "internal/taskstate/taskstate.go",
				StartLine: 22,
				EndLine:   22,
				Excerpt:   "type RuntimeTaskState struct {",
			}},
		},
	}, &fixture.state)

	if fixture.state.HasMutations() {
		t.Fatal("structured observation without FileChange must not update turn mutation state")
	}
	if len(fixture.agent.changeStack) != 0 {
		t.Fatalf("changeStack len = %d, want 0", len(fixture.agent.changeStack))
	}
	assertAgentHistoryUnchanged(t, fixture.agent)

	snapshot := fixture.taskLedger.Snapshot()
	if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, []string{"internal/taskstate/taskstate.go"}) {
		t.Fatalf("ledger touched paths = %v, want [internal/taskstate/taskstate.go]", got)
	}
	evidence := snapshot.Evidence.Items()
	if len(evidence) != 1 ||
		evidence[0].Path() != "internal/taskstate/taskstate.go" ||
		evidence[0].ToolCallID() != "structured-search" {
		t.Fatalf("ledger evidence = %#v", evidence)
	}
}

func TestMutationTracker_RecordToolExecutionResult_UsesExecutionErrorForBashLedger(t *testing.T) {
	fixture := newLedgerMutationTrackerFixture(t)

	fixture.tracker.RecordToolExecutionResult(&tools.ToolCall{
		Tool: "bash",
		Args: map[string]string{"command": "go test ./internal/agent"},
	}, toolruntime.Result{
		Result: "exit status 1",
		Error:  true,
	}, &fixture.state)

	snapshot := fixture.taskLedger.Snapshot()
	if got := snapshot.LastPassedTests.Results(); len(got) != 0 {
		t.Fatalf("LastPassedTests = %#v, want empty", got)
	}
	failed := snapshot.LastFailedTests.Results()
	if len(failed) != 1 || failed[0].Command() != "go test ./internal/agent" || failed[0].Status() != "failed" {
		t.Fatalf("LastFailedTests = %#v, want failed go test observation", failed)
	}
}

func TestMutationTracker_RecordToolResult_UsesInvocationCWDForRelativeLedgerPaths(t *testing.T) {
	root := t.TempDir()
	invocationCWD := filepath.Join(root, "pkg")
	taskLedger := taskstate.NewStoreWithRoot(root)
	a := &Agent{
		Runtime: &AgentRuntime{
			InvocationCWD: invocationCWD,
			TaskLedger:    taskLedger,
		},
		agentConversationState: agentConversationState{
			session: history.NewSession("test-model"),
		},
	}
	tracker := a.mutationTracker()
	state := turnsupport.NewMutationState()

	tracker.RecordToolResult(&tools.ToolCall{
		ID:   "read-1",
		Tool: "read_file",
	}, "📄 File: foo.go\n7: type RuntimeTaskState struct {}", nil, &state)

	snapshot := taskLedger.Snapshot()
	if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, []string{"pkg/foo.go"}) {
		t.Fatalf("ledger touched paths = %v, want [pkg/foo.go]", got)
	}
	if got := snapshot.Evidence.Items(); len(got) != 1 || got[0].Path() != "pkg/foo.go" {
		t.Fatalf("ledger evidence = %#v, want one item for pkg/foo.go", got)
	}
}

func TestTrackDeferredDiagnostics_StrReplacePrefersFileChangePath(t *testing.T) {
	t.Run("prefers detail path", func(t *testing.T) {
		a := &Agent{}
		tracker := a.mutationTracker()
		change := &tools.FileChange{
			FilePath: "display/path.go",
			Details: []tools.FileChangeDetail{
				{FilePath: "/resolved/detail.go"},
			},
		}

		tracker.trackDeferredDiagnostics(&tools.ToolCall{
			Tool: "str_replace",
			Args: map[string]string{"path": "args/fallback.go"},
		}, "Successfully replaced", change)

		if len(a.pendingLSPFiles) != 1 || a.pendingLSPFiles[0] != "/resolved/detail.go" {
			t.Fatalf("pendingLSPFiles = %v, want [/resolved/detail.go]", a.pendingLSPFiles)
		}
	})

	t.Run("falls back to change file path", func(t *testing.T) {
		a := &Agent{}
		tracker := a.mutationTracker()
		change := &tools.FileChange{FilePath: "/resolved/from-change.go"}

		tracker.trackDeferredDiagnostics(&tools.ToolCall{
			Tool: "str_replace",
			Args: map[string]string{"path": "args/fallback.go"},
		}, "Successfully replaced", change)

		if len(a.pendingLSPFiles) != 1 || a.pendingLSPFiles[0] != "/resolved/from-change.go" {
			t.Fatalf("pendingLSPFiles = %v, want [/resolved/from-change.go]", a.pendingLSPFiles)
		}
	})

	t.Run("falls back to tool args when change missing", func(t *testing.T) {
		a := &Agent{}
		tracker := a.mutationTracker()

		tracker.trackDeferredDiagnostics(&tools.ToolCall{
			Tool: "str_replace",
			Args: map[string]string{"path": "args/fallback.go"},
		}, "Successfully replaced", nil)

		if len(a.pendingLSPFiles) != 1 || a.pendingLSPFiles[0] != "args/fallback.go" {
			t.Fatalf("pendingLSPFiles = %v, want [args/fallback.go]", a.pendingLSPFiles)
		}
	})
}
