package ledger

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

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
