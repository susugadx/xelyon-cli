package groq

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
	defaultGroqDiagnosticSmokeTimeout         = 120 * time.Second
	defaultGroqDiagnosticSmokeMaxOutputTokens = 64
	groqDiagnosticSmokeToolName               = "xelyon_groq_doctor_probe"
)

type groqDiagnosticSmokeRequest = providerdiag.TextToolSmokeRequest

func runGroqDiagnosticSmoke(ctx context.Context, cfg *config.Config, report DiagnosticReport, options DiagnosticOptions) (DiagnosticSmokeResult, error) {
	timeout := options.SmokeTimeout
	if timeout <= 0 {
		timeout = defaultGroqDiagnosticSmokeTimeout
	}
	maxOutputTokens := options.MaxOutputTokens
	if maxOutputTokens <= 0 {
		maxOutputTokens = defaultGroqDiagnosticSmokeMaxOutputTokens
	}

	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	smokeCfg := groqDiagnosticPolicyConfig(cfg, report.Model, report.CatalogModel, maxOutputTokens)
	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}

	provider := New(os.Getenv(groqAPIKeyEnv))
	provider.SetMCPTools(nil)
	defer provider.ClearToolChoice()

	result := DiagnosticSmokeResult{Ran: true, Route: report.Route}
	started := time.Now()
	for _, request := range groqDiagnosticSmokeRequests(options, report.FunctionCallingEnabled) {
		if request.ToolPayload && !report.FunctionCallingEnabled {
			providerdiag.AddTextToolSmokeRequestResult(
				&result,
				providerdiag.NewSkippedTextToolSmokeRequest(request, report.Route, groqDiagnosticDisabledToolSkipReason()),
			)
			continue
		}

		requestResult, err := runGroqDiagnosticSmokeRequest(smokeCtx, smokeCfg, provider, report, request, output)
		providerdiag.AddTextToolSmokeRequestResult(&result, requestResult)
		if err != nil {
			result.Duration = time.Since(started).Round(time.Millisecond).String()
			return result, err
		}
	}
	result.Duration = time.Since(started).Round(time.Millisecond).String()
	return result, nil
}

func groqDiagnosticSmokeRequests(options DiagnosticOptions, functionCallingEnabled bool) []groqDiagnosticSmokeRequest {
	return providerdiag.TextToolSmokeRequests(providerdiag.TextToolSmokeRequestOptions{
		TextSmoke:              options.TextSmoke,
		ToolSmoke:              options.ToolSmoke,
		FunctionCallingEnabled: functionCallingEnabled,
		ProviderSlug:           "groq",
		ToolName:               groqDiagnosticSmokeToolName,
		ToolExpectedValue:      "groq-tool-ok",
	})
}

func groqDiagnosticDisabledToolSkipReason() string {
	return fmt.Sprintf("Groq function calling payloads are disabled (%s=0)", groqFunctionCallingEnv)
}

func runGroqDiagnosticSmokeRequest(
	ctx context.Context,
	cfg *config.Config,
	provider *Provider,
	report DiagnosticReport,
	request groqDiagnosticSmokeRequest,
	output io.Writer,
) (DiagnosticSmokeRequestResult, error) {
	requestCtx := newGroqDiagnosticSmokeRequestContext(ctx, cfg, request, output)
	applyGroqDiagnosticToolChoice(provider, request)

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
	costEstimate := cost.EstimateRequestCostWithCacheForConfig(cfg, "groq", report.Model, usage)

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
		if !providerdiag.ContentHasToolCall(content, groqDiagnosticSmokeToolName) {
			result.Error = fmt.Sprintf("tool smoke response did not include %s function_call", groqDiagnosticSmokeToolName)
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

func newGroqDiagnosticSmokeRequestContext(ctx context.Context, cfg *config.Config, request groqDiagnosticSmokeRequest, output io.Writer) context.Context {
	return providerdiag.NewTextToolSmokeRequestContext(
		ctx,
		cfg,
		request,
		providerdiag.NoopDiagnosticToolDefinitions(groqDiagnosticSmokeToolName, "Groq"),
		output,
	)
}

func applyGroqDiagnosticToolChoice(provider *Provider, request groqDiagnosticSmokeRequest) {
	if request.ToolPayload {
		provider.SetToolChoice(groqDiagnosticSmokeToolName)
		return
	}
	provider.ClearToolChoice()
}

func groqSmokeErrorIsToolFailure(smoke DiagnosticSmokeResult) bool {
	for _, request := range smoke.Requests {
		if request.ToolPayload && request.Ran && strings.TrimSpace(request.Error) != "" {
			return true
		}
	}
	return false
}
