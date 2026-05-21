package agent

import (
	"bytes"
	"context"
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
