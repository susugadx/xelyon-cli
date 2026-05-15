package kimi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

const (
	defaultKimiDiagnosticSmokeTimeout         = 120 * time.Second
	defaultKimiDiagnosticSmokeMaxOutputTokens = 64
)

func runKimiDiagnosticSmoke(ctx context.Context, cfg *config.Config, report DiagnosticReport, options DiagnosticOptions) (DiagnosticSmokeResult, error) {
	timeout := options.SmokeTimeout
	if timeout <= 0 {
		timeout = defaultKimiDiagnosticSmokeTimeout
	}
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultKimiDiagnosticSmokeMaxOutputTokens
	}

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	catalogModel := report.CatalogModel
	if strings.TrimSpace(catalogModel) == "" {
		catalogModel = report.Model
	}
	smokeCfg := kimiDiagnosticPolicyConfig(cfg, report.Model, catalogModel, maxOutputTokens)

	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}

	provider := New(os.Getenv(kimiAPIKeyEnv))
	result := DiagnosticSmokeResult{Ran: true}
	plan := buildKimiDiagnosticRequestPlan(options)

	for _, request := range plan.Requests {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			result.Requests = append(result.Requests, newKimiDiagnosticSkippedToolSmokeRequest(request))
			continue
		}

		requestResult, err := runKimiDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report.Model, request, output)
		result.Requests = append(result.Requests, requestResult)
		result.addRequestObservation(requestResult)
		if requestResult.Content != "" {
			result.Content = requestResult.Content
		}
		if request.ImagePayload {
			result.ImagePayload = true
			if !requestResult.PromptCacheKeyPresent {
				result.markLastRequestError("image smoke request did not include prompt_cache_key")
				return result, fmt.Errorf("image smoke request did not include prompt_cache_key")
			}
		}
		if request.WebSearchPayload {
			result.WebSearchPayload = true
			if !requestResult.PromptCacheKeyPresent {
				result.markLastRequestError("web search smoke request did not include prompt_cache_key")
				return result, fmt.Errorf("web search smoke request did not include prompt_cache_key")
			}
		}
		if request.ToolPayload && err == nil {
			result.ToolPayload = true
			if !diagnosticSmokeContentHasToolCall(requestResult.Content) {
				err := fmt.Errorf("tool smoke response did not include %s function_call", diagnosticSmokeToolName)
				result.markLastRequestError(err.Error())
				return result, err
			}
		}
		if err != nil {
			return result, err
		}
	}

	if plan.RunTextSmoke {
		first := diagnosticSmokePromptCacheKey(result.Requests, kimiDiagnosticSmokeCacheFirstName)
		second := diagnosticSmokePromptCacheKey(result.Requests, kimiDiagnosticSmokeCacheSecondName)
		if first == "" || second == "" || first != second {
			return result, fmt.Errorf("session-aware prompt_cache_key mismatch: first=%q second=%q", first, second)
		}
	}

	return result, nil
}

func (r *DiagnosticSmokeResult) markLastRequestError(message string) {
	if r == nil || len(r.Requests) == 0 {
		return
	}
	r.Requests[len(r.Requests)-1].Error = message
}

func diagnosticSmokePromptCacheKey(requests []DiagnosticSmokeRequestResult, name string) string {
	for _, request := range requests {
		if request.Name == name {
			return request.PromptCacheKey
		}
	}
	return ""
}

func diagnosticUsageObservation(usage api.Usage) DiagnosticUsageObservation {
	return DiagnosticUsageObservation{
		InputTokens:              usage.InputTokens,
		OutputTokens:             usage.OutputTokens,
		ThinkingTokens:           usage.ThinkingTokens,
		CachedInputTokens:        usage.CachedInputTokens,
		WebSearchCallCount:       usage.WebSearchCalls,
		WebSearchCallFeeEstimate: usage.StorageCost,
		SearchResultTotalTokens:  usage.WebSearchResultTokens,
	}
}

func (r *DiagnosticSmokeResult) addRequestObservation(request DiagnosticSmokeRequestResult) {
	if request.Skipped {
		return
	}
	if request.UsageObserved {
		r.UsageObserved = true
	}
	r.CachedInputTokens += request.Usage.CachedInputTokens
	r.WebSearchCallCount += request.Usage.WebSearchCallCount
	r.WebSearchCallFeeEstimate += request.Usage.WebSearchCallFeeEstimate
	r.SearchResultTotalTokens += request.Usage.SearchResultTotalTokens
	r.WebSearchUsageObserved = r.WebSearchUsageObserved || request.WebSearchUsageObserved
}

func (u DiagnosticUsageObservation) webSearchUsageObserved() bool {
	return u.WebSearchCallCount > 0
}

func isTransientKimiSmokeError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	message := strings.ToLower(err.Error())
	if strings.Contains(message, "429") ||
		strings.Contains(message, "rate limit") ||
		strings.Contains(message, "timeout") ||
		strings.Contains(message, "deadline exceeded") ||
		strings.Contains(message, "api error (5") ||
		strings.Contains(message, "status 5") {
		return true
	}
	return false
}
