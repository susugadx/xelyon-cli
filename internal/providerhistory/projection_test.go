package providerhistory

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ledger"
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
	pointer := ledger.EvidencePointer{Path: "src/main.go", StartLine: 7, EndLine: 9, Source: "read_file", ToolCallID: "call_read_old"}

	result := Project(ProjectionInput{
		Messages: history,
		Policy: Policy{
			Mode:             Apply,
			EvidencePointers: []ledger.EvidencePointer{pointer},
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
	applied := AppliedEvidencePointers(result.Report)
	if !reflect.DeepEqual(applied, []ledger.EvidencePointer{pointer}) {
		t.Fatalf("AppliedEvidencePointers() = %#v, want %#v", applied, []ledger.EvidencePointer{pointer})
	}
	applied[0].Path = "mutated.go"
	if result.Report.Candidates[0].EvidencePointers[0].Path != "src/main.go" {
		t.Fatalf("AppliedEvidencePointers() did not return a defensive copy: %#v", result.Report.Candidates[0].EvidencePointers)
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
			EvidencePointers:                       []ledger.EvidencePointer{{Path: "src/main.go", StartLine: 1, Source: "read_file", ToolCallID: "call_read_old"}},
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
	if result.History[4].Content == commandOutput || !strings.Contains(result.History[4].Content, "successful test command output") {
		t.Fatalf("command output = %q, want successful command placeholder", result.History[4].Content)
	}
	if result.Report.CommandEditDryRun.CommandReplacedCount != 1 || !result.Report.ResponsesChainDisabled {
		t.Fatalf("command report = %#v, want command replacement and response chain disabled", result.Report.CommandEditDryRun)
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
	replacement := providerHistoryTestWriteFileContentArgument(t, result.History[0].ToolCalls[0].Function.Arguments, "src/generated.go")
	if replacement == content || !strings.HasPrefix(replacement, "[omitted old write_file.content; path=src/generated.go]") {
		t.Fatalf("projected write_file.content = %q, want placeholder", replacement)
	}
	if history[0].ToolCalls[0].Function.Arguments != args {
		t.Fatalf("raw write_file arguments mutated to %s", history[0].ToolCalls[0].Function.Arguments)
	}
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
		KeptReasonCounts: map[string]int{"missing_evidence_pointer": 1},
		Candidates: []ReductionCandidate{{
			ToolName:         "read_file",
			EvidencePointers: []ledger.EvidencePointer{{Path: "src/main.go", StartLine: 1, EndLine: 2}},
		}},
		CommandEditDryRun: CommandEditDryRunReport{
			CandidateReasonCounts: map[string]int{"command_success_output": 1},
			KeptReasonCounts:      map[string]int{"latest_tool_result": 1},
			Candidates:            []CommandEditDryRunCandidate{{ToolName: "bash"}},
			Kept:                  []CommandEditDryRunCandidate{{ToolName: "write_file"}},
		},
	}

	cloned := CloneProjectionReport(report)
	cloned.KeptReasonCounts["missing_evidence_pointer"] = 2
	cloned.Candidates[0].EvidencePointers[0].Path = "mutated.go"
	cloned.CommandEditDryRun.CandidateReasonCounts["command_success_output"] = 2
	cloned.CommandEditDryRun.KeptReasonCounts["latest_tool_result"] = 2
	cloned.CommandEditDryRun.Candidates[0].ToolName = "command"
	cloned.CommandEditDryRun.Kept[0].ToolName = "apply_patch"

	if report.KeptReasonCounts["missing_evidence_pointer"] != 1 ||
		report.Candidates[0].EvidencePointers[0].Path != "src/main.go" ||
		report.CommandEditDryRun.CandidateReasonCounts["command_success_output"] != 1 ||
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
			EvidencePointers: []ledger.EvidencePointer{{Path: "src/main.go", StartLine: 1, Source: "read_file", ToolCallID: "call_read_old"}},
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

func providerHistoryTestAssertByteMetrics(t *testing.T, original, projected []api.Message, report ProjectionReport) {
	t.Helper()
	originalBytes := providerHistoryContentBytes(original)
	projectedBytes := providerHistoryContentBytes(projected)
	if report.OriginalBytes != originalBytes || report.ProjectedBytes != projectedBytes {
		t.Fatalf("byte metrics = original %d projected %d, want %d/%d", report.OriginalBytes, report.ProjectedBytes, originalBytes, projectedBytes)
	}
	wantSaved := 0
	if originalBytes > projectedBytes {
		wantSaved = originalBytes - projectedBytes
	}
	if report.EstimatedSavedBytes != wantSaved {
		t.Fatalf("EstimatedSavedBytes = %d, want %d", report.EstimatedSavedBytes, wantSaved)
	}
	wantSavedTokens := providerHistoryApproxSavedTokens(original, projected)
	if report.ApproxSavedTokens != wantSavedTokens {
		t.Fatalf("ApproxSavedTokens = %d, want %d", report.ApproxSavedTokens, wantSavedTokens)
	}
}
