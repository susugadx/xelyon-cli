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

type kimiDiagnosticSmokeRequest struct {
	Name             string
	SystemPrompt     string
	UserContent      string
	Thinking         bool
	SessionID        string
	ToolPayload      bool
	ImagePayload     bool
	WebSearchPayload bool
}

const (
	kimiDiagnosticSmokeCacheFirstName  = "thinking_off_cache_first"
	kimiDiagnosticSmokeCacheSecondName = "thinking_off_cache_second"
	kimiDiagnosticSmokeThinkingName    = "thinking_on"
	kimiDiagnosticSmokeImageName       = "image_smoke"
	kimiDiagnosticSmokeWebSearchName   = "web_search_smoke"
	kimiDiagnosticSmokeToolName        = "tool_smoke"
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

	smokeCfg := config.CloneConfig(cfg)
	catalogModel := report.CatalogModel
	if strings.TrimSpace(catalogModel) == "" {
		catalogModel = report.Model
	}
	smokeCfg.SetProviderModelConfig("kimi", config.ProviderModelConfig{
		DefaultModel:    report.Model,
		CatalogModel:    catalogModel,
		MaxOutputTokens: maxOutputTokens,
		ModelOverrides: map[string]config.ModelOverride{
			report.Model: {
				CatalogModel:    catalogModel,
				MaxOutputTokens: maxOutputTokens,
			},
		},
	})

	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}

	provider := New(os.Getenv(kimiAPIKeyEnv))
	result := DiagnosticSmokeResult{Ran: true}
	requests, runTextSmoke := kimiDiagnosticSmokeRequests(options, report.FunctionCallingEnabled)

	for _, request := range requests {
		requestResult, err := runKimiDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report.Model, request, output)
		result.Requests = append(result.Requests, requestResult)
		result.addRequestObservation(requestResult)
		if requestResult.Content != "" {
			result.Content = requestResult.Content
		}
		if request.ImagePayload {
			result.ImagePayload = true
			if !requestResult.PromptCacheKeyPresent {
				return result, fmt.Errorf("image smoke request did not include prompt_cache_key")
			}
		}
		if request.WebSearchPayload {
			result.WebSearchPayload = true
			if !requestResult.PromptCacheKeyPresent {
				return result, fmt.Errorf("web search smoke request did not include prompt_cache_key")
			}
		}
		if request.ToolPayload && err == nil {
			result.ToolPayload = true
			if !diagnosticSmokeContentHasToolCall(requestResult.Content) {
				return result, fmt.Errorf("tool smoke response did not include %s function_call", diagnosticSmokeToolName)
			}
		}
		if err != nil {
			return result, err
		}
	}

	if runTextSmoke {
		first := diagnosticSmokePromptCacheKey(result.Requests, kimiDiagnosticSmokeCacheFirstName)
		second := diagnosticSmokePromptCacheKey(result.Requests, kimiDiagnosticSmokeCacheSecondName)
		if first == "" || second == "" || first != second {
			return result, fmt.Errorf("session-aware prompt_cache_key mismatch: first=%q second=%q", first, second)
		}
	}

	return result, nil
}

func kimiDiagnosticSmokeRequests(options DiagnosticOptions, functionCallingEnabled bool) ([]kimiDiagnosticSmokeRequest, bool) {
	runTextSmoke := options.TextSmoke || options.ToolSmoke || (!options.ImageSmoke && !options.WebSearchSmoke)
	var requests []kimiDiagnosticSmokeRequest
	if runTextSmoke {
		requests = append(requests, kimiDiagnosticTextSmokeRequests()...)
	}
	if options.ImageSmoke {
		requests = append(requests, kimiDiagnosticImageSmokeRequest())
	}
	if options.WebSearchSmoke {
		requests = append(requests, kimiDiagnosticWebSearchSmokeRequest())
	}
	if options.ToolSmoke && functionCallingEnabled {
		requests = append(requests, kimiDiagnosticToolSmokeRequest())
	}
	return requests, runTextSmoke
}

func kimiDiagnosticTextSmokeRequests() []kimiDiagnosticSmokeRequest {
	return []kimiDiagnosticSmokeRequest{
		{
			Name:         kimiDiagnosticSmokeCacheFirstName,
			SystemPrompt: "Reply briefly.",
			UserContent:  "Reply with: xelyon kimi doctor cache one",
			Thinking:     false,
			SessionID:    "xelyon-kimi-doctor-cache",
		},
		{
			Name:         kimiDiagnosticSmokeCacheSecondName,
			SystemPrompt: "Reply briefly.",
			UserContent:  "Reply with: xelyon kimi doctor cache two",
			Thinking:     false,
			SessionID:    "xelyon-kimi-doctor-cache",
		},
		{
			Name:         kimiDiagnosticSmokeThinkingName,
			SystemPrompt: "Think briefly, then reply briefly.",
			UserContent:  "Reply with: xelyon kimi doctor thinking ok",
			Thinking:     true,
			SessionID:    "xelyon-kimi-doctor-thinking",
		},
	}
}

func kimiDiagnosticImageSmokeRequest() kimiDiagnosticSmokeRequest {
	return kimiDiagnosticSmokeRequest{
		Name:         kimiDiagnosticSmokeImageName,
		SystemPrompt: "Reply briefly.",
		UserContent:  "Look at the attached tiny diagnostic image and reply with a short non-empty response.",
		Thinking:     false,
		SessionID:    "xelyon-kimi-doctor-image",
		ImagePayload: true,
	}
}

func kimiDiagnosticWebSearchSmokeRequest() kimiDiagnosticSmokeRequest {
	return kimiDiagnosticSmokeRequest{
		Name:             kimiDiagnosticSmokeWebSearchName,
		SystemPrompt:     "Use web search and reply briefly.",
		UserContent:      "Search the web for Moonshot AI Kimi API web search pricing and reply with one short non-empty summary.",
		Thinking:         false,
		SessionID:        "xelyon-kimi-doctor-web-search",
		WebSearchPayload: true,
	}
}

func kimiDiagnosticToolSmokeRequest() kimiDiagnosticSmokeRequest {
	return kimiDiagnosticSmokeRequest{
		Name:         kimiDiagnosticSmokeToolName,
		SystemPrompt: "Use the diagnostic tool.",
		UserContent:  `Call xelyon_kimi_doctor_probe exactly once with {"value":"kimi-tool-ok"} and do not answer in prose.`,
		Thinking:     false,
		SessionID:    "xelyon-kimi-doctor-tool",
		ToolPayload:  true,
	}
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
