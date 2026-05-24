package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/review"
)

func TestAgentReviewModelCompleteReviewPassesPromptAsSingleUserMessage(t *testing.T) {
	var captured struct {
		systemPrompt string
		history      []api.Message
		model        string
		updateMode   string
		toolsOff     bool
		toolCount    int
		mergedTools  int
		compacted    int
		activeCtx    int
		cacheNS      string
	}
	provider := &scriptedChatProvider{
		name: "openai",
		chatWithToolsFn: func(_ int, ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
			captured.systemPrompt = systemPrompt
			captured.history = append([]api.Message(nil), history...)
			captured.model = model
			captured.updateMode = api.AssistantUpdateModeFromContext(ctx)
			captured.toolsOff = api.IsToolUseDisabled(ctx)
			captured.toolCount = len(api.ToolDefinitionsFromContext(ctx))
			captured.mergedTools = len(api.ToolDefinitionsWithAdditional(ctx, []api.ToolDefinition{{
				Name:        "mcp_fake_tool",
				Description: "fake mcp tool",
				Parameters:  map[string]interface{}{"type": "object"},
			}}))
			captured.compacted = len(api.CompactedInputItemsFromContext(ctx))
			captured.activeCtx = len(api.ActiveContextBlocksFromContext(ctx))
			captured.cacheNS = api.ProviderCacheNamespaceFromContext(ctx)
			return `{"ok":true}`, nil
		},
	}
	agent := newReviewAgentForTest(t, provider)
	agent.isCompactedMode = true
	agent.compactedItems = []api.InputItem{{Type: "message", Role: "user", Content: "existing compacted chat"}}

	resp, err := (agentReviewModel{agent: agent}).CompleteReview(contextWithInheritedActiveContext(), review.ReviewModelRequest{
		Phase:  review.ReviewModelPhaseReport,
		Prompt: "return review json",
	})
	if err != nil {
		t.Fatalf("CompleteReview() error = %v", err)
	}
	if resp.Content != `{"ok":true}` {
		t.Fatalf("Content = %q, want raw provider response", resp.Content)
	}
	if captured.systemPrompt != "" {
		t.Fatalf("systemPrompt = %q, want empty", captured.systemPrompt)
	}
	if len(captured.history) != 1 || captured.history[0].Role != "user" || captured.history[0].Content != "return review json" {
		t.Fatalf("history = %#v, want single user prompt", captured.history)
	}
	if captured.model != "review-model" {
		t.Fatalf("model = %q, want review-model", captured.model)
	}
	if captured.updateMode != api.AssistantUpdatesOff {
		t.Fatalf("assistant update mode = %q, want off", captured.updateMode)
	}
	if !captured.toolsOff {
		t.Fatal("tool use disabled = false, want true for review model call")
	}
	if captured.toolCount != 0 {
		t.Fatalf("tool definitions = %d, want 0 for review model call", captured.toolCount)
	}
	if captured.mergedTools != 0 {
		t.Fatalf("merged tool definitions = %d, want 0 for isolated review model call", captured.mergedTools)
	}
	if captured.compacted != 0 {
		t.Fatalf("compacted input items = %d, want 0 for isolated review model call", captured.compacted)
	}
	if captured.activeCtx != 0 {
		t.Fatalf("active context blocks = %d, want 0 for isolated review model call", captured.activeCtx)
	}
	if captured.cacheNS != reviewModelProviderCacheNamespace {
		t.Fatalf("provider cache namespace = %q, want %q", captured.cacheNS, reviewModelProviderCacheNamespace)
	}
}

func TestAgentReviewModelCompleteReviewWrapsProviderErrorWithPhase(t *testing.T) {
	provider := &scriptedChatProvider{
		name: "openai",
		chatWithToolsFn: func(_ int, _ context.Context, _ string, _ []api.Message, _ string) (string, error) {
			return "", errors.New("provider failed")
		},
	}
	agent := newReviewAgentForTest(t, provider)

	_, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), review.ReviewModelRequest{
		Phase:  review.ReviewModelPhaseProbePlan,
		Prompt: "plan",
	})
	if err == nil {
		t.Fatal("CompleteReview() error = nil, want provider error")
	}
	if got := err.Error(); !strings.Contains(got, "review model probe_plan") || !strings.Contains(got, "provider failed") {
		t.Fatalf("CompleteReview() error = %q, want phase and provider error", got)
	}
}

func TestAgentReviewModelCompleteReviewRestoresResponseID(t *testing.T) {
	provider := &reviewResponseIDProvider{}
	provider.name = "openai"
	provider.responseID = "resp_original"
	provider.chatWithToolsFn = func(_ int, _ context.Context, _ string, _ []api.Message, _ string) (string, error) {
		if got := provider.GetResponseID(); got != "" {
			t.Fatalf("response ID during review call = %q, want empty", got)
		}
		provider.SetResponseID("resp_review")
		return `{"ok":true}`, nil
	}
	agent := newReviewAgentForTest(t, provider)

	if _, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), review.ReviewModelRequest{
		Phase:  review.ReviewModelPhaseReport,
		Prompt: "report",
	}); err != nil {
		t.Fatalf("CompleteReview() error = %v", err)
	}
	if got := provider.GetResponseID(); got != "resp_original" {
		t.Fatalf("response ID after review call = %q, want resp_original", got)
	}
}

type reviewResponseIDProvider struct {
	scriptedChatProvider
	responseID string
}

func (p *reviewResponseIDProvider) HasCachedResponseID() bool {
	return p.responseID != ""
}

func (p *reviewResponseIDProvider) SetResponseID(id string) {
	p.responseID = strings.TrimSpace(id)
}

func (p *reviewResponseIDProvider) GetResponseID() string {
	return p.responseID
}
