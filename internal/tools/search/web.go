package search

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/cache"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

var (
	webSearchCacheMu       sync.Mutex
	webSearchCache         *cache.Cache
	webSearchCacheSettings webSearchCacheConfig
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
	searchProvider := resolveSearchProvider(cfg, execCtx.ProviderName, execCtx.ProviderConfigKey)
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

	return result
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

func normalizeCacheKey(searchOwnerKey, query string) string {
	searchOwnerKey = normalizeProviderName(searchOwnerKey)
	if searchOwnerKey == "" {
		searchOwnerKey = "default"
	}
	return searchOwnerKey + ":" + normalizeQuery(query)
}

func resolveSearchProvider(cfg *config.Config, mainProvider, mainProviderConfigKey string) string {
	if cfg != nil {
		provider := normalizeProviderName(cfg.WebSearch.Provider)
		if isNativeSearchProvider(provider) {
			return provider
		}
	}

	provider := normalizeProviderName(mainProviderConfigKey)
	if isNativeSearchProvider(provider) {
		return provider
	}

	provider = normalizeProviderName(mainProvider)
	if isNativeSearchProvider(provider) {
		return provider
	}

	return ""
}

func resolveSearchModel(cfg *config.Config, searchProvider, mainProvider, mainModel string) string {
	if config.SameProviderRuntimeIdentity(searchProvider, mainProvider) {
		return mainModel
	}
	if cfg == nil {
		return ""
	}
	return cfg.GetEffectiveModelForProvider(searchProvider)
}

func isNativeSearchProvider(provider string) bool {
	entry, ok := llmcatalog.ProviderDescriptorFor(provider)
	return ok && entry.NativeWebSearch
}

func webSearchProviderError() string {
	return fmt.Sprintf(`Web search requires a provider with native search support.
Set web_search.provider in config.yaml to one of: %s

Example:
  web_search:
    provider: gemini

Gemini API key is free at https://aistudio.google.com/apikey`, strings.Join(llmcatalog.NativeWebSearchProviderKeys(true), ", "))
}

func normalizeProviderName(providerName string) string {
	return config.NormalizeProviderName(providerName)
}
