package openaisubscription

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

const (
	defaultSubscriptionDiagnosticSmokeTimeout = 120 * time.Second
	subscriptionDiagnosticSmokeToolName       = "xelyon_openai_subscription_doctor_probe"
)

type SubscriptionDiagnosticOptions struct {
	Config               *config.Config
	Model                string
	CatalogModel         string
	RunSmoke             bool
	TextSmoke            bool
	ToolSmoke            bool
	RetentionSmoke       bool
	CacheSmoke           bool
	CompactSmoke         bool
	ThinkingSmoke        bool
	WebSearchSmoke       bool
	Capabilities         bool
	RequiredCapabilities []string
	PrintRequest         bool
	SmokeTimeout         time.Duration
	SmokeOutput          io.Writer
}

func (o SubscriptionDiagnosticOptions) requiresAuthCheck() bool {
	return o.localCapabilityRequest().RequiresAuthCheck()
}

func (o SubscriptionDiagnosticOptions) requiresEndpointCheck() bool {
	return o.localCapabilityRequest().RequiresExternalSetupCheck()
}

func (o SubscriptionDiagnosticOptions) localCapabilityRequest() providerdiag.LocalCapabilityRequest {
	return providerdiag.LocalCapabilityRequest{
		Capabilities:         o.Capabilities,
		RequiredCapabilities: o.RequiredCapabilities,
		RunSmoke:             o.RunSmoke,
		PrintRequest:         o.PrintRequest,
	}
}

type SubscriptionDiagnosticReport struct {
	Provider             string                                `json:"provider"`
	DisplayName          string                                `json:"display_name"`
	Endpoint             string                                `json:"endpoint"`
	Model                string                                `json:"model"`
	ModelSource          string                                `json:"model_source"`
	CatalogModel         string                                `json:"catalog_model"`
	CatalogModelSource   string                                `json:"catalog_model_source"`
	Route                string                                `json:"route"`
	RuntimeMode          string                                `json:"runtime_mode"`
	Billing              string                                `json:"billing"`
	APICost              string                                `json:"api_cost"`
	AuthState            string                                `json:"auth_state"`
	Account              string                                `json:"account,omitempty"`
	AuthFile             string                                `json:"auth_file,omitempty"`
	Originator           string                                `json:"originator"`
	PromptCacheKey       string                                `json:"prompt_cache_key"`
	PromptCacheRetention string                                `json:"prompt_cache_retention"`
	Store                string                                `json:"store"`
	PreviousResponseID   string                                `json:"previous_response_id"`
	ContextManagement    string                                `json:"context_management"`
	FunctionCalling      bool                                  `json:"function_calling"`
	Checks               []DiagnosticCheck                     `json:"checks"`
	Capabilities         *DiagnosticCapabilities               `json:"capabilities,omitempty"`
	RequestPreview       *SubscriptionDiagnosticRequestPreview `json:"request_preview,omitempty"`
	Smoke                *SubscriptionDiagnosticSmokeResult    `json:"smoke,omitempty"`
}

type SubscriptionDiagnosticRequestPreview struct {
	Requests []SubscriptionDiagnosticRequestPreviewRequest `json:"requests"`
}

type SubscriptionDiagnosticRequestPreviewRequest struct {
	Name             string            `json:"name"`
	ToolPayload      bool              `json:"tool_payload"`
	RetentionPayload bool              `json:"retention_payload"`
	CachePayload     bool              `json:"cache_payload"`
	ThinkingPayload  bool              `json:"thinking_payload"`
	CompactPayload   bool              `json:"compact_payload"`
	WebSearchPayload bool              `json:"web_search_payload"`
	Route            string            `json:"route"`
	Method           string            `json:"method,omitempty"`
	URL              string            `json:"url,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	Body             any               `json:"body,omitempty"`
	Skipped          bool              `json:"skipped,omitempty"`
	SkipReason       string            `json:"skip_reason,omitempty"`
}

type SubscriptionDiagnosticSmokeResult struct {
	Ran                bool                                       `json:"ran"`
	ToolPayload        bool                                       `json:"tool_payload"`
	RetentionPayload   bool                                       `json:"retention_payload"`
	CachePayload       bool                                       `json:"cache_payload"`
	ThinkingPayload    bool                                       `json:"thinking_payload"`
	CompactPayload     bool                                       `json:"compact_payload"`
	WebSearchPayload   bool                                       `json:"web_search_payload"`
	WebSearchCallCount int                                        `json:"web_search_call_count,omitempty"`
	Route              string                                     `json:"route"`
	Content            string                                     `json:"content,omitempty"`
	Duration           string                                     `json:"duration,omitempty"`
	UsageObserved      bool                                       `json:"usage_observed"`
	Usage              providerdiag.SmokeUsage                    `json:"usage"`
	Cost               providerdiag.SmokeCost                     `json:"cost"`
	Requests           []SubscriptionDiagnosticSmokeRequestResult `json:"requests,omitempty"`
}

type SubscriptionDiagnosticSmokeRequestResult struct {
	Name               string                  `json:"name"`
	Ran                bool                    `json:"ran"`
	Skipped            bool                    `json:"skipped,omitempty"`
	SkipReason         string                  `json:"skip_reason,omitempty"`
	ToolPayload        bool                    `json:"tool_payload"`
	RetentionPayload   bool                    `json:"retention_payload"`
	CachePayload       bool                    `json:"cache_payload"`
	ThinkingPayload    bool                    `json:"thinking_payload"`
	CompactPayload     bool                    `json:"compact_payload"`
	WebSearchPayload   bool                    `json:"web_search_payload"`
	WebSearchCallCount int                     `json:"web_search_call_count,omitempty"`
	Route              string                  `json:"route"`
	Content            string                  `json:"content,omitempty"`
	Duration           string                  `json:"duration,omitempty"`
	UsageObserved      bool                    `json:"usage_observed"`
	Usage              providerdiag.SmokeUsage `json:"usage"`
	Cost               providerdiag.SmokeCost  `json:"cost"`
	Error              string                  `json:"error,omitempty"`
}

type subscriptionDiagnosticSmokeRequest struct {
	Name             string
	SystemPrompt     string
	UserContent      string
	ToolPayload      bool
	RetentionPayload bool
	CachePayload     bool
	ThinkingPayload  bool
	CompactPayload   bool
	WebSearchPayload bool
}

func DiagnoseOpenAISubscription(ctx context.Context, options SubscriptionDiagnosticOptions) SubscriptionDiagnosticReport {
	cfg := config.CloneConfig(options.Config)
	model, modelSource := providerdiag.ResolveProviderDiagnosticModel(cfg, subscriptionProviderKey, options.Model, SubscriptionDefaultModel())
	catalogModel, catalogSource := providerdiag.ResolveProviderDiagnosticCatalogModel(cfg, subscriptionProviderKey, model, options.CatalogModel)
	authCfg := DefaultSubscriptionAuthConfig()
	authStatus := ReadSubscriptionAuthStatus(authCfg)

	report := SubscriptionDiagnosticReport{
		Provider:             subscriptionProviderKey,
		DisplayName:          subscriptionDisplayName,
		Endpoint:             RedactSubscriptionEndpointForDisplay(authCfg.Endpoint),
		Model:                model,
		ModelSource:          modelSource,
		CatalogModel:         catalogModel,
		CatalogModelSource:   catalogSource,
		Route:                DiagnosticRouteResponsesStreaming,
		RuntimeMode:          "full_payload",
		Billing:              "ChatGPT subscription",
		APICost:              "N/A",
		AuthState:            string(authStatus.State),
		Account:              authStatus.AccountIDMasked,
		AuthFile:             authStatus.AuthFilePath,
		Originator:           authCfg.Originator,
		PromptCacheKey:       "enabled",
		PromptCacheRetention: "omitted_by_policy",
		Store:                "false",
		PreviousResponseID:   "unsupported",
		ContextManagement:    "disabled",
		FunctionCalling:      true,
	}

	report.addSubscriptionLocalChecks(authCfg, authStatus, options.requiresAuthCheck(), options.requiresEndpointCheck())
	if options.Capabilities {
		report.addSubscriptionCapabilities(ctx, cfg)
	}
	report.addSubscriptionRequiredCapabilities(ctx, cfg, options.RequiredCapabilities)
	if options.CompactSmoke {
		report.addSubscriptionCompactEndpointCheck(authCfg)
	}
	if options.PrintRequest {
		report.addSubscriptionRequestPreview(ctx, cfg, options)
	}
	if options.RunSmoke && !options.PrintRequest {
		report.runSubscriptionSmokeIfReady(ctx, cfg, options)
	}
	return report
}

func (r SubscriptionDiagnosticReport) HasFailures() bool {
	for _, check := range r.Checks {
		if check.Status == DiagnosticStatusFail {
			return true
		}
	}
	return false
}

func (r SubscriptionDiagnosticReport) SummaryStatus() DiagnosticStatus {
	if r.HasFailures() {
		return DiagnosticStatusFail
	}
	for _, check := range r.Checks {
		if check.Status == DiagnosticStatusWarn {
			return DiagnosticStatusWarn
		}
	}
	return DiagnosticStatusOK
}

func (r *SubscriptionDiagnosticReport) addCheck(status DiagnosticStatus, name, message, detail, suggestion string) {
	r.Checks = append(r.Checks, DiagnosticCheck{
		Name:       name,
		Status:     status,
		Message:    message,
		Detail:     strings.TrimSpace(detail),
		Suggestion: strings.TrimSpace(suggestion),
	})
}

func (r *SubscriptionDiagnosticReport) addSubscriptionLocalChecks(authCfg SubscriptionAuthConfig, authStatus SubscriptionAuthStatus, includeAuth, includeEndpoint bool) {
	if includeAuth {
		status, message, suggestion := subscriptionDiagnosticAuthCheckFields(authStatus)
		r.addCheck(status, "auth", message, authStatus.Message, suggestion)
	}
	if includeEndpoint {
		if endpoint, err := validateSubscriptionResponsesEndpoint(authCfg.Endpoint); err != nil {
			r.addCheck(DiagnosticStatusFail, "endpoint", "subscription endpoint is invalid", RedactSubscriptionSecrets(err.Error()), "Set "+subscriptionEndpointEnv+" to a valid HTTPS endpoint for tests only")
		} else {
			r.addCheck(DiagnosticStatusOK, "endpoint", "subscription endpoint is configured", RedactSubscriptionEndpointForDisplay(endpoint), "")
		}
		if originator, err := validateSubscriptionOriginatorForRequest(authCfg.Originator); err != nil {
			r.addCheck(DiagnosticStatusFail, "originator", "subscription originator is not xelyon", err.Error(), "Remove "+subscriptionOriginatorEnv+" or set it to xelyon")
		} else {
			r.addCheck(DiagnosticStatusOK, "originator", "subscription originator is honest xelyon", originator, "")
		}
	}
	if api.IsRegisteredProvider(subscriptionProviderKey) {
		r.addCheck(DiagnosticStatusOK, "provider_registration", "openai_subscription provider is registered", "", "")
	} else {
		r.addCheck(DiagnosticStatusFail, "provider_registration", "openai_subscription provider is not registered", "", "")
	}
	if err := ValidateSubscriptionModel(r.Model); err != nil {
		r.addCheck(DiagnosticStatusFail, "model", "subscription model is unsupported", err.Error(), "")
	} else {
		r.addCheck(DiagnosticStatusOK, "model", "subscription model is supported by XELYON request builder", r.Model, "")
	}
	r.addCheck(DiagnosticStatusOK, "route", "subscription always uses Responses streaming", r.Route, "")
	r.addCheck(DiagnosticStatusOK, "function_calling", "subscription function calling is enabled", "", "")
	r.addCheck(DiagnosticStatusOK, "prompt_cache_key", "prompt_cache_key is enabled", "stable key derived from model and system prompt", "")
	r.addCheck(DiagnosticStatusOK, "prompt_cache_retention", "prompt_cache_retention is omitted by subscription policy", "", "")
	r.addCheck(DiagnosticStatusOK, "store", "subscription runtime forces store=false", "", "")
	r.addCheck(DiagnosticStatusOK, "previous_response_id", "previous_response_id is expected unsupported and omitted", "", "")
	r.addCheck(DiagnosticStatusOK, "context_management", "context_management is disabled for subscription", "", "")
	r.addCheck(DiagnosticStatusOK, "cost", "OpenAI Platform API cost is not estimated", "N/A (ChatGPT subscription)", "")
}

func subscriptionDiagnosticAuthCheckFields(authStatus SubscriptionAuthStatus) (DiagnosticStatus, string, string) {
	if authStatus.LoggedIn {
		return DiagnosticStatusOK, "openai_subscription is logged in", ""
	}
	if authStatus.State == SubscriptionAuthStateTokenExpired {
		return DiagnosticStatusWarn,
			"openai_subscription token is expired and will be refreshed by request path",
			"Run: " + subscriptionLoginCommand + " if refresh fails"
	}
	if authStatus.State == SubscriptionAuthStatePermissionUnsafe {
		return DiagnosticStatusFail, "openai_subscription is not ready", "Fix openai_subscription auth file/directory permissions or run: " + subscriptionLoginCommand
	}
	return DiagnosticStatusFail, "openai_subscription is not ready", "Run: " + subscriptionLoginCommand
}

func (r *SubscriptionDiagnosticReport) addSubscriptionCapabilities(ctx context.Context, cfg *config.Config) {
	capabilities := buildSubscriptionDiagnosticCapabilities(ctx, cfg, *r)
	r.Capabilities = &capabilities
	r.addCheck(
		DiagnosticStatusOK,
		"capabilities",
		"OpenAI Subscription model capabilities were resolved",
		providerdiag.DiagnosticCapabilitiesDetail(capabilities),
		"",
	)
}

func buildSubscriptionDiagnosticCapabilities(ctx context.Context, cfg *config.Config, report SubscriptionDiagnosticReport) DiagnosticCapabilities {
	capabilities := providerdiag.DiagnosticCapabilitiesFromSnapshot(subscriptionDiagnosticCapabilitySnapshot(ctx, cfg, report))
	capabilities.Provider = subscriptionProviderKey
	return capabilities
}

func subscriptionDiagnosticCapabilitySnapshot(ctx context.Context, cfg *config.Config, report SubscriptionDiagnosticReport) providerdiag.CapabilitySnapshot {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if ctx == nil {
		ctx = context.Background()
	}
	policy := providerdiag.SimpleProviderCatalogPolicy(cfg, subscriptionProviderKey, report.Model, report.CatalogModel)
	reasoning := subscriptionResponsesReasoningConfig(config.WithContext(ctx, cfg), openairesponses.NewModelIdentity(report.Model, report.CatalogModel))
	return providerdiag.CapabilitySnapshot{
		RequestModel:                   report.Model,
		CatalogModel:                   report.CatalogModel,
		Route:                          report.Route,
		RouteReason:                    "subscription endpoint uses streaming Responses-shaped full payload requests",
		ResponsesAPI:                   true,
		ResponsesStreaming:             true,
		ResponsesStreamingAvailability: providerdiag.KnownCapabilityAvailability(true),
		ChatCompletions:                false,
		FunctionCalling:                true,
		ImageInput:                     providerdiag.KnownCapabilityAvailability(false),
		WebSearch:                      providerdiag.KnownCapabilityAvailability(true),
		Thinking:                       providerdiag.KnownCapabilityAvailability(reasoning != nil),
		LocalModelAvailable:            providerdiag.KnownCapabilityAvailability(false),
		Retention: providerdiag.RetentionSnapshot{
			Supported:          true,
			Store:              false,
			PreviousResponseID: false,
			PersistResponseID:  false,
			SessionPersistence: false,
			Detail:             "full_payload mode; store=false; previous_response_id unsupported",
		},
		ServerCompaction: providerdiag.ServerCompactionSnapshot{
			Enabled:        false,
			RequestPayload: false,
			LocalFallback:  true,
			Detail:         "context_management disabled for subscription; local provider-facing reduction is fallback",
		},
		ContextWindowTokens: policy.ContextWindowTokens,
		ContextWindowKnown:  policy.ContextWindowKnown,
		MaxOutput:           policy.MaxOutput,
		Pricing:             cost.PricingInfo{PricingUnavailable: true},
	}
}

func (r *SubscriptionDiagnosticReport) addSubscriptionCompactEndpointCheck(authCfg SubscriptionAuthConfig) {
	if r == nil {
		return
	}
	compactEndpoint := strings.TrimSpace(authCfg.CompactEndpoint)
	if compactEndpoint == "" {
		r.addCheck(DiagnosticStatusWarn, "compact_api", "subscription Compact API endpoint is not configured", "runtime falls back to provider-facing reduction/local summary", "")
		return
	}
	validatedEndpoint, err := validateSubscriptionCompactEndpoint(compactEndpoint)
	if err != nil {
		if subscriptionCompactEndpointForbidden(compactEndpoint) {
			r.addCheck(DiagnosticStatusFail, "compact_api", "subscription Compact API endpoint must not use OpenAI Platform API", RedactSubscriptionEndpointForDisplay(compactEndpoint), "Unset "+subscriptionCompactEndpointEnv+" or point it at a subscription Compact endpoint")
			return
		}
		r.addCheck(DiagnosticStatusFail, "compact_api", "subscription Compact API endpoint is invalid", RedactSubscriptionSecrets(err.Error()), "Set "+subscriptionCompactEndpointEnv+" to a valid subscription Compact endpoint or empty to disable it")
		return
	}
	r.addCheck(DiagnosticStatusOK, "compact_api", "subscription Compact API endpoint is configured for smoke probing", RedactSubscriptionEndpointForDisplay(validatedEndpoint), "This is a ChatGPT/Codex subscription backend endpoint, not an OpenAI Platform API endpoint")
}

func (r *SubscriptionDiagnosticReport) addSubscriptionRequiredCapabilities(ctx context.Context, cfg *config.Config, required []string) {
	diagnostic := providerdiag.NewRequiredCapabilityDiagnostic(
		subscriptionDiagnosticCapabilitySnapshot(ctx, cfg, *r),
		required,
		providerdiag.RequiredCapabilityDiagnosticOptions{
			ProviderName:                  "OpenAI Subscription",
			MissingTarget:                 "model/configuration",
			UnknownAvailabilitySuggestion: "Set --catalog-model or provider_models.openai_subscription.catalog_model to the underlying subscription model before requiring catalog-dependent capabilities",
		},
	)
	if !diagnostic.Requested {
		return
	}
	status := DiagnosticStatusFail
	if diagnostic.Satisfied {
		status = DiagnosticStatusOK
	}
	r.addCheck(status, diagnostic.Name, diagnostic.Message, diagnostic.Detail, diagnostic.Suggestion)
}

func (r *SubscriptionDiagnosticReport) addSubscriptionRequestPreview(ctx context.Context, cfg *config.Config, options SubscriptionDiagnosticOptions) {
	preview, err := buildSubscriptionDiagnosticRequestPreview(ctx, cfg, *r, options)
	r.RequestPreview = &preview
	if err != nil {
		r.addCheck(DiagnosticStatusFail, "request_preview", "subscription request preview could not be built", err.Error(), "")
		return
	}
	r.addCheck(DiagnosticStatusOK, "request_preview", "subscription request preview was built without sending a live request", fmt.Sprintf("requests=%d", len(preview.Requests)), "")
}

func buildSubscriptionDiagnosticRequestPreview(ctx context.Context, cfg *config.Config, report SubscriptionDiagnosticReport, options SubscriptionDiagnosticOptions) (SubscriptionDiagnosticRequestPreview, error) {
	baseCfg := subscriptionDiagnosticSmokeConfig(cfg, report.Model, options)
	provider := NewSubscription()
	preview := SubscriptionDiagnosticRequestPreview{}
	for _, request := range subscriptionDiagnosticSmokeRequests(options) {
		if request.CompactPayload {
			compactEndpoint, err := validateSubscriptionCompactEndpoint(DefaultSubscriptionAuthConfig().CompactEndpoint)
			if err == nil {
				preview.Requests = append(preview.Requests, SubscriptionDiagnosticRequestPreviewRequest{
					Name:           request.Name,
					CompactPayload: true,
					Route:          diagnosticRouteSubscriptionCompact,
					Method:         "POST",
					URL:            RedactSubscriptionEndpointForDisplay(compactEndpoint),
					Headers:        subscriptionDiagnosticPreviewHeaders(report.Account, report.Originator),
					Body:           subscriptionDiagnosticCompactPreviewBody(report.Model),
				})
				continue
			}
			preview.Requests = append(preview.Requests, SubscriptionDiagnosticRequestPreviewRequest{
				Name:           request.Name,
				CompactPayload: true,
				Route:          diagnosticRouteSubscriptionCompact,
				Skipped:        true,
				SkipReason:     RedactSubscriptionSecrets(err.Error()),
			})
			continue
		}
		if request.WebSearchPayload {
			body := buildSubscriptionWebSearchRequest(config.WithContext(ctx, baseCfg), request.UserContent, report.Model)
			preview.Requests = append(preview.Requests, SubscriptionDiagnosticRequestPreviewRequest{
				Name:             request.Name,
				WebSearchPayload: true,
				Route:            report.Route,
				Method:           "POST",
				URL:              report.Endpoint,
				Headers:          subscriptionDiagnosticPreviewHeaders(report.Account, report.Originator),
				Body:             subscriptionDiagnosticWebSearchPreviewBody(body),
			})
			continue
		}
		requestCtx := newSubscriptionDiagnosticSmokeRequestContext(ctx, baseCfg, request, io.Discard)
		configureSubscriptionDiagnosticProviderForRequest(provider, request)
		body := provider.buildChatResponsesRequest(requestCtx, request.SystemPrompt, []api.Message{{Role: "user", Content: request.UserContent}}, report.Model)
		preview.Requests = append(preview.Requests, SubscriptionDiagnosticRequestPreviewRequest{
			Name:             request.Name,
			ToolPayload:      request.ToolPayload,
			RetentionPayload: request.RetentionPayload,
			CachePayload:     request.CachePayload,
			ThinkingPayload:  request.ThinkingPayload,
			Route:            report.Route,
			Method:           "POST",
			URL:              report.Endpoint,
			Headers:          subscriptionDiagnosticPreviewHeaders(report.Account, report.Originator),
			Body:             subscriptionDiagnosticPreviewBody(body),
		})
	}
	return preview, nil
}
