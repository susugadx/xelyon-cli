package agent

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/history"
)

func TestToolExecutionContext_PrefersActiveModelOwnerForProviderConfigKey(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"claude": {
			DefaultModel: "claude-custom",
		},
		"anthropic": {
			DefaultModel: "anthropic-custom",
		},
	})
	runtime := NewAgentRuntimeWithConfig(cfg)

	agent := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "claude",
		CurrentModel:      "anthropic-custom",
		CurrentProvider:   &MockProvider{name: "claude"},
		Runtime:           runtime,
	}

	execCtx := agent.toolExecutionContext(context.Background(), nil, io.Discard, io.Discard)

	if execCtx.ProviderName != "claude" {
		t.Fatalf("ProviderName = %q, want %q", execCtx.ProviderName, "claude")
	}
	if execCtx.ProviderConfigKey != "anthropic" {
		t.Fatalf("ProviderConfigKey = %q, want %q", execCtx.ProviderConfigKey, "anthropic")
	}
}

func TestToolExecutionContext_IncludesSessionPromptCacheScope(t *testing.T) {
	agent := &Agent{
		Runtime: NewAgentRuntimeWithConfig(newProjectMapDisabledConfig()),
		agentConversationState: agentConversationState{
			session: &history.Session{ID: " session-tool "},
		},
	}

	execCtx := agent.toolExecutionContext(context.Background(), nil, io.Discard, io.Discard)

	for name, ctx := range map[string]context.Context{
		"request": execCtx.EffectiveContext(),
		"prompt":  execCtx.EffectivePromptContext(),
	} {
		scope, ok := api.PromptCacheScopeFromContext(ctx)
		if !ok {
			t.Fatalf("%s context PromptCacheScopeFromContext() ok = false, want true", name)
		}
		if scope.SessionID != "session-tool" || scope.TaskID != "" {
			t.Fatalf("%s context scope = %+v, want session-tool only", name, scope)
		}
	}
}

func TestToolExecutionContext_UsageAttributionAddsRequestOwnerCost(t *testing.T) {
	agent := NewAgentWithRuntime("llama3", &mockProvider{name: "ollama"}, false, NewAgentRuntimeWithConfig(newProjectMapDisabledConfig()))
	t.Cleanup(agent.Cleanup)

	execCtx := agent.toolExecutionContext(context.Background(), nil, io.Discard, io.Discard)
	if execCtx.UsageAttribution == nil {
		t.Fatal("UsageAttribution = nil, want stats callback")
	}

	execCtx.UsageAttribution("kimi", "kimi-k2.6", api.Usage{InputTokens: 10, OutputTokens: 4, CachedInputTokens: 2})

	agent.statsMu.Lock()
	defer agent.statsMu.Unlock()
	if agent.Stats.InputTokens != 10 || agent.Stats.OutputTokens != 4 || agent.Stats.CachedInputTokens != 2 {
		t.Fatalf("Stats usage = input %d output %d cached %d, want 10/4/2", agent.Stats.InputTokens, agent.Stats.OutputTokens, agent.Stats.CachedInputTokens)
	}
	if agent.Stats.LastUsage == nil || agent.Stats.LastUsage.InputTokens != 10 {
		t.Fatalf("LastUsage = %+v, want callback usage", agent.Stats.LastUsage)
	}
	if agent.Stats.Provider != "ollama" || agent.Stats.Model != "llama3" {
		t.Fatalf("Stats owner mutated to %s/%s, want ollama/llama3", agent.Stats.Provider, agent.Stats.Model)
	}
	if agent.Stats.AccumulatedCost <= 0 || agent.Stats.CostUnknown {
		t.Fatalf("AccumulatedCost = %f CostUnknown = %t, want known Kimi token cost", agent.Stats.AccumulatedCost, agent.Stats.CostUnknown)
	}
	if got := agent.Stats.EstimatedCostForConfig(agent.cfg()); got <= 0 {
		t.Fatalf("EstimatedCostForConfig = %f, want Kimi cost even when chat provider is Ollama", got)
	}
}

func TestToolExecutionContext_PromptIOIgnoresRequestDeadlineAndUsesExplicitCancel(t *testing.T) {
	requestCtx, cancelRequest := context.WithTimeout(
		context.WithValue(context.Background(), requestPromptContextKey{}, "tool-prompt"),
		time.Nanosecond,
	)
	defer cancelRequest()
	<-requestCtx.Done()

	explicitCancelCtx, cancelExplicit := context.WithCancel(context.Background())
	defer cancelExplicit()

	agent := &Agent{
		Runtime: NewAgentRuntimeWithConfig(newProjectMapDisabledConfig()),
		agentRequestState: agentRequestState{
			requestPromptCancelCtx: explicitCancelCtx,
		},
	}

	execCtx := agent.toolExecutionContext(requestCtx, nil, io.Discard, io.Discard)
	promptIO := execCtx.PromptIO()
	promptCtx := promptIO.PromptContext()

	if _, ok := promptCtx.Deadline(); ok {
		t.Fatal("PromptIO context should not inherit the API request deadline")
	}
	if err := promptCtx.Err(); err != nil {
		t.Fatalf("PromptIO context should stay active after request deadline, got %v", err)
	}
	if got := promptCtx.Value(requestPromptContextKey{}); got != "tool-prompt" {
		t.Fatalf("PromptIO context marker = %v, want tool-prompt", got)
	}

	cancelExplicit()

	select {
	case <-promptCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("PromptIO context should be cancelled by explicit request cancel")
	}
}

func TestBeginChatRequestContext_ToolPromptContextUsesInterruptScope(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	cfg.APIRetry.Timeout = 1
	agent := &Agent{Runtime: NewAgentRuntimeWithConfig(cfg)}

	requestCtx, cleanup := agent.beginChatRequestContext()
	defer cleanup()

	execCtx := agent.toolExecutionContext(requestCtx, nil, io.Discard, io.Discard)
	promptIO := execCtx.PromptIO()
	promptCtx := promptIO.PromptContext()

	if _, ok := promptCtx.Deadline(); ok {
		t.Fatal("tool prompt context should not expose the API request deadline")
	}

	agent.cancelActiveRequest("signal: interrupt")

	select {
	case <-promptCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("tool prompt context should be cancelled by request interrupt")
	}
	select {
	case <-requestCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("request context should be cancelled by request interrupt")
	}
}
