package agent

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/review"
	reviewmodelinput "github.com/susugadx/xelyon-cli/internal/review/modelinput"
)

func reviewModelRequestForAgentTest(phase review.ReviewModelPhase, prompt string) review.ReviewModelRequest {
	return review.ReviewModelRequest{
		Phase:        phase,
		SystemPrompt: reviewmodelinput.BuildReviewSystemPrompt(),
		Prompt:       prompt,
	}
}

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

	resp, err := (agentReviewModel{agent: agent}).CompleteReview(contextWithInheritedActiveContext(), reviewModelRequestForAgentTest(review.ReviewModelPhaseReport, "return review json"))
	if err != nil {
		t.Fatalf("CompleteReview() error = %v", err)
	}
	if resp.Content != `{"ok":true}` {
		t.Fatalf("Content = %q, want raw provider response", resp.Content)
	}
	if captured.systemPrompt != reviewmodelinput.BuildReviewSystemPrompt() {
		t.Fatalf("systemPrompt = %q, want reviewer constitution", captured.systemPrompt)
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

func TestAgentReviewModelCompleteReviewRejectsEmptySystemPrompt(t *testing.T) {
	provider := &scriptedChatProvider{
		name: "openai",
		chatWithToolsFn: func(_ int, _ context.Context, _ string, _ []api.Message, _ string) (string, error) {
			t.Fatal("provider should not be called when review system prompt is empty")
			return "", nil
		},
	}
	agent := newReviewAgentForTest(t, provider)

	_, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), review.ReviewModelRequest{
		Phase:  review.ReviewModelPhaseReport,
		Prompt: "return review json",
	})
	if err == nil {
		t.Fatal("CompleteReview() error = nil, want empty system prompt validation error")
	}
	if got := err.Error(); !strings.Contains(got, "review model report") || !strings.Contains(got, "system prompt is empty") {
		t.Fatalf("CompleteReview() error = %q, want phase and empty system prompt", got)
	}
	if provider.callCount != 0 {
		t.Fatalf("provider calls = %d, want 0", provider.callCount)
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

	_, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), reviewModelRequestForAgentTest(review.ReviewModelPhaseProbePlan, "plan"))
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

	if _, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), reviewModelRequestForAgentTest(review.ReviewModelPhaseReport, "report")); err != nil {
		t.Fatalf("CompleteReview() error = %v", err)
	}
	if got := provider.GetResponseID(); got != "resp_original" {
		t.Fatalf("response ID after review call = %q, want resp_original", got)
	}
}

func TestAgentReviewModelThinkingInheritsRuntimeConfig(t *testing.T) {
	var captured struct {
		enabled bool
		level   string
	}
	provider := &scriptedChatProvider{
		name: "openai",
		chatWithToolsFn: func(_ int, ctx context.Context, _ string, _ []api.Message, _ string) (string, error) {
			captured.enabled = api.IsThinkingEnabled(ctx)
			captured.level = config.FromContext(ctx).Thinking.Level
			return `{"ok":true}`, nil
		},
	}
	agent := newReviewAgentForTest(t, provider)
	cfg := agent.cfg()
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "xhigh"
	cfg.Review.Thinking = config.ReviewThinkingConfig{Mode: config.ReviewThinkingModeInherit}
	agent.setRuntimeConfig(cfg)

	if _, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), reviewModelRequestForAgentTest(review.ReviewModelPhaseReport, "report")); err != nil {
		t.Fatalf("CompleteReview() error = %v", err)
	}
	if !captured.enabled {
		t.Fatal("review thinking enabled = false, want inherited true")
	}
	if captured.level != "xhigh" {
		t.Fatalf("review thinking level = %q, want inherited xhigh", captured.level)
	}
}

func TestAgentReviewModelThinkingInheritAppliesLevelOverrideWithoutMutatingRuntimeConfig(t *testing.T) {
	var captured struct {
		enabled bool
		level   string
	}
	provider := &scriptedChatProvider{
		name: "openai",
		chatWithToolsFn: func(_ int, ctx context.Context, _ string, _ []api.Message, _ string) (string, error) {
			captured.enabled = api.IsThinkingEnabled(ctx)
			captured.level = config.FromContext(ctx).Thinking.Level
			return `{"ok":true}`, nil
		},
	}
	agent := newReviewAgentForTest(t, provider)
	cfg := agent.cfg()
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "low"
	cfg.Review.Thinking = config.ReviewThinkingConfig{
		Mode:  config.ReviewThinkingModeInherit,
		Level: "high",
	}
	agent.setRuntimeConfig(cfg)

	if _, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), reviewModelRequestForAgentTest(review.ReviewModelPhaseReport, "report")); err != nil {
		t.Fatalf("CompleteReview() error = %v", err)
	}
	if !captured.enabled {
		t.Fatal("review thinking enabled = false, want inherited true")
	}
	if captured.level != "high" {
		t.Fatalf("review thinking level = %q, want review override high", captured.level)
	}
	if agent.cfg().Thinking.Level != "low" {
		t.Fatalf("runtime Thinking.Level = %q, want low", agent.cfg().Thinking.Level)
	}
}

func TestAgentReviewModelThinkingOffOverridesWithoutMutatingRuntimeConfig(t *testing.T) {
	var captured struct {
		enabled bool
		level   string
	}
	provider := &scriptedChatProvider{
		name: "openai",
		chatWithToolsFn: func(_ int, ctx context.Context, _ string, _ []api.Message, _ string) (string, error) {
			captured.enabled = api.IsThinkingEnabled(ctx)
			captured.level = config.FromContext(ctx).Thinking.Level
			return `{"ok":true}`, nil
		},
	}
	agent := newReviewAgentForTest(t, provider)
	cfg := agent.cfg()
	cfg.Thinking.Enabled = true
	cfg.Thinking.Level = "xhigh"
	cfg.Review.Thinking = config.ReviewThinkingConfig{
		Mode:  config.ReviewThinkingModeOff,
		Level: "high",
	}
	agent.setRuntimeConfig(cfg)

	if _, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), reviewModelRequestForAgentTest(review.ReviewModelPhaseReport, "report")); err != nil {
		t.Fatalf("CompleteReview() error = %v", err)
	}
	if captured.enabled {
		t.Fatal("review thinking enabled = true, want off override")
	}
	if captured.level != "xhigh" {
		t.Fatalf("review thinking level = %q, want runtime xhigh when mode=off", captured.level)
	}
	if !agent.cfg().Thinking.Enabled {
		t.Fatal("runtime Thinking.Enabled mutated to false")
	}
	if agent.cfg().Thinking.Level != "xhigh" {
		t.Fatalf("runtime Thinking.Level = %q, want xhigh", agent.cfg().Thinking.Level)
	}
}

func TestAgentReviewModelThinkingOnOverridesLevelWithoutMutatingRuntimeConfig(t *testing.T) {
	var captured struct {
		enabled bool
		level   string
	}
	provider := &scriptedChatProvider{
		name: "openai",
		chatWithToolsFn: func(_ int, ctx context.Context, _ string, _ []api.Message, _ string) (string, error) {
			captured.enabled = api.IsThinkingEnabled(ctx)
			captured.level = config.FromContext(ctx).Thinking.Level
			return `{"ok":true}`, nil
		},
	}
	agent := newReviewAgentForTest(t, provider)
	cfg := agent.cfg()
	cfg.Thinking.Enabled = false
	cfg.Thinking.Level = "low"
	cfg.Review.Thinking = config.ReviewThinkingConfig{
		Mode:  config.ReviewThinkingModeOn,
		Level: "high",
	}
	agent.setRuntimeConfig(cfg)

	if _, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), reviewModelRequestForAgentTest(review.ReviewModelPhaseReport, "report")); err != nil {
		t.Fatalf("CompleteReview() error = %v", err)
	}
	if !captured.enabled {
		t.Fatal("review thinking enabled = false, want on override")
	}
	if captured.level != "high" {
		t.Fatalf("review thinking level = %q, want review override high", captured.level)
	}
	if agent.cfg().Thinking.Enabled {
		t.Fatal("runtime Thinking.Enabled mutated to true")
	}
	if agent.cfg().Thinking.Level != "low" {
		t.Fatalf("runtime Thinking.Level = %q, want low", agent.cfg().Thinking.Level)
	}
}

func TestAgentReviewModelUsesConfiguredProviderModelWithoutMutatingSession(t *testing.T) {
	currentProvider := &reviewResponseIDProvider{}
	currentProvider.name = "openai"
	currentProvider.responseID = "current_resp"
	currentProvider.chatWithToolsFn = func(_ int, _ context.Context, _ string, _ []api.Message, _ string) (string, error) {
		t.Fatal("current provider should not be called when review.provider is configured")
		return "", nil
	}
	agent := newReviewAgentForTest(t, currentProvider)
	agent.History = []api.Message{{Role: "user", Content: "existing chat"}}

	cfg := agent.cfg()
	cfg.Review = config.ReviewConfig{Provider: "ollama", Model: "review-dedicated"}
	agent.setRuntimeConfig(cfg)

	var captured struct {
		factoryProvider string
		model           string
		history         []api.Message
	}
	configuredProvider := &reviewResponseIDProvider{}
	configuredProvider.name = "ollama"
	configuredProvider.responseID = "configured_resp"
	configuredProvider.chatWithToolsFn = func(_ int, _ context.Context, _ string, history []api.Message, model string) (string, error) {
		if got := configuredProvider.GetResponseID(); got != "" {
			t.Fatalf("configured provider response ID during review call = %q, want empty", got)
		}
		configuredProvider.SetResponseID("review_resp")
		captured.model = model
		captured.history = append([]api.Message(nil), history...)
		return `{"configured":true}`, nil
	}
	restoreFactory := replaceReviewModelProviderFactoryForTest(t, func(providerName string) (api.Provider, error) {
		captured.factoryProvider = providerName
		return configuredProvider, nil
	})
	defer restoreFactory()

	resp, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), reviewModelRequestForAgentTest(review.ReviewModelPhaseReport, "review prompt"))
	if err != nil {
		t.Fatalf("CompleteReview() error = %v", err)
	}
	if resp.Content != `{"configured":true}` {
		t.Fatalf("Content = %q, want configured response", resp.Content)
	}
	if captured.factoryProvider != "ollama" {
		t.Fatalf("factory provider = %q, want ollama", captured.factoryProvider)
	}
	if captured.model != "review-dedicated" {
		t.Fatalf("model = %q, want review-dedicated", captured.model)
	}
	if len(captured.history) != 1 || captured.history[0].Content != "review prompt" {
		t.Fatalf("history = %#v, want single review prompt", captured.history)
	}
	if currentProvider.callCount != 0 {
		t.Fatalf("current provider calls = %d, want 0", currentProvider.callCount)
	}
	if got := currentProvider.GetResponseID(); got != "current_resp" {
		t.Fatalf("current provider response ID = %q, want current_resp", got)
	}
	if got := configuredProvider.GetResponseID(); got != "configured_resp" {
		t.Fatalf("configured provider response ID after review call = %q, want configured_resp", got)
	}
	if agent.CurrentProvider != currentProvider {
		t.Fatal("agent CurrentProvider was mutated")
	}
	if agent.CurrentModel != "review-model" {
		t.Fatalf("agent CurrentModel = %q, want review-model", agent.CurrentModel)
	}
	if len(agent.History) != 1 || agent.History[0].Content != "existing chat" {
		t.Fatalf("agent history mutated: %#v", agent.History)
	}
}

func TestAgentReviewModelConfiguredProviderFallsBackToProviderDefaultModel(t *testing.T) {
	provider := &scriptedChatProvider{name: "openai"}
	agent := newReviewAgentForTest(t, provider)

	cfg := agent.cfg()
	cfg.Review.Provider = "ollama"
	cfg.SetProviderModelConfig("ollama", config.ProviderModelConfig{DefaultModel: "review-default"})
	agent.setRuntimeConfig(cfg)

	var capturedModel string
	configuredProvider := &scriptedChatProvider{
		name: "ollama",
		chatWithToolsFn: func(_ int, _ context.Context, _ string, _ []api.Message, model string) (string, error) {
			capturedModel = model
			return `{"ok":true}`, nil
		},
	}
	restoreFactory := replaceReviewModelProviderFactoryForTest(t, func(providerName string) (api.Provider, error) {
		if providerName != "ollama" {
			t.Fatalf("factory provider = %q, want ollama", providerName)
		}
		return configuredProvider, nil
	})
	defer restoreFactory()

	if _, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), reviewModelRequestForAgentTest(review.ReviewModelPhaseReport, "report")); err != nil {
		t.Fatalf("CompleteReview() error = %v", err)
	}
	if capturedModel != "review-default" {
		t.Fatalf("model = %q, want provider default review-default", capturedModel)
	}
}

func TestAgentReviewModelConfiguredProviderAttributesUsageToReviewModel(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")

	provider := &scriptedChatProvider{name: "openai"}
	agent := newReviewAgentForTest(t, provider)

	cfg := agent.cfg()
	cfg.Review = config.ReviewConfig{Provider: "gemini", Model: "gemini-3.1-pro"}
	agent.setRuntimeConfig(cfg)

	usage := api.Usage{InputTokens: 100_000, OutputTokens: 100_000}
	var configuredProvider *scriptedChatProvider
	configuredProvider = &scriptedChatProvider{
		name: "gemini",
		chatWithToolsFn: func(_ int, _ context.Context, _ string, _ []api.Message, _ string) (string, error) {
			if configuredProvider.usageCallback == nil {
				t.Fatal("review provider usage callback was not configured")
			}
			configuredProvider.usageCallback(usage)
			return `{"ok":true}`, nil
		},
	}
	restoreFactory := replaceReviewModelProviderFactoryForTest(t, func(providerName string) (api.Provider, error) {
		if providerName != "gemini" {
			t.Fatalf("factory provider = %q, want gemini", providerName)
		}
		return configuredProvider, nil
	})
	defer restoreFactory()

	if _, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), reviewModelRequestForAgentTest(review.ReviewModelPhaseReport, "report")); err != nil {
		t.Fatalf("CompleteReview() error = %v", err)
	}

	if agent.Stats.InputTokens != usage.InputTokens || agent.Stats.OutputTokens != usage.OutputTokens {
		t.Fatalf("usage = (%d, %d), want (%d, %d)", agent.Stats.InputTokens, agent.Stats.OutputTokens, usage.InputTokens, usage.OutputTokens)
	}
	const wantGeminiCost = 1.4
	if math.Abs(agent.Stats.AccumulatedCost-wantGeminiCost) > 0.000001 {
		t.Fatalf("AccumulatedCost = %.6f, want %.6f for review gemini/gemini-3.1-pro", agent.Stats.AccumulatedCost, wantGeminiCost)
	}
	if agent.Stats.CostUnknown {
		t.Fatal("CostUnknown = true, want false for known review model pricing")
	}
}

func TestAgentReviewModelRejectsReviewModelWithoutProvider(t *testing.T) {
	provider := &scriptedChatProvider{name: "openai"}
	agent := newReviewAgentForTest(t, provider)

	cfg := agent.cfg()
	cfg.Review.Model = "orphan-review-model"
	agent.setRuntimeConfig(cfg)

	_, err := (agentReviewModel{agent: agent}).CompleteReview(context.Background(), reviewModelRequestForAgentTest(review.ReviewModelPhaseReport, "report"))
	if err == nil {
		t.Fatal("CompleteReview() error = nil, want review.model validation error")
	}
	if !strings.Contains(err.Error(), "review.model requires review.provider") {
		t.Fatalf("CompleteReview() error = %q, want review.model requires review.provider", err)
	}
	if provider.callCount != 0 {
		t.Fatalf("provider calls = %d, want 0", provider.callCount)
	}
}

func replaceReviewModelProviderFactoryForTest(t *testing.T, factory func(string) (api.Provider, error)) func() {
	t.Helper()
	previous := newReviewModelProvider
	newReviewModelProvider = factory
	return func() {
		newReviewModelProvider = previous
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
