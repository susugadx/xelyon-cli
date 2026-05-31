package taskstate

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRecorder_RecordToolObservation_ReadSearchGatherEvidence(t *testing.T) {
	root := t.TempDir()
	store := NewStoreWithRoot(root)
	recorder := store.Recorder()

	recorder.RecordToolObservation(ToolObservation{
		ToolName:   "read_file",
		ToolCallID: "call-read",
		Result: strings.Join([]string{
			"📄 File: " + filepath.Join(root, "internal/taskstate/taskstate.go") + ":10-12 [L1]",
			"10: func run() {",
			"11: }",
		}, "\n"),
	})
	recorder.RecordToolObservation(ToolObservation{
		ToolName:   "search_code",
		ToolCallID: "call-search",
		Result: strings.Join([]string{
			"Found 1 match(es) in 1 file(s)",
			"",
			"📄 internal/agent/agent.go (1 match(es)) [L2]",
			"  [def]     >   42 │ func runAgent() {}",
			"",
			"Recommended reads:",
			"  - internal/agent/next.go:9 | caller to inspect",
			"  - internal/agent/client.go:4 in UseBuilder | scoped caller",
			"  - internal/agent/span.go:4-8 in UseBuilder | scoped range",
		}, "\n"),
	})
	recorder.RecordToolObservation(ToolObservation{
		ToolName:   "gather_context",
		ToolCallID: "call-gather",
		Result: strings.Join([]string{
			"Route: direct",
			"",
			"📄 File: internal/tools/file/read.go:5-5",
			"5: func read() {}",
		}, "\n"),
	})

	snapshot := store.Snapshot()
	wantTouched := []string{
		"internal/taskstate/taskstate.go",
		"internal/agent/agent.go",
		"internal/tools/file/read.go",
	}
	if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, wantTouched) {
		t.Fatalf("TouchedFiles.Paths() = %v, want %v", got, wantTouched)
	}

	evidence := snapshot.Evidence.Items()
	if len(evidence) != 4 {
		t.Fatalf("Evidence len = %d, want 4: %#v", len(evidence), evidence)
	}
	if got := evidence[0].Path(); got != "internal/taskstate/taskstate.go" {
		t.Fatalf("read evidence path = %q", got)
	}
	if evidence[0].StartLine() != 10 || evidence[0].EndLine() != 10 {
		t.Fatalf("read evidence lines = %d-%d, want 10-10", evidence[0].StartLine(), evidence[0].EndLine())
	}
	if evidence[0].Source() != "read_file" || evidence[0].ToolCallID() != "call-read" {
		t.Fatalf("read evidence source/id = %q/%q", evidence[0].Source(), evidence[0].ToolCallID())
	}
	if evidence[0].FileHash() != "" || evidence[0].Stale() {
		t.Fatalf("read evidence hash/stale = %q/%v, want empty/false", evidence[0].FileHash(), evidence[0].Stale())
	}
	if got := evidence[2].Path(); got != "internal/agent/agent.go" {
		t.Fatalf("search evidence path = %q", got)
	}
	if evidence[2].StartLine() != 42 || !strings.Contains(evidence[2].Excerpt(), "runAgent") {
		t.Fatalf("search evidence = line %d excerpt %q", evidence[2].StartLine(), evidence[2].Excerpt())
	}

	reads := snapshot.RecommendedReads.Items()
	if len(reads) != 3 {
		t.Fatalf("RecommendedReads len = %d, want 3", len(reads))
	}
	if reads[0].Path() != "internal/agent/next.go" ||
		reads[0].Reason() != "caller to inspect" ||
		reads[0].Source() != "search_code" ||
		reads[0].ToolCallID() != "call-search" {
		t.Fatalf("RecommendedReads[0] = path=%q reason=%q source=%q id=%q", reads[0].Path(), reads[0].Reason(), reads[0].Source(), reads[0].ToolCallID())
	}
	if reads[1].Path() != "internal/agent/client.go" || reads[1].Reason() != "scoped caller" {
		t.Fatalf("RecommendedReads[1] = path=%q reason=%q, want internal/agent/client.go/scoped caller", reads[1].Path(), reads[1].Reason())
	}
	if reads[2].Path() != "internal/agent/span.go" || reads[2].Reason() != "scoped range" {
		t.Fatalf("RecommendedReads[2] = path=%q reason=%q, want internal/agent/span.go/scoped range", reads[2].Path(), reads[2].Reason())
	}
}

func TestRecorder_RecordToolObservation_SearchCodeSymbolItemsAreEvidence(t *testing.T) {
	root := t.TempDir()
	store := NewStoreWithRoot(root)
	recorder := store.Recorder()

	recorder.RecordToolObservation(ToolObservation{
		ToolName:   "search_code",
		ToolCallID: "call-symbol",
		Result: strings.Join([]string{
			"── function Build (L3) in internal/agent/builder.go ──",
			"Definition:",
			"  3: func Build() {}",
			"",
			"Callers (1):",
			"  - internal/agent/client.go:4 in UseBuilder | _ = Build()",
			"",
			"Related Tests (1):",
			"  - internal/agent/client_test.go:12 | func TestBuild",
			"",
			"Recommended reads:",
			"  - internal/agent/next.go:9 | inspect later",
		}, "\n"),
	})

	snapshot := store.Snapshot()
	wantTouched := []string{
		"internal/agent/builder.go",
		"internal/agent/client.go",
		"internal/agent/client_test.go",
	}
	if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, wantTouched) {
		t.Fatalf("TouchedFiles.Paths() = %v, want %v", got, wantTouched)
	}

	evidence := snapshot.Evidence.Items()
	if len(evidence) != 4 {
		t.Fatalf("Evidence len = %d, want 4: %#v", len(evidence), evidence)
	}
	if evidence[2].Path() != "internal/agent/client.go" ||
		evidence[2].StartLine() != 4 ||
		evidence[2].Excerpt() != "_ = Build()" {
		t.Fatalf("caller evidence = %#v", evidence[2])
	}
	if evidence[3].Path() != "internal/agent/client_test.go" ||
		evidence[3].StartLine() != 12 ||
		evidence[3].Excerpt() != "func TestBuild" {
		t.Fatalf("test evidence = %#v", evidence[3])
	}

	reads := snapshot.RecommendedReads.Items()
	if len(reads) != 1 || reads[0].Path() != "internal/agent/next.go" || reads[0].Reason() != "inspect later" {
		t.Fatalf("RecommendedReads = %#v", reads)
	}
}
