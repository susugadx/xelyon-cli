package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

const (
	providerHistoryRequestEvidencePath        = "README.md"
	providerHistoryRequestExpectedPlaceholder = "[omitted old read_file result; evidence: README.md:L1-L3 source=read_file]"
)

type providerFacingHistoryMutationProbe struct {
	name             string
	supportsImages   bool
	response         string
	capturedHistory  []api.Message
	imageUserMessage string
	imageCalls       int
}

func (p *providerFacingHistoryMutationProbe) Name() string {
	if p.name != "" {
		return p.name
	}
	return "openai"
}

func (p *providerFacingHistoryMutationProbe) SupportsImages() bool { return p.supportsImages }

func (p *providerFacingHistoryMutationProbe) IsFunctionCallingEnabled() bool { return true }

func (p *providerFacingHistoryMutationProbe) ChatWithTools(_ context.Context, _ string, history []api.Message, _ string) (string, error) {
	p.capture(history)
	mutateProviderFacingHistoryForTest(history)
	if p.response != "" {
		return p.response, nil
	}
	return "provider response", nil
}

func (p *providerFacingHistoryMutationProbe) ChatWithImage(_ context.Context, _ string, history []api.Message, userMessage string, image *api.ImageData, _ string) (string, error) {
	if image == nil {
		return "", fmt.Errorf("image is required")
	}
	p.imageCalls++
	p.imageUserMessage = userMessage
	p.capture(history)
	mutateProviderFacingHistoryForTest(history)
	if p.response != "" {
		return p.response, nil
	}
	return "image response", nil
}

func (p *providerFacingHistoryMutationProbe) capture(messages []api.Message) {
	p.capturedHistory = api.CloneMessages(messages)
}

func providerHistoryMessageContents(messages []api.Message) []string {
	contents := make([]string, len(messages))
	for i, msg := range messages {
		contents[i] = msg.Content
	}
	return contents
}

func mutateProviderFacingHistoryForTest(messages []api.Message) {
	if len(messages) == 0 {
		return
	}
	messages[0].Content = "provider mutated content"
	if len(messages[0].ToolCalls) > 0 {
		messages[0].ToolCalls[0].ID = "provider_mutated_call"
		if len(messages[0].ToolCalls[0].ThoughtParts) > 0 {
			messages[0].ToolCalls[0].ThoughtParts[0]["text"] = "provider mutated thought"
		}
	}
}

func seedProviderHistoryReductionRequestFixture(t *testing.T, agent *Agent, callID string) string {
	t.Helper()
	oldRead := strings.Repeat("old provider-facing read_file output\n", 5)
	agent.Runtime.Options.EnableProviderHistoryReduction = true
	agent.Runtime.TaskLedger = providerHistoryTaskLedgerWithEvidence(t,
		providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: callID, Path: providerHistoryRequestEvidencePath, StartLine: 1, EndLine: 3},
	)
	agent.History = providerHistoryReductionRequestHistory(callID, oldRead)
	return oldRead
}

func seedProviderHistoryReductionRequestFixtureWithoutEvidence(agent *Agent, callID string) string {
	oldRead := strings.Repeat("old provider-facing read_file output without evidence\n", 5)
	agent.Runtime.Options.EnableProviderHistoryReduction = true
	agent.History = providerHistoryReductionRequestHistory(callID, oldRead)
	return oldRead
}

func assertProviderRequestHistoryReductionApplied(t *testing.T, agent *Agent, provider *providerFacingHistoryMutationProbe, oldRead, wantCurrentPrompt string) {
	t.Helper()
	if len(provider.capturedHistory) < 5 {
		t.Fatalf("provider history length = %d, want projected fixture history", len(provider.capturedHistory))
	}
	if provider.capturedHistory[1].Content != providerHistoryRequestExpectedPlaceholder {
		t.Fatalf("provider old tool result = %q, want %q", provider.capturedHistory[1].Content, providerHistoryRequestExpectedPlaceholder)
	}
	if provider.capturedHistory[4].Content != providerHistoryReductionLatestToolOutput {
		t.Fatalf("provider latest tool result = %q, want raw latest tool result", provider.capturedHistory[4].Content)
	}
	if wantCurrentPrompt != "" && !strings.Contains(provider.capturedHistory[len(provider.capturedHistory)-1].Content, wantCurrentPrompt) {
		t.Fatalf("provider current prompt = %q, want %q", provider.capturedHistory[len(provider.capturedHistory)-1].Content, wantCurrentPrompt)
	}
	if agent.History[1].Content != oldRead {
		t.Fatalf("Agent.History[1].Content = %q, want raw old read", agent.History[1].Content)
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.Mode != ProviderHistoryReductionApply || report.ReplacedCount != 1 || report.EstimatedSavedBytes <= 0 {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want apply report with one replacement", report)
	}
}
