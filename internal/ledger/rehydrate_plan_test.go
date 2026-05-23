package ledger

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRehydratePlan_PlansEditTargetMissingRecentEvidence(t *testing.T) {
	root := t.TempDir()
	state := RuntimeTaskState{}
	plan := BuildRehydratePlan(state, rehydratePlanTestWorkspace(root), RehydratePlanOptions{
		TargetPaths: []string{"src/main.go"},
		OldEvidencePointers: []EvidencePointer{
			oldRehydratePlanPointer("src/main.go", 10, 20, "read_file", "call_old_read"),
		},
	})

	want := RehydratePlan{Items: []RehydratePlanItem{{
		Path:       "src/main.go",
		StartLine:  10,
		EndLine:    20,
		Source:     "read_file",
		Reason:     RehydratePlanReasonEditTargetMissingEvidence,
		ToolCallID: "call_old_read",
	}}}
	if !reflect.DeepEqual(plan, want) {
		t.Fatalf("BuildRehydratePlan() = %#v, want %#v", plan, want)
	}
}

func TestRehydratePlan_SkipsTargetWithNonStaleRecentEvidence(t *testing.T) {
	root := t.TempDir()
	state := RuntimeTaskState{
		Evidence: Evidence{items: []evidenceFact{{
			path:      "src/main.go",
			startLine: 1,
			endLine:   2,
			source:    "read_file",
			excerpt:   "fresh evidence",
		}}},
	}
	plan := BuildRehydratePlan(state, rehydratePlanTestWorkspace(root), RehydratePlanOptions{
		TargetPaths: []string{"src/main.go"},
		OldEvidencePointers: []EvidencePointer{
			oldRehydratePlanPointer("src/main.go", 10, 20, "read_file", ""),
		},
	})

	if len(plan.Items) != 0 {
		t.Fatalf("BuildRehydratePlan() = %#v, want no items with fresh recent evidence", plan)
	}
}

func TestRehydratePlan_FiltersInvalidTargetsPointersAndSources(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	state := RuntimeTaskState{}
	plan := BuildRehydratePlan(state, rehydratePlanTestWorkspace(root), RehydratePlanOptions{
		TargetPaths: []string{
			"src/main.go",
			"locator:abc",
			filepath.Join(outside, "outside.go"),
		},
		OldEvidencePointers: []EvidencePointer{
			oldRehydratePlanPointer("src/main.go", 1, 2, "bash", "call_bash"),
			oldRehydratePlanPointer("src/main.go", 0, 2, "read_file", "call_zero"),
			oldRehydratePlanPointer("../outside.go", 1, 2, "read_file", "call_escape"),
			oldRehydratePlanPointer("src/main.go", 3, 4, "search_code", "call_search"),
		},
	})

	want := RehydratePlan{Items: []RehydratePlanItem{{
		Path:       "src/main.go",
		StartLine:  3,
		EndLine:    4,
		Source:     "search_code",
		Reason:     RehydratePlanReasonEditTargetMissingEvidence,
		ToolCallID: "call_search",
	}}}
	if !reflect.DeepEqual(plan, want) {
		t.Fatalf("BuildRehydratePlan() = %#v, want %#v", plan, want)
	}
}

func TestRehydratePlan_DedupesSamePathRange(t *testing.T) {
	root := t.TempDir()
	plan := BuildRehydratePlan(RuntimeTaskState{}, rehydratePlanTestWorkspace(root), RehydratePlanOptions{
		TargetPaths: []string{"src/main.go"},
		OldEvidencePointers: []EvidencePointer{
			oldRehydratePlanPointer("src/main.go", 5, 8, "read_file", "call_read"),
			oldRehydratePlanPointer("src/main.go", 5, 8, "search_code", "call_search"),
		},
	})

	if len(plan.Items) != 1 || plan.Items[0].ToolCallID != "call_read" {
		t.Fatalf("BuildRehydratePlan() = %#v, want one deduped item keeping first pointer", plan)
	}
}

func TestRehydratePlan_StalePointerAndReadinessWarningUseStaleReason(t *testing.T) {
	root := t.TempDir()
	stalePointer := oldRehydratePlanPointer("src/other.go", 1, 3, "read_file", "call_stale")
	stalePointer.Stale = true
	plan := BuildRehydratePlan(RuntimeTaskState{}, rehydratePlanTestWorkspace(root), RehydratePlanOptions{
		EditReadinessObservations: []EditReadinessObservation{{
			Path:           "src/main.go",
			NormalizedPath: "src/main.go",
			Reasons:        []EditReadinessReason{EditReadinessReasonStaleEvidence},
		}},
		OldEvidencePointers: []EvidencePointer{
			oldRehydratePlanPointer("src/main.go", 1, 3, "read_file", "call_warning"),
			stalePointer,
		},
		TargetPaths: []string{"src/other.go"},
	})

	if len(plan.Items) != 2 {
		t.Fatalf("BuildRehydratePlan() = %#v, want two stale-reason items", plan)
	}
	for _, item := range plan.Items {
		if item.Reason != RehydratePlanReasonStaleEvidenceRequiresRefresh || !item.Stale {
			t.Fatalf("plan item = %#v, want stale reason and Stale=true", item)
		}
	}
}

func TestRehydratePlan_StaleReadinessObservationBypassesRecentEvidenceSkip(t *testing.T) {
	root := t.TempDir()
	state := RuntimeTaskState{
		Evidence: Evidence{items: []evidenceFact{{
			path:       "src/main.go",
			startLine:  1,
			endLine:    3,
			source:     "read_file",
			toolCallID: "call_old",
			excerpt:    "fresh but stale-by-observation",
		}}},
	}
	plan := BuildRehydratePlan(state, rehydratePlanTestWorkspace(root), RehydratePlanOptions{
		EditReadinessObservations: []EditReadinessObservation{{
			Path:           "src/main.go",
			NormalizedPath: "src/main.go",
			Reasons:        []EditReadinessReason{EditReadinessReasonStaleEvidence},
		}},
		OldEvidencePointers: []EvidencePointer{
			oldRehydratePlanPointer("src/main.go", 1, 3, "read_file", "call_old"),
		},
	})

	want := RehydratePlan{Items: []RehydratePlanItem{{
		Path:       "src/main.go",
		StartLine:  1,
		EndLine:    3,
		Source:     "read_file",
		Reason:     RehydratePlanReasonStaleEvidenceRequiresRefresh,
		ToolCallID: "call_old",
		Stale:      true,
	}}}
	if !reflect.DeepEqual(plan, want) {
		t.Fatalf("BuildRehydratePlan() = %#v, want stale refresh item despite recent evidence %#v", plan, want)
	}
}

func TestRehydratePlan_NormalizedObservationPathUsesRepoRootBase(t *testing.T) {
	root := t.TempDir()
	writeRehydratePlanTestFile(t, root, "src/main.go")
	writeRehydratePlanTestFile(t, root, "subdir/src/main.go")

	plan := BuildRehydratePlan(RuntimeTaskState{}, EvidenceRehydrateOptions{
		RepoRoot:      root,
		InvocationCWD: filepath.Join(root, "subdir"),
	}, RehydratePlanOptions{
		EditReadinessObservations: []EditReadinessObservation{{
			Path:           "src/main.go",
			NormalizedPath: "src/main.go",
			Reasons:        []EditReadinessReason{EditReadinessReasonNoRecentRead},
		}},
		OldEvidencePointers: []EvidencePointer{
			oldRehydratePlanPointer("src/main.go", 4, 6, "read_file", "call_repo_root"),
		},
	})

	want := RehydratePlan{Items: []RehydratePlanItem{{
		Path:       "src/main.go",
		StartLine:  4,
		EndLine:    6,
		Source:     "read_file",
		Reason:     RehydratePlanReasonEditTargetMissingEvidence,
		ToolCallID: "call_repo_root",
	}}}
	if !reflect.DeepEqual(plan, want) {
		t.Fatalf("BuildRehydratePlan() = %#v, want normalized path to stay repo-relative %#v", plan, want)
	}
}

func TestRehydratePlan_UsesChangedFilesButNotTouchedFiles(t *testing.T) {
	root := t.TempDir()
	state := RuntimeTaskState{
		ChangedFiles: ChangedFiles{files: []fileFact{{path: "src/changed.go"}}},
		TouchedFiles: TouchedFiles{files: []fileFact{{path: "src/touched.go"}}},
	}
	plan := BuildRehydratePlan(state, rehydratePlanTestWorkspace(root), RehydratePlanOptions{
		OldEvidencePointers: []EvidencePointer{
			oldRehydratePlanPointer("src/changed.go", 1, 2, "gather_context", "call_changed"),
			oldRehydratePlanPointer("src/touched.go", 1, 2, "read_file", "call_touched"),
		},
	})

	want := RehydratePlan{Items: []RehydratePlanItem{{
		Path:       "src/changed.go",
		StartLine:  1,
		EndLine:    2,
		Source:     "gather_context",
		Reason:     RehydratePlanReasonOmittedProviderHistory,
		ToolCallID: "call_changed",
	}}}
	if !reflect.DeepEqual(plan, want) {
		t.Fatalf("BuildRehydratePlan() = %#v, want %#v", plan, want)
	}
}

func TestRehydratePlan_AppliesItemAndLineLimits(t *testing.T) {
	root := t.TempDir()
	plan := BuildRehydratePlan(RuntimeTaskState{}, rehydratePlanTestWorkspace(root), RehydratePlanOptions{
		TargetPaths: []string{"a.go", "b.go", "c.go"},
		OldEvidencePointers: []EvidencePointer{
			oldRehydratePlanPointer("a.go", 1, 100, "read_file", "call_a"),
			oldRehydratePlanPointer("b.go", 1, 100, "read_file", "call_b"),
			oldRehydratePlanPointer("c.go", 1, 100, "read_file", "call_c"),
		},
		MaxItems:        3,
		MaxLinesPerItem: 5,
		MaxTotalLines:   8,
	})

	want := RehydratePlan{Items: []RehydratePlanItem{
		{Path: "a.go", StartLine: 1, EndLine: 5, Source: "read_file", Reason: RehydratePlanReasonEditTargetMissingEvidence, ToolCallID: "call_a"},
		{Path: "b.go", StartLine: 1, EndLine: 3, Source: "read_file", Reason: RehydratePlanReasonEditTargetMissingEvidence, ToolCallID: "call_b"},
	}}
	if !reflect.DeepEqual(plan, want) {
		t.Fatalf("BuildRehydratePlan() = %#v, want %#v", plan, want)
	}
}

func TestRehydratePlan_AppliesMaxItemsLimit(t *testing.T) {
	root := t.TempDir()
	plan := BuildRehydratePlan(RuntimeTaskState{}, rehydratePlanTestWorkspace(root), RehydratePlanOptions{
		TargetPaths: []string{"a.go", "b.go", "c.go"},
		OldEvidencePointers: []EvidencePointer{
			oldRehydratePlanPointer("a.go", 1, 2, "read_file", "call_a"),
			oldRehydratePlanPointer("b.go", 1, 2, "read_file", "call_b"),
			oldRehydratePlanPointer("c.go", 1, 2, "read_file", "call_c"),
		},
		MaxItems:        2,
		MaxLinesPerItem: 80,
		MaxTotalLines:   240,
	})

	want := RehydratePlan{Items: []RehydratePlanItem{
		{Path: "a.go", StartLine: 1, EndLine: 2, Source: "read_file", Reason: RehydratePlanReasonEditTargetMissingEvidence, ToolCallID: "call_a"},
		{Path: "b.go", StartLine: 1, EndLine: 2, Source: "read_file", Reason: RehydratePlanReasonEditTargetMissingEvidence, ToolCallID: "call_b"},
	}}
	if !reflect.DeepEqual(plan, want) {
		t.Fatalf("BuildRehydratePlan() = %#v, want %#v", plan, want)
	}
}

func TestRehydratePlanStore_UsesRecordedEditReadinessObservations(t *testing.T) {
	root := t.TempDir()
	store := NewStoreWithRoot(root)
	store.RecordEditReadinessObservation(EditReadinessObservation{
		Path:           "src/main.go",
		NormalizedPath: "src/main.go",
		Status:         EditReadinessStatusWarning,
		Reasons:        []EditReadinessReason{EditReadinessReasonNoRecentRead},
	})

	plan := store.BuildRehydratePlan(RehydratePlanOptions{
		OldEvidencePointers: []EvidencePointer{
			oldRehydratePlanPointer("src/main.go", 1, 2, "read_file", "call_old"),
		},
	})

	if len(plan.Items) != 1 || plan.Items[0].Reason != RehydratePlanReasonEditTargetMissingEvidence {
		t.Fatalf("Store.BuildRehydratePlan() = %#v, want one edit target item", plan)
	}
}

func rehydratePlanTestWorkspace(root string) EvidenceRehydrateOptions {
	return EvidenceRehydrateOptions{RepoRoot: root, InvocationCWD: root}
}

func oldRehydratePlanPointer(path string, startLine, endLine int, source, toolCallID string) EvidencePointer {
	return EvidencePointer{
		Path:       path,
		StartLine:  startLine,
		EndLine:    endLine,
		Source:     source,
		ToolCallID: toolCallID,
		PathBase:   EvidencePointerPathBaseRepoRoot,
	}
}

func writeRehydratePlanTestFile(t *testing.T, root, path string) {
	t.Helper()
	fullPath := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(fullPath), err)
	}
	if err := os.WriteFile(fullPath, []byte("package test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", fullPath, err)
	}
}
