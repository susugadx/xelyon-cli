package ledger

import (
	"reflect"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestRecorder_ZeroAndNilAreNoop(t *testing.T) {
	var zero Recorder
	zero.RecordToolObservation(ToolObservation{Change: &tools.FileChange{FilePath: "src/main.go"}})
	zero.RecordChangedFile("src/changed.go")
	zero.RecordTouchedFile("src/read.go")
	zero.RecordEvidence("evidence", "source")
	zero.RecordRecommendedRead("src/next.go", "reason")
	zero.RecordTestObservation(TestObservation{Command: "go test", ExitCode: 1, Status: "failed", Output: "failed"})
	zero.SetLastFailedTests([]TestResult{NewTestResult("go test", "failed", "failed")})
	zero.SetLastPassedTests([]TestResult{NewTestResult("go test", "ok", "passed")})

	var nilRecorder *Recorder
	nilRecorder.RecordToolObservation(ToolObservation{Change: &tools.FileChange{FilePath: "src/main.go"}})
	nilRecorder.RecordChangedFile("src/changed.go")
	nilRecorder.RecordTouchedFile("src/read.go")
	nilRecorder.RecordEvidence("evidence", "source")
	nilRecorder.RecordRecommendedRead("src/next.go", "reason")
	nilRecorder.RecordTestObservation(TestObservation{Command: "go test", ExitCode: 1, Status: "failed", Output: "failed"})
	nilRecorder.SetLastFailedTests([]TestResult{NewTestResult("go test", "failed", "failed")})
	nilRecorder.SetLastPassedTests([]TestResult{NewTestResult("go test", "ok", "passed")})
}

func TestStore_SnapshotReturnsDefensiveCopy(t *testing.T) {
	root := t.TempDir()
	store := NewStoreWithRoot(root)
	recorder := store.Recorder()
	recorder.RecordChangedFile("src/main.go")
	recorder.RecordTouchedFile("src/read.go")
	recorder.RecordToolObservation(ToolObservation{
		ToolName: "read_file",
		Result:   "📄 File: src/read.go\n12: line 12",
	})
	recorder.RecordRecommendedRead("src/next.go", "follow-up")
	recorder.SetLastFailedTests([]TestResult{NewTestResultWithExitCode("go test ./...", 1, "failed", "failed")})
	recorder.SetLastPassedTests([]TestResult{NewTestResultWithExitCode("go test ./internal/ledger", 0, "passed", "ok")})

	snapshot := store.Snapshot()
	snapshot.ChangedFiles.files[0].path = "src/changed.go"
	snapshot.TouchedFiles.files[0].path = "src/touched.go"
	snapshot.Evidence.items[0].excerpt = "mutated"
	snapshot.RecommendedReads.items[0].path = "src/mutated.go"
	snapshot.LastFailedTests.results[0] = NewTestResult("mutated", "mutated", "mutated")
	snapshot.LastPassedTests.results[0] = NewTestResult("mutated", "mutated", "mutated")

	fresh := store.Snapshot()
	if got := fresh.ChangedFiles.Paths(); !reflect.DeepEqual(got, []string{"src/main.go"}) {
		t.Fatalf("ChangedFiles after snapshot mutation = %v, want [src/main.go]", got)
	}
	if got := fresh.TouchedFiles.Paths(); !reflect.DeepEqual(got, []string{"src/read.go"}) {
		t.Fatalf("TouchedFiles after snapshot mutation = %v, want [src/read.go]", got)
	}
	if got := fresh.Evidence.Items()[0].Text(); got != "line 12" {
		t.Fatalf("Evidence text after snapshot mutation = %q, want %q", got, "line 12")
	}
	if got := fresh.RecommendedReads.Items()[0].Path(); got != "src/next.go" {
		t.Fatalf("RecommendedReads path after snapshot mutation = %q, want %q", got, "src/next.go")
	}
	if got := fresh.LastFailedTests.Results()[0].Command(); got != "go test ./..." {
		t.Fatalf("LastFailedTests command after snapshot mutation = %q, want %q", got, "go test ./...")
	}
	if got := fresh.LastPassedTests.Results()[0].Command(); got != "go test ./internal/ledger" {
		t.Fatalf("LastPassedTests command after snapshot mutation = %q, want %q", got, "go test ./internal/ledger")
	}

	paths := fresh.ChangedFiles.Paths()
	paths[0] = "src/from-paths.go"
	if got := store.Snapshot().ChangedFiles.Paths(); !reflect.DeepEqual(got, []string{"src/main.go"}) {
		t.Fatalf("ChangedFiles after Paths mutation = %v, want [src/main.go]", got)
	}
}
