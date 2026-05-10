package ledger

import (
	"reflect"
	"strings"
	"testing"
)

func TestRecorder_RecordToolObservation_BashTestsAndTouchedPaths(t *testing.T) {
	root := t.TempDir()
	store := NewStoreWithRoot(root)
	recorder := store.Recorder()

	recorder.RecordToolObservation(ToolObservation{
		ToolName: "bash",
		Args:     map[string]string{"command": "go test ./internal/ledger"},
		Result:   "ok\tgithub.com/susugadx/xelyon-cli/internal/ledger\t0.01s\ninternal/ledger/ledger.go:12: ok",
	})
	recorder.RecordToolObservation(ToolObservation{
		ToolName: "bash",
		Args:     map[string]string{"command": "go test -v ./internal/ledger"},
		Result:   "=== RUN   TestErrorHandling\n--- PASS: TestErrorHandling (0.00s)\nPASS",
	})
	recorder.RecordToolObservation(ToolObservation{
		ToolName: "bash",
		Args:     map[string]string{"command": "go test ./internal/agent"},
		Result:   "Error: exit status 1\nOutput: internal/agent/agent.go:20: failure",
		Error:    true,
	})
	recorder.RecordToolObservation(ToolObservation{
		ToolName: "bash",
		Args:     map[string]string{"command": "rg RuntimeTaskState"},
		Result:   "internal/ledger/ledger.go:11:type RuntimeTaskState struct {}",
	})

	snapshot := store.Snapshot()
	wantTouched := []string{"internal/ledger/ledger.go", "internal/agent/agent.go"}
	if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, wantTouched) {
		t.Fatalf("TouchedFiles.Paths() = %v, want %v", got, wantTouched)
	}
	if got := snapshot.LastPassedTests.Results(); len(got) != 2 ||
		got[0].Command() != "go test ./internal/ledger" ||
		got[1].Command() != "go test -v ./internal/ledger" ||
		got[1].ExitCode() != 0 {
		t.Fatalf("LastPassedTests = %#v", got)
	}
	if got := snapshot.LastFailedTests.Results(); len(got) != 1 || got[0].Command() != "go test ./internal/agent" || got[0].ExitCode() != 1 {
		t.Fatalf("LastFailedTests = %#v", got)
	}
}

func TestRecorder_RecordToolObservation_CancelledBashTestIsNotPassed(t *testing.T) {
	store := NewStoreWithRoot(t.TempDir())
	recorder := store.Recorder()

	recorder.RecordToolObservation(ToolObservation{
		ToolName: "bash",
		Args:     map[string]string{"command": "go test ./..."},
		Result:   "Cancelled by user",
	})
	recorder.RecordToolObservation(ToolObservation{
		ToolName: "bash",
		Args:     map[string]string{"command": "go test ./internal/ledger"},
		Result:   "Command interrupted.\nPartial output:\nok\tgithub.com/susugadx/xelyon-cli/internal/ledger\t0.01s",
	})

	snapshot := store.Snapshot()
	if got := snapshot.LastPassedTests.Results(); len(got) != 0 {
		t.Fatalf("LastPassedTests = %#v, want none", got)
	}
	failed := snapshot.LastFailedTests.Results()
	if len(failed) != 2 {
		t.Fatalf("LastFailedTests len = %d, want 2: %#v", len(failed), failed)
	}
	for _, result := range failed {
		if result.Status() != "failed" || result.ExitCode() != -1 {
			t.Fatalf("cancelled result = status %q exit %d, want failed/-1", result.Status(), result.ExitCode())
		}
	}
}

func TestRecorder_TestObservationCapsExcerptAndExitCode(t *testing.T) {
	store := NewStoreWithRoot(t.TempDir())
	recorder := store.Recorder()
	longOutput := strings.Repeat("x", maxTestExcerptBytes+50)

	for i := 0; i < 6; i++ {
		recorder.RecordTestObservation(TestObservation{
			Command:  "go test ./failed" + string(rune('0'+i)),
			ExitCode: i + 1,
			Status:   "failed",
			Output:   longOutput,
		})
		recorder.RecordTestObservation(TestObservation{
			Command:  "go test ./passed" + string(rune('0'+i)),
			ExitCode: 0,
			Status:   "passed",
			Output:   "ok",
		})
	}

	snapshot := store.Snapshot()
	failed := snapshot.LastFailedTests.Results()
	passed := snapshot.LastPassedTests.Results()
	if len(failed) != maxFailedTestResults {
		t.Fatalf("failed len = %d, want %d", len(failed), maxFailedTestResults)
	}
	if len(passed) != maxPassedTestResults {
		t.Fatalf("passed len = %d, want %d", len(passed), maxPassedTestResults)
	}
	if failed[0].Command() != "go test ./failed1" || failed[len(failed)-1].ExitCode() != 6 {
		t.Fatalf("failed cap/order = %#v", failed)
	}
	if !strings.Contains(failed[0].Output(), "... (truncated)") || len(failed[0].Output()) > maxTestExcerptBytes {
		t.Fatalf("failed excerpt was not capped as expected: len=%d", len(failed[0].Output()))
	}
	if passed[0].Command() != "go test ./passed1" || passed[len(passed)-1].Status() != "passed" {
		t.Fatalf("passed cap/order = %#v", passed)
	}
}
