package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/review"
)

func TestCompressHistoryDoesNotUseProviderHistoryReductionRequestProjection(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "summary"}
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	staleReport := seedProviderHistoryReductionInternalCallFixture(t, agent)

	if err := agent.CompressHistory(1); err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}

	if providerHistoryMessagesContainReductionPlaceholder(provider.capturedChatHistory) {
		t.Fatalf("summary request history contains provider reduction placeholder: %#v", provider.capturedChatHistory)
	}
	assertLastProviderHistoryProjectionReportPreserved(t, agent.Runtime, staleReport)
}

func TestCompactAPIDoesNotUseProviderHistoryReductionRequestProjection(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", supportsCompact: true}
	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", config.DefaultConfig())
	staleReport := seedProviderHistoryReductionInternalCallFixture(t, agent)

	if err := agent.CompressWithCompactAPI(context.Background()); err != nil {
		t.Fatalf("CompressWithCompactAPI() error = %v", err)
	}

	if providerHistoryInputItemsContainReductionPlaceholder(provider.capturedCompactInput) {
		t.Fatalf("compact input contains provider reduction placeholder: %#v", provider.capturedCompactInput)
	}
	assertLastProviderHistoryProjectionReportPreserved(t, agent.Runtime, staleReport)
}

func TestGeminiApplyPatchRepairDoesNotUseProviderHistoryReductionRequestProjection(t *testing.T) {
	var captured []api.Message
	provider := &scriptedChatProvider{
		name: "gemini",
		chatWithToolsFn: func(_ int, _ context.Context, _ string, history []api.Message, _ string) (string, error) {
			captured = api.CloneMessages(history)
			return validAddFilePatch, nil
		},
	}
	var out bytes.Buffer
	agent := newChatRequestTestAgent(t, provider, &out)
	staleReport := seedProviderHistoryReductionInternalCallFixture(t, agent)

	if _, err := agent.requestGeminiApplyPatchRepair(context.Background(), "bad patch", "parse error"); err != nil {
		t.Fatalf("requestGeminiApplyPatchRepair() error = %v", err)
	}

	if providerHistoryMessagesContainReductionPlaceholder(captured) {
		t.Fatalf("repair request history contains provider reduction placeholder: %#v", captured)
	}
	assertLastProviderHistoryProjectionReportPreserved(t, agent.Runtime, staleReport)
}

func TestReviewModelDoesNotUseProviderHistoryReductionRequestProjection(t *testing.T) {
	var captured []api.Message
	provider := &scriptedChatProvider{
		name: "openai",
		chatWithToolsFn: func(_ int, _ context.Context, _ string, history []api.Message, _ string) (string, error) {
			captured = api.CloneMessages(history)
			return `{"ok":true}`, nil
		},
	}
	agent := newReviewAgentForTest(t, provider)
	staleReport := seedProviderHistoryReductionInternalCallFixture(t, agent)

	if _, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), review.ReviewModelRequest{
		Phase:  review.ReviewModelPhaseReport,
		Prompt: "review prompt",
	}); err != nil {
		t.Fatalf("CompleteReview() error = %v", err)
	}

	if providerHistoryMessagesContainReductionPlaceholder(captured) {
		t.Fatalf("review request history contains provider reduction placeholder: %#v", captured)
	}
	assertLastProviderHistoryProjectionReportPreserved(t, agent.Runtime, staleReport)
}

func seedProviderHistoryReductionInternalCallFixture(t *testing.T, agent *Agent) ProviderHistoryProjectionReport {
	t.Helper()
	oldRead := strings.Repeat("old internal-call read_file output\n", 12)
	agent.Runtime.Options.EnableProviderHistoryReduction = true
	agent.Runtime.TaskLedger = providerHistoryTaskLedgerWithEvidence(t,
		providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_internal_old", Path: "internal.go", StartLine: 1},
	)
	staleReport := ProviderHistoryProjectionReport{
		Mode:                ProviderHistoryReductionApply,
		CandidateCount:      7,
		ReplacedCount:       7,
		EstimatedSavedBytes: 700,
	}
	agent.Runtime.LastProviderHistoryProjectionReport = staleReport
	agent.History = []api.Message{
		{Role: "user", Content: "inspect"},
		providerHistoryAssistantToolCall("call_internal_old", "read_file"),
		providerHistoryToolResult("call_internal_old", "read_file", oldRead),
		{Role: "assistant", Content: "after old read"},
		providerHistoryAssistantToolCall("call_internal_latest", "read_file"),
		providerHistoryToolResult("call_internal_latest", "read_file", "latest read"),
		{Role: "assistant", Content: "done"},
	}
	return staleReport
}
