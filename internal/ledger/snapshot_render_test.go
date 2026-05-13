package ledger

import (
	"reflect"
	"strconv"
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
		"- internal/ledger/ledger.go:L20-L22 source=read_file id=call_x file_hash=hash stale=true excerpt=\"type RuntimeTaskState struct { ChangedFiles ChangedFiles }\"",
		"Recommended reads:",
		"- internal/agent/runtime.go reason=\"owner boundary\" source=read_file id=call_x",
		"Last failed tests:",
		"- failed: go test ./internal/agent exit=1 excerpt=\"FAIL internal/agent panic trace\"",
		"Last passed tests:",
		"- passed: go test ./internal/ledger",
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
			NewTestResultWithExitCode("go test ./internal/ledger", 0, "passed", "ok\ninternal/ledger"),
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
		"- passed: go test ./internal/ledger excerpt=\"ok internal/ledger\"",
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

func populatedSnapshotRenderState() RuntimeTaskState {
	return RuntimeTaskState{
		ChangedFiles: ChangedFiles{files: []fileFact{{path: "src/main.go"}}},
		TouchedFiles: TouchedFiles{files: []fileFact{{path: "docs/commands.md"}}},
		Evidence: Evidence{items: []evidenceFact{{
			path:       "internal/ledger/ledger.go",
			startLine:  20,
			endLine:    22,
			source:     "read_file",
			toolCallID: "call_x",
			fileHash:   "hash",
			stale:      true,
			excerpt:    "type RuntimeTaskState struct {\nChangedFiles ChangedFiles\n}",
		}}},
		RecommendedReads: RecommendedReads{items: []recommendedReadFact{{
			path:       "internal/agent/runtime.go",
			reason:     "owner boundary",
			source:     "read_file",
			toolCallID: "call_x",
		}}},
		LastFailedTests: LastFailedTests{results: []TestResult{
			NewTestResultWithExitCode("go test ./internal/agent", 1, "failed", "FAIL internal/agent\npanic trace"),
		}},
		LastPassedTests: LastPassedTests{results: []TestResult{
			NewTestResultWithExitCode("go test ./internal/ledger", 0, "passed", ""),
		}},
	}
}

func snapshotRenderStateWithThreeFactsPerSection() RuntimeTaskState {
	return RuntimeTaskState{
		ChangedFiles: ChangedFiles{files: numberedFileFacts("src/changed_", 3)},
		TouchedFiles: TouchedFiles{files: numberedFileFacts("src/touched_", 3)},
		Evidence: Evidence{items: []evidenceFact{
			{path: "src/evidence_1.go", startLine: 1, endLine: 1, excerpt: "one"},
			{path: "src/evidence_2.go", startLine: 2, endLine: 2, excerpt: "two"},
			{path: "src/evidence_3.go", startLine: 3, endLine: 3, excerpt: "three"},
		}},
		RecommendedReads: RecommendedReads{items: numberedRecommendedReadFacts("src/read_", 3)},
		LastFailedTests:  LastFailedTests{results: numberedFailedTestResults(3)},
		LastPassedTests:  LastPassedTests{results: numberedPassedTestResults(3, "")},
	}
}

func snapshotRenderStateExceedingDefaultLimits() RuntimeTaskState {
	return RuntimeTaskState{
		ChangedFiles:     ChangedFiles{files: numberedFileFacts("src/changed_", 21)},
		TouchedFiles:     TouchedFiles{files: numberedFileFacts("src/touched_", 31)},
		Evidence:         Evidence{items: numberedEvidenceFacts("src/evidence_", 21)},
		RecommendedReads: RecommendedReads{items: numberedRecommendedReadFacts("src/read_", 11)},
		LastFailedTests:  LastFailedTests{results: numberedFailedTestResults(4)},
		LastPassedTests:  LastPassedTests{results: numberedPassedTestResults(4, "passed")},
	}
}

func numberedFileFacts(prefix string, count int) []fileFact {
	files := make([]fileFact, 0, count)
	for i := 1; i <= count; i++ {
		files = append(files, fileFact{path: prefix + strconv.Itoa(i) + ".go"})
	}
	return files
}

func numberedEvidenceFacts(prefix string, count int) []evidenceFact {
	items := make([]evidenceFact, 0, count)
	for i := 1; i <= count; i++ {
		items = append(items, evidenceFact{
			path:      prefix + strconv.Itoa(i) + ".go",
			startLine: i,
			endLine:   i,
			excerpt:   "evidence " + strconv.Itoa(i),
		})
	}
	return items
}

func numberedRecommendedReadFacts(prefix string, count int) []recommendedReadFact {
	items := make([]recommendedReadFact, 0, count)
	for i := 1; i <= count; i++ {
		items = append(items, recommendedReadFact{path: prefix + strconv.Itoa(i) + ".go"})
	}
	return items
}

func numberedFailedTestResults(count int) []TestResult {
	return numberedTestResults("failed", 1, count, "failed")
}

func numberedPassedTestResults(count int, excerptPrefix string) []TestResult {
	return numberedTestResults("passed", 0, count, excerptPrefix)
}

func numberedTestResults(status string, exitCode, count int, excerptPrefix string) []TestResult {
	results := make([]TestResult, 0, count)
	for i := 1; i <= count; i++ {
		command := "go test ./" + status + strconv.Itoa(i)
		excerpt := ""
		if excerptPrefix != "" {
			excerpt = excerptPrefix + " " + strconv.Itoa(i)
		}
		results = append(results, NewTestResultWithExitCode(command, exitCode, status, excerpt))
	}
	return results
}

func assertSnapshotRenderEquals(t *testing.T, got string, wantLines ...string) {
	t.Helper()
	want := snapshotText(wantLines...)
	if got != want {
		t.Fatalf("snapshot =\n%s\nwant =\n%s", got, want)
	}
}

func snapshotText(lines ...string) string {
	return strings.Join(lines, "\n")
}

func assertSnapshotRenderContains(t *testing.T, output string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(output, fragment) {
			t.Fatalf("snapshot missing %q:\n%s", fragment, output)
		}
	}
}

func assertSnapshotRenderOmits(t *testing.T, output string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if strings.Contains(output, fragment) {
			t.Fatalf("snapshot should not contain %q:\n%s", fragment, output)
		}
	}
}
