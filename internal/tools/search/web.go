package search

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/cache"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/utilitymodel"
)

var (
	webSearchCacheMu       sync.Mutex
	webSearchCache         *cache.Cache
	webSearchCacheSettings webSearchCacheConfig
	runUtilityModelTask    = utilitymodel.Run
)

type webSearchCacheConfig struct {
	Enabled bool
	Size    int
	TTL     int
}

// ExecuteWebSearch はネイティブ Web 検索を実行し、必要に応じて結果キャッシュを利用する。
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

	cfg := execCtx.EffectiveConfig()
	searchProvider := resolveSearchProvider(cfg, execCtx.ProviderName)
	if searchProvider == "" {
		return webSearchProviderError()
	}
	searchModel := resolveSearchModel(cfg, searchProvider, execCtx.ProviderName, execCtx.Model)

	requestCtx := tools.WithRegistry(context.Background(), execCtx.EffectiveRegistry())
	requestCtx = tools.WithConfig(requestCtx, cfg)

	result, cached, err := searchWithCache(requestCtx, cfg, searchProvider, query, searchModel)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if cached {
		out.Green.Printf("🔍 Web search (cached): %s\n", query)
	} else {
		out.Green.Printf("🔍 Searching the web (%s): %s\n", searchProvider, query)
	}

	return compactWebSearchResultWithUtilityModel(execCtx, query, result)
}

func searchWithCache(ctx context.Context, cfg *config.Config, provider, query, model string) (string, bool, error) {
	searchCache := getWebSearchCache(cfg)
	cacheKey := normalizeCacheKey(provider, query)

	if searchCache != nil {
		if cached, err := searchCache.Get(cacheKey); err == nil {
			return string(cached), true, nil
		}
	}

	result, err := websearch.SearchWithContext(ctx, provider, query, model)
	if err != nil {
		return "", false, err
	}

	if searchCache != nil {
		searchCache.Set(cacheKey, []byte(result), 0)
	}

	return result, false, nil
}

func getWebSearchCache(cfg *config.Config) *cache.Cache {
	settings := newWebSearchCacheConfig(cfg)

	webSearchCacheMu.Lock()
	defer webSearchCacheMu.Unlock()

	if webSearchCache != nil && webSearchCacheSettings == settings {
		return webSearchCache
	}
	if webSearchCache == nil && webSearchCacheSettings == settings {
		return nil
	}

	webSearchCacheSettings = settings
	if !settings.Enabled || settings.Size <= 0 {
		webSearchCache = nil
		return nil
	}

	webSearchCache = cache.New(cache.Config{
		Enabled:    true,
		Capacity:   settings.Size,
		DefaultTTL: time.Duration(settings.TTL) * time.Second,
	}, nil)
	return webSearchCache
}

func newWebSearchCacheConfig(cfg *config.Config) webSearchCacheConfig {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return webSearchCacheConfig{
		Enabled: cfg.WebSearch.CacheEnabled,
		Size:    cfg.WebSearch.CacheSize,
		TTL:     cfg.WebSearch.CacheTTL,
	}
}

func normalizeQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}

func normalizeCacheKey(provider, query string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		provider = "default"
	}
	return provider + ":" + normalizeQuery(query)
}

func resolveSearchProvider(cfg *config.Config, mainProvider string) string {
	if cfg != nil {
		provider := normalizeProviderName(cfg.WebSearch.Provider)
		if isNativeSearchProvider(provider) {
			return provider
		}
	}

	provider := normalizeProviderName(mainProvider)
	if isNativeSearchProvider(provider) {
		return provider
	}

	return ""
}

func resolveSearchModel(cfg *config.Config, searchProvider, mainProvider, mainModel string) string {
	if searchProvider == normalizeProviderName(mainProvider) {
		return mainModel
	}
	if cfg == nil {
		return ""
	}
	if providerConfig, ok := cfg.ProviderModels[searchProvider]; ok {
		return providerConfig.DefaultModel
	}
	return ""
}

func isNativeSearchProvider(provider string) bool {
	switch provider {
	case "openai", "gemini", "claude":
		return true
	default:
		return false
	}
}

func webSearchProviderError() string {
	return `Web search requires a provider with native search support.
Set web_search.provider in config.yaml to one of: openai, gemini, claude

Example:
  web_search:
    provider: gemini

Gemini API key is free at https://aistudio.google.com/apikey`
}

func normalizeProviderName(providerName string) string {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case "anthropic":
		return "claude"
	default:
		return strings.ToLower(strings.TrimSpace(providerName))
	}
}

const (
	// webSearchUtilityModelMinTokens は utility model を使う最低入力サイズ。
	webSearchUtilityModelMinTokens = 1200
)

func compactWebSearchResultWithUtilityModel(execCtx tools.ExecutionContext, query, result string) string {
	compacted := result
	if !shouldUseUtilityModelForWebSearch(execCtx.EffectiveConfig(), result) {
		return compacted
	}

	systemPrompt, userPrompt := buildWebSearchUtilityPrompts(query, result)
	utilityResult, err := runUtilityModelTask(execCtx.EffectiveContext(), execCtx.EffectiveConfig(), utilitymodel.TaskWebSearchCompaction, systemPrompt, userPrompt)
	if err != nil {
		return compacted
	}

	utilityResult = normalizeUtilityWebSearchOutput(utilityResult)
	if utilityResult == "" {
		return compacted
	}

	return utilityResult
}

func shouldUseUtilityModelForWebSearch(cfg *config.Config, result string) bool {
	if !utilitymodel.EnabledForTask(cfg, utilitymodel.TaskWebSearchCompaction) {
		return false
	}

	result = strings.TrimSpace(result)
	if result == "" || result == "No results found." {
		return false
	}

	return token.EstimateTokenCount(result) >= webSearchUtilityModelMinTokens
}

func buildWebSearchUtilityPrompts(query, result string) (systemPrompt, userPrompt string) {
	systemPrompt = `You are a utility model for XELYON.
Rewrite raw native web-search output into a shorter canonical result.

Rules:
- Do not add facts that are not present in the raw result.
- Preserve dates, numbers, names, and URLs exactly when present.
- Keep the summary concise and source-preserving.
- Use exactly this output format when results exist:
Summary:
- ...
Sources:
- Title (URL)
- If there are no results, return exactly: No results found.`

	userPrompt = fmt.Sprintf("Query: %s\n\nRaw web search result:\n%s", query, result)
	return systemPrompt, userPrompt
}

func normalizeUtilityWebSearchOutput(result string) string {
	result = strings.TrimSpace(result)
	if result == "" {
		return ""
	}

	if strings.HasPrefix(result, "```") {
		result = strings.TrimPrefix(result, "```")
		result = strings.TrimSuffix(result, "```")
		result = strings.TrimSpace(result)
		if nl := strings.IndexByte(result, '\n'); nl >= 0 {
			firstLine := strings.TrimSpace(result[:nl])
			if !strings.Contains(firstLine, ":") {
				result = strings.TrimSpace(result[nl+1:])
			}
		}
	}

	if result == "No results found." {
		return result
	}
	if strings.Contains(result, "Summary:") {
		return result
	}
	if strings.Contains(result, "Sources:") {
		return ""
	}
	return "Summary:\n" + result
}
