package search

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

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

	requestCtx := webSearchRequestContext(execCtx, cfg, searchProvider, searchModel.Model)

	result, cached, err := searchWithCache(requestCtx, cfg, searchProvider, query, searchModel.Model)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	ownerLabel := webSearchOwnerLabel(searchProvider, searchModel)
	if cached {
		out.Green.Printf("🔍 Web search (cached, %s): %s\n", ownerLabel, query)
	} else {
		out.Green.Printf("🔍 Searching the web (%s): %s\n", ownerLabel, query)
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
	requestCtx := webSearchAPIRequestContext(ctx, cfg, req.UsageCallback, req.UsageAttribution, searchProvider, searchModel.Model)

	raw, cached, err := searchWithCache(requestCtx, cfg, searchProvider, req.Query, searchModel.Model)
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
		Model:            searchModel.Model,
		Cached:           cached,
		Raw:              raw,
		Results:          results,
		ResultsTruncated: resultsTruncated,
	}, nil
}
