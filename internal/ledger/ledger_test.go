package ledger

import (
	"reflect"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestRecorder_ZeroAndNilAreNoop(t *testing.T) {
	var zero Recorder
	zero.RecordToolObservation(&tools.FileChange{FilePath: "/src/main.go"})
	zero.RecordChangedFile("/src/changed.go")
	zero.RecordTouchedFile("/src/read.go")
	zero.RecordEvidence("evidence", "source")
	zero.RecordRecommendedRead("/src/next.go", "reason")
	zero.SetLastFailedTests([]TestResult{NewTestResult("go test", "failed", "failed")})
	zero.SetLastPassedTests([]TestResult{NewTestResult("go test", "ok", "passed")})

	var nilRecorder *Recorder
	nilRecorder.RecordToolObservation(&tools.FileChange{FilePath: "/src/main.go"})
	nilRecorder.RecordChangedFile("/src/changed.go")
	nilRecorder.RecordTouchedFile("/src/read.go")
	nilRecorder.RecordEvidence("evidence", "source")
	nilRecorder.RecordRecommendedRead("/src/next.go", "reason")
	nilRecorder.SetLastFailedTests([]TestResult{NewTestResult("go test", "failed", "failed")})
	nilRecorder.SetLastPassedTests([]TestResult{NewTestResult("go test", "ok", "passed")})
}

func TestRecorder_RecordToolObservation_ChangedFilesAreOrderedAndDeduped(t *testing.T) {
	store := NewStore()
	recorder := store.Recorder()

	recorder.RecordToolObservation(&tools.FileChange{
		FilePath: "/src/fallback.go",
		Details: []tools.FileChangeDetail{
			{FilePath: "/src/a.go"},
			{FilePath: ""},
			{FilePath: "/src/a.go"},
			{FilePath: "/src/b.go"},
		},
	})
	recorder.RecordToolObservation(&tools.FileChange{FilePath: "/src/a.go"})
	recorder.RecordToolObservation(&tools.FileChange{FilePath: "/src/c.go"})
	recorder.RecordChangedFile("")

	got := store.Snapshot().ChangedFiles.Paths()
	want := []string{"/src/a.go", "/src/b.go", "/src/c.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ChangedFiles.Paths() = %v, want %v", got, want)
	}
}

func TestStore_SnapshotReturnsDefensiveCopy(t *testing.T) {
	store := NewStore()
	recorder := store.Recorder()
	recorder.RecordChangedFile("/src/main.go")
	recorder.RecordTouchedFile("/src/read.go")
	recorder.RecordEvidence("line 12", "read_file")
	recorder.RecordRecommendedRead("/src/next.go", "follow-up")
	recorder.SetLastFailedTests([]TestResult{NewTestResult("go test ./...", "failed", "failed")})
	recorder.SetLastPassedTests([]TestResult{NewTestResult("go test ./internal/ledger", "ok", "passed")})

	snapshot := store.Snapshot()
	snapshot.ChangedFiles.files[0].path = "/src/changed.go"
	snapshot.TouchedFiles.files[0].path = "/src/touched.go"
	snapshot.Evidence.items[0].text = "mutated"
	snapshot.RecommendedReads.items[0].path = "/src/mutated.go"
	snapshot.LastFailedTests.results[0] = NewTestResult("mutated", "mutated", "mutated")
	snapshot.LastPassedTests.results[0] = NewTestResult("mutated", "mutated", "mutated")

	fresh := store.Snapshot()
	if got := fresh.ChangedFiles.Paths(); !reflect.DeepEqual(got, []string{"/src/main.go"}) {
		t.Fatalf("ChangedFiles after snapshot mutation = %v, want [/src/main.go]", got)
	}
	if got := fresh.TouchedFiles.Paths(); !reflect.DeepEqual(got, []string{"/src/read.go"}) {
		t.Fatalf("TouchedFiles after snapshot mutation = %v, want [/src/read.go]", got)
	}
	if got := fresh.Evidence.Items()[0].Text(); got != "line 12" {
		t.Fatalf("Evidence text after snapshot mutation = %q, want %q", got, "line 12")
	}
	if got := fresh.RecommendedReads.Items()[0].Path(); got != "/src/next.go" {
		t.Fatalf("RecommendedReads path after snapshot mutation = %q, want %q", got, "/src/next.go")
	}
	if got := fresh.LastFailedTests.Results()[0].Command(); got != "go test ./..." {
		t.Fatalf("LastFailedTests command after snapshot mutation = %q, want %q", got, "go test ./...")
	}
	if got := fresh.LastPassedTests.Results()[0].Command(); got != "go test ./internal/ledger" {
		t.Fatalf("LastPassedTests command after snapshot mutation = %q, want %q", got, "go test ./internal/ledger")
	}

	paths := fresh.ChangedFiles.Paths()
	paths[0] = "/src/from-paths.go"
	if got := store.Snapshot().ChangedFiles.Paths(); !reflect.DeepEqual(got, []string{"/src/main.go"}) {
		t.Fatalf("ChangedFiles after Paths mutation = %v, want [/src/main.go]", got)
	}
}

func TestRecorder_LastTestSettersReplace(t *testing.T) {
	store := NewStore()
	recorder := store.Recorder()

	recorder.SetLastFailedTests([]TestResult{
		NewTestResult("go test ./...", "first", "failed"),
		NewTestResult("go test ./internal/agent", "second", "failed"),
	})
	recorder.SetLastFailedTests([]TestResult{
		NewTestResult("go test ./internal/ledger", "latest", "failed"),
	})
	recorder.SetLastPassedTests([]TestResult{
		NewTestResult("go test ./...", "ok", "passed"),
	})
	recorder.SetLastPassedTests(nil)

	snapshot := store.Snapshot()
	failed := snapshot.LastFailedTests.Results()
	if len(failed) != 1 {
		t.Fatalf("LastFailedTests len = %d, want 1", len(failed))
	}
	if got := failed[0].Command(); got != "go test ./internal/ledger" {
		t.Fatalf("LastFailedTests command = %q, want %q", got, "go test ./internal/ledger")
	}
	if got := failed[0].Output(); got != "latest" {
		t.Fatalf("LastFailedTests output = %q, want %q", got, "latest")
	}
	if got := failed[0].Status(); got != "failed" {
		t.Fatalf("LastFailedTests status = %q, want %q", got, "failed")
	}
	if got := snapshot.LastPassedTests.Len(); got != 0 {
		t.Fatalf("LastPassedTests len = %d, want 0", got)
	}
}
