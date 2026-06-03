package agent

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestNormalModeRequestAppliesProviderHistoryReductionWhenRuntimeGateEnabled(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{}
	agent := newChatRequestTestAgent(t, provider, &out)
	oldRead := seedProviderHistoryReductionRequestFixture(t, agent, "call_normal_old")
	agent.session.AddMessageFromAPI(agent.History[0], agent.CurrentModel)
	agent.session.AddMessageFromAPI(agent.History[1], agent.CurrentModel)
	agent.session.AddToolExecution("read_file", map[string]string{"path": "README.md"}, oldRead, true, agent.CurrentModel)

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	assertProviderRequestHistoryReductionApplied(t, agent, provider, oldRead, "next request")
	if agent.session.Messages[1].Content != oldRead {
		t.Fatalf("session conversation tool content = %q, want raw old read", agent.session.Messages[1].Content)
	}
	if agent.session.Messages[2].ToolExecution == nil || agent.session.Messages[2].ToolExecution.ResultPreview != oldRead {
		t.Fatalf("session tool execution = %#v, want raw old read audit entry", agent.session.Messages[2].ToolExecution)
	}
}

func TestNormalModeRequestKeepsReductionCandidateWithoutEvidencePointer(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{}
	agent := newChatRequestTestAgent(t, provider, &out)
	oldRead := seedProviderHistoryReductionRequestFixtureWithoutEvidence(agent, "call_missing_evidence")

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	if provider.capturedHistory[1].Content != oldRead {
		t.Fatalf("provider old tool result = %q, want raw candidate without evidence", provider.capturedHistory[1].Content)
	}
	if provider.capturedResponseIDChainDisabled {
		t.Fatal("provider request context disabled response ID chain without replacement")
	}
	if agent.History[1].Content != oldRead {
		t.Fatalf("Agent.History[1].Content = %q, want raw old read", agent.History[1].Content)
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.Mode != ProviderHistoryReductionApply || report.CandidateCount != 1 || report.ReplacedCount != 0 || report.ResponsesChainDisabled {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want one unapplied apply candidate without chain disable", report)
	}
	candidate := candidateByToolCallID(report, "call_missing_evidence")
	if candidate == nil || candidate.ReplacementApplied || candidate.KeepReason != "missing_evidence_pointer" {
		t.Fatalf("candidate = %#v, want missing_evidence_pointer without replacement", candidate)
	}
}

func TestNormalModeRequestDryRunReportsCandidatesWithoutChangingProviderPayload(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{}
	agent := newChatRequestTestAgent(t, provider, &out)
	oldRead := seedProviderHistoryReductionRequestFixture(t, agent, "call_normal_dry_run")
	agent.Runtime.Options.ProviderHistoryReductionMode = ProviderHistoryReductionDryRun
	agent.Runtime.Options.ProviderHistoryReductionModeSet = true

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	if provider.capturedHistory[1].Content != oldRead {
		t.Fatalf("provider old tool result = %q, want raw dry-run payload", provider.capturedHistory[1].Content)
	}
	if provider.capturedResponseIDChainDisabled {
		t.Fatal("provider request context disabled response ID chain in dry-run mode")
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.Mode != ProviderHistoryReductionDryRun || report.CandidateCount != 1 || report.ReplacedCount != 0 || report.ResponsesChainDisabled {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want dry-run candidate report without replacement or chain disable", report)
	}
	if agent.History[1].Content != oldRead {
		t.Fatalf("Agent.History[1].Content = %q, want raw old read", agent.History[1].Content)
	}
}

func TestNormalModeRequestDryRunReportsCommandEditWithoutChangingProviderPayload(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{}
	agent := newChatRequestTestAgent(t, provider, &out)
	commandOutput := strings.Repeat("command dry-run output\n", 12)
	writeArgs := providerHistoryJSONArguments(t, map[string]string{
		"path":    "generated.txt",
		"content": strings.Repeat("line\n", 40),
	})
	agent.Runtime.Options.ProviderHistoryReductionMode = ProviderHistoryReductionDryRun
	agent.Runtime.Options.ProviderHistoryReductionModeSet = true
	agent.History = []api.Message{
		{Role: "user", Content: "inspect command and edit history"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_cmd", "bash", map[string]string{"command": "ls -la"})),
		providerHistoryToolResult("call_cmd", "bash", commandOutput),
		{Role: "assistant", Content: "command checked"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithArguments("call_write", "write_file", writeArgs)),
		providerHistoryToolResult("call_write", "write_file", "wrote generated.txt"),
		{Role: "assistant", Content: "write done"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "ready"},
	}
	for _, msg := range agent.History {
		agent.session.AddMessageFromAPI(msg, agent.CurrentModel)
	}
	beforeHistory := api.CloneMessages(agent.History)
	beforeSession := append(agent.session.Messages[:0:0], agent.session.Messages...)

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	if provider.capturedHistory[2].Content != commandOutput {
		t.Fatalf("provider command output = %q, want raw command output", provider.capturedHistory[2].Content)
	}
	if provider.capturedHistory[4].ToolCalls[0].Function.Arguments != writeArgs {
		t.Fatalf("provider write_file args = %q, want raw args", provider.capturedHistory[4].ToolCalls[0].Function.Arguments)
	}
	if provider.capturedResponseIDChainDisabled {
		t.Fatal("provider request context disabled response ID chain for command/edit dry-run candidates")
	}
	for i, want := range beforeHistory {
		if !reflect.DeepEqual(agent.History[i], want) {
			t.Fatalf("Agent.History[%d] changed after command/edit dry-run request:\n got %#v\nwant %#v", i, agent.History[i], want)
		}
	}
	for i, want := range beforeSession {
		if !reflect.DeepEqual(agent.session.Messages[i], want) {
			t.Fatalf("session.Messages[%d] changed after command/edit dry-run request:\n got %#v\nwant %#v", i, agent.session.Messages[i], want)
		}
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.Mode != ProviderHistoryReductionDryRun || report.ResponsesChainDisabled {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want dry-run report without response chain disable", report)
	}
	if report.CommandEditDryRun.CommandCandidates != 1 || report.CommandEditDryRun.EditArgCandidates != 1 || report.CommandEditDryRun.ReplacementStatus != providerHistoryCommandEditReplacementStatusNotImplemented {
		t.Fatalf("CommandEditDryRun = %#v, want one command and one edit dry-run candidate", report.CommandEditDryRun)
	}
}

func TestNormalModeRequestApplyReplacesSuccessfulCommandOutputOnlyInProviderPayload(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{}
	agent := newChatRequestTestAgent(t, provider, &out)
	commandOutput := providerHistoryLargeSuccessfulTestOutput()
	writeArgs := providerHistoryJSONArguments(t, map[string]string{
		"path":    "generated.txt",
		"content": strings.Repeat("line\n", 80),
	})
	agent.Runtime.Options.EnableProviderHistoryReduction = true
	agent.History = []api.Message{
		{Role: "user", Content: "inspect command and edit history"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_cmd", "bash", map[string]string{"command": providerHistorySuccessfulTestCommand})),
		providerHistoryToolResult("call_cmd", "bash", commandOutput),
		{Role: "assistant", Content: "command checked"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithArguments("call_write", "write_file", writeArgs)),
		providerHistoryToolResult("call_write", "write_file", "wrote generated.txt"),
		{Role: "assistant", Content: "write done"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "ready"},
	}
	for _, msg := range agent.History {
		agent.session.AddMessageFromAPI(msg, agent.CurrentModel)
	}
	beforeHistory := api.CloneMessages(agent.History)
	beforeSession := append(agent.session.Messages[:0:0], agent.session.Messages...)

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	assertProviderHistoryCommandContentReplacement(t, provider.capturedHistory[2].Content, commandOutput, providerHistorySuccessfulTestReplacementLabel)
	if provider.capturedHistory[4].ToolCalls[0].Function.Arguments != writeArgs {
		t.Fatalf("provider write_file args = %q, want raw args", provider.capturedHistory[4].ToolCalls[0].Function.Arguments)
	}
	if !provider.capturedResponseIDChainDisabled {
		t.Fatal("provider request context did not disable response ID chain for command replacement")
	}
	for i, want := range beforeHistory {
		if !reflect.DeepEqual(agent.History[i], want) {
			t.Fatalf("Agent.History[%d] changed after command replacement request:\n got %#v\nwant %#v", i, agent.History[i], want)
		}
	}
	for i, want := range beforeSession {
		if !reflect.DeepEqual(agent.session.Messages[i], want) {
			t.Fatalf("session.Messages[%d] changed after command replacement request:\n got %#v\nwant %#v", i, agent.session.Messages[i], want)
		}
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.Mode != ProviderHistoryReductionApply || report.ReplacedCount != 0 || !report.ResponsesChainDisabled {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want command-only apply report with response chain disabled", report)
	}
	if report.CommandEditDryRun.CommandReplacedCount != 1 ||
		report.CommandEditDryRun.ReplacementStatus != providerHistoryCommandEditReplacementStatusPartialApply ||
		report.CommandEditDryRun.CommandReplacementSavedBytes <= 0 ||
		report.CommandEditDryRun.ApproxCommandReplacementSavedTokens < providerHistoryCommandReplacementMinSavedTokens {
		t.Fatalf("CommandEditDryRun = %#v, want one command replacement with savings", report.CommandEditDryRun)
	}
}

func TestNormalModeRequestApplyReplacesSuccessfulWriteFileContentOnlyInProviderPayload(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{}
	agent := newChatRequestTestAgent(t, provider, &out)
	path := "generated/request.go"
	content := providerHistoryLargeWriteFileContent()
	writeArgs := providerHistoryWriteFileArguments(t, path, content)
	writeResult := providerHistoryWriteFileSuccess(content, path)
	agent.Runtime.Options.EnableProviderHistoryReduction = true
	agent.History = []api.Message{
		{Role: "user", Content: "inspect write history"},
		providerHistoryAssistantToolCalls(providerHistoryToolCallWithArguments("call_write", "write_file", writeArgs)),
		providerHistoryToolResult("call_write", "write_file", writeResult),
		{Role: "assistant", Content: "write done"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "ready"},
	}
	for _, msg := range agent.History {
		agent.session.AddMessageFromAPI(msg, agent.CurrentModel)
	}
	beforeHistory := api.CloneMessages(agent.History)
	beforeSession := append(agent.session.Messages[:0:0], agent.session.Messages...)

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	assertProviderHistoryWriteFileContentReplacement(t, provider.capturedHistory[1].ToolCalls[0].Function.Arguments, path, content)
	if provider.capturedHistory[2].Content != writeResult {
		t.Fatalf("provider write_file result = %q, want raw success output", provider.capturedHistory[2].Content)
	}
	if !provider.capturedResponseIDChainDisabled {
		t.Fatal("provider request context did not disable response ID chain for write_file.content replacement")
	}
	for i, want := range beforeHistory {
		if !reflect.DeepEqual(agent.History[i], want) {
			t.Fatalf("Agent.History[%d] changed after write_file.content replacement request:\n got %#v\nwant %#v", i, agent.History[i], want)
		}
	}
	for i, want := range beforeSession {
		if !reflect.DeepEqual(agent.session.Messages[i], want) {
			t.Fatalf("session.Messages[%d] changed after write_file.content replacement request:\n got %#v\nwant %#v", i, agent.session.Messages[i], want)
		}
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.Mode != ProviderHistoryReductionApply ||
		report.ReplacedCount != 0 ||
		report.CommandEditDryRun.EditArgReplacedCount != 1 ||
		report.CommandEditDryRun.EditArgReplacementSavedBytes <= 0 ||
		report.CommandEditDryRun.ApproxEditArgReplacementSavedTokens < providerHistoryEditArgReplacementMinSavedTokens ||
		!report.ResponsesChainDisabled {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want write_file.content-only apply report with response chain disabled", report)
	}
}

func TestNormalModeRequestApplyReplacesSuccessfulApplyPatchAndStrReplaceArgsOnlyInProviderPayload(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{responseID: "resp_old"}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.session.ResponseID = "resp_old"
	patchPath := "generated/request_patch.go"
	replacePath := "generated/request_replace.go"
	patch := providerHistoryLargeApplyPatch(patchPath)
	oldStr := providerHistoryLargeStrReplaceText("old request line")
	newStr := providerHistoryLargeStrReplaceText("new request line")
	patchArgs := providerHistoryJSONAnyArguments(t, map[string]any{"patch": patch})
	replaceArgs := providerHistoryStrReplaceArguments(t, replacePath, oldStr, newStr)
	patchResult := providerHistoryApplyPatchSuccess(nil, []string{patchPath}, nil)
	replaceResult := "Successfully replaced text in " + replacePath + " (lines 3-20 → 3-21)"
	agent.Runtime.Options.EnableProviderHistoryReduction = true
	agent.History = []api.Message{
		{Role: "user", Content: "inspect edit history"},
		providerHistoryAssistantToolCalls(
			providerHistoryToolCallWithArguments("call_patch", "apply_patch", patchArgs),
			providerHistoryToolCallWithArguments("call_replace", "str_replace", replaceArgs),
		),
		providerHistoryToolResult("call_patch", "apply_patch", patchResult),
		providerHistoryToolResult("call_replace", "str_replace", replaceResult),
		{Role: "assistant", Content: "edits done"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "ready"},
	}
	for _, msg := range agent.History {
		agent.session.AddMessageFromAPI(msg, agent.CurrentModel)
	}
	beforeHistory := api.CloneMessages(agent.History)
	beforeSession := append(agent.session.Messages[:0:0], agent.session.Messages...)

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	assertProviderHistoryApplyPatchArgReplacement(t, provider.capturedHistory[1].ToolCalls[0].Function.Arguments, patchPath, patch)
	assertProviderHistoryStrReplaceArgReplacement(t, provider.capturedHistory[1].ToolCalls[1].Function.Arguments, replacePath, oldStr, newStr)
	if provider.capturedHistory[2].Content != patchResult || provider.capturedHistory[3].Content != replaceResult {
		t.Fatalf("provider edit results = %q / %q, want raw success outputs", provider.capturedHistory[2].Content, provider.capturedHistory[3].Content)
	}
	if !provider.capturedResponseIDChainDisabled {
		t.Fatal("provider request context did not disable response ID chain for edit argument replacements")
	}
	if provider.GetResponseID() != "" {
		t.Fatalf("provider response ID = %q, want cleared before edit argument replacement request", provider.GetResponseID())
	}
	if agent.session.ResponseID != "" {
		t.Fatalf("session.ResponseID = %q, want cleared before edit argument replacement request is saved", agent.session.ResponseID)
	}
	for i, want := range beforeHistory {
		if !reflect.DeepEqual(agent.History[i], want) {
			t.Fatalf("Agent.History[%d] changed after edit argument replacement request:\n got %#v\nwant %#v", i, agent.History[i], want)
		}
	}
	for i, want := range beforeSession {
		if !reflect.DeepEqual(agent.session.Messages[i], want) {
			t.Fatalf("session.Messages[%d] changed after edit argument replacement request:\n got %#v\nwant %#v", i, agent.session.Messages[i], want)
		}
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.Mode != ProviderHistoryReductionApply ||
		report.ReplacedCount != 0 ||
		report.CommandEditDryRun.EditArgReplacedCount != 2 ||
		report.CommandEditDryRun.EditArgReplacementSavedBytes <= 0 ||
		report.CommandEditDryRun.ApproxEditArgReplacementSavedTokens < providerHistoryEditArgReplacementMinSavedTokens*2 ||
		!report.ResponsesChainDisabled {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want two edit-arg replacements with response chain disabled", report)
	}
}

func TestNormalModeRequestAutoUsesDryRunEffectiveMode(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{}
	agent := newChatRequestTestAgent(t, provider, &out)
	oldRead := seedProviderHistoryReductionRequestFixture(t, agent, "call_normal_auto")
	agent.Runtime.Options.ProviderHistoryReductionMode = ProviderHistoryReductionAuto
	agent.Runtime.Options.ProviderHistoryReductionModeSet = true

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	if provider.capturedHistory[1].Content != oldRead {
		t.Fatalf("provider old tool result = %q, want raw auto/dry-run payload", provider.capturedHistory[1].Content)
	}
	if provider.capturedResponseIDChainDisabled {
		t.Fatal("provider request context disabled response ID chain in auto dry-run mode")
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.Mode != ProviderHistoryReductionDryRun || report.CandidateCount != 1 || report.ReplacedCount != 0 || report.ResponsesChainDisabled {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want dry-run report for auto mode without chain disable", report)
	}
}

func TestNormalModeRequestAppliesProviderHistoryReductionPreservesInferredToolName(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{name: "gemini"}
	agent := newChatRequestTestAgent(t, provider, &out)
	oldRead := seedProviderHistoryReductionRequestFixture(t, agent, "call_missing_stored_name")
	agent.History[1].ToolName = ""

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	assertProviderRequestHistoryReductionApplied(t, agent, provider, oldRead, "next request")
	if provider.capturedHistory[1].ToolName != "read_file" {
		t.Fatalf("provider old tool name = %q, want inferred read_file for reduced tool response", provider.capturedHistory[1].ToolName)
	}
	if agent.History[1].ToolName != "" {
		t.Fatalf("Agent.History[1].ToolName = %q, want raw history unchanged", agent.History[1].ToolName)
	}
}

func TestNormalModeRequestReplacementClearsSavedResponseContext(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{responseID: "resp_old"}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.session.ResponseID = "resp_old"
	oldRead := seedProviderHistoryReductionRequestFixture(t, agent, "call_normal_clear_response_id")

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	assertProviderRequestHistoryReductionApplied(t, agent, provider, oldRead, "next request")
	if provider.GetResponseID() != "" {
		t.Fatalf("provider response ID = %q, want cleared before reduced request", provider.GetResponseID())
	}
	if agent.session.ResponseID != "" {
		t.Fatalf("session.ResponseID = %q, want cleared before reduced request is saved", agent.session.ResponseID)
	}
}

func TestImageRequestAppliesProviderHistoryReductionToPastHistoryWhenRuntimeGateEnabled(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{supportsImages: true}
	agent := newChatRequestTestAgent(t, provider, &out)
	oldRead := seedProviderHistoryReductionRequestFixture(t, agent, "call_image_apply_old")
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png", Path: "test.png", Size: 4}

	if err := agent.chatInternal("describe image", image); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	assertProviderRequestHistoryReductionApplied(t, agent, provider, oldRead, "")
	if len(provider.capturedHistory) != 6 {
		t.Fatalf("image provider history length = %d, want only projected past history", len(provider.capturedHistory))
	}
	if strings.Contains(strings.Join(providerHistoryMessageContents(provider.capturedHistory), "\n"), "describe image") {
		t.Fatalf("image provider history should exclude current prompt, got %#v", provider.capturedHistory)
	}
	if !strings.Contains(provider.imageUserMessage, "describe image") {
		t.Fatalf("image userMessage = %q, want current prompt", provider.imageUserMessage)
	}
}

func TestHeadlessRequestAppliesProviderHistoryReductionWhenRuntimeGateEnabled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &providerFacingHistoryMutationProbe{}
	runner := newHeadlessRunner("headless query", "test-model", provider, newProjectMapDisabledConfig())
	t.Cleanup(runner.agent.Cleanup)
	oldRead := seedProviderHistoryReductionRequestFixture(t, runner.agent, "call_headless_apply_old")

	if _, err := runner.requestAssistantResponse(context.Background(), 0); err != nil {
		t.Fatalf("requestAssistantResponse() error = %v", err)
	}

	assertProviderRequestHistoryReductionApplied(t, runner.agent, provider, oldRead, "")
}

func TestPlanInvestigationRequestAppliesProviderHistoryReductionWhenRuntimeGateEnabled(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{response: "investigation done"}
	agent := newChatRequestTestAgent(t, provider, &out)
	oldRead := seedProviderHistoryReductionRequestFixture(t, agent, "call_plan_apply_old")

	response, err := newPlanInvestigationRunner(agent, context.Background()).requestResponse()
	if err != nil {
		t.Fatalf("requestResponse() error = %v", err)
	}
	if response != "investigation done" {
		t.Fatalf("response = %q, want investigation done", response)
	}
	assertProviderRequestHistoryReductionApplied(t, agent, provider, oldRead, "")
}
