package taskstate

import (
	"reflect"
	"strings"
	"testing"
)

func TestRenderCurrentTaskStateSnapshot_EmptyState(t *testing.T) {
	got := RenderCurrentTaskStateSnapshot(RuntimeTaskState{}, SnapshotRenderOptions{})
	assertSnapshotRenderEquals(t, got,
		"<current_task_state>",
		"CurrentTaskState:",
		"- No runtime task facts recorded yet.",
		"</current_task_state>",
	)
}

func TestRuntimeTaskStateIsEmpty_MatchesSnapshotRenderer(t *testing.T) {
	empty := RuntimeTaskState{}
	if !empty.IsEmpty() {
		t.Fatal("empty RuntimeTaskState IsEmpty() = false, want true")
	}
	assertSnapshotRenderContains(t, RenderCurrentTaskStateSnapshot(empty, SnapshotRenderOptions{}), "- No runtime task facts recorded yet.")

	populated := populatedSnapshotRenderState()
	if populated.IsEmpty() {
		t.Fatal("populated RuntimeTaskState IsEmpty() = true, want false")
	}
	assertSnapshotRenderOmits(t, RenderCurrentTaskStateSnapshot(populated, SnapshotRenderOptions{}), "- No runtime task facts recorded yet.")
}

func TestRenderCurrentTaskStateSnapshot_CompactFormat(t *testing.T) {
	state := populatedSnapshotRenderState()

	got := RenderCurrentTaskStateSnapshot(state, SnapshotRenderOptions{})
	assertSnapshotRenderEquals(t, got,
		"<current_task_state>",
		"CurrentTaskState:",
		"Changed files:",
		"- src/main.go",
		"Recently touched files:",
		"- docs/commands.md",
		"Evidence pointers:",
		"- internal/taskstate/taskstate.go:L20-L22 source=read_file id=call_x file_hash=hash stale=true excerpt=\"type RuntimeTaskState struct { ChangedFiles ChangedFiles }\"",
		"Recommended reads:",
		"- internal/agent/runtime.go reason=\"owner boundary\" source=read_file id=call_x",
		"Last failed tests:",
		"- failed: go test ./internal/agent exit=1 excerpt=\"FAIL internal/agent panic trace\"",
		"Last passed tests:",
		"- passed: go test ./internal/taskstate",
		"</current_task_state>",
	)
}

func TestRenderCurrentTaskStateSnapshot_SectionLimits(t *testing.T) {
	state := snapshotRenderStateWithThreeFactsPerSection()
	opts := SnapshotRenderOptions{
		ChangedFilesLimit:     2,
		TouchedFilesLimit:     2,
		EvidenceLimit:         2,
		RecommendedReadsLimit: 2,
		FailedTestsLimit:      2,
		PassedTestsLimit:      2,
	}

	got := RenderCurrentTaskStateSnapshot(state, opts)

	if count := strings.Count(got, "- ... 1 more omitted"); count != 6 {
		t.Fatalf("omitted line count = %d, want 6:\n%s", count, got)
	}
	assertSnapshotRenderOmits(t, got,
		"src/changed_3.go",
		"src/touched_3.go",
		"src/evidence_3.go",
		"src/read_3.go",
		"go test ./failed3",
		"go test ./passed3",
	)
}

func TestRenderCurrentTaskStateSnapshot_NonPositiveLimitsUseDefaults(t *testing.T) {
	state := snapshotRenderStateExceedingDefaultLimits()
	opts := SnapshotRenderOptions{
		ChangedFilesLimit:     -1,
		TouchedFilesLimit:     -1,
		EvidenceLimit:         -1,
		RecommendedReadsLimit: -1,
		FailedTestsLimit:      -1,
		PassedTestsLimit:      -1,
		ExcerptRuneLimit:      -1,
	}

	got := RenderCurrentTaskStateSnapshot(state, opts)

	assertSnapshotRenderContains(t, got,
		"- src/changed_20.go",
		"- src/touched_30.go",
		"- src/evidence_20.go:L20-L20 excerpt=\"evidence 20\"",
		"- src/read_10.go",
		"- failed: go test ./failed3 exit=1 excerpt=\"failed 3\"",
		"- passed: go test ./passed3 excerpt=\"passed 3\"",
	)
	if count := strings.Count(got, "- ... 1 more omitted"); count != 6 {
		t.Fatalf("default omitted line count = %d, want 6:\n%s", count, got)
	}
	assertSnapshotRenderOmits(t, got,
		"- src/changed_21.go",
		"- src/touched_31.go",
		"- src/evidence_21.go",
		"- src/read_11.go",
		"go test ./failed4",
		"go test ./passed4",
	)
}

func TestRenderCurrentTaskStateSnapshot_ExcerptSingleLineAndDefaultTruncation(t *testing.T) {
	const hiddenTail = "SHOULD_NOT_APPEAR"
	longExcerpt := strings.Repeat("x", 150) + "\n" + hiddenTail
	state := RuntimeTaskState{
		Evidence: Evidence{items: []evidenceFact{{
			path:      "src/long.go",
			startLine: 10,
			endLine:   12,
			source:    "read_file",
			excerpt:   longExcerpt,
		}}},
	}

	got := RenderCurrentTaskStateSnapshot(state, SnapshotRenderOptions{})

	assertSnapshotRenderContains(t, got, `excerpt="`+strings.Repeat("x", 137)+`..."`)
	assertSnapshotRenderOmits(t, got, hiddenTail)
}

func TestRenderCurrentTaskStateSnapshot_FileHashAndStaleAreConditional(t *testing.T) {
	state := RuntimeTaskState{
		Evidence: Evidence{items: []evidenceFact{
			{path: "src/fresh.go", startLine: 1, endLine: 1, source: "read_file", excerpt: "fresh"},
			{path: "src/stale.go", startLine: 2, endLine: 2, source: "read_file", fileHash: "hash", stale: true, excerpt: "stale"},
		}},
	}

	got := RenderCurrentTaskStateSnapshot(state, SnapshotRenderOptions{})
	assertSnapshotRenderEquals(t, got,
		"<current_task_state>",
		"CurrentTaskState:",
		"Evidence pointers:",
		"- src/fresh.go:L1-L1 source=read_file excerpt=\"fresh\"",
		"- src/stale.go:L2-L2 source=read_file file_hash=hash stale=true excerpt=\"stale\"",
		"</current_task_state>",
	)
}

func TestRenderCurrentTaskStateSnapshot_OptionalAttributesAndPassedExcerpt(t *testing.T) {
	state := RuntimeTaskState{
		Evidence: Evidence{items: []evidenceFact{{
			path:      "src/evidence.go",
			startLine: 3,
			endLine:   4,
			excerpt:   "line with spaces",
		}}},
		RecommendedReads: RecommendedReads{items: []recommendedReadFact{{
			path: "src/next.go",
		}}},
		LastPassedTests: LastPassedTests{results: []TestResult{
			NewTestResultWithExitCode("go test ./internal/taskstate", 0, "passed", "ok\ninternal/taskstate"),
		}},
	}

	got := RenderCurrentTaskStateSnapshot(state, SnapshotRenderOptions{})
	assertSnapshotRenderEquals(t, got,
		"<current_task_state>",
		"CurrentTaskState:",
		"Evidence pointers:",
		"- src/evidence.go:L3-L4 excerpt=\"line with spaces\"",
		"Recommended reads:",
		"- src/next.go",
		"Last passed tests:",
		"- passed: go test ./internal/taskstate excerpt=\"ok internal/taskstate\"",
		"</current_task_state>",
	)
}

func TestRenderCurrentTaskStateSnapshot_Deterministic(t *testing.T) {
	state := populatedSnapshotRenderState()
	opts := SnapshotRenderOptions{EvidenceLimit: 5, ExcerptRuneLimit: 80}

	first := RenderCurrentTaskStateSnapshot(state, opts)
	second := RenderCurrentTaskStateSnapshot(state, opts)

	if first != second {
		t.Fatalf("snapshot changed between renders:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestRenderCurrentTaskStateSnapshot_DoesNotMutateRuntimeTaskState(t *testing.T) {
	state := populatedSnapshotRenderState()
	before := state.clone()

	_ = RenderCurrentTaskStateSnapshot(state, SnapshotRenderOptions{ExcerptRuneLimit: 20})

	if !reflect.DeepEqual(state, before) {
		t.Fatalf("state mutated:\ngot:  %#v\nwant: %#v", state, before)
	}
}
