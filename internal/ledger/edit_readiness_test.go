package ledger

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestRuntimeTaskState_EditReadinessHelpers(t *testing.T) {
	state := RuntimeTaskState{
		ChangedFiles: ChangedFiles{files: []fileFact{{path: "src/changed.go"}}},
		TouchedFiles: TouchedFiles{files: []fileFact{{path: "src/read.go"}}},
		Evidence: Evidence{items: []evidenceFact{{
			path:       "src/read.go",
			startLine:  2,
			endLine:    4,
			source:     "read_file",
			toolCallID: "call_read",
			fileHash:   "sha256:old",
			excerpt:    "evidence",
		}}},
	}

	if !state.WasRecentlyChanged("src/changed.go") {
		t.Fatal("WasRecentlyChanged() = false, want true")
	}
	if !state.WasRecentlyTouched("src/read.go") {
		t.Fatal("WasRecentlyTouched() = false, want true")
	}
	if !state.HasRecentEvidenceForPath("src/read.go") {
		t.Fatal("HasRecentEvidenceForPath() = false, want true")
	}
	if state.HasRecentEvidenceForPath("src/missing.go") {
		t.Fatal("HasRecentEvidenceForPath(missing) = true, want false")
	}

	pointers := state.EvidencePointersForPath("src/read.go")
	if len(pointers) != 1 || pointers[0].Path != "src/read.go" || pointers[0].StartLine != 2 {
		t.Fatalf("EvidencePointersForPath() = %#v", pointers)
	}
	pointers[0].Path = "mutated.go"
	if got := state.EvidencePointersForPath("src/read.go")[0].Path; got != "src/read.go" {
		t.Fatalf("EvidencePointersForPath() returned non-defensive copy, got %q", got)
	}
}

func TestStore_CheckEditReadiness_WithRecentEvidenceIsOK(t *testing.T) {
	store := NewStoreWithRoot(t.TempDir())
	store.Recorder().RecordToolObservation(ToolObservation{
		ToolName:   "read_file",
		ToolCallID: "call_read",
		Structured: &tools.RuntimeObservation{
			Evidence: []tools.ObservationEvidence{{
				Path:      "src/main.go",
				StartLine: 1,
				EndLine:   1,
				Excerpt:   "package main",
			}},
		},
	})

	observation := store.CheckEditReadiness(context.Background(), EditReadinessTarget{
		Path:       "src/main.go",
		ToolName:   "str_replace",
		ToolCallID: "call_edit",
	}, EditReadinessOptions{})

	if observation.Status != EditReadinessStatusOK {
		t.Fatalf("status = %q, want ok: %#v", observation.Status, observation)
	}
	if observation.NormalizedPath != "src/main.go" {
		t.Fatalf("normalized path = %q, want src/main.go", observation.NormalizedPath)
	}
	if len(observation.EvidencePointers) != 1 || observation.EvidencePointers[0].ToolCallID != "call_read" {
		t.Fatalf("evidence pointers = %#v", observation.EvidencePointers)
	}
	if len(observation.Reasons) != 0 {
		t.Fatalf("reasons = %#v, want empty", observation.Reasons)
	}
}

func TestStore_CheckEditReadiness_WarningReasonsWithoutEvidence(t *testing.T) {
	tests := []struct {
		name       string
		seed       func(*Store)
		wantReason EditReadinessReason
	}{
		{
			name: "touched-only",
			seed: func(store *Store) {
				store.Recorder().RecordTouchedFile("src/main.go")
			},
			wantReason: EditReadinessReasonEvidenceRangeMissing,
		},
		{
			name: "changed-only",
			seed: func(store *Store) {
				store.Recorder().RecordChangedFile("src/main.go")
			},
			wantReason: EditReadinessReasonNoRecentRead,
		},
		{
			name:       "not-in-ledger",
			seed:       func(*Store) {},
			wantReason: EditReadinessReasonPathNotInLedger,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStoreWithRoot(t.TempDir())
			tt.seed(store)

			observation := store.CheckEditReadiness(context.Background(), EditReadinessTarget{
				Path:     "src/main.go",
				ToolName: "write_file",
			}, EditReadinessOptions{})

			if observation.Status != EditReadinessStatusWarning {
				t.Fatalf("status = %q, want warning: %#v", observation.Status, observation)
			}
			if !reflect.DeepEqual(observation.Reasons, []EditReadinessReason{tt.wantReason}) {
				t.Fatalf("reasons = %#v, want [%s]", observation.Reasons, tt.wantReason)
			}
		})
	}
}

func TestStore_CheckEditReadiness_RehydrateStaleEvidence(t *testing.T) {
	root := t.TempDir()
	writeLedgerTestFile(t, root, "src/main.go", "new content\n")
	store := NewStoreWithRoot(root)
	store.state.Evidence.record(evidenceFact{
		path:      "src/main.go",
		startLine: 1,
		endLine:   1,
		source:    "read_file",
		fileHash:  "sha256:old",
		excerpt:   "old content",
	})

	observation := store.CheckEditReadiness(context.Background(), EditReadinessTarget{
		Path:     "src/main.go",
		ToolName: "str_replace",
	}, EditReadinessOptions{RehydrateEvidence: true})

	if observation.Status != EditReadinessStatusWarning {
		t.Fatalf("status = %q, want warning: %#v", observation.Status, observation)
	}
	if !reflect.DeepEqual(observation.Reasons, []EditReadinessReason{EditReadinessReasonStaleEvidence}) {
		t.Fatalf("reasons = %#v, want stale_evidence", observation.Reasons)
	}
	if len(observation.RehydrateResults) != 1 || !observation.RehydrateResults[0].Stale || observation.RehydrateResults[0].Reason != "" {
		t.Fatalf("rehydrate results = %#v, want one stale success", observation.RehydrateResults)
	}
}

func TestStore_CheckEditReadiness_RehydrateFailuresPreserveReason(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		startLine  int
		endLine    int
		content    string
		wantReason EvidenceRehydrateErrorReason
	}{
		{
			name:       "missing-file",
			path:       "src/missing.go",
			startLine:  1,
			endLine:    1,
			wantReason: EvidenceRehydrateReasonMissingFile,
		},
		{
			name:       "range-out-of-bounds",
			path:       "src/main.go",
			startLine:  2,
			endLine:    2,
			content:    "only line\n",
			wantReason: EvidenceRehydrateReasonRangeOutOfBounds,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if tt.content != "" {
				writeLedgerTestFile(t, root, tt.path, tt.content)
			}
			store := NewStoreWithRoot(root)
			store.state.Evidence.record(evidenceFact{
				path:      tt.path,
				startLine: tt.startLine,
				endLine:   tt.endLine,
				source:    "read_file",
				excerpt:   "evidence",
			})

			observation := store.CheckEditReadiness(context.Background(), EditReadinessTarget{
				Path:     tt.path,
				ToolName: "str_replace",
			}, EditReadinessOptions{RehydrateEvidence: true})

			if observation.Status != EditReadinessStatusWarning {
				t.Fatalf("status = %q, want warning: %#v", observation.Status, observation)
			}
			if !reflect.DeepEqual(observation.Reasons, []EditReadinessReason{EditReadinessReasonRehydrateFailed}) {
				t.Fatalf("reasons = %#v, want rehydrate_failed", observation.Reasons)
			}
			if len(observation.RehydrateResults) != 1 || observation.RehydrateResults[0].Reason != tt.wantReason {
				t.Fatalf("rehydrate results = %#v, want reason %q", observation.RehydrateResults, tt.wantReason)
			}
		})
	}
}

func TestStore_EditReadinessObservationsAreInternalAndDefensive(t *testing.T) {
	store := NewStoreWithRoot(t.TempDir())
	store.RecordEditReadinessObservation(EditReadinessObservation{
		Path:           "src/main.go",
		NormalizedPath: "src/main.go",
		ToolName:       "write_file",
		Status:         EditReadinessStatusWarning,
		Reasons:        []EditReadinessReason{EditReadinessReasonPathNotInLedger},
		EvidencePointers: []EvidencePointer{{
			Path:      "src/main.go",
			StartLine: 1,
			EndLine:   1,
		}},
		RehydrateResults: []EvidenceRehydrateResult{{
			Path:   "src/main.go",
			Reason: EvidenceRehydrateReasonMissingFile,
		}},
	})

	if !store.Snapshot().IsEmpty() {
		t.Fatalf("Snapshot() = %#v, want no runtime task facts from edit readiness observation", store.Snapshot())
	}
	rendered := RenderCurrentTaskStateSnapshot(store.Snapshot(), SnapshotRenderOptions{})
	if strings.Contains(rendered, "path_not_in_ledger") || strings.Contains(rendered, "EditReadiness") {
		t.Fatalf("snapshot render should not include edit readiness observation:\n%s", rendered)
	}

	observations := store.EditReadinessObservations()
	observations[0].Reasons[0] = EditReadinessReasonNoRecentRead
	observations[0].EvidencePointers[0].Path = "mutated.go"
	observations[0].RehydrateResults[0].Reason = EvidenceRehydrateReasonInvalidPath

	fresh := store.EditReadinessObservations()
	if fresh[0].Reasons[0] != EditReadinessReasonPathNotInLedger ||
		fresh[0].EvidencePointers[0].Path != "src/main.go" ||
		fresh[0].RehydrateResults[0].Reason != EvidenceRehydrateReasonMissingFile {
		t.Fatalf("EditReadinessObservations() returned non-defensive copy: %#v", fresh)
	}

	store.Reset()
	if got := store.EditReadinessObservations(); len(got) != 0 {
		t.Fatalf("observations after Reset() = %#v, want empty", got)
	}
}
