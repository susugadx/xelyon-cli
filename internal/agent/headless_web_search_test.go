package agent

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/config"
	searchtool "github.com/susugadx/xelyon-cli/internal/tools/search"
)

func TestHeadlessWebSearchCanonicalizesOpenAISubscriptionAlias(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	const query = "headless subscription web search query"
	var gotQuery string
	var gotModel string
	websearch.RegisterWithContextForTest(t, "openai_subscription", func(ctx context.Context, query, model string) (string, error) {
		gotQuery = query
		gotModel = model
		callback := websearch.UsageCallbackFromContext(ctx)
		if callback == nil {
			t.Fatal("UsageCallbackFromContext() = nil, want headless usage callback")
		}
		callback(api.Usage{InputTokens: 11, OutputTokens: 7})
		return "Summary:\nsubscription search result\n\nSources:\n\n1. Docs\n   URL: https://example.test/subscription-search", nil
	})

	cfg := newProjectMapDisabledConfig()
	cfg.WebSearch.Provider = "chatgpt"
	cfg.WebSearch.CacheEnabled = false
	cfg.SetProviderModelConfig("openai_subscription", config.ProviderModelConfig{DefaultModel: "gpt-5.5"})
	provider := &headlessOpenAISubscriptionProvider{}
	runner := newHeadlessRunnerWithOptions("probe", "gpt-5.5", provider, cfg, HeadlessRunOptions{})
	t.Cleanup(runner.agent.Cleanup)

	execCtx := runner.agent.toolExecutionContext(context.Background(), nil, io.Discard, io.Discard)
	result := searchtool.ExecuteWebSearch(execCtx, query)

	if !strings.Contains(result, "subscription search result") {
		t.Fatalf("ExecuteWebSearch() = %q, want subscription result", result)
	}
	if gotQuery != query {
		t.Fatalf("adapter query = %q, want %q", gotQuery, query)
	}
	if gotModel != "gpt-5.5" {
		t.Fatalf("adapter model = %q, want gpt-5.5", gotModel)
	}

	runner.agent.statsMu.Lock()
	defer runner.agent.statsMu.Unlock()
	if runner.agent.Stats.InputTokens != 11 || runner.agent.Stats.OutputTokens != 7 {
		t.Fatalf("headless stats usage = input %d output %d, want 11/7", runner.agent.Stats.InputTokens, runner.agent.Stats.OutputTokens)
	}
	if runner.agent.Stats.Provider != "openai_subscription" || runner.agent.Stats.Model != "gpt-5.5" {
		t.Fatalf("headless stats owner = %s/%s, want openai_subscription/gpt-5.5", runner.agent.Stats.Provider, runner.agent.Stats.Model)
	}
}

type headlessOpenAISubscriptionProvider struct{}

func (p *headlessOpenAISubscriptionProvider) Name() string { return "OpenAI Subscription" }

func (p *headlessOpenAISubscriptionProvider) ProviderConfigKey() string {
	return "openai_subscription"
}

func (p *headlessOpenAISubscriptionProvider) SupportsImages() bool { return false }

func (p *headlessOpenAISubscriptionProvider) IsFunctionCallingEnabled() bool { return true }

func (p *headlessOpenAISubscriptionProvider) ChatWithTools(context.Context, string, []api.Message, string) (string, error) {
	return "done", nil
}

func (p *headlessOpenAISubscriptionProvider) ChatWithImage(context.Context, string, []api.Message, string, *api.ImageData, string) (string, error) {
	return "done", nil
}
