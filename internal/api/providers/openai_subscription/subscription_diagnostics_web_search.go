package openaisubscription

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func runSubscriptionDiagnosticWebSearchSmokeRequest(ctx context.Context, provider *SubscriptionProvider, report SubscriptionDiagnosticReport, request subscriptionDiagnosticSmokeRequest) (SubscriptionDiagnosticSmokeRequestResult, error) {
	result := SubscriptionDiagnosticSmokeRequestResult{
		Name:             request.Name,
		WebSearchPayload: true,
		Route:            report.Route,
		Cost:             subscriptionDiagnosticSmokeCost(),
	}
	endpoint, err := validateSubscriptionResponsesEndpoint(DefaultSubscriptionAuthConfig().Endpoint)
	if err != nil {
		result.Error = RedactSubscriptionSecrets(err.Error())
		return result, errors.New(result.Error)
	}
	var usage api.Usage
	usageObserved := false
	requestCtx := websearch.WithUsageCallback(ctx, func(observed api.Usage) {
		usage.Add(observed)
		usageObserved = usageObserved || observed.HasTokenObservation()
	})
	started := time.Now()
	searchResult, err := provider.runSubscriptionWebSearch(requestCtx, endpoint, request.UserContent, report.Model)
	result.Ran = true
	result.Duration = time.Since(started).Round(time.Millisecond).String()
	result.WebSearchCallCount = searchResult.WebSearchCallCount
	result.UsageObserved = usageObserved
	result.Usage = providerdiag.SmokeUsageFromAPIUsage(usage)
	if err == nil || subscriptionWebSearchResultHasContent(searchResult) {
		result.Content = strings.TrimSpace(formatSubscriptionWebSearchResult(searchResult))
	}
	if err == nil {
		err = validateSubscriptionWebSearchSmokeResult(searchResult)
	}
	if err != nil {
		result.Error = RedactSubscriptionSecrets(err.Error())
		return result, errors.New(result.Error)
	}
	return result, nil
}

func subscriptionWebSearchResultHasContent(result subscriptionWebSearchResult) bool {
	return strings.TrimSpace(result.Summary) != "" || len(result.Sources) > 0
}

func subscriptionDiagnosticWebSearchPreviewBody(body subscriptionWebSearchRequest) map[string]any {
	return map[string]any{
		"model":                  body.Model,
		"stream":                 body.Stream,
		"store":                  body.Store,
		"instructions":           presenceLabel(body.Instructions),
		"input":                  presenceLabel(body.Input),
		"tools":                  subscriptionDiagnosticWebSearchToolsPreview(body.Tools),
		"tool_choice":            body.ToolChoice,
		"prompt_cache_key":       presenceLabel(body.PromptCacheKey),
		"previous_response_id":   "omitted",
		"context_management":     "omitted",
		"prompt_cache_retention": "omitted",
		"max_output_tokens":      "omitted",
		"include":                "omitted",
	}
}

func subscriptionDiagnosticWebSearchToolsPreview(tools []subscriptionWebSearchTool) []map[string]string {
	out := make([]map[string]string, 0, len(tools))
	for _, tool := range tools {
		out = append(out, map[string]string{"type": tool.Type})
	}
	return out
}
