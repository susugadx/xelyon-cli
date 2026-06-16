package claude

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
	defaultClaudeDiagnosticSmokeTimeout         = 120 * time.Second
	defaultClaudeDiagnosticSmokeMaxOutputTokens = 64
)

func runClaudeDiagnosticSmoke(ctx context.Context, cfg *config.Config, report DiagnosticReport, options DiagnosticOptions) (DiagnosticSmokeResult, error) {
	timeout := options.SmokeTimeout
	if timeout <= 0 {
		timeout = defaultClaudeDiagnosticSmokeTimeout
	}
	maxOutputTokens := claudeDiagnosticRequestMaxOutputTokens(options)

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	smokeCfg := claudeDiagnosticPolicyConfig(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}

	provider := New(os.Getenv(anthropicAPIKeyEnv))
	provider.SetMCPTools(nil)
	defer provider.ClearToolChoice()

	result := DiagnosticSmokeResult{Ran: true}
	started := time.Now()
	plan := buildClaudeDiagnosticRequestPlan(options, report.FunctionCallingEnabled)
	for _, request := range plan.Requests {
		if request.skipped(report.FunctionCallingEnabled) {
			providerdiag.AddThinkingMultimodalSmokeRequestResult(&result, newClaudeDiagnosticSkippedToolSmokeRequest(request))
			continue
		}
		requestResult, err := runClaudeDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report, request, output)
		providerdiag.AddThinkingMultimodalSmokeRequestResult(&result, requestResult)
		if err != nil {
			result.Duration = time.Since(started).Round(time.Millisecond).String()
			return result, err
		}
	}
	result.Duration = time.Since(started).Round(time.Millisecond).String()
	return result, nil
}

func runClaudeDiagnosticSmokeRequest(
	ctx context.Context,
	cfg *config.Config,
	provider *Provider,
	report DiagnosticReport,
	request claudeDiagnosticRequest,
	output io.Writer,
) (DiagnosticSmokeRequestResult, error) {
	requestCtx := newClaudeDiagnosticRequestContext(ctx, cfg, request, output)

	var usage api.Usage
	usageObserved := false
	provider.SetUsageCallback(func(observed api.Usage) {
		usage.Add(observed)
		usageObserved = usageObserved || observed.HasTokenObservation()
	})

	started := time.Now()
	content, err := runClaudeDiagnosticSmokePayload(requestCtx, provider, report, request)
	elapsed := time.Since(started).Round(time.Millisecond)
	costEstimate := cost.EstimateRequestCostWithCacheForConfig(cfg, "claude", report.Model, usage)

	result := request.smokeBase()
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
		if !claudeDiagnosticSmokeContentHasToolCall(content) {
			result.Error = fmt.Sprintf("tool smoke response did not include %s function_call", claudeDiagnosticToolName)
			return result, errors.New(result.Error)
		}
		return result, nil
	}
	if request.WebSearchPayload {
		if !claudeDiagnosticWebSearchContentHasResult(content) {
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

func runClaudeDiagnosticSmokePayload(ctx context.Context, provider *Provider, report DiagnosticReport, request claudeDiagnosticRequest) (string, error) {
	applyClaudeDiagnosticToolChoice(ctx, provider, request, report.CatalogModel)

	switch {
	case request.ImagePayload:
		return provider.ChatWithImage(ctx, request.SystemPrompt, nil, request.UserContent, claudeDiagnosticImage(), report.Model)
	case request.WebSearchPayload:
		return provider.webSearch(ctx, request.UserContent, report.Model)
	default:
		return provider.ChatWithTools(ctx, request.SystemPrompt, []api.Message{{Role: "user", Content: request.UserContent}}, report.Model)
	}
}

func claudeDiagnosticSmokeContentHasToolCall(content string) bool {
	return providerdiag.ContentHasToolCall(content, claudeDiagnosticToolName)
}

func claudeDiagnosticWebSearchContentHasResult(content string) bool {
	content = strings.TrimSpace(content)
	return content != "" && content != "No results found." && (strings.Contains(content, "Summary:") || strings.Contains(content, "Sources:"))
}

func claudeDiagnosticImage() *api.ImageData {
	return &api.ImageData{
		MediaType: "image/png",
		Base64:    claudeDiagnosticPNGBase64,
	}
}

const claudeDiagnosticPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII="
