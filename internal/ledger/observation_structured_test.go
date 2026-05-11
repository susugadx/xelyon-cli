package ledger

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

type structuredObservationRecorderFixture struct {
	store    *Store
	recorder *Recorder
}

func newStructuredObservationRecorderFixture(t *testing.T) structuredObservationRecorderFixture {
	t.Helper()

	store := NewStoreWithRoot(t.TempDir())
	return structuredObservationRecorderFixture{
		store:    store,
		recorder: store.Recorder(),
	}
}

func TestRecorder_RecordToolObservation_PrefersStructuredObservation(t *testing.T) {
	root := t.TempDir()
	invocationCWD := filepath.Join(root, "pkg")
	for _, path := range []string{
		filepath.Join(root, "root.go"),
		filepath.Join(invocationCWD, "target.py"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir test file dir %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte("def Foo():\n    pass\n"), 0o644); err != nil {
			t.Fatalf("write test file %s: %v", path, err)
		}
	}
	store := NewStoreWithWorkspace(root, invocationCWD)
	recorder := store.Recorder()

	recorder.RecordToolObservation(ToolObservation{
		ToolName:   "search_code",
		ToolCallID: "structured-search",
		Result: strings.Join([]string{
			"📄 root.go (1 match(es)) [L1]",
			"  [def]     >   1 │ func misleading() {}",
		}, "\n"),
		Structured: &tools.RuntimeObservation{
			TouchedFiles: []tools.ObservationPath{{
				Path:         "target.py",
				ResolvedPath: filepath.Join(invocationCWD, "target.py"),
			}},
			Evidence: []tools.ObservationEvidence{{
				Path:         "target.py",
				ResolvedPath: filepath.Join(invocationCWD, "target.py"),
				StartLine:    1,
				EndLine:      1,
				Excerpt:      "def Foo():",
			}},
			RecommendedReads: []tools.ObservationRecommendedRead{{
				Path:         "target.py",
				ResolvedPath: filepath.Join(invocationCWD, "target.py"),
				Reason:       "structured follow-up",
			}},
		},
	})

	snapshot := store.Snapshot()
	if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, []string{"pkg/target.py"}) {
		t.Fatalf("TouchedFiles.Paths() = %v, want [pkg/target.py]", got)
	}
	evidence := snapshot.Evidence.Items()
	if len(evidence) != 1 ||
		evidence[0].Path() != "pkg/target.py" ||
		evidence[0].Source() != "search_code" ||
		evidence[0].ToolCallID() != "structured-search" ||
		evidence[0].Excerpt() != "def Foo():" {
		t.Fatalf("Evidence = %#v, want structured target.py evidence", evidence)
	}
	reads := snapshot.RecommendedReads.Items()
	if len(reads) != 1 || reads[0].Path() != "pkg/target.py" || reads[0].Reason() != "structured follow-up" {
		t.Fatalf("RecommendedReads = %#v, want structured target.py read", reads)
	}
}

func TestRecorder_RecordToolObservation_DropsInvalidStructuredEvidenceButKeepsTouchedPath(t *testing.T) {
	for _, tt := range []struct {
		name        string
		evidence    tools.ObservationEvidence
		wantTouched []string
	}{
		{
			name: "start line zero",
			evidence: tools.ObservationEvidence{
				Path:      "pkg/target.py",
				StartLine: 0,
				EndLine:   1,
				Excerpt:   "def target():",
			},
			wantTouched: []string{"pkg/target.py"},
		},
		{
			name: "end before start",
			evidence: tools.ObservationEvidence{
				Path:      "pkg/target.py",
				StartLine: 3,
				EndLine:   2,
				Excerpt:   "def target():",
			},
			wantTouched: []string{"pkg/target.py"},
		},
		{
			name: "empty excerpt",
			evidence: tools.ObservationEvidence{
				Path:      "pkg/target.py",
				StartLine: 1,
				EndLine:   1,
				Excerpt:   "",
			},
			wantTouched: []string{"pkg/target.py"},
		},
		{
			name: "empty path",
			evidence: tools.ObservationEvidence{
				Path:      "",
				StartLine: 1,
				EndLine:   1,
				Excerpt:   "def target():",
			},
			wantTouched: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newStructuredObservationRecorderFixture(t)

			fixture.recorder.RecordToolObservation(ToolObservation{
				ToolName:   "custom_tool",
				ToolCallID: "invalid-structured",
				Structured: &tools.RuntimeObservation{
					Evidence: []tools.ObservationEvidence{tt.evidence},
				},
			})

			snapshot := fixture.store.Snapshot()
			if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, tt.wantTouched) {
				t.Fatalf("TouchedFiles.Paths() = %v, want %v", got, tt.wantTouched)
			}
			if got := snapshot.Evidence.Items(); len(got) != 0 {
				t.Fatalf("Evidence = %#v, want empty", got)
			}
		})
	}
}

func TestRecorder_RecordToolObservation_NormalizesStructuredLineOnlyEvidence(t *testing.T) {
	fixture := newStructuredObservationRecorderFixture(t)

	fixture.recorder.RecordToolObservation(ToolObservation{
		ToolName:   "search_code",
		ToolCallID: "structured-symbol",
		Result: strings.Join([]string{
			"📄 rendered-fallback.py (1 match(es)) [L9]",
			"  [ref]     >   9 │ def fallback():",
		}, "\n"),
		Structured: &tools.RuntimeObservation{
			Evidence: []tools.ObservationEvidence{
				{
					Path:      "pkg/definition.py",
					StartLine: 3,
					EndLine:   5,
					Excerpt:   "def target():",
				},
				{
					Path:      "pkg/caller.py",
					StartLine: 12,
					EndLine:   0,
					Excerpt:   "target()",
				},
			},
		},
	})

	evidence := fixture.store.Snapshot().Evidence.Items()
	if len(evidence) != 2 {
		t.Fatalf("Evidence = %#v, want definition and line-only caller evidence", evidence)
	}
	if evidence[0].Path() != "pkg/definition.py" ||
		evidence[0].StartLine() != 3 ||
		evidence[0].EndLine() != 5 {
		t.Fatalf("definition evidence = %#v, want pkg/definition.py:3-5", evidence[0])
	}
	if evidence[1].Path() != "pkg/caller.py" ||
		evidence[1].StartLine() != 12 ||
		evidence[1].EndLine() != 12 {
		t.Fatalf("line-only evidence = %#v, want pkg/caller.py:12-12", evidence[1])
	}
}

func TestRecorder_RecordToolObservation_StructuredTouchedStillRecordsRenderedRecommendedReads(t *testing.T) {
	fixture := newStructuredObservationRecorderFixture(t)

	fixture.recorder.RecordToolObservation(ToolObservation{
		ToolName:   "search_code",
		ToolCallID: "structured-touched",
		Result: strings.Join([]string{
			"Recommended reads:",
			"  - pkg/next.py:7 | inspect caller",
		}, "\n"),
		Structured: &tools.RuntimeObservation{
			TouchedFiles: []tools.ObservationPath{{
				Path: "pkg/target.py",
			}},
		},
	})

	snapshot := fixture.store.Snapshot()
	if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, []string{"pkg/target.py"}) {
		t.Fatalf("TouchedFiles.Paths() = %v, want [pkg/target.py]", got)
	}
	reads := snapshot.RecommendedReads.Items()
	if len(reads) != 1 || reads[0].Path() != "pkg/next.py" || reads[0].Reason() != "inspect caller" {
		t.Fatalf("RecommendedReads = %#v, want rendered pkg/next.py read", reads)
	}
}

func TestRecorder_RecordToolObservation_FallsBackWhenStructuredPathsAreInvalid(t *testing.T) {
	fixture := newStructuredObservationRecorderFixture(t)

	fixture.recorder.RecordToolObservation(ToolObservation{
		ToolName:   "search_code",
		ToolCallID: "invalid-structured-paths",
		Result: strings.Join([]string{
			"Found 1 match(es) in 1 file(s)",
			"",
			"📄 pkg/fallback.py (1 match(es)) [L3]",
			"  [ref]     >   3 │ def fallback():",
			"",
			"Recommended reads:",
			"  - pkg/next.py:9 | fallback follow-up",
		}, "\n"),
		Structured: &tools.RuntimeObservation{
			TouchedFiles: []tools.ObservationPath{{
				Path: "https://example.com/pkg/target.py",
			}},
			Evidence: []tools.ObservationEvidence{{
				Path:      "pkg/*.py",
				StartLine: 1,
				EndLine:   1,
				Excerpt:   "def target():",
			}},
			RecommendedReads: []tools.ObservationRecommendedRead{{
				Path:   "pkg/*.py",
				Reason: "invalid structured read",
			}},
		},
	})

	snapshot := fixture.store.Snapshot()
	if got := snapshot.TouchedFiles.Paths(); !reflect.DeepEqual(got, []string{"pkg/fallback.py"}) {
		t.Fatalf("TouchedFiles.Paths() = %v, want [pkg/fallback.py]", got)
	}
	evidence := snapshot.Evidence.Items()
	if len(evidence) != 1 ||
		evidence[0].Path() != "pkg/fallback.py" ||
		evidence[0].StartLine() != 3 ||
		evidence[0].Excerpt() != "def fallback():" {
		t.Fatalf("Evidence = %#v, want rendered fallback evidence", evidence)
	}
	reads := snapshot.RecommendedReads.Items()
	if len(reads) != 1 || reads[0].Path() != "pkg/next.py" || reads[0].Reason() != "fallback follow-up" {
		t.Fatalf("RecommendedReads = %#v, want rendered fallback read", reads)
	}
}

func TestRecorder_RecordToolObservation_DedupesStructuredAndRenderedRecommendedReads(t *testing.T) {
	fixture := newStructuredObservationRecorderFixture(t)

	fixture.recorder.RecordToolObservation(ToolObservation{
		ToolName:   "search_code",
		ToolCallID: "duplicate-reads",
		Result: strings.Join([]string{
			"Recommended reads:",
			"  - pkg/next.py:7 | rendered reason",
		}, "\n"),
		Structured: &tools.RuntimeObservation{
			RecommendedReads: []tools.ObservationRecommendedRead{{
				Path:   "pkg/next.py",
				Reason: "structured reason",
			}},
		},
	})

	reads := fixture.store.Snapshot().RecommendedReads.Items()
	if len(reads) != 1 ||
		reads[0].Path() != "pkg/next.py" ||
		reads[0].Reason() != "structured reason" {
		t.Fatalf("RecommendedReads = %#v, want one structured pkg/next.py read", reads)
	}
}
