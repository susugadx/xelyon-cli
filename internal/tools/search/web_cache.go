package search

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/cache"
	"github.com/susugadx/xelyon-cli/internal/config"
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
