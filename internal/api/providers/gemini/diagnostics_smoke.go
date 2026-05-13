package gemini

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

const (
	defaultGeminiDiagnosticSmokeTimeout         = 120 * time.Second
	defaultGeminiDiagnosticSmokeMaxOutputTokens = 64
	geminiDiagnosticToolName                    = "xelyon_gemini_doctor_probe"
)

func runGeminiDiagnosticSmoke(ctx context.Context, cfg *config.Config, report DiagnosticReport, options DiagnosticOptions) (DiagnosticSmokeResult, error) {
	timeout := options.SmokeTimeout
	if timeout <= 0 {
		timeout = defaultGeminiDiagnosticSmokeTimeout
	}
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultGeminiDiagnosticSmokeMaxOutputTokens
	}

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	smokeCfg := geminiDiagnosticPolicyConfig(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}

	provider := New(os.Getenv(geminiAPIKeyEnv))
	provider.SetMCPTools(nil)

	result := DiagnosticSmokeResult{Ran: true}
	started := time.Now()
	for _, request := range geminiDiagnosticRequests(options) {
		requestResult, err := runGeminiDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report, request, output)
		result.Requests = append(result.Requests, requestResult)
		result.addRequestObservation(requestResult)
		if err != nil {
			result.Duration = time.Since(started).Round(time.Millisecond).String()
			return result, err
		}
	}
	result.Duration = time.Since(started).Round(time.Millisecond).String()
	return result, nil
}

func runGeminiDiagnosticSmokeRequest(
	ctx context.Context,
	cfg *config.Config,
	provider *Provider,
	report DiagnosticReport,
	request geminiDiagnosticRequest,
	output io.Writer,
) (DiagnosticSmokeRequestResult, error) {
	requestCtx := newGeminiDiagnosticRequestContext(ctx, cfg, request, output)

	var usage api.Usage
	usageObserved := false
	provider.SetUsageCallback(func(observed api.Usage) {
		usage.Add(observed)
		usageObserved = usageObserved || observed.HasTokenObservation()
	})

	started := time.Now()
	content, err := runGeminiDiagnosticSmokePayload(requestCtx, provider, report, request)
	elapsed := time.Since(started).Round(time.Millisecond)
	costEstimate := cost.EstimateRequestCostWithCacheForConfig(cfg, "gemini", report.Model, usage)

	result := DiagnosticSmokeRequestResult{
		Name:             request.Name,
		Ran:              true,
		ToolPayload:      request.ToolPayload,
		ImagePayload:     request.ImagePayload,
		WebSearchPayload: request.WebSearchPayload,
		Route:            geminiDiagnosticRequestRoute(request),
		Content:          strings.TrimSpace(content),
		Duration:         elapsed.String(),
		UsageObserved:    usageObserved,
		Usage:            providerdiag.SmokeUsageFromAPIUsage(usage),
		Cost:             providerdiag.SmokeCostFromEstimate(costEstimate),
	}
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	if request.ToolPayload {
		if !geminiDiagnosticSmokeContentHasToolCall(content) {
			result.Error = fmt.Sprintf("tool smoke response did not include %s function_call", geminiDiagnosticToolName)
			return result, errors.New(result.Error)
		}
		return result, nil
	}
	if request.WebSearchPayload {
		if !geminiDiagnosticWebSearchContentHasResult(content) {
			result.Error = "web search smoke response did not include summary or sources"
			return result, errors.New(result.Error)
		}
		return result, nil
	}
	if strings.TrimSpace(content) == "" {
		result.Error = fmt.Sprintf("%s smoke response content is empty", request.Name)
		return result, errors.New(result.Error)
	}
	return result, nil
}

func runGeminiDiagnosticSmokePayload(ctx context.Context, provider *Provider, report DiagnosticReport, request geminiDiagnosticRequest) (string, error) {
	switch {
	case request.ImagePayload:
		return provider.ChatWithImage(ctx, request.SystemPrompt, nil, request.UserContent, geminiDiagnosticImage(), report.Model)
	case request.WebSearchPayload:
		return provider.webSearch(ctx, request.UserContent, report.Model)
	default:
		return provider.ChatWithTools(ctx, request.SystemPrompt, []api.Message{{Role: "user", Content: request.UserContent}}, report.Model)
	}
}

func (r *DiagnosticSmokeResult) addRequestObservation(request DiagnosticSmokeRequestResult) {
	if request.ToolPayload {
		r.ToolPayload = true
	}
	if request.ImagePayload {
		r.ImagePayload = true
	}
	if request.WebSearchPayload {
		r.WebSearchPayload = true
	}
	switch {
	case r.Route == "":
		r.Route = request.Route
	case r.Route != request.Route:
		r.Route = "mixed"
	}
	if strings.TrimSpace(r.Content) == "" {
		r.Content = request.Content
	}
	r.Usage = providerdiag.AddSmokeUsage(r.Usage, request.Usage)
	r.Cost = providerdiag.AddSmokeCost(r.Cost, request.Cost)
	r.UsageObserved = r.UsageObserved || request.UsageObserved
}

func geminiSmokeErrorIsToolFailure(smoke DiagnosticSmokeResult) bool {
	for _, request := range smoke.Requests {
		if request.ToolPayload && request.Ran && strings.TrimSpace(request.Error) != "" {
			return true
		}
	}
	return false
}

func geminiSmokeErrorIsWebSearchFailure(smoke DiagnosticSmokeResult) bool {
	for _, request := range smoke.Requests {
		if request.WebSearchPayload && request.Ran && strings.TrimSpace(request.Error) != "" {
			return true
		}
	}
	return false
}

func geminiDiagnosticSmokeContentHasToolCall(content string) bool {
	return providerdiag.ContentHasToolCall(content, geminiDiagnosticToolName)
}

func geminiDiagnosticWebSearchContentHasResult(content string) bool {
	content = strings.TrimSpace(content)
	return content != "" && content != "No results found." && (strings.Contains(content, "Summary:") || strings.Contains(content, "Sources:"))
}

func geminiDiagnosticImage() *api.ImageData {
	return &api.ImageData{
		MediaType: "image/png",
		Base64:    geminiDiagnosticPNGBase64,
	}
}

const geminiDiagnosticPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII="
