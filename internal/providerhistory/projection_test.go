package providerhistory

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

func TestProjectDryRunDetectsReductionCandidatesAndReportMetrics(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "inspect the repo"},
		providerHistoryTestAssistantToolCall("call_read_old", "read_file"),
		providerHistoryTestToolResult("call_read_old", "read_file", "old read_file output"),
		{Role: "assistant", Content: "I read it"},
		providerHistoryTestAssistantToolCalls(
			providerHistoryTestToolCall("call_search_old", "search_code"),
			providerHistoryTestToolCall("call_gather_old", "gather_context"),
		),
		providerHistoryTestToolResult("call_search_old", "", "old search_code output"),
		providerHistoryTestToolResult("call_gather_old", "gather_context", "old gather_context output"),
		{Role: "assistant", Content: "I have enough evidence"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest read_file output"),
		{Role: "assistant", Content: "final answer"},
	}

	result := Project(ProjectionInput{
		Messages: history,
		Policy:   Policy{Mode: DryRun},
	})

	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("dry-run projection changed history:\n got %#v\nwant %#v", result.History, history)
	}
	report := result.Report
	if report.Mode != DryRun {
		t.Fatalf("report mode = %v, want dry-run", report.Mode)
	}
	if report.ToolResultCount != 4 || report.CandidateCount != 3 || report.ReplacedCount != 0 || report.KeptCount != 4 {
		t.Fatalf("report counts = tool %d candidates %d replaced %d kept %d, want 4/3/0/4", report.ToolResultCount, report.CandidateCount, report.ReplacedCount, report.KeptCount)
	}
	if report.ReplacementStatus != providerHistoryReplacementStatusNotImplemented {
		t.Fatalf("ReplacementStatus = %q, want not_implemented for dry-run", report.ReplacementStatus)
	}
	if got, want := providerHistoryTestCandidateTools(report), []string{"read_file", "search_code", "gather_context"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("candidate tools = %#v, want %#v", got, want)
	}
	if latest := providerHistoryTestKeptByToolCallID(report, "call_latest"); latest == nil || latest.KeepReason != "latest_tool_result" {
		t.Fatalf("latest kept entry = %#v, want latest_tool_result", latest)
	}
	providerHistoryTestAssertByteMetrics(t, history, result.History, report)
}

func TestProjectApplyEvidenceReductionUsesPolicyPointersAndDefensiveCopies(t *testing.T) {
	oldRead := strings.Repeat("important evidence line\n", 240)
	history := providerHistoryTestReductionHistory("call_read_old", oldRead)
	pointer := taskstate.EvidencePointer{Path: "src/main.go", StartLine: 7, EndLine: 9, Source: "read_file", ToolCallID: "call_read_old"}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:             Apply,
			EvidencePointers: []taskstate.EvidencePointer{pointer},
		},
	})

	if result.History[1].Content == oldRead || !strings.Contains(result.History[1].Content, "[omitted old read_file result; evidence:") {
		t.Fatalf("projected read_file result = %q, want evidence placeholder", result.History[1].Content)
	}
	if history[1].Content != oldRead {
		t.Fatalf("raw history mutated to %q", history[1].Content)
	}
	if result.Report.ReplacedCount != 1 || !result.Report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want one replacement and response chain disabled", result.Report)
	}
	if result.Report.ReplacementStatus != providerHistoryReplacementStatusApply {
		t.Fatalf("ReplacementStatus = %q, want apply when all candidates are replaced", result.Report.ReplacementStatus)
	}
	applied := AppliedEvidencePointers(result.Report)
	if !reflect.DeepEqual(applied, []taskstate.EvidencePointer{pointer}) {
		t.Fatalf("AppliedEvidencePointers() = %#v, want %#v", applied, []taskstate.EvidencePointer{pointer})
	}
	applied[0].Path = "mutated.go"
	if result.Report.Candidates[0].EvidencePointers[0].Path != "src/main.go" {
		t.Fatalf("AppliedEvidencePointers() did not return a defensive copy: %#v", result.Report.Candidates[0].EvidencePointers)
	}
}

func TestProjectApplyEvidenceReductionSyncsOpenAIResponsesReplayOutput(t *testing.T) {
	oldRead := strings.Repeat("important evidence line\n", 240)
	history := providerHistoryTestReductionHistory("call_read_old", oldRead)
	history[1].SetOpenAIResponsesInputItems([]api.InputItem{{
		Type:   "function_call_output",
		CallID: "call_read_old",
		Output: oldRead,
	}})
	pointer := taskstate.EvidencePointer{Path: "src/main.go", StartLine: 7, EndLine: 9, Source: "read_file", ToolCallID: "call_read_old"}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:             Apply,
			EvidencePointers: []taskstate.EvidencePointer{pointer},
		},
	})

	items := result.History[1].OpenAIResponsesInputItems()
	if len(items) != 1 {
		t.Fatalf("len(OpenAIResponsesInputItems) = %d, want 1", len(items))
	}
	if items[0].Output == oldRead || items[0].Output != result.History[1].Content {
		t.Fatalf("replay function_call_output = %q, want projected content %q", items[0].Output, result.History[1].Content)
	}
	if history[1].OpenAIResponsesInputItems()[0].Output != oldRead {
		t.Fatalf("raw history replay metadata was mutated: %#v", history[1].OpenAIResponsesInputItems())
	}
}

func TestProjectApplyKeepsEvidenceReductionWhenActiveContextTransportUnsupportedButAppliesCommand(t *testing.T) {
	oldRead := strings.Repeat("old read output\n", 220)
	commandOutput := providerHistoryTestLargeSuccessfulTestOutput()
	history := []api.Message{
		providerHistoryTestAssistantToolCall("call_read_old", "read_file"),
		providerHistoryTestToolResult("call_read_old", "read_file", oldRead),
		{Role: "assistant", Content: "after read"},
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_test", "bash", map[string]string{"command": "go test ./internal/providerhistory"})),
		providerHistoryTestToolResult("call_test", "bash", commandOutput),
		{Role: "assistant", Content: "tests passed"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:                                   Apply,
			EvidencePointers:                       []taskstate.EvidencePointer{{Path: "src/main.go", StartLine: 1, Source: "read_file", ToolCallID: "call_read_old"}},
			EvidenceReductionRequiresActiveContext: true,
			ActiveContextTransportAvailable:        false,
		},
	})

	read := providerHistoryTestCandidateByToolCallID(result.Report, "call_read_old")
	if read == nil || read.ReplacementApplied || read.KeepReason != "active_context_transport_unsupported" {
		t.Fatalf("read candidate = %#v, want kept by active_context_transport_unsupported", read)
	}
	if result.History[1].Content != oldRead {
		t.Fatalf("read projection changed to %q, want original old read", result.History[1].Content)
	}
	if result.History[4].Content == commandOutput || !strings.Contains(result.History[4].Content, "successful validation command output") {
		t.Fatalf("command output = %q, want successful command placeholder", result.History[4].Content)
	}
	if result.Report.CommandEditDryRun.CommandReplacedCount != 1 || !result.Report.ResponsesChainDisabled {
		t.Fatalf("command report = %#v, want command replacement and response chain disabled", result.Report.CommandEditDryRun)
	}
	if result.Report.ReplacementStatus != providerHistoryReplacementStatusPartialApply {
		t.Fatalf("ReplacementStatus = %q, want partial_apply when read candidate is kept and command is replaced", result.Report.ReplacementStatus)
	}
}

func TestProjectReplacesOldWriteFileContentOnlyAfterSuccessfulMatchingResult(t *testing.T) {
	content := strings.Repeat("package main\n\nfunc generated() string { return \"x\" }\n", 260)
	args := providerHistoryTestJSONArguments(t, map[string]string{"path": "src/generated.go", "content": content})
	history := providerHistoryTestWriteFileHistory("call_write", args, providerHistoryTestWriteFileSuccess(content, "src/generated.go"))

	result := Project(ProjectionInput{
		Messages: history,
		Policy:   Policy{Mode: Apply},
	})

	if result.Report.CommandEditDryRun.EditArgReplacedCount != 1 || !result.Report.ResponsesChainDisabled {
		t.Fatalf("command/edit report = %#v, want one write_file.content replacement", result.Report.CommandEditDryRun)
	}
	if result.Report.EstimatedSavedBytes != result.Report.CommandEditDryRun.EditArgReplacementSavedBytes ||
		result.Report.ApproxSavedTokens != result.Report.CommandEditDryRun.ApproxEditArgReplacementSavedTokens {
		t.Fatalf("top-level savings = bytes %d tokens %d, want write_file.content savings %d/%d", result.Report.EstimatedSavedBytes, result.Report.ApproxSavedTokens, result.Report.CommandEditDryRun.EditArgReplacementSavedBytes, result.Report.CommandEditDryRun.ApproxEditArgReplacementSavedTokens)
	}
	replacement := providerHistoryTestWriteFileContentArgument(t, result.History[0].ToolCalls[0].Function.Arguments, "src/generated.go")
	if replacement == content || !strings.HasPrefix(replacement, "[omitted old write_file.content; path=src/generated.go]") {
		t.Fatalf("projected write_file.content = %q, want placeholder", replacement)
	}
	if history[0].ToolCalls[0].Function.Arguments != args {
		t.Fatalf("raw write_file arguments mutated to %s", history[0].ToolCalls[0].Function.Arguments)
	}
}

func TestProjectDryRunEstimatesEditArgSavingsWithoutChangingPayload(t *testing.T) {
	content := strings.Repeat("package dryrun\n\nfunc generated() string { return \"x\" }\n", 260)
	args := providerHistoryTestJSONArguments(t, map[string]string{"path": "src/dryrun.go", "content": content})
	history := providerHistoryTestWriteFileHistory("call_write", args, providerHistoryTestWriteFileSuccess(content, "src/dryrun.go"))

	result := Project(ProjectionInput{
		Messages: history,
		Policy:   Policy{Mode: DryRun},
	})

	if !reflect.DeepEqual(result.History, history) {
		t.Fatalf("dry-run projection changed history:\n got %#v\nwant %#v", result.History, history)
	}
	report := result.Report.CommandEditDryRun
	if report.EditArgCandidates != 1 ||
		report.EditArgReplacedCount != 0 ||
		report.EditArgEstimatedSavedBytes <= 0 ||
		report.ApproxEditArgSavedTokens < providerHistoryEditArgReplacementMinSavedTokens ||
		result.Report.ResponsesChainDisabled {
		t.Fatalf("dry-run report = %#v / top-level %#v, want edit-arg estimate without replacement", report, result.Report)
	}
	if result.Report.EstimatedSavedBytes != report.EditArgEstimatedSavedBytes ||
		result.Report.ApproxSavedTokens != report.ApproxEditArgSavedTokens {
		t.Fatalf("top-level dry-run savings = bytes %d tokens %d, want edit estimate %d/%d", result.Report.EstimatedSavedBytes, result.Report.ApproxSavedTokens, report.EditArgEstimatedSavedBytes, report.ApproxEditArgSavedTokens)
	}
}

func TestProjectApplyTotalsContentAndEditArgSavingsWithoutDoubleCounting(t *testing.T) {
	oldRead := strings.Repeat("old evidence line for provider projection\n", 240)
	writeContent := strings.Repeat("package generated\n\nfunc value() string { return \"x\" }\n", 260)
	writeArgs := providerHistoryTestJSONArguments(t, map[string]string{"path": "src/generated.go", "content": writeContent})
	history := []api.Message{
		providerHistoryTestAssistantToolCall("call_read", "read_file"),
		providerHistoryTestToolResult("call_read", "read_file", oldRead),
		{Role: "assistant", Content: "read done"},
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithArguments("call_write", "write_file", writeArgs)),
		providerHistoryTestToolResult("call_write", "write_file", providerHistoryTestWriteFileSuccess(writeContent, "src/generated.go")),
		{Role: "assistant", Content: "write done"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:             Apply,
			EvidencePointers: []taskstate.EvidencePointer{{Path: "src/main.go", StartLine: 1, Source: "read_file", ToolCallID: "call_read"}},
		},
	})

	if result.Report.ReplacedCount != 1 || result.Report.CommandEditDryRun.EditArgReplacedCount != 1 {
		t.Fatalf("report = %#v, want one content and one edit-arg replacement", result.Report)
	}
	wantBytes := result.Report.ContentReplacementSavedBytes + result.Report.CommandEditDryRun.EditArgReplacementSavedBytes
	wantTokens := result.Report.ApproxContentReplacementSavedTokens + result.Report.CommandEditDryRun.ApproxEditArgReplacementSavedTokens
	if result.Report.EstimatedSavedBytes != wantBytes || result.Report.ApproxSavedTokens != wantTokens {
		t.Fatalf("top-level savings = bytes %d tokens %d, want content+edit %d/%d", result.Report.EstimatedSavedBytes, result.Report.ApproxSavedTokens, wantBytes, wantTokens)
	}
	providerHistoryTestAssertByteMetrics(t, history, result.History, result.Report)
}

func TestProjectPreservesNilAndEmptyInputShape(t *testing.T) {
	if got := Project(ProjectionInput{}).History; got != nil {
		t.Fatalf("Project(nil).History = %#v, want nil", got)
	}
	empty := []api.Message{}
	got := Project(ProjectionInput{Messages: empty}).History
	if got == nil || len(got) != 0 {
		t.Fatalf("Project(empty).History = %#v, want non-nil empty slice", got)
	}
}

func TestCloneProjectionReportCopiesNestedState(t *testing.T) {
	report := ProjectionReport{
		KeptReasonCounts:             map[string]int{"missing_evidence_pointer": 1},
		ContentReplacementToolCounts: map[string]int{"list_dir": 1},
		FutureFamilyKeptReasonCounts: map[string]int{"wait_agent_freeform_output_keep": 1},
		Candidates: []ReductionCandidate{{
			ToolName:         "read_file",
			EvidencePointers: []taskstate.EvidencePointer{{Path: "src/main.go", StartLine: 1, EndLine: 2}},
		}},
		CommandEditDryRun: CommandEditDryRunReport{
			CandidateReasonCounts:              map[string]int{"validation_success": 1},
			CommandReplacementClassifierCounts: map[string]int{"validation": 1},
			KeptReasonCounts:                   map[string]int{"latest_tool_result": 1},
			Candidates:                         []CommandEditDryRunCandidate{{ToolName: "bash"}},
			Kept:                               []CommandEditDryRunCandidate{{ToolName: "write_file"}},
		},
	}

	cloned := CloneProjectionReport(report)
	cloned.KeptReasonCounts["missing_evidence_pointer"] = 2
	cloned.ContentReplacementToolCounts["list_dir"] = 2
	cloned.FutureFamilyKeptReasonCounts["wait_agent_freeform_output_keep"] = 2
	cloned.Candidates[0].EvidencePointers[0].Path = "mutated.go"
	cloned.CommandEditDryRun.CandidateReasonCounts["validation_success"] = 2
	cloned.CommandEditDryRun.CommandReplacementClassifierCounts["validation"] = 2
	cloned.CommandEditDryRun.KeptReasonCounts["latest_tool_result"] = 2
	cloned.CommandEditDryRun.Candidates[0].ToolName = "command"
	cloned.CommandEditDryRun.Kept[0].ToolName = "apply_patch"

	if report.KeptReasonCounts["missing_evidence_pointer"] != 1 ||
		report.ContentReplacementToolCounts["list_dir"] != 1 ||
		report.FutureFamilyKeptReasonCounts["wait_agent_freeform_output_keep"] != 1 ||
		report.Candidates[0].EvidencePointers[0].Path != "src/main.go" ||
		report.CommandEditDryRun.CandidateReasonCounts["validation_success"] != 1 ||
		report.CommandEditDryRun.CommandReplacementClassifierCounts["validation"] != 1 ||
		report.CommandEditDryRun.KeptReasonCounts["latest_tool_result"] != 1 ||
		report.CommandEditDryRun.Candidates[0].ToolName != "bash" ||
		report.CommandEditDryRun.Kept[0].ToolName != "write_file" {
		t.Fatalf("original report mutated: %#v", report)
	}
}

func TestSyntheticProjectionReportsSavedBytesAndTokens(t *testing.T) {
	oldRead := strings.Repeat("line from old read\n", 300)
	commandOutput := providerHistoryTestLargeSuccessfulTestOutput()
	history := []api.Message{
		providerHistoryTestAssistantToolCall("call_read_old", "read_file"),
		providerHistoryTestToolResult("call_read_old", "read_file", oldRead),
		{Role: "assistant", Content: "after read"},
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_test", "bash", map[string]string{"command": "go test ./..."})),
		providerHistoryTestToolResult("call_test", "bash", commandOutput),
		{Role: "assistant", Content: "after tests"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:             Apply,
			EvidencePointers: []taskstate.EvidencePointer{{Path: "src/main.go", StartLine: 1, Source: "read_file", ToolCallID: "call_read_old"}},
		},
	})

	if result.Report.OriginalMessageCount != len(history) || result.Report.ProjectedMessageCount != len(history) {
		t.Fatalf("message counts = %d/%d, want %d", result.Report.OriginalMessageCount, result.Report.ProjectedMessageCount, len(history))
	}
	if result.Report.EstimatedSavedBytes <= 0 || result.Report.ApproxSavedTokens <= 0 {
		t.Fatalf("saved metrics = bytes %d tokens %d, want positive", result.Report.EstimatedSavedBytes, result.Report.ApproxSavedTokens)
	}
	providerHistoryTestAssertByteMetrics(t, history, result.History, result.Report)
}

func providerHistoryTestToolCall(id, name string) api.OpenAIToolCall {
	return providerHistoryTestToolCallWithArguments(id, name, "{}")
}

func providerHistoryTestToolCallWithArguments(id, name, arguments string) api.OpenAIToolCall {
	return api.OpenAIToolCall{
		ID:       id,
		Type:     "function",
		Function: api.OpenAIToolCallFunction{Name: name, Arguments: arguments},
	}
}

func providerHistoryTestToolCallWithJSONArguments(t *testing.T, id, name string, arguments map[string]string) api.OpenAIToolCall {
	t.Helper()
	return providerHistoryTestToolCallWithArguments(id, name, providerHistoryTestJSONArguments(t, arguments))
}

func providerHistoryTestJSONArguments(t *testing.T, arguments map[string]string) string {
	t.Helper()
	data, err := json.Marshal(arguments)
	if err != nil {
		t.Fatalf("json.Marshal(%#v) error = %v", arguments, err)
	}
	return string(data)
}

func providerHistoryTestAssistantToolCall(id, name string) api.Message {
	return providerHistoryTestAssistantToolCalls(providerHistoryTestToolCall(id, name))
}

func providerHistoryTestAssistantToolCalls(toolCalls ...api.OpenAIToolCall) api.Message {
	return api.Message{
		Role:      "assistant",
		Content:   "calling tools",
		ToolCalls: toolCalls,
	}
}

func providerHistoryTestToolResult(id, name, content string) api.Message {
	return api.Message{
		Role:       "tool",
		Content:    content,
		ToolCallID: id,
		ToolName:   name,
	}
}

func providerHistoryTestReductionHistory(callID, oldRead string) []api.Message {
	return []api.Message{
		providerHistoryTestAssistantToolCall(callID, "read_file"),
		providerHistoryTestToolResult(callID, "read_file", oldRead),
		{Role: "assistant", Content: "after old read"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest read_file output"),
		{Role: "assistant", Content: "done"},
	}
}

func providerHistoryTestWriteFileHistory(callID, args, result string) []api.Message {
	return []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithArguments(callID, "write_file", args)),
		providerHistoryTestToolResult(callID, "write_file", result),
		{Role: "assistant", Content: "write done"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "done"},
	}
}

func providerHistoryTestLargeSuccessfulTestOutput() string {
	return strings.Repeat("ok\tgithub.com/susugadx/xelyon-cli/internal/providerhistory\t0.001s\n", 260)
}

func providerHistoryTestNumberedLines(prefix string, count int) string {
	var b strings.Builder
	for i := 1; i <= count; i++ {
		fmt.Fprintf(&b, "%s-%03d\n", prefix, i)
	}
	return b.String()
}

func providerHistoryTestWriteFileSuccess(content, path string) string {
	return "Successfully wrote " + strconv.Itoa(len(content)) + " bytes (" + strconv.Itoa(strings.Count(content, "\n")+1) + " lines) to " + path
}

func providerHistoryTestWriteFileContentArgument(t *testing.T, args, path string) string {
	t.Helper()
	var fields map[string]string
	if err := json.Unmarshal([]byte(args), &fields); err != nil {
		t.Fatalf("projected write_file arguments are not valid JSON: %v\nargs=%s", err, args)
	}
	if fields["path"] != path {
		t.Fatalf("projected path = %q, want %q in args %s", fields["path"], path, args)
	}
	return fields["content"]
}

func providerHistoryTestCandidateTools(report ProjectionReport) []string {
	tools := make([]string, len(report.Candidates))
	for i, candidate := range report.Candidates {
		tools[i] = candidate.ToolName
	}
	return tools
}

func providerHistoryTestCandidateByToolCallID(report ProjectionReport, id string) *ReductionCandidate {
	for i := range report.Candidates {
		if report.Candidates[i].ToolCallID == id {
			return &report.Candidates[i]
		}
	}
	return nil
}

func providerHistoryTestKeptByToolCallID(report ProjectionReport, id string) *ReductionCandidate {
	for i := range report.Kept {
		if report.Kept[i].ToolCallID == id {
			return &report.Kept[i]
		}
	}
	return nil
}

func providerHistoryTestCommandCandidateByToolCallID(report CommandEditDryRunReport, id string) *CommandEditDryRunCandidate {
	for i := range report.Candidates {
		if report.Candidates[i].ToolCallID == id {
			return &report.Candidates[i]
		}
	}
	return nil
}

func providerHistoryTestRawOutputStore(t *testing.T) RawOutputArtifactStore {
	t.Helper()
	store, err := rawoutputs.OpenStore(rawoutputs.Root(t.TempDir()), rawoutputs.StoreOptions{})
	if err != nil {
		t.Fatalf("rawoutputs.OpenStore() error = %v", err)
	}
	return store
}

func providerHistoryTestRawOutputSpyStore(t *testing.T) *providerHistoryMaterializeSpyStore {
	t.Helper()
	store, err := rawoutputs.OpenStore(rawoutputs.Root(t.TempDir()), rawoutputs.StoreOptions{})
	if err != nil {
		t.Fatalf("rawoutputs.OpenStore() error = %v", err)
	}
	return &providerHistoryMaterializeSpyStore{inner: store}
}

type providerHistoryMaterializeSpyStore struct {
	inner            *rawoutputs.Store
	createCalls      int
	materializeCalls int
	lastLegacy       rawoutputs.LegacyMaterializeRequest
}

func (s *providerHistoryMaterializeSpyStore) Create(ctx context.Context, req rawoutputs.CreateRequest) (rawoutputs.CreateResult, error) {
	s.createCalls++
	return s.inner.Create(ctx, req)
}

func (s *providerHistoryMaterializeSpyStore) MaterializeLegacy(ctx context.Context, req rawoutputs.LegacyMaterializeRequest) (rawoutputs.CreateResult, error) {
	s.materializeCalls++
	s.lastLegacy = req
	return s.inner.MaterializeLegacy(ctx, req)
}

func (s *providerHistoryMaterializeSpyStore) Verify(ctx context.Context, ref rawoutputs.RawOutputRef) (rawoutputs.VerifyResult, error) {
	return s.inner.Verify(ctx, ref)
}

func assertProviderHistoryLegacyMaterialize(t *testing.T, store *providerHistoryMaterializeSpyStore, surface rawoutputs.Surface) {
	t.Helper()
	if store.createCalls != 0 || store.materializeCalls != 1 {
		t.Fatalf("raw output materialization calls = create %d legacy %d, want Create=0 MaterializeLegacy=1", store.createCalls, store.materializeCalls)
	}
	if store.lastLegacy.Ambiguous || strings.TrimSpace(store.lastLegacy.ExactSourceID) == "" {
		t.Fatalf("legacy source identity = %q ambiguous=%t, want exact non-ambiguous", store.lastLegacy.ExactSourceID, store.lastLegacy.Ambiguous)
	}
	if store.lastLegacy.Surface != surface {
		t.Fatalf("legacy materialize surface = %s, want %s", store.lastLegacy.Surface, surface)
	}
	if strings.TrimSpace(store.lastLegacy.Source.ToolCallID) == "" ||
		strings.TrimSpace(store.lastLegacy.Source.ToolName) == "" {
		t.Fatalf("legacy source metadata = %#v, want tool identity", store.lastLegacy.Source)
	}
}

func providerHistoryTestAssertByteMetrics(t *testing.T, original, projected []api.Message, report ProjectionReport) {
	t.Helper()
	originalBytes := providerHistoryContentBytes(original)
	projectedBytes := providerHistoryContentBytes(projected)
	if report.OriginalBytes != originalBytes || report.ProjectedBytes != projectedBytes {
		t.Fatalf("byte metrics = original %d projected %d, want %d/%d", report.OriginalBytes, report.ProjectedBytes, originalBytes, projectedBytes)
	}
	wantContentSaved, wantContentTokens := providerHistoryTestContentReplacementSavings(original, report)
	if report.ContentReplacementSavedBytes != wantContentSaved {
		t.Fatalf("ContentReplacementSavedBytes = %d, want %d", report.ContentReplacementSavedBytes, wantContentSaved)
	}
	if report.ApproxContentReplacementSavedTokens != wantContentTokens {
		t.Fatalf("ApproxContentReplacementSavedTokens = %d, want %d", report.ApproxContentReplacementSavedTokens, wantContentTokens)
	}
	wantTotalSaved := wantContentSaved
	wantTotalTokens := wantContentTokens
	switch report.Mode {
	case Apply:
		wantTotalSaved += report.CommandEditDryRun.CommandReplacementSavedBytes + report.CommandEditDryRun.EditArgReplacementSavedBytes + report.CommandEditDryRun.ArtifactBackedCommandReplacementSavedBytes
		wantTotalTokens += report.CommandEditDryRun.ApproxCommandReplacementSavedTokens + report.CommandEditDryRun.ApproxEditArgReplacementSavedTokens + report.CommandEditDryRun.ApproxArtifactBackedCommandReplacementSavedTokens
	case DryRun:
		wantTotalSaved += report.CommandEditDryRun.CommandEstimatedSavedBytes + report.CommandEditDryRun.EditArgEstimatedSavedBytes
		wantTotalTokens += report.CommandEditDryRun.ApproxCommandSavedTokens + report.CommandEditDryRun.ApproxEditArgSavedTokens
	}
	if report.EstimatedSavedBytes != wantTotalSaved {
		t.Fatalf("EstimatedSavedBytes = %d, want provider-facing total %d", report.EstimatedSavedBytes, wantTotalSaved)
	}
	if report.ApproxSavedTokens != wantTotalTokens {
		t.Fatalf("ApproxSavedTokens = %d, want provider-facing total %d", report.ApproxSavedTokens, wantTotalTokens)
	}
}

func providerHistoryTestContentReplacementSavings(original []api.Message, report ProjectionReport) (int, int) {
	if report.Mode != Apply && report.Mode != DryRun {
		return 0, 0
	}
	totalBytes := 0
	totalTokens := 0
	for _, candidate := range report.Candidates {
		if report.Mode == Apply && !candidate.ReplacementApplied {
			continue
		}
		if candidate.SuggestedReplacementText == "" || candidate.HistoryIndex < 0 || candidate.HistoryIndex >= len(original) {
			continue
		}
		originalContent := original[candidate.HistoryIndex].Content
		savedBytes, savedTokens, _, ok := providerHistoryContentReplacementEligibility(originalContent, candidate.SuggestedReplacementText)
		if !ok {
			continue
		}
		totalBytes += savedBytes
		totalTokens += savedTokens
	}
	return totalBytes, totalTokens
}
