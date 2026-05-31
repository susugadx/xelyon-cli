package search

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/cache"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

var (
	webSearchURLRE               = regexp.MustCompile(`https?://[^\s<>"'` + "`" + `\]\)}]+`)
	webSearchResultIndexPrefixRE = regexp.MustCompile(`^\d+[.)]\s*`)
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

// WebSearchRequest は非対話 Web 検索 API の入力を表す。
type WebSearchRequest struct {
	Config                *config.Config
	MainProvider          string
	MainProviderConfigKey string
	MainModel             string
	Query                 string
	MaxResults            int
	UsageCallback         api.UsageCallback
	UsageAttribution      tools.UsageAttributionCallback
}

// WebSearchResponse は非対話 Web 検索 API の結果を表す。
type WebSearchResponse struct {
	Provider         string
	Model            string
	Cached           bool
	Raw              string
	Results          []WebSearchResult
	ResultsTruncated bool
}

// WebSearchResult は provider の検索回答から抽出した URL 付き検索結果。
type WebSearchResult struct {
	Title        string
	URL          string
	Snippet      string
	SourceDomain string
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

	requestCtx := webSearchRequestContext(execCtx, cfg, searchProvider, searchModel)

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

// SearchWeb は provider 解決・cache・native web search registry を使って非対話検索を実行する。
func SearchWeb(ctx context.Context, req WebSearchRequest) (WebSearchResponse, error) {
	if strings.TrimSpace(req.Query) == "" {
		return WebSearchResponse{}, fmt.Errorf("query is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	cfg := req.Config
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	searchProvider := resolveSearchProvider(cfg, req.MainProvider, req.MainProviderConfigKey)
	if searchProvider == "" {
		return WebSearchResponse{}, errors.New(webSearchProviderError())
	}
	searchModel := resolveSearchModel(cfg, searchProvider, req.MainProvider, req.MainModel)
	requestCtx := tools.WithConfig(ctx, cfg)
	requestCtx = api.WithAssistantUpdateMode(requestCtx, api.AssistantUpdatesOff)
	if callback := webSearchProviderUsageCallback(req.UsageCallback, req.UsageAttribution, searchProvider, searchModel); callback != nil {
		requestCtx = websearch.WithUsageCallback(requestCtx, callback)
	}

	raw, cached, err := searchWithCache(requestCtx, cfg, searchProvider, req.Query, searchModel)
	if err != nil {
		return WebSearchResponse{}, err
	}

	results := ParseWebSearchResults(raw)
	resultsTruncated := req.MaxResults > 0 && len(results) > req.MaxResults
	if resultsTruncated {
		results = results[:req.MaxResults]
	}
	return WebSearchResponse{
		Provider:         searchProvider,
		Model:            searchModel,
		Cached:           cached,
		Raw:              raw,
		Results:          results,
		ResultsTruncated: resultsTruncated,
	}, nil
}

func webSearchRequestContext(execCtx tools.ExecutionContext, cfg *config.Config, searchProvider, searchModel string) context.Context {
	requestCtx := execCtx.EffectiveContext()
	requestCtx = tools.WithRegistry(requestCtx, execCtx.EffectiveRegistry())
	requestCtx = tools.WithConfig(requestCtx, cfg)
	requestCtx = api.WithAssistantUpdateMode(requestCtx, api.AssistantUpdatesOff)
	return websearch.WithUsageCallback(requestCtx, webSearchUsageCallback(execCtx, searchProvider, searchModel))
}

func webSearchUsageCallback(execCtx tools.ExecutionContext, provider, model string) api.UsageCallback {
	return webSearchProviderUsageCallback(nil, execCtx.UsageAttribution, provider, model)
}

func webSearchProviderUsageCallback(legacy api.UsageCallback, attribution tools.UsageAttributionCallback, provider, model string) api.UsageCallback {
	if legacy == nil && attribution == nil {
		return nil
	}
	return func(usage api.Usage) {
		if legacy != nil {
			legacy(usage)
		}
		if attribution != nil {
			attribution(provider, model, usage)
		}
	}
}

// ParseWebSearchResults は provider のテキスト回答から URL 付き検索結果を抽出する。
func ParseWebSearchResults(raw string) []WebSearchResult {
	lines := strings.Split(raw, "\n")
	results := make([]WebSearchResult, 0)
	seen := make(map[string]struct{})
	for i, line := range lines {
		urls := webSearchURLRE.FindAllString(line, -1)
		for _, rawURL := range urls {
			cleanURL := cleanWebSearchResultURL(rawURL)
			if cleanURL == "" {
				continue
			}
			if _, exists := seen[cleanURL]; exists {
				continue
			}
			seen[cleanURL] = struct{}{}
			results = append(results, WebSearchResult{
				Title:        webSearchResultTitle(lines, i, cleanURL),
				URL:          cleanURL,
				Snippet:      webSearchResultSnippet(lines, i),
				SourceDomain: webSearchResultDomain(cleanURL),
			})
		}
	}
	return results
}

func cleanWebSearchResultURL(rawURL string) string {
	candidate := strings.TrimSpace(rawURL)
	candidate = strings.TrimRight(candidate, ".,;:!?")
	parsed, err := url.Parse(candidate)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return ""
	}
	return parsed.String()
}

func webSearchResultTitle(lines []string, urlLine int, resultURL string) string {
	for i := urlLine; i >= 0 && i >= urlLine-2; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.Contains(line, "URL:") {
			continue
		}
		line = strings.TrimSpace(webSearchResultIndexPrefixRE.ReplaceAllString(line, ""))
		if line != "" && !strings.Contains(line, resultURL) {
			return line
		}
	}
	if domain := webSearchResultDomain(resultURL); domain != "" {
		return domain
	}
	return resultURL
}

func webSearchResultSnippet(lines []string, urlLine int) string {
	start := max(0, urlLine-1)
	end := min(len(lines), urlLine+2)
	parts := make([]string, 0, end-start)
	for _, line := range lines[start:end] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts = append(parts, line)
	}
	snippet := strings.Join(parts, " ")
	if len(snippet) > 500 {
		snippet = snippet[:500]
	}
	return snippet
}

func webSearchResultDomain(resultURL string) string {
	parsed, err := url.Parse(resultURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
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
