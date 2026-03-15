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

	return CompactWebSearchResult(query, result)
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

// ── web_search result compaction ──

const (
	// webSearchMaxSummaryLines は tool-native result に残すサマリー行数上限。
	webSearchMaxSummaryLines = 15

	// webSearchMaxSources は tool-native result に残すソース数上限。
	webSearchMaxSources = 7

	// webSearchCompactMinLen は compaction を適用する最小バイト数。
	// これ未満の結果はそのまま返す。
	webSearchCompactMinLen = 600
)

// CompactWebSearchResult はプロバイダーから返された web_search 結果を
// summary-first / source-preserving な canonical 形式に変換する。
// query を先頭に付与して、history 上で検索意図が失われないようにする。
// 全プロバイダーが "Summary:\n..." + "Sources:\n..." 形式を返す前提。
func CompactWebSearchResult(query, result string) string {
	result = strings.TrimSpace(result)
	if result == "" || result == "No results found." {
		return result
	}

	// query ヘッダーを付与（provider result には含まれないため常に追加）
	queryHeader := fmt.Sprintf("Query: %s\n", query)

	if len(result) < webSearchCompactMinLen {
		return queryHeader + result
	}

	summary, sourcesBlock := splitWebSearchSections(result)

	// summary truncation
	summary = truncateWebSearchSummary(summary)

	// sources compaction
	sourcesBlock = compactWebSearchSources(sourcesBlock)

	var b strings.Builder
	b.WriteString(queryHeader)
	if summary != "" {
		b.WriteString(summary)
	}
	if sourcesBlock != "" {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(sourcesBlock)
	}

	return b.String()
}

// splitWebSearchSections は "Summary:" と "Sources:" のセクションを分離する。
func splitWebSearchSections(result string) (summary, sources string) {
	sourcesIdx := strings.Index(result, "\nSources:\n")
	if sourcesIdx < 0 {
		// "Sources:" がない場合は全体を summary として扱う
		return result, ""
	}
	return strings.TrimSpace(result[:sourcesIdx]), strings.TrimSpace(result[sourcesIdx+1:])
}

// truncateWebSearchSummary はサマリーを最大行数に切り詰める。
func truncateWebSearchSummary(summary string) string {
	lines := strings.Split(summary, "\n")
	if len(lines) <= webSearchMaxSummaryLines {
		return summary
	}
	truncated := strings.Join(lines[:webSearchMaxSummaryLines], "\n")
	return truncated + "\n[... summary truncated]"
}

// compactWebSearchSources はソースセクションを compact な単行形式に変換する。
// 入力形式（全プロバイダー共通）:
//
//	Sources:
//
//	1. Title
//	   URL: https://...
//
//	2. Title
//	   URL: https://...
//
// 出力形式:
//
//	Sources:
//	- Title (https://...)
//	- Title (https://...)
func compactWebSearchSources(sourcesBlock string) string {
	if sourcesBlock == "" {
		return ""
	}

	lines := strings.Split(sourcesBlock, "\n")

	var b strings.Builder
	b.WriteString("Sources:")
	count := 0

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])

		// skip header and empty lines
		if trimmed == "" || trimmed == "Sources:" {
			continue
		}

		// numbered entry: "1. Title"
		if len(trimmed) > 2 && trimmed[0] >= '1' && trimmed[0] <= '9' {
			dotIdx := strings.Index(trimmed, ". ")
			if dotIdx < 0 || dotIdx > 3 {
				continue
			}
			title := trimmed[dotIdx+2:]

			// look ahead for URL line
			url := ""
			if i+1 < len(lines) {
				nextTrimmed := strings.TrimSpace(lines[i+1])
				if strings.HasPrefix(nextTrimmed, "URL: ") {
					url = strings.TrimPrefix(nextTrimmed, "URL: ")
					i++ // consume the URL line
				}
			}

			count++
			if count > webSearchMaxSources {
				// current entry (count-th) is already omitted; count remaining after it
				remaining := 1 + countRemainingEntries(lines[i+1:])
				fmt.Fprintf(&b, "\n[+%d more sources]", remaining)
				break
			}

			if url != "" {
				fmt.Fprintf(&b, "\n- %s (%s)", title, url)
			} else {
				fmt.Fprintf(&b, "\n- %s", title)
			}
		}
	}

	if count == 0 {
		return sourcesBlock // couldn't parse → return as-is
	}

	return b.String()
}

// countRemainingEntries は残りの行から番号付きエントリの数を数える。
func countRemainingEntries(lines []string) int {
	count := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 2 && trimmed[0] >= '1' && trimmed[0] <= '9' {
			if dotIdx := strings.Index(trimmed, ". "); dotIdx >= 0 && dotIdx <= 3 {
				count++
			}
		}
	}
	return count
}
