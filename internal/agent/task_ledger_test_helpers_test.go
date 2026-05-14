package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ledger"
)

func newTaskLedgerWithPassedTest(t *testing.T) *ledger.Store {
	t.Helper()
	store := ledger.NewStoreWithRoot(t.TempDir())
	store.Recorder().SetLastPassedTests([]ledger.TestResult{
		ledger.NewTestResultWithExitCode("go test ./internal/ledger", 0, "passed", "ok"),
	})
	return store
}
