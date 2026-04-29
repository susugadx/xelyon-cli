package bedrock

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const (
	bedrockSmokeEnabledEnv       = "XELYON_BEDROCK_SMOKE"
	bedrockSmokeClaudeModelEnv   = "XELYON_BEDROCK_SMOKE_CLAUDE_MODEL"
	bedrockSmokeConverseModelEnv = "XELYON_BEDROCK_SMOKE_CONVERSE_MODEL"
)

func TestBedrockLiveSmoke_ClaudeMessagesRoute(t *testing.T) {
	requireBedrockLiveSmoke(t)
	t.Setenv("BEDROCK_FUNCTION_CALLING", "1")

	model := bedrockSmokeModel(bedrockSmokeClaudeModelEnv, defaultModel)

	t.Run("text", func(t *testing.T) {
		cfg := bedrockSmokeConfig(model)
		p := newBedrockLiveSmokeProvider(t, cfg)
		var usage api.Usage
		p.SetUsageCallback(func(u api.Usage) {
			usage = u
		})

		ctx, cancel := bedrockSmokeContext(t, cfg, nil)
		defer cancel()

		got, err := p.ChatWithTools(ctx, "You are a Bedrock smoke test.", []api.Message{
			{Role: "user", Content: "Reply with the exact token bedrock-claude-smoke-ok."},
		}, model)
		if err != nil {
			t.Fatalf("Claude text smoke failed: %v", err)
		}
		if !strings.Contains(strings.ToLower(got), "bedrock-claude-smoke-ok") {
			t.Fatalf("Claude text response = %q, want smoke token", got)
		}
		assertBedrockSmokeUsage(t, usage)
	})

	t.Run("tool_use", func(t *testing.T) {
		cfg := bedrockSmokeConfig(model)
		p := newBedrockLiveSmokeProvider(t, cfg)
		ctx, cancel := bedrockSmokeContext(t, cfg, bedrockSmokeTools())
		defer cancel()

		got, err := p.ChatWithTools(ctx, "You are a Bedrock smoke test. Use tools when directly asked.", []api.Message{
			{Role: "user", Content: `Call smoke_echo exactly once with {"value":"bedrock-claude-tool-ok"} and do not answer in prose.`},
		}, model)
		if err != nil {
			t.Fatalf("Claude tool smoke failed: %v", err)
		}
		assertBedrockSmokeToolCall(t, got, "smoke_echo")
	})

	t.Run("image", func(t *testing.T) {
		cfg := bedrockSmokeConfig(model)
		p := newBedrockLiveSmokeProvider(t, cfg)
		ctx, cancel := bedrockSmokeContext(t, cfg, nil)
		defer cancel()

		got, err := p.ChatWithImage(ctx, "You are a Bedrock smoke test.", nil, "What is the dominant color? Answer in one word.", bedrockSmokePNG(), model)
		if err != nil {
			t.Fatalf("Claude image smoke failed: %v", err)
		}
		if strings.TrimSpace(got) == "" {
			t.Fatal("Claude image smoke returned empty response")
		}
	})

	t.Run("thinking", func(t *testing.T) {
		cfg := bedrockSmokeConfig(model)
		cfg.Thinking.Enabled = true
		cfg.Thinking.Level = "low"
		p := newBedrockLiveSmokeProvider(t, cfg)
		ctx, cancel := bedrockSmokeContext(t, cfg, nil)
		defer cancel()

		got, err := p.ChatWithTools(ctx, "You are a Bedrock smoke test.", []api.Message{
			{Role: "user", Content: "Think briefly and then reply with the exact token bedrock-claude-thinking-ok."},
		}, model)
		if err != nil {
			t.Fatalf("Claude thinking smoke failed: %v", err)
		}
		if !strings.Contains(strings.ToLower(got), "bedrock-claude-thinking-ok") {
			t.Fatalf("Claude thinking response = %q, want smoke token", got)
		}
	})
}

func TestBedrockLiveSmoke_ConverseRoute(t *testing.T) {
	requireBedrockLiveSmoke(t)
	t.Setenv("BEDROCK_FUNCTION_CALLING", "1")

	model := bedrockSmokeModel(bedrockSmokeConverseModelEnv, "amazon.nova-pro-v1:0")

	t.Run("text_and_usage", func(t *testing.T) {
		cfg := bedrockSmokeConfig(model)
		p := newBedrockLiveSmokeProvider(t, cfg)
		var usage api.Usage
		p.SetUsageCallback(func(u api.Usage) {
			usage = u
		})

		ctx, cancel := bedrockSmokeContext(t, cfg, nil)
		defer cancel()

		got, err := p.ChatWithTools(ctx, "You are a Bedrock Converse smoke test.", []api.Message{
			{Role: "user", Content: "Reply with the exact token bedrock-converse-smoke-ok."},
		}, model)
		if err != nil {
			t.Fatalf("Converse text smoke failed: %v", err)
		}
		if !strings.Contains(strings.ToLower(got), "bedrock-converse-smoke-ok") {
			t.Fatalf("Converse text response = %q, want smoke token", got)
		}
		assertBedrockSmokeUsage(t, usage)
	})

	t.Run("tool_use", func(t *testing.T) {
		cfg := bedrockSmokeConfig(model)
		p := newBedrockLiveSmokeProvider(t, cfg)
		ctx, cancel := bedrockSmokeContext(t, cfg, bedrockSmokeTools())
		defer cancel()

		got, err := p.ChatWithTools(ctx, "You are a Bedrock Converse smoke test. Use tools when directly asked.", []api.Message{
			{Role: "user", Content: `Call smoke_echo exactly once with {"value":"bedrock-converse-tool-ok"} and do not answer in prose.`},
		}, model)
		if err != nil {
			t.Fatalf("Converse tool smoke failed: %v", err)
		}
		assertBedrockSmokeToolCall(t, got, "smoke_echo")
	})
}

func requireBedrockLiveSmoke(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("Bedrock live smoke is skipped in short mode")
	}
	if os.Getenv(bedrockSmokeEnabledEnv) != "1" {
		t.Skipf("set %s=1 to run Bedrock live smoke tests", bedrockSmokeEnabledEnv)
	}
}

func bedrockSmokeModel(envName, fallback string) string {
	if model := strings.TrimSpace(os.Getenv(envName)); model != "" {
		return model
	}
	return fallback
}

func newBedrockLiveSmokeProvider(t *testing.T, cfg *config.Config) *Provider {
	t.Helper()
	p, err := New()
	if err != nil {
		t.Fatalf("failed to create Bedrock provider: %v", err)
	}
	p.SetRuntimeConfig(cfg)
	return p
}

func bedrockSmokeConfig(model string) *config.Config {
	cfg := config.DefaultConfig()
	cfg.General.UILanguage = "en"
	cfg.PromptCache.Enabled = false
	cfg.Compression.ClaudeCompaction = false
	cfg.Streaming.IdleTimeoutSeconds = 120
	cfg.ProviderModels["bedrock"] = config.ProviderModelConfig{
		DefaultModel:     model,
		MaxOutputTokens:  128,
		AnthropicVersion: bedrockAnthropicVersion,
		ModelOverrides: map[string]config.ModelOverride{
			model: {MaxOutputTokens: 128},
		},
	}
	return cfg
}

func bedrockSmokeContext(t *testing.T, cfg *config.Config, tools []api.ToolDefinition) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	ctx = config.WithContext(ctx, cfg)
	ctx = ui.WithRuntime(ctx, ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesOff)
	if len(tools) > 0 {
		ctx = api.WithToolDefinitions(ctx, tools)
	}
	return ctx, cancel
}

func bedrockSmokeTools() []api.ToolDefinition {
	return []api.ToolDefinition{
		{
			Name:        "smoke_echo",
			Description: "Echo a smoke-test marker. Use this tool when the user asks to call smoke_echo.",
			Parameters: map[string]any{
				"type":     "object",
				"required": []any{"value"},
				"properties": map[string]any{
					"value": map[string]any{
						"type":        "string",
						"description": "Smoke-test marker value.",
					},
				},
			},
		},
	}
}

func assertBedrockSmokeToolCall(t *testing.T, got, tool string) {
	t.Helper()
	if !strings.Contains(got, `"tool":"`+tool+`"`) {
		t.Fatalf("response = %q, want %q tool call JSON", got, tool)
	}
}

func assertBedrockSmokeUsage(t *testing.T, usage api.Usage) {
	t.Helper()
	if usage.InputTokens <= 0 {
		t.Fatalf("usage.InputTokens = %d, want > 0; usage=%#v", usage.InputTokens, usage)
	}
	if usage.OutputTokens <= 0 && usage.ThinkingTokens <= 0 {
		t.Fatalf("usage output/thinking tokens are empty; usage=%#v", usage)
	}
}

func bedrockSmokePNG() *api.ImageData {
	const redPNG16x16 = "iVBORw0KGgoAAAANSUhEUgAAABAAAAAQCAIAAACQkWg2AAAAFklEQVR42mP4z8BAEmIY1TCqYfhqAACQ+f8B8u7oVwAAAABJRU5ErkJggg=="
	return &api.ImageData{
		Path:      "bedrock-smoke-red-16x16.png",
		MediaType: "image/png",
		Base64:    redPNG16x16,
		Size:      79,
	}
}
