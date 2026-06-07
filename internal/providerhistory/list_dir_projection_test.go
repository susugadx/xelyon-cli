package providerhistory

import (
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

func TestProjectApplyReplacesOldListDirStructuredResult(t *testing.T) {
	oldListDir := providerHistoryTestLargeListDirResult()
	history := providerHistoryTestListDirHistory(t, "call_list", "./internal", oldListDir)

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                                   Apply,
			EvidenceReductionRequiresActiveContext: true,
			ActiveContextTransportAvailable:        false,
		},
	})

	replacement := result.History[1].Content
	for _, want := range []string{"[omitted old list_dir result;", "path=internal", "entries=7", "depth=2"} {
		if !strings.Contains(replacement, want) {
			t.Fatalf("list_dir replacement = %q, want %q", replacement, want)
		}
	}
	if strings.Contains(replacement, "/abs/project") {
		t.Fatalf("list_dir replacement leaked result header absolute path: %q", replacement)
	}
	if history[1].Content != oldListDir {
		t.Fatalf("raw history mutated to %q", history[1].Content)
	}
	if result.Report.ReplacedCount != 1 || !result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want one replacement and response chain disabled", result.Report)
	}
	if got := result.Report.ContentReplacementToolCounts["list_dir"]; got != 1 {
		t.Fatalf("ContentReplacementToolCounts[list_dir] = %d, want 1 in %#v", got, result.Report.ContentReplacementToolCounts)
	}
	if pointers := AppliedEvidencePointers(result.Report); len(pointers) != 0 {
		t.Fatalf("AppliedEvidencePointers() = %#v, want list_dir to stay out of evidence", pointers)
	}
	plan := BuildRehydratePlan(taskstate.NewStoreWithRoot(t.TempDir()), result.Report, []string{"internal"})
	if len(plan.Items) != 0 {
		t.Fatalf("BuildRehydratePlan() = %#v, want empty for list_dir-only replacement", plan)
	}
	providerHistoryTestAssertByteMetrics(t, history, result.History, result.Report)
}

func TestAppliedEvidencePointersIgnoresStructuredListDirCandidates(t *testing.T) {
	pointer := taskstate.EvidencePointer{Path: "internal", StartLine: 1, Source: "list_dir", ToolCallID: "call_list"}
	report := ProjectionReport{
		Candidates: []ReductionCandidate{{
			ToolName:           "list_dir",
			ToolCallID:         "call_list",
			ReplacementApplied: true,
			EvidencePointers:   []taskstate.EvidencePointer{pointer},
		}},
	}

	if pointers := AppliedEvidencePointers(report); len(pointers) != 0 {
		t.Fatalf("AppliedEvidencePointers() = %#v, want structured list_dir ignored", pointers)
	}
}

func TestProjectApplyKeepsEvidenceBackedCandidateWhenTransportUnsupportedButAppliesListDir(t *testing.T) {
	oldRead := strings.Repeat("old evidence line\n", 240)
	oldListDir := providerHistoryTestLargeListDirResult()
	history := []api.Message{
		providerHistoryTestAssistantToolCall("call_read", "read_file"),
		providerHistoryTestToolResult("call_read", "read_file", oldRead),
		{Role: "assistant", Content: "after read"},
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_list", "list_dir", map[string]string{"path": "."})),
		providerHistoryTestToolResult("call_list", "list_dir", oldListDir),
		{Role: "assistant", Content: "after list"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                                   Apply,
			EvidencePointers:                       []taskstate.EvidencePointer{{Path: "src/main.go", StartLine: 1, Source: "read_file", ToolCallID: "call_read"}},
			EvidenceReductionRequiresActiveContext: true,
			ActiveContextTransportAvailable:        false,
		},
	})

	read := providerHistoryTestCandidateByToolCallID(result.Report, "call_read")
	if read == nil || read.ReplacementApplied || read.KeepReason != "active_context_transport_unsupported" {
		t.Fatalf("read candidate = %#v, want active-context keep", read)
	}
	list := providerHistoryTestCandidateByToolCallID(result.Report, "call_list")
	if list == nil || !list.ReplacementApplied {
		t.Fatalf("list_dir candidate = %#v, want replacement despite unsupported active context", list)
	}
	if result.History[1].Content != oldRead {
		t.Fatalf("read projection changed to %q, want original", result.History[1].Content)
	}
	if !strings.Contains(result.History[4].Content, "path=.") {
		t.Fatalf("list_dir projection = %q, want dot path placeholder", result.History[4].Content)
	}
	if result.Report.ReplacementStatus != providerHistoryReplacementStatusPartialApply || !result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want partial apply with chain disabled", result.Report)
	}
}

func TestProjectApplyKeepsUnsafeOrMalformedListDirResults(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		content string
		reason  string
	}{
		{name: "unsafe path", path: "../outside", content: providerHistoryTestLargeListDirResult(), reason: "unsafe_list_dir_path"},
		{name: "absolute path", path: "/tmp/project", content: providerHistoryTestLargeListDirResult(), reason: "unsafe_list_dir_path"},
		{name: "error result", path: "internal", content: "Error: /abs/project is not a directory", reason: "list_dir_result_not_success"},
		{name: "missing summary", path: "internal", content: "📂 /abs/project\nfiles: main.go (10 bytes)", reason: "list_dir_summary_unparseable"},
		{name: "malformed summary", path: "internal", content: "📂 /abs/project\nsummary: dirs=3, files=4", reason: "list_dir_summary_unparseable"},
		{name: "below threshold", path: "internal", content: "📂 /abs/project\nsummary: depth=1, dirs=1, files=1\nfiles: main.go (10 bytes)\n", reason: "replacement_below_min_saved_tokens"},
		{name: "not smaller", path: "internal", content: "📂 /abs/project\nsummary: depth=1, dirs=0, files=0", reason: "replacement_not_smaller"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := providerHistoryTestListDirHistory(t, "call_list", tt.path, tt.content)
			result := Project(ProjectionInput{Messages: history, Policy: Policy{Mode: Apply}})

			if !reflect.DeepEqual(result.History, history) {
				t.Fatalf("projection changed for kept list_dir:\n got %#v\nwant %#v", result.History, history)
			}
			candidate := providerHistoryTestCandidateByToolCallID(result.Report, "call_list")
			if candidate == nil || candidate.ReplacementApplied || candidate.KeepReason != tt.reason {
				t.Fatalf("candidate = %#v, want kept reason %q", candidate, tt.reason)
			}
			if result.Report.ResponsesChainDisabled {
				t.Fatalf("ResponsesChainDisabled = true, want false for kept list_dir")
			}
			providerHistoryTestAssertByteMetrics(t, history, result.History, result.Report)
		})
	}
}

func TestProjectDryRunListDirSavingsRespectThreshold(t *testing.T) {
	large := providerHistoryTestListDirHistory(t, "call_large", "internal/providerhistory", providerHistoryTestLargeListDirResult())
	largeResult := Project(ProjectionInput{Messages: large, Policy: Policy{Mode: DryRun}})
	if !reflect.DeepEqual(largeResult.History, large) {
		t.Fatalf("dry-run changed large list_dir history")
	}
	if largeResult.Report.EstimatedSavedBytes <= 0 || largeResult.Report.ApproxSavedTokens < providerHistoryContentReplacementMinSavedTokens {
		t.Fatalf("large dry-run savings = bytes %d tokens %d, want thresholded positive", largeResult.Report.EstimatedSavedBytes, largeResult.Report.ApproxSavedTokens)
	}
	if got := largeResult.Report.ContentReplacementToolCounts["list_dir"]; got != 1 {
		t.Fatalf("large dry-run tool counts = %#v, want list_dir:1", largeResult.Report.ContentReplacementToolCounts)
	}

	smallContent := "📂 /abs/project\nsummary: depth=1, dirs=1, files=1\nfiles: main.go (10 bytes)\n"
	small := providerHistoryTestListDirHistory(t, "call_small", "internal/providerhistory", smallContent)
	smallResult := Project(ProjectionInput{Messages: small, Policy: Policy{Mode: DryRun}})
	if smallResult.Report.EstimatedSavedBytes != 0 || smallResult.Report.ApproxSavedTokens != 0 {
		t.Fatalf("small dry-run savings = bytes %d tokens %d, want zero below threshold", smallResult.Report.EstimatedSavedBytes, smallResult.Report.ApproxSavedTokens)
	}
	if len(smallResult.Report.ContentReplacementToolCounts) != 0 {
		t.Fatalf("small dry-run tool counts = %#v, want none below threshold", smallResult.Report.ContentReplacementToolCounts)
	}
}

func TestProjectKeepsLatestAndNoLaterAssistantListDirResults(t *testing.T) {
	history := []api.Message{
		providerHistoryTestAssistantToolCall("call_old", "list_dir"),
		providerHistoryTestToolResult("call_old", "list_dir", providerHistoryTestLargeListDirResult()),
		{Role: "assistant", Content: "after old"},
		providerHistoryTestAssistantToolCall("call_latest", "list_dir"),
		providerHistoryTestToolResult("call_latest", "list_dir", providerHistoryTestLargeListDirResult()),
		{Role: "assistant", Content: "done"},
	}
	result := Project(ProjectionInput{Messages: history, Policy: Policy{Mode: Apply}})
	if latest := providerHistoryTestKeptByToolCallID(result.Report, "call_latest"); latest == nil || latest.KeepReason != "latest_tool_result" {
		t.Fatalf("latest kept entry = %#v, want latest_tool_result", latest)
	}

	noLater := []api.Message{
		providerHistoryTestAssistantToolCalls(
			providerHistoryTestToolCallWithJSONArguments(t, "call_list", "list_dir", map[string]string{"path": "internal"}),
			providerHistoryTestToolCallWithJSONArguments(t, "call_bash", "bash", map[string]string{"command": "echo ok"}),
		),
		providerHistoryTestToolResult("call_list", "list_dir", providerHistoryTestLargeListDirResult()),
		providerHistoryTestToolResult("call_bash", "bash", "ok\nProcess exited with code 0"),
		{Role: "user", Content: "continue"},
	}
	noLaterResult := Project(ProjectionInput{Messages: noLater, Policy: Policy{Mode: Apply}})
	if kept := providerHistoryTestKeptByToolCallID(noLaterResult.Report, "call_list"); kept == nil || kept.KeepReason != "no_later_assistant_message" {
		t.Fatalf("no-later kept entry = %#v, want no_later_assistant_message", kept)
	}
}

func providerHistoryTestListDirHistory(t *testing.T, callID, path, content string) []api.Message {
	t.Helper()
	return []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, callID, "list_dir", map[string]string{"path": path, "depth": "2"})),
		providerHistoryTestToolResult(callID, "list_dir", content),
		{Role: "assistant", Content: "after list"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "done"},
	}
}

func providerHistoryTestLargeListDirResult() string {
	return "📂 /abs/project/internal\nsummary: depth=2, dirs=3, files=4\n" +
		strings.Repeat("dirs: providerhistory/, agent/, api/\nfiles: reduction.go (100 bytes), projection.go (200 bytes)\n", 220)
}
