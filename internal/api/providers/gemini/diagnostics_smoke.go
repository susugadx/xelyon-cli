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
	timeout := geminiDiagnosticSmokeTimeout(options)
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultGeminiDiagnosticSmokeMaxOutputTokens
	}

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
		requestResult, err := runGeminiDiagnosticSmokeRequestWithTimeout(ctx, timeout, smokeCfg, provider, report, request, output)
		providerdiag.AddMultimodalSmokeRequestResult(&result, requestResult)
		if err != nil {
			result.Duration = time.Since(started).Round(time.Millisecond).String()
			return result, err
		}
	}
	result.Duration = time.Since(started).Round(time.Millisecond).String()
	return result, nil
}

func geminiDiagnosticSmokeTimeout(options DiagnosticOptions) time.Duration {
	if options.SmokeTimeout > 0 {
		return options.SmokeTimeout
	}
	return defaultGeminiDiagnosticSmokeTimeout
}

func runGeminiDiagnosticSmokeRequestWithTimeout(
	ctx context.Context,
	timeout time.Duration,
	cfg *config.Config,
	provider *Provider,
	report DiagnosticReport,
	request geminiDiagnosticRequest,
	output io.Writer,
) (DiagnosticSmokeRequestResult, error) {
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return runGeminiDiagnosticSmokeRequest(requestCtx, cfg, provider, report, request, output)
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

	result := providerdiag.NewMultimodalSmokeRequestResult(request.multimodalSmokeRequest())
	result.Ran = true
	result.Content = strings.TrimSpace(content)
	result.Duration = elapsed.String()
	result.UsageObserved = usageObserved
	result.Usage = providerdiag.SmokeUsageFromAPIUsage(usage)
	result.Cost = providerdiag.SmokeCostFromEstimate(costEstimate)
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
