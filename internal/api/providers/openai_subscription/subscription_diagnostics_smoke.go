package openaisubscription

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func (r *SubscriptionDiagnosticReport) runSubscriptionSmokeIfReady(ctx context.Context, cfg *config.Config, options SubscriptionDiagnosticOptions) {
	if failures := subscriptionDiagnosticSmokeReadinessFailures(r.Checks, options); len(failures) > 0 {
		r.addCheck(DiagnosticStatusWarn, "smoke", "subscription smoke skipped because readiness checks failed", "failed checks: "+strings.Join(failures, ", "), "Fix readiness checks before running --smoke")
		return
	}
	result, err := runSubscriptionDiagnosticSmoke(ctx, cfg, *r, options)
	r.Smoke = &result
	if options.CompactSmoke {
		r.addSubscriptionCompactSmokeResultCheck(result)
	}
	if err != nil {
		r.addCheck(DiagnosticStatusFail, "smoke", "subscription smoke failed", err.Error(), "")
		return
	}
	r.addCheck(DiagnosticStatusOK, "smoke", "subscription smoke passed", fmt.Sprintf("requests=%d", len(result.Requests)), "")
}

func subscriptionDiagnosticSmokeReadinessFailures(checks []DiagnosticCheck, options SubscriptionDiagnosticOptions) []string {
	blockingChecks := map[string]bool{
		"auth":                  true,
		"endpoint":              true,
		"originator":            true,
		"provider_registration": true,
		"model":                 true,
	}
	if options.CompactSmoke {
		blockingChecks["compact_api"] = true
	}
	var failures []string
	for _, check := range checks {
		if check.Status == DiagnosticStatusFail && blockingChecks[check.Name] {
			failures = append(failures, check.Name)
		}
	}
	sort.Strings(failures)
	return failures
}

func (r *SubscriptionDiagnosticReport) addSubscriptionCompactSmokeResultCheck(result SubscriptionDiagnosticSmokeResult) {
	if r == nil {
		return
	}
	for _, request := range result.Requests {
		if !request.CompactPayload {
			continue
		}
		if request.Skipped {
			r.addCheck(DiagnosticStatusWarn, "compact_smoke", "subscription Compact API smoke skipped", request.SkipReason, "")
			return
		}
		if strings.TrimSpace(request.Error) != "" {
			r.addCheck(DiagnosticStatusWarn, "compact_smoke", "subscription Compact API is unsupported or returned an incompatible response", request.Error, "")
			return
		}
		r.addCheck(DiagnosticStatusOK, "compact_smoke", "subscription Compact API smoke passed", request.Content, "")
		return
	}
}

func runSubscriptionDiagnosticSmoke(ctx context.Context, cfg *config.Config, report SubscriptionDiagnosticReport, options SubscriptionDiagnosticOptions) (SubscriptionDiagnosticSmokeResult, error) {
	timeout := options.SmokeTimeout
	if timeout <= 0 {
		timeout = defaultSubscriptionDiagnosticSmokeTimeout
	}
	smokeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	baseCfg := subscriptionDiagnosticSmokeConfig(cfg, report.Model, options)
	output := options.SmokeOutput
	if output == nil {
		output = io.Discard
	}
	provider := NewSubscription()
	result := SubscriptionDiagnosticSmokeResult{Ran: true}
	started := time.Now()
	for _, request := range subscriptionDiagnosticSmokeRequests(options) {
		if request.CompactPayload {
			requestResult, err := runSubscriptionDiagnosticCompactSmokeRequest(smokeCtx, provider, report, request)
			subscriptionDiagnosticAddSmokeRequest(&result, requestResult)
			if err != nil {
				result.Duration = time.Since(started).Round(time.Millisecond).String()
				return result, err
			}
			continue
		}
		requestResult, err := runSubscriptionDiagnosticSmokeRequest(smokeCtx, baseCfg, provider, report, request, output)
		subscriptionDiagnosticAddSmokeRequest(&result, requestResult)
		if err != nil {
			result.Duration = time.Since(started).Round(time.Millisecond).String()
			return result, err
		}
	}
	result.Duration = time.Since(started).Round(time.Millisecond).String()
	return result, nil
}

func runSubscriptionDiagnosticCompactSmokeRequest(ctx context.Context, provider *SubscriptionProvider, report SubscriptionDiagnosticReport, request subscriptionDiagnosticSmokeRequest) (SubscriptionDiagnosticSmokeRequestResult, error) {
	result := SubscriptionDiagnosticSmokeRequestResult{
		Name:           request.Name,
		CompactPayload: true,
		Route:          diagnosticRouteSubscriptionCompact,
		Cost:           subscriptionDiagnosticSmokeCost(),
	}
	compactEndpoint := strings.TrimSpace(DefaultSubscriptionAuthConfig().CompactEndpoint)
	if compactEndpoint == "" {
		result.Skipped = true
		result.SkipReason = "subscription Compact API endpoint is not configured"
		return result, nil
	}
	started := time.Now()
	response, err := runSubscriptionCompactProbe(ctx, provider, compactEndpoint, report.Model)
	result.Ran = true
	result.Duration = time.Since(started).Round(time.Millisecond).String()
	if err != nil {
		result.Error = RedactSubscriptionSecrets(err.Error())
		if subscriptionCompactEndpointForbidden(compactEndpoint) {
			return result, errors.New(result.Error)
		}
		return result, nil
	}
	result.Content = fmt.Sprintf("compact output items=%d", len(response.Output))
	if response.Usage != nil {
		result.UsageObserved = true
		result.Usage = providerdiag.SmokeUsage{
			InputTokens:  response.Usage.InputTokens,
			OutputTokens: response.Usage.OutputTokens,
		}
	}
	return result, nil
}

func runSubscriptionDiagnosticSmokeRequest(ctx context.Context, cfg *config.Config, provider *SubscriptionProvider, report SubscriptionDiagnosticReport, request subscriptionDiagnosticSmokeRequest, output io.Writer) (SubscriptionDiagnosticSmokeRequestResult, error) {
	requestCtx := newSubscriptionDiagnosticSmokeRequestContext(ctx, cfg, request, output)
	configureSubscriptionDiagnosticProviderForRequest(provider, request)
	var usage api.Usage
	usageObserved := false
	provider.SetUsageCallback(func(observed api.Usage) {
		usage.Add(observed)
		usageObserved = usageObserved || observed.HasTokenObservation()
	})
	observedRequests, restoreObserver := observeSubscriptionDiagnosticResponsesRequests(provider)
	defer restoreObserver()

	started := time.Now()
	content, err := provider.ChatWithTools(requestCtx, request.SystemPrompt, []api.Message{{Role: "user", Content: request.UserContent}}, report.Model)
	if err == nil && request.ToolPayload {
		content, err = runSubscriptionDiagnosticToolContinuation(requestCtx, provider, request, report.Model, content)
	}
	elapsed := time.Since(started).Round(time.Millisecond)
	observed := observedRequests()
	result := SubscriptionDiagnosticSmokeRequestResult{
		Name:             request.Name,
		Ran:              true,
		ToolPayload:      request.ToolPayload,
		RetentionPayload: request.RetentionPayload,
		CachePayload:     request.CachePayload,
		ThinkingPayload:  request.ThinkingPayload,
		Route:            report.Route,
		Content:          strings.TrimSpace(content),
		Duration:         elapsed.String(),
		UsageObserved:    usageObserved,
		Usage:            providerdiag.SmokeUsageFromAPIUsage(usage),
		Cost:             subscriptionDiagnosticSmokeCost(),
	}
	if err != nil {
		result.Error = RedactSubscriptionSecrets(err.Error())
		return result, errors.New(result.Error)
	}
	wantReasoning := subscriptionResponsesReasoningConfig(requestCtx, openairesponses.NewModelIdentity(report.Model, report.CatalogModel)) != nil
	if err := validateSubscriptionDiagnosticObservedRequest(request, observed, result, wantReasoning); err != nil {
		result.Error = err.Error()
		return result, err
	}
	return result, nil
}

func runSubscriptionDiagnosticToolContinuation(ctx context.Context, provider *SubscriptionProvider, request subscriptionDiagnosticSmokeRequest, model, firstContent string) (string, error) {
	calls := tools.ParseToolCalls(firstContent)
	if len(calls) == 0 {
		return firstContent, fmt.Errorf("tool smoke response did not include %s function_call", subscriptionDiagnosticSmokeToolName)
	}
	call := calls[0]
	assistant := api.Message{
		Role: "assistant",
		ToolCalls: []api.OpenAIToolCall{{
			ID:   call.ID,
			Type: "function",
			Function: api.OpenAIToolCallFunction{
				Name:      call.Tool,
				Arguments: toolruntime.ArgsToJSON(call.RawArgs),
			},
		}},
	}
	toolResult := api.Message{Role: "tool", ToolCallID: call.ID, ToolName: call.Tool, Content: `{"value":"subscription-tool-ok"}`}
	provider.ClearToolChoice()
	return provider.ChatWithTools(ctx, "Reply briefly after the diagnostic tool result.", []api.Message{
		{Role: "user", Content: request.UserContent},
		assistant,
		toolResult,
	}, model)
}

func validateSubscriptionDiagnosticObservedRequest(request subscriptionDiagnosticSmokeRequest, observed []ResponsesRequest, result SubscriptionDiagnosticSmokeRequestResult, wantReasoning bool) error {
	if len(observed) == 0 {
		return fmt.Errorf("%s smoke did not build a Responses request", request.Name)
	}
	for i, req := range observed {
		if req.Store {
			return fmt.Errorf("%s smoke request[%d] set store=true", request.Name, i)
		}
		if strings.TrimSpace(req.PreviousResponseID) != "" {
			return fmt.Errorf("%s smoke request[%d] sent previous_response_id", request.Name, i)
		}
		if len(req.ContextManagement) != 0 {
			return fmt.Errorf("%s smoke request[%d] sent context_management", request.Name, i)
		}
		if strings.TrimSpace(req.PromptCacheKey) == "" {
			return fmt.Errorf("%s smoke request[%d] did not send prompt_cache_key", request.Name, i)
		}
	}
	if request.ThinkingPayload {
		reasoning := observed[0].Reasoning
		if wantReasoning && reasoning == nil {
			return fmt.Errorf("%s smoke did not send requested reasoning payload", request.Name)
		}
		if !wantReasoning && reasoning != nil {
			return fmt.Errorf("%s smoke sent unexpected reasoning payload", request.Name)
		}
	}
	if strings.TrimSpace(result.Content) == "" && !request.ToolPayload {
		return fmt.Errorf("%s smoke response content is empty", request.Name)
	}
	return nil
}

func observeSubscriptionDiagnosticResponsesRequests(provider *SubscriptionProvider) (func() []ResponsesRequest, func()) {
	if provider == nil {
		return func() []ResponsesRequest { return nil }, func() {}
	}
	previousObserver := provider.responsesRequestObserver
	observed := make([]ResponsesRequest, 0, 2)
	provider.responsesRequestObserver = func(request ResponsesRequest) {
		observed = append(observed, request)
		if previousObserver != nil {
			previousObserver(request)
		}
	}
	return func() []ResponsesRequest { return append([]ResponsesRequest(nil), observed...) }, func() {
		provider.responsesRequestObserver = previousObserver
	}
}

func subscriptionDiagnosticSmokeRequests(options SubscriptionDiagnosticOptions) []subscriptionDiagnosticSmokeRequest {
	textSmoke := options.TextSmoke || (!options.ToolSmoke && !options.RetentionSmoke && !options.CacheSmoke && !options.CompactSmoke && !options.ThinkingSmoke)
	var requests []subscriptionDiagnosticSmokeRequest
	if textSmoke {
		requests = append(requests, subscriptionDiagnosticSmokeRequest{Name: "text", SystemPrompt: "Reply briefly.", UserContent: "Reply with: xelyon openai subscription doctor ok"})
	}
	if options.ToolSmoke {
		requests = append(requests, subscriptionDiagnosticSmokeRequest{Name: "tool", SystemPrompt: "Use the diagnostic tool.", UserContent: `Call xelyon_openai_subscription_doctor_probe exactly once with {"value":"subscription-tool-ok"} and do not answer in prose.`, ToolPayload: true})
	}
	if options.RetentionSmoke {
		requests = append(requests, subscriptionDiagnosticSmokeRequest{Name: "retention", SystemPrompt: "Reply briefly.", UserContent: "Reply with: xelyon subscription full payload retention ok", RetentionPayload: true})
	}
	if options.CacheSmoke {
		requests = append(requests, subscriptionDiagnosticSmokeRequest{Name: "cache", SystemPrompt: "Reply briefly.", UserContent: "Reply with: xelyon subscription cache ok", CachePayload: true})
	}
	if options.ThinkingSmoke {
		requests = append(requests, subscriptionDiagnosticSmokeRequest{Name: "thinking", SystemPrompt: "Reply briefly.", UserContent: "Reply with: xelyon subscription thinking ok", ThinkingPayload: true})
	}
	if options.CompactSmoke {
		requests = append(requests, subscriptionDiagnosticSmokeRequest{Name: "compact", CompactPayload: true})
	}
	return requests
}

func subscriptionDiagnosticSmokeConfig(cfg *config.Config, model string, options SubscriptionDiagnosticOptions) *config.Config {
	out := config.CloneConfig(cfg)
	if out == nil {
		out = config.DefaultConfig()
	}
	out.Responses.Store = false
	out.Responses.PersistResponseID = false
	pCfg, _ := out.GetProviderModelConfig(subscriptionProviderKey)
	pCfg.DefaultModel = model
	out.SetProviderModelConfig(subscriptionProviderKey, pCfg)
	return out
}

func newSubscriptionDiagnosticSmokeRequestContext(ctx context.Context, cfg *config.Config, request subscriptionDiagnosticSmokeRequest, output io.Writer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		output = io.Discard
	}
	requestCtx := ui.WithRuntime(ctx, ui.NewRuntime(strings.NewReader(""), output, output))
	requestCtx = api.WithAssistantUpdateMode(requestCtx, api.AssistantUpdatesOff)
	if request.ToolPayload {
		requestCtx = api.WithToolDefinitions(requestCtx, providerdiag.NoopDiagnosticToolDefinitions(subscriptionDiagnosticSmokeToolName, subscriptionDisplayName))
	} else {
		requestCtx = api.WithToolDefinitions(requestCtx, nil)
		requestCtx = api.WithToolUseDisabled(requestCtx)
	}
	return config.WithContext(requestCtx, cfg)
}

func configureSubscriptionDiagnosticProviderForRequest(provider *SubscriptionProvider, request subscriptionDiagnosticSmokeRequest) {
	if provider == nil {
		return
	}
	if request.ToolPayload {
		provider.SetMCPTools(providerdiag.NoopDiagnosticToolDefinitions(subscriptionDiagnosticSmokeToolName, subscriptionDisplayName))
		provider.SetToolChoice(subscriptionDiagnosticSmokeToolName)
		return
	}
	provider.SetMCPTools(nil)
	provider.ClearToolChoice()
}

func subscriptionDiagnosticPreviewHeaders(maskedAccount, originator string) map[string]string {
	headers := providerdiag.JSONHeaders()
	headers["Authorization"] = "Bearer <redacted>"
	if strings.TrimSpace(maskedAccount) != "" {
		headers["ChatGPT-Account-Id"] = "<redacted>"
	}
	headers["originator"] = strings.TrimSpace(originator)
	headers["User-Agent"] = "xelyon/<version> (<os> <arch>)"
	return headers
}

func subscriptionDiagnosticPreviewBody(body ResponsesRequest) map[string]any {
	preview := map[string]any{
		"model":                        body.Model,
		"stream":                       body.Stream,
		"store":                        body.Store,
		"instructions":                 presenceLabel(body.Instructions),
		"input_items":                  subscriptionDiagnosticInputShape(body.Input),
		"tools_count":                  len(body.Tools),
		"prompt_cache_key":             presenceLabel(body.PromptCacheKey),
		"prompt_cache_retention":       presenceLabel(body.PromptCacheRetention),
		"previous_response_id_present": strings.TrimSpace(body.PreviousResponseID) != "",
		"context_management_count":     len(body.ContextManagement),
		"max_output_tokens":            subscriptionDiagnosticMaxOutputTokensPreview(body.MaxOutputTokens),
	}
	if body.Reasoning != nil {
		preview["reasoning_effort"] = body.Reasoning.Effort
	}
	if body.ToolChoice != nil {
		preview["tool_choice"] = body.ToolChoice
	}
	return preview
}

func subscriptionDiagnosticMaxOutputTokensPreview(value int) any {
	if value <= 0 {
		return "omitted"
	}
	return value
}

func subscriptionDiagnosticCompactPreviewBody(model string) map[string]any {
	return map[string]any{
		"model":        model,
		"instructions": "present",
		"input_items":  subscriptionDiagnosticInputShape(subscriptionCompactProbeInput()),
	}
}

func subscriptionDiagnosticInputShape(input any) []map[string]any {
	items, ok := input.([]api.InputItem)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]any{
			"type":      item.Type,
			"role":      item.Role,
			"call_id":   presenceLabel(item.CallID),
			"name":      presenceLabel(item.Name),
			"content":   presenceLabel(item.Content),
			"output":    presenceLabel(item.Output),
			"encrypted": presenceLabel(item.EncryptedContent),
		})
	}
	return out
}

func presenceLabel(value any) string {
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return "omitted"
		}
		return "present"
	case nil:
		return "omitted"
	default:
		return "present"
	}
}

func subscriptionDiagnosticSmokeCost() providerdiag.SmokeCost {
	return providerdiag.SmokeCost{PricingUnavailable: true}
}

func subscriptionDiagnosticAddSmokeRequest(result *SubscriptionDiagnosticSmokeResult, request SubscriptionDiagnosticSmokeRequestResult) {
	if result == nil {
		return
	}
	result.Requests = append(result.Requests, request)
	if request.Skipped {
		return
	}
	if strings.TrimSpace(result.Route) == "" {
		result.Route = request.Route
	} else if result.Route != request.Route {
		result.Route = "mixed"
	}
	if request.ToolPayload {
		result.ToolPayload = true
	}
	if request.RetentionPayload {
		result.RetentionPayload = true
	}
	if request.CachePayload {
		result.CachePayload = true
	}
	if request.ThinkingPayload {
		result.ThinkingPayload = true
	}
	if request.CompactPayload {
		result.CompactPayload = true
	}
	if request.Content != "" {
		result.Content = request.Content
	}
	result.Usage = providerdiag.AddSmokeUsage(result.Usage, request.Usage)
	result.UsageObserved = result.UsageObserved || request.UsageObserved
	result.Cost = subscriptionDiagnosticSmokeCost()
}
