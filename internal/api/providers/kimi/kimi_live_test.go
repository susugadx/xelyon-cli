//go:build live

package kimi

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

const kimiLiveRequestTimeout = 300 * time.Second

func TestKimiLiveSmoke(t *testing.T) {
	apiKey := requireKimiLiveAPIKey(t)
	t.Setenv("KIMI_FUNCTION_CALLING", "0")

	t.Run("k2.6 thinking off", func(t *testing.T) {
		p := New(apiKey)
		content, _, _ := runKimiLiveChat(t, p, kimiLiveConfig("kimi", "kimi-k2.6", false), "kimi-k2.6", "xelyon-kimi-live-off", "Reply with: xelyon kimi live off ok")
		if strings.TrimSpace(content) == "" {
			t.Fatal("Kimi live thinking off returned empty content")
		}
	})

	t.Run("k2.6 thinking on", func(t *testing.T) {
		p := New(apiKey)
		content, _, _ := runKimiLiveChat(t, p, kimiLiveConfig("kimi", "kimi-k2.6", true), "kimi-k2.6", "xelyon-kimi-live-on", "Think briefly, then reply with: xelyon kimi live thinking ok")
		if strings.TrimSpace(content) == "" {
			t.Fatal("Kimi live thinking on returned empty content")
		}
	})

	t.Run("k2.5 normal", func(t *testing.T) {
		p := New(apiKey)
		content, _, _ := runKimiLiveChat(t, p, kimiLiveConfig("kimi", "kimi-k2.5", false), "kimi-k2.5", "xelyon-kimi-live-k25", "Reply with: xelyon kimi live k25 ok")
		if strings.TrimSpace(content) == "" {
			t.Fatal("Kimi live k2.5 returned empty content")
		}
	})

	t.Run("moonshot alias provider", func(t *testing.T) {
		provider, err := api.NewProvider("moonshot")
		if err != nil {
			t.Fatalf("NewProvider(moonshot) error = %v", err)
		}
		p, ok := provider.(*Provider)
		if !ok {
			t.Fatalf("NewProvider(moonshot) type = %T, want *kimi.Provider", provider)
		}
		content, _, _ := runKimiLiveChat(t, p, kimiLiveConfig("moonshot", "kimi-k2.5", false), "", "xelyon-kimi-live-moonshot", "Reply with: xelyon kimi live moonshot ok")
		if strings.TrimSpace(content) == "" {
			t.Fatal("Kimi live moonshot alias returned empty content")
		}
	})

	t.Run("usage callback", func(t *testing.T) {
		p := New(apiKey)
		_, usage, observed := runKimiLiveChat(t, p, kimiLiveConfig("kimi", "kimi-k2.6", false), "kimi-k2.6", "xelyon-kimi-live-usage", "Reply with: xelyon kimi live usage ok")
		if !observed {
			t.Fatal("usage callback was not observed")
		}
		if usage.InputTokens <= 0 || usage.OutputTokens <= 0 {
			t.Fatalf("usage = %+v, want input/output tokens", usage)
		}
	})

	t.Run("session prompt cache observation", func(t *testing.T) {
		p := New(apiKey)
		cfg := kimiLiveConfig("kimi", "kimi-k2.6", false)
		sessionID := "xelyon-kimi-live-cache"
		_, firstUsage, firstObserved := runKimiLiveChat(t, p, cfg, "kimi-k2.6", sessionID, "Reply with: xelyon kimi live cache one")
		_, secondUsage, secondObserved := runKimiLiveChat(t, p, cfg, "kimi-k2.6", sessionID, "Reply with: xelyon kimi live cache two")
		if !firstObserved || !secondObserved {
			t.Fatalf("usage observed first=%t second=%t, want both true", firstObserved, secondObserved)
		}
		t.Logf("Kimi live cached_tokens first=%d second=%d", firstUsage.CachedInputTokens, secondUsage.CachedInputTokens)
	})
}

func TestKimiLiveToolSmoke(t *testing.T) {
	apiKey := requireKimiLiveAPIKey(t)
	if os.Getenv("XELYON_KIMI_TOOL_SMOKE") != "1" {
		t.Skip("set XELYON_KIMI_TOOL_SMOKE=1 to run the live Kimi tool smoke test")
	}
	t.Setenv("KIMI_FUNCTION_CALLING", "1")

	cfg := kimiLiveConfig("kimi", "kimi-k2.6", false)
	ctx, cancel := kimiLiveContext(cfg, "xelyon-kimi-live-tool")
	defer cancel()
	ctx = api.WithToolDefinitions(ctx, diagnosticSmokeToolDefinitions())

	p := New(apiKey)
	p.SetToolChoice(diagnosticSmokeToolName)
	var usage api.Usage
	p.SetUsageCallback(func(observed api.Usage) {
		usage = observed
	})

	content, err := p.ChatWithTools(
		ctx,
		"Use the diagnostic tool.",
		[]api.Message{{Role: "user", Content: `Call xelyon_kimi_doctor_probe exactly once with {"value":"kimi-live-tool-ok"} and do not answer in prose.`}},
		"kimi-k2.6",
	)
	if err != nil {
		skipIfTransientKimiLiveError(t, err)
		t.Fatalf("Kimi live tool smoke failed: %v", err)
	}
	if !diagnosticSmokeContentHasToolCall(content) {
		t.Fatalf("tool smoke response = %q, want %s tool call JSON", content, diagnosticSmokeToolName)
	}
	t.Logf("Kimi live tool smoke cached_tokens=%d", usage.CachedInputTokens)
}

func requireKimiLiveAPIKey(t *testing.T) string {
	t.Helper()
	apiKey := strings.TrimSpace(os.Getenv(kimiAPIKeyEnv))
	if apiKey == "" {
		t.Skipf("set %s to run live Kimi smoke tests", kimiAPIKeyEnv)
	}
	return apiKey
}

func runKimiLiveChat(t *testing.T, p *Provider, cfg *config.Config, model, sessionID, userContent string) (string, api.Usage, bool) {
	t.Helper()
	ctx, cancel := kimiLiveContext(cfg, sessionID)
	defer cancel()

	var usage api.Usage
	usageObserved := false
	p.SetUsageCallback(func(observed api.Usage) {
		usage = observed
		usageObserved = true
	})

	content, err := p.ChatWithTools(
		ctx,
		"Reply briefly.",
		[]api.Message{{Role: "user", Content: userContent}},
		model,
	)
	if err != nil {
		skipIfTransientKimiLiveError(t, err)
		t.Fatalf("Kimi live ChatWithTools() error = %v", err)
	}
	return content, usage, usageObserved
}

func kimiLiveConfig(provider, model string, thinking bool) *config.Config {
	cfg := config.DefaultConfig()
	cfg.Thinking.Enabled = thinking
	cfg.Thinking.Level = "high"
	cfg.SetProviderModelConfig(provider, config.ProviderModelConfig{
		DefaultModel:    model,
		CatalogModel:    model,
		MaxOutputTokens: 64,
		ModelOverrides: map[string]config.ModelOverride{
			model: {
				CatalogModel:    model,
				MaxOutputTokens: 64,
			},
		},
	})
	return cfg
}

func kimiLiveContext(cfg *config.Config, sessionID string) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), kimiLiveRequestTimeout)
	ctx = config.WithContext(ctx, cfg)
	ctx = uiruntime.WithRuntime(ctx, uiruntime.NewRuntime(strings.NewReader(""), io.Discard, io.Discard))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	ctx = api.WithPromptCacheScope(ctx, api.PromptCacheScope{SessionID: sessionID})
	ctx = api.WithToolDefinitions(ctx, nil)
	return ctx, cancel
}

func skipIfTransientKimiLiveError(t *testing.T, err error) {
	t.Helper()
	if isTransientKimiSmokeError(err) {
		t.Skipf("live Kimi API returned transient condition: %v", err)
	}
}
