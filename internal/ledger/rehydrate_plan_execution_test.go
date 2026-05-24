package ledger

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRehydratePlanExecution_ReadsCurrentFileRange(t *testing.T) {
	root := t.TempDir()
	writeLedgerTestFile(t, root, "src/main.go", "package main\nfunc main() {}\n")
	plan := RehydratePlan{Items: []RehydratePlanItem{{
		Path:       "src/main.go",
		StartLine:  1,
		EndLine:    2,
		Source:     "read_file",
		Reason:     RehydratePlanReasonOmittedProviderHistory,
		ToolCallID: "call_read",
	}}}

	report := ExecuteRehydratePlan(context.Background(), plan, rehydratePlanTestWorkspace(root), RehydratePlanExecutionOptions{})

	if report.Truncated || len(report.Failures) != 0 || len(report.Block.Items) != 1 {
		t.Fatalf("ExecuteRehydratePlan() = %#v, want one successful item", report)
	}
	item := report.Block.Items[0]
	if item.Path != "src/main.go" ||
		item.StartLine != 1 ||
		item.EndLine != 2 ||
		item.Source != "read_file" ||
		item.Reason != RehydratePlanReasonOmittedProviderHistory ||
		item.ToolCallID != "call_read" ||
		item.Content != "package main\nfunc main() {}" ||
		item.CurrentFileHash == "" ||
		item.Stale {
		t.Fatalf("rehydrated item = %#v, want current src/main.go range", item)
	}
}

func TestRehydratePlanExecution_RejectsUnsafeAndUnreadableItems(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	writeLedgerTestFile(t, root, "one.go", "one\n")
	writeLedgerTestFile(t, root, "binary.go", "a\x00b\n")
	writeLedgerTestFile(t, outside, "outside.go", "outside\n")
	if err := os.Symlink(filepath.Join(outside, "outside.go"), filepath.Join(root, "link.go")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	plan := RehydratePlan{Items: []RehydratePlanItem{
		executionPlanItem(filepath.Join(outside, "outside.go"), 1, 1),
		executionPlanItem("../outside.go", 1, 1),
		executionPlanItem("*.go", 1, 1),
		executionPlanItem("missing.go", 1, 1),
		executionPlanItem("binary.go", 1, 1),
		executionPlanItem("one.go", 0, 1),
		executionPlanItem("one.go", 1, 2),
		executionPlanItem("link.go", 1, 1),
	}}

	report := ExecuteRehydratePlan(context.Background(), plan, rehydratePlanTestWorkspace(root), RehydratePlanExecutionOptions{})

	if len(report.Block.Items) != 0 {
		t.Fatalf("success items = %#v, want none", report.Block.Items)
	}
	want := []EvidenceRehydrateErrorReason{
		EvidenceRehydrateReasonPathEscape,
		EvidenceRehydrateReasonPathEscape,
		EvidenceRehydrateReasonInvalidPath,
		EvidenceRehydrateReasonMissingFile,
		EvidenceRehydrateReasonBinaryFile,
		EvidenceRehydrateReasonInvalidRange,
		EvidenceRehydrateReasonRangeOutOfBounds,
		EvidenceRehydrateReasonPathEscape,
	}
	if len(report.Failures) != len(want) {
		t.Fatalf("failures = %#v, want %d failures", report.Failures, len(want))
	}
	for i, failure := range report.Failures {
		if failure.ErrorReason != want[i] {
			t.Fatalf("failure[%d].ErrorReason = %q, want %q; report=%#v", i, failure.ErrorReason, want[i], report)
		}
	}
}

func TestRehydratePlanExecution_StaleCombinesPlanAndFileHash(t *testing.T) {
	root := t.TempDir()
	writeLedgerTestFile(t, root, "a.go", "a\n")
	writeLedgerTestFile(t, root, "b.go", "b\n")
	matching, err := RehydrateEvidencePointer(context.Background(), EvidencePointer{
		Path:      "b.go",
		StartLine: 1,
		EndLine:   1,
		PathBase:  EvidencePointerPathBaseRepoRoot,
	}, rehydratePlanTestWorkspace(root))
	if err != nil {
		t.Fatalf("RehydrateEvidencePointer() error = %v", err)
	}
	plan := RehydratePlan{Items: []RehydratePlanItem{
		{
			Path:      "a.go",
			StartLine: 1,
			EndLine:   1,
			Source:    "read_file",
			Reason:    RehydratePlanReasonStaleEvidenceRequiresRefresh,
			Stale:     true,
		},
		{
			Path:      "b.go",
			StartLine: 1,
			EndLine:   1,
			Source:    "read_file",
			Reason:    RehydratePlanReasonOmittedProviderHistory,
			FileHash:  matching.CurrentFileHash + "-old",
		},
	}}

	report := ExecuteRehydratePlan(context.Background(), plan, rehydratePlanTestWorkspace(root), RehydratePlanExecutionOptions{})

	if len(report.Block.Items) != 2 {
		t.Fatalf("items = %#v, want two stale items", report.Block.Items)
	}
	for _, item := range report.Block.Items {
		if !item.Stale {
			t.Fatalf("item = %#v, want Stale=true from plan or hash mismatch", item)
		}
	}
}

func TestRehydratePlanExecution_RespectsItemLineAndByteBudgets(t *testing.T) {
	t.Run("max items", func(t *testing.T) {
		root := t.TempDir()
		writeLedgerTestFile(t, root, "a.go", "a\n")
		writeLedgerTestFile(t, root, "b.go", "b\n")
		plan := RehydratePlan{Items: []RehydratePlanItem{
			executionPlanItem("a.go", 1, 1),
			executionPlanItem("b.go", 1, 1),
		}}

		report := ExecuteRehydratePlan(context.Background(), plan, rehydratePlanTestWorkspace(root), RehydratePlanExecutionOptions{MaxItems: 1})

		if !report.Truncated || !report.Block.Truncated || len(report.Block.Items) != 1 || report.Block.Items[0].Path != "a.go" {
			t.Fatalf("report = %#v, want one item and Truncated=true", report)
		}
	})

	t.Run("line budget", func(t *testing.T) {
		root := t.TempDir()
		writeLedgerTestFile(t, root, "a.go", "a1\na2\n")
		writeLedgerTestFile(t, root, "b.go", "b1\nb2\n")
		plan := RehydratePlan{Items: []RehydratePlanItem{
			executionPlanItem("a.go", 1, 2),
			executionPlanItem("b.go", 1, 2),
		}}

		report := ExecuteRehydratePlan(context.Background(), plan, rehydratePlanTestWorkspace(root), RehydratePlanExecutionOptions{MaxTotalLines: 3})

		if !report.Truncated || len(report.Block.Items) != 1 || report.Block.Items[0].Path != "a.go" {
			t.Fatalf("report = %#v, want second item omitted by line budget", report)
		}
	})

	t.Run("byte budget", func(t *testing.T) {
		root := t.TempDir()
		writeLedgerTestFile(t, root, "large.go", strings.Repeat("large line\n", 20))
		plan := RehydratePlan{Items: []RehydratePlanItem{executionPlanItem("large.go", 1, 20)}}

		report := ExecuteRehydratePlan(context.Background(), plan, rehydratePlanTestWorkspace(root), RehydratePlanExecutionOptions{MaxTotalBytes: 120})

		if !report.Truncated || len(report.Block.Items) != 0 || RenderRehydratedEvidenceBlock(report.Block) != "" {
			t.Fatalf("report = %#v, want byte-over item omitted without partial render", report)
		}
	})
}

func TestRehydratePlanStoreExecuteUsesWorkspace(t *testing.T) {
	root := t.TempDir()
	writeLedgerTestFile(t, root, "src/main.go", "one\n")
	store := NewStoreWithRoot(root)
	plan := RehydratePlan{Items: []RehydratePlanItem{executionPlanItem("src/main.go", 1, 1)}}

	report := store.ExecuteRehydratePlan(context.Background(), plan, RehydratePlanExecutionOptions{})

	if len(report.Block.Items) != 1 || report.Block.Items[0].Content != "one" {
		t.Fatalf("Store.ExecuteRehydratePlan() = %#v, want one current file item", report)
	}
}

func TestRehydratedEvidenceRenderIncludesMarkersMetadataAndLineNumbers(t *testing.T) {
	block := RehydratedEvidenceBlock{Items: []RehydratedEvidenceItem{{
		Path:       "src/main.go",
		StartLine:  10,
		EndLine:    11,
		Source:     "read_file",
		Reason:     RehydratePlanReasonStaleEvidenceRequiresRefresh,
		ToolCallID: "call_read",
		Content:    "line ten\nline eleven",
		Stale:      true,
	}}}

	rendered := RenderRehydratedEvidenceBlock(block)

	for _, want := range []string{
		RehydratedEvidenceStartMarker,
		"RehydratedEvidence:",
		"- path: src/main.go",
		"  range: L10-L11",
		"  source: read_file",
		"  reason: stale_evidence_requires_refresh",
		"  stale: true",
		"  tool_call_id: call_read",
		"  warning: stale evidence",
		"    L10: line ten",
		"    L11: line eleven",
		RehydratedEvidenceEndMarker,
	} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered block missing %q:\n%s", want, rendered)
		}
	}
}

func TestRehydratedEvidenceRenderEmptyBlock(t *testing.T) {
	if got := RenderRehydratedEvidenceBlock(RehydratedEvidenceBlock{}); got != "" {
		t.Fatalf("RenderRehydratedEvidenceBlock(empty) = %q, want empty string", got)
	}
}

func executionPlanItem(path string, startLine, endLine int) RehydratePlanItem {
	return RehydratePlanItem{
		Path:       path,
		StartLine:  startLine,
		EndLine:    endLine,
		Source:     "read_file",
		Reason:     RehydratePlanReasonOmittedProviderHistory,
		ToolCallID: "call_read",
	}
}
