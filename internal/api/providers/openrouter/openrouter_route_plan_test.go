package openrouter

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestResolveOpenRouterRoutePlanUsesAnthropicSkinForClaudeContextManagement(t *testing.T) {
	plan := resolveOpenRouterRoutePlan(config.DefaultConfig(), "https://openrouter.ai/api/v1/chat/completions", "anthropic/claude-sonnet-4.6")

	if plan.Route != DiagnosticRouteAnthropicMessages {
		t.Fatalf("Route = %q, want anthropic_messages", plan.Route)
	}
	if plan.APIURL != "https://openrouter.ai/api/v1/messages" {
		t.Fatalf("APIURL = %q, want Anthropic Skin URL", plan.APIURL)
	}
	if !strings.Contains(plan.Reason, "/v1/messages is selected") {
		t.Fatalf("Reason = %q, want Anthropic Skin reason", plan.Reason)
	}
}

func TestResolveOpenRouterRoutePlanUsesChatCompletionsWhenClaudeContextManagementDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ClaudeCompaction = false
	cfg.Compression.ClearToolUses = false

	plan := resolveOpenRouterRoutePlan(cfg, "https://openrouter.ai/api/v1/chat/completions", "anthropic/claude-sonnet-4.6")

	if plan.Route != DiagnosticRouteChatCompletions {
		t.Fatalf("Route = %q, want chat_completions", plan.Route)
	}
	if plan.APIURL != "https://openrouter.ai/api/v1/chat/completions" {
		t.Fatalf("APIURL = %q, want Chat Completions URL", plan.APIURL)
	}
	if !strings.Contains(plan.Reason, "Claude but OpenRouter Claude context management is disabled") {
		t.Fatalf("Reason = %q, want disabled Claude context management reason", plan.Reason)
	}
}
