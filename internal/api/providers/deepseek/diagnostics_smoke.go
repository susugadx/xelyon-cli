package deepseek

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
	defaultDeepSeekDiagnosticSmokeTimeout         = 120 * time.Second
	defaultDeepSeekDiagnosticSmokeMaxOutputTokens = 64
	deepSeekDiagnosticSmokeToolName               = "xelyon_deepseek_doctor_probe"
)

type deepSeekDiagnosticSmokeRequest = providerdiag.TextToolSmokeRequest

func runDeepSeekDiagnosticSmoke(ctx context.Context, cfg *config.Config, report DiagnosticReport, options DiagnosticOptions) (DiagnosticSmokeResult, error) {
	timeout := options.SmokeTimeout
	if timeout <= 0 {
		timeout = defaultDeepSeekDiagnosticSmokeTimeout
	}
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultDeepSeekDiagnosticSmokeMaxOutputTokens
	}

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	smokeCfg := deepSeekDiagnosticPolicyConfig(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}

	provider := New(os.Getenv(deepSeekAPIKeyEnv))
	provider.SetMCPTools(nil)
	defer provider.ClearToolChoice()

	result := DiagnosticSmokeResult{Ran: true, Route: report.Route}
	started := time.Now()
	for _, request := range deepSeekDiagnosticSmokeRequests(options, report.FunctionCallingEnabled) {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			providerdiag.AddTextToolSmokeRequestResult(
				&result,
				providerdiag.NewSkippedTextToolSmokeRequest(request, report.Route, deepSeekDiagnosticDisabledToolSkipReason()),
			)
			continue
		}

		requestResult, err := runDeepSeekDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report, request, output)
		providerdiag.AddTextToolSmokeRequestResult(&result, requestResult)
		if err != nil {
			result.Duration = time.Since(started).Round(time.Millisecond).String()
			return result, err
		}
	}
	result.Duration = time.Since(started).Round(time.Millisecond).String()
	return result, nil
}

func deepSeekDiagnosticSmokeRequests(options DiagnosticOptions, functionCallingEnabled bool) []deepSeekDiagnosticSmokeRequest {
	return providerdiag.TextToolSmokeRequests(providerdiag.TextToolSmokeRequestOptions{
		TextSmoke:              options.TextSmoke,
		ToolSmoke:              options.ToolSmoke,
		FunctionCallingEnabled: functionCallingEnabled,
		ProviderSlug:           "deepseek",
		ToolName:               deepSeekDiagnosticSmokeToolName,
		ToolExpectedValue:      "deepseek-tool-ok",
	})
}

func deepSeekDiagnosticDisabledToolSkipReason() string {
	return fmt.Sprintf("DeepSeek function calling payloads are disabled (%s=0)", deepSeekFunctionCallingEnv)
}

func runDeepSeekDiagnosticSmokeRequest(
	ctx context.Context,
	cfg *config.Config,
	provider *Provider,
	report DiagnosticReport,
	request deepSeekDiagnosticSmokeRequest,
	output io.Writer,
) (DiagnosticSmokeRequestResult, error) {
	requestCtx := newDeepSeekDiagnosticSmokeRequestContext(ctx, cfg, request, output)
	applyDeepSeekDiagnosticToolChoice(provider, request)

	var usage api.Usage
	usageObserved := false
	provider.SetUsageCallback(func(observed api.Usage) {
		usage.Add(observed)
		usageObserved = usageObserved || observed.HasTokenObservation()
	})

	started := time.Now()
	content, err := provider.ChatWithTools(
		requestCtx,
		request.SystemPrompt,
		[]api.Message{{Role: "user", Content: request.UserContent}},
		report.Model,
	)
	elapsed := time.Since(started).Round(time.Millisecond)
	costEstimate := cost.EstimateRequestCostWithCacheForConfig(cfg, "deepseek", report.Model, usage)

	result := DiagnosticSmokeRequestResult{
		Name:          request.Name,
		Ran:           true,
		ToolPayload:   request.ToolPayload,
		Route:         report.Route,
		Content:       strings.TrimSpace(content),
		Duration:      elapsed.String(),
		UsageObserved: usageObserved,
		Usage:         providerdiag.SmokeUsageFromAPIUsage(usage),
		Cost:          providerdiag.SmokeCostFromEstimate(costEstimate),
	}
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	if request.ToolPayload {
		if !providerdiag.ContentHasToolCall(content, deepSeekDiagnosticSmokeToolName) {
			result.Error = fmt.Sprintf("tool smoke response did not include %s function_call", deepSeekDiagnosticSmokeToolName)
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

func newDeepSeekDiagnosticSmokeRequestContext(ctx context.Context, cfg *config.Config, request deepSeekDiagnosticSmokeRequest, output io.Writer) context.Context {
	return providerdiag.NewTextToolSmokeRequestContext(
		ctx,
		cfg,
		request,
		providerdiag.NoopDiagnosticToolDefinitions(deepSeekDiagnosticSmokeToolName, "DeepSeek"),
		output,
	)
}

func applyDeepSeekDiagnosticToolChoice(provider *Provider, request deepSeekDiagnosticSmokeRequest) {
	if request.ToolPayload {
		provider.SetToolChoice(deepSeekDiagnosticSmokeToolName)
		return
	}
	provider.ClearToolChoice()
}

func deepSeekSmokeErrorIsToolFailure(smoke DiagnosticSmokeResult) bool {
	for _, request := range smoke.Requests {
		if request.ToolPayload && request.Ran && strings.TrimSpace(request.Error) != "" {
			return true
		}
	}
	return false
}
