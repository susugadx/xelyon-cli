package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

func newTaskLedgerWithPassedTest(t *testing.T) *taskstate.Store {
	t.Helper()
	store := taskstate.NewStoreWithRoot(t.TempDir())
	store.Recorder().SetLastPassedTests([]taskstate.TestResult{
		taskstate.NewTestResultWithExitCode("go test ./internal/taskstate", 0, "passed", "ok"),
	})
	return store
}

func assertTaskLedgerReset(t *testing.T, agent *Agent, action string) {
	t.Helper()
	if !agent.Runtime.TaskLedger.Snapshot().IsEmpty() {
		t.Fatalf("task ledger should be reset after %s: %#v", action, agent.Runtime.TaskLedger.Snapshot())
	}
}

func assertTaskLedgerPreserved(t *testing.T, agent *Agent, action string) {
	t.Helper()
	if agent.Runtime.TaskLedger.Snapshot().IsEmpty() {
		t.Fatalf("task ledger was reset after %s", action)
	}
}
