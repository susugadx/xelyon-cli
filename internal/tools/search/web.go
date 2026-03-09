package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api/providers/serper"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// ExecuteWebSearch executes web search using a provider-native search if available,
// then falls back to Serper. Cache is shared across all implementations.
func ExecuteWebSearch(execCtx tools.ExecutionContext, query string) string {
	if query == "" {
		return "Error: query is required"
	}

	out := execCtx.Output()

	// 確認プロンプト（--auto-approve / config で自動承認可能）
	dec := common.ConfirmWithAutoApproveDecisionAndOptions(execCtx.PromptIO(), execCtx.ConfirmOptions(), "web_search",
		fmt.Sprintf("Execute web search: %s", query))
	switch dec.Action {
	case common.ConfirmNo:
		return "User rejected web search"
	case common.ConfirmComment:
		return fmt.Sprintf("User feedback: %s", dec.Comment)
	}

	requestCtx := tools.WithRegistry(context.Background(), execCtx.EffectiveRegistry())
	requestCtx = tools.WithConfig(requestCtx, execCtx.EffectiveConfig())

	result, cached, source, err := executeWebSearchWithFallback(requestCtx, out, execCtx.EffectiveConfig(), query, execCtx.ProviderName, execCtx.Model)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if cached {
		out.Green.Printf("🔍 Web search (cached): %s\n", query)
	} else {
		out.Green.Printf("🔍 Searching the web (%s): %s\n", source, query)
	}

	return result
}

func executeWebSearchWithFallback(ctx context.Context, out common.Output, cfg *config.Config, query, providerName, model string) (string, bool, string, error) {
	providerName = normalizeProviderName(providerName)
	cacheScope := cacheScopeForProvider(providerName)

	searchSource := "serper"
	result, cached, err := serper.SearchWithCacheAndConfig(cfg, cacheScope, query, func(q string) (string, error) {
		output, source, err := searchWithProvider(ctx, out, q, providerName, model)
		searchSource = source
		return output, err
	})
	return result, cached, searchSource, err
}

func searchWithProvider(ctx context.Context, out common.Output, query, providerName, model string) (string, string, error) {
	switch providerName {
	case "gemini":
		result, err := websearch.SearchWithContext(ctx, providerName, query, model)
		if err == nil {
			return result, "gemini", nil
		}
		out.Yellow.Printf("⚠️  Gemini native web search failed, falling back to Serper: %v\n", err)
	case "claude":
		result, err := websearch.SearchWithContext(ctx, providerName, query, model)
		if err == nil {
			return result, "claude", nil
		}
		out.Yellow.Printf("⚠️  Claude native web search failed, falling back to Serper: %v\n", err)
	case "openai":
		result, err := websearch.SearchWithContext(ctx, providerName, query, model)
		if err == nil {
			return result, "openai", nil
		}
		out.Yellow.Printf("⚠️  OpenAI native web search failed, falling back to Serper: %v\n", err)
	}

	result, err := serper.WebSearch(query)
	if err != nil {
		return "", "serper", fmt.Errorf("serper fallback failed: %w", err)
	}
	return result, "serper", nil
}

func normalizeProviderName(providerName string) string {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case "anthropic":
		return "claude"
	default:
		return strings.ToLower(strings.TrimSpace(providerName))
	}
}

func cacheScopeForProvider(providerName string) string {
	switch providerName {
	case "gemini", "claude", "openai":
		return providerName
	default:
		return "serper"
	}
}
