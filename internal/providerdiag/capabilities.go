package providerdiag

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/cost"
)

// DiagnosticCapabilities は doctor が解決した provider capability の共通 DTO。
type DiagnosticCapabilities struct {
	Provider                 string                               `json:"provider,omitempty"`
	Model                    string                               `json:"model,omitempty"`
	Deployment               string                               `json:"deployment,omitempty"`
	CatalogModel             string                               `json:"catalog_model,omitempty"`
	Route                    string                               `json:"route,omitempty"`
	RouteReason              string                               `json:"route_reason,omitempty"`
	ResponsesAPI             bool                                 `json:"responses_api"`
	ResponsesStreaming       bool                                 `json:"responses_streaming"`
	ResponsesStreamingKnown  bool                                 `json:"responses_streaming_known"`
	ChatCompletions          bool                                 `json:"chat_completions"`
	FunctionCalling          bool                                 `json:"function_calling"`
	ImageInput               bool                                 `json:"image_input"`
	ImageInputKnown          bool                                 `json:"image_input_known"`
	WebSearch                bool                                 `json:"web_search"`
	WebSearchKnown           bool                                 `json:"web_search_known"`
	Thinking                 bool                                 `json:"thinking"`
	ThinkingKnown            bool                                 `json:"thinking_known"`
	LocalModelAvailable      bool                                 `json:"local_model_available"`
	LocalModelAvailableKnown bool                                 `json:"local_model_available_known"`
	Retention                DiagnosticRetentionCapability        `json:"retention"`
	ServerCompaction         DiagnosticServerCompactionCapability `json:"server_compaction"`
	ContextWindowTokens      int                                  `json:"context_window_tokens,omitempty"`
	ContextWindowKnown       bool                                 `json:"context_window_known"`
	MaxOutputTokens          int                                  `json:"max_output_tokens,omitempty"`
	MaxOutputTokensKnown     bool                                 `json:"max_output_tokens_known"`
	MaxOutputTokensSource    string                               `json:"max_output_tokens_source,omitempty"`
	MaxOutputRuntimeFallback int                                  `json:"max_output_runtime_fallback,omitempty"`
	Pricing                  DiagnosticPricingCapability          `json:"pricing"`
}

// DiagnosticRetentionCapability は Responses API の retention 設定由来の能力を表す。
type DiagnosticRetentionCapability struct {
	Supported          bool   `json:"supported"`
	Store              bool   `json:"store"`
	PreviousResponseID bool   `json:"previous_response_id"`
	PersistResponseID  bool   `json:"persist_response_id"`
	SessionPersistence bool   `json:"session_persistence"`
	Detail             string `json:"detail,omitempty"`
}

// DiagnosticServerCompactionCapability は server-side context compaction の能力を表す。
type DiagnosticServerCompactionCapability struct {
	Enabled                  bool   `json:"enabled"`
	RequestPayload           bool   `json:"request_payload"`
	CompactThreshold         int    `json:"compact_threshold,omitempty"`
	LocalFallback            bool   `json:"local_fallback"`
	SkipLocalAutoCompression bool   `json:"skip_local_auto_compression"`
	Detail                   string `json:"detail,omitempty"`
}

// DiagnosticPricingCapability は catalog/config から解決した pricing 能力を表す。
type DiagnosticPricingCapability struct {
	Available           bool    `json:"available"`
	InputCostPerM       float64 `json:"input_cost_per_m,omitempty"`
	CachedInputCostPerM float64 `json:"cached_input_cost_per_m,omitempty"`
	OutputCostPerM      float64 `json:"output_cost_per_m,omitempty"`
	Detail              string  `json:"detail,omitempty"`
}

// DiagnosticCapabilitiesFromSnapshot は共通 snapshot を doctor JSON DTO に投影する。
func DiagnosticCapabilitiesFromSnapshot(snapshot CapabilitySnapshot) DiagnosticCapabilities {
	return DiagnosticCapabilities{
		Model:                    snapshot.RequestModel,
		CatalogModel:             snapshot.CatalogModel,
		Route:                    snapshot.Route,
		RouteReason:              snapshot.RouteReason,
		ResponsesAPI:             snapshot.ResponsesAPI,
		ResponsesStreaming:       snapshot.ResponsesStreaming,
		ResponsesStreamingKnown:  snapshot.ResponsesStreamingAvailability.Known,
		ChatCompletions:          snapshot.ChatCompletions,
		FunctionCalling:          snapshot.FunctionCalling,
		ImageInput:               capabilityAvailable(snapshot.ImageInput),
		ImageInputKnown:          snapshot.ImageInput.Known,
		WebSearch:                capabilityAvailable(snapshot.WebSearch),
		WebSearchKnown:           snapshot.WebSearch.Known,
		Thinking:                 capabilityAvailable(snapshot.Thinking),
		ThinkingKnown:            snapshot.Thinking.Known,
		LocalModelAvailable:      capabilityAvailable(snapshot.LocalModelAvailable),
		LocalModelAvailableKnown: snapshot.LocalModelAvailable.Known,
		Retention:                DiagnosticRetentionCapabilityFromSnapshot(snapshot.Retention),
		ServerCompaction:         DiagnosticServerCompactionCapabilityFromSnapshot(snapshot.ServerCompaction),
		ContextWindowTokens:      snapshot.ContextWindowTokens,
		ContextWindowKnown:       snapshot.ContextWindowKnown,
		MaxOutputTokens:          snapshot.MaxOutput.CapabilityTokens(),
		MaxOutputTokensKnown:     snapshot.MaxOutput.Available,
		MaxOutputTokensSource:    snapshot.MaxOutput.CapabilitySource(),
		MaxOutputRuntimeFallback: snapshot.MaxOutput.RuntimeFallback,
		Pricing:                  DiagnosticPricingCapabilityFromPricing(snapshot.Pricing),
	}
}

// DiagnosticRetentionCapabilityFromSnapshot は retention snapshot を DTO に投影する。
func DiagnosticRetentionCapabilityFromSnapshot(snapshot RetentionSnapshot) DiagnosticRetentionCapability {
	return DiagnosticRetentionCapability(snapshot)
}

// DiagnosticServerCompactionCapabilityFromSnapshot は server compaction snapshot を DTO に投影する。
func DiagnosticServerCompactionCapabilityFromSnapshot(snapshot ServerCompactionSnapshot) DiagnosticServerCompactionCapability {
	return DiagnosticServerCompactionCapability(snapshot)
}

// DiagnosticPricingCapabilityFromPricing は pricing metadata を DTO に投影する。
func DiagnosticPricingCapabilityFromPricing(pricing cost.PricingInfo) DiagnosticPricingCapability {
	return DiagnosticPricingCapability{
		Available:           !pricing.PricingUnavailable,
		InputCostPerM:       pricing.InputCostPerM,
		CachedInputCostPerM: pricing.CachedInputCostPerM,
		OutputCostPerM:      pricing.OutputCostPerM,
		Detail:              PricingDetail(pricing),
	}
}

// DiagnosticCapabilitiesDetail は capabilities check の detail を返す。
func DiagnosticCapabilitiesDetail(capabilities DiagnosticCapabilities) string {
	features := []string{
		fmt.Sprintf("responses_api=%t", capabilities.ResponsesAPI),
		capabilityDetail("responses_streaming", capabilities.ResponsesStreaming, capabilities.ResponsesStreamingKnown),
		fmt.Sprintf("function_calling=%t", capabilities.FunctionCalling),
		capabilityDetail("image_input", capabilities.ImageInput, capabilities.ImageInputKnown),
		capabilityDetail("web_search", capabilities.WebSearch, capabilities.WebSearchKnown),
		capabilityDetail("thinking", capabilities.Thinking, capabilities.ThinkingKnown),
		capabilityDetail("local_model_available", capabilities.LocalModelAvailable, capabilities.LocalModelAvailableKnown),
	}
	features = append(features,
		fmt.Sprintf("previous_response_id=%t", capabilities.Retention.PreviousResponseID),
		fmt.Sprintf("server_compaction=%t", capabilities.ServerCompaction.RequestPayload),
	)
	if capabilities.ContextWindowKnown {
		features = append(features, fmt.Sprintf("context_window=%d", capabilities.ContextWindowTokens))
	} else {
		features = append(features, "context_window=unknown")
	}
	if capabilities.MaxOutputTokensKnown {
		features = append(features, fmt.Sprintf("max_output_tokens=%d (%s)", capabilities.MaxOutputTokens, capabilities.MaxOutputTokensSource))
	} else if capabilities.MaxOutputRuntimeFallback > 0 {
		features = append(features, fmt.Sprintf("max_output_tokens=missing (runtime_fallback=%d)", capabilities.MaxOutputRuntimeFallback))
	} else {
		features = append(features, "max_output_tokens=unknown")
	}
	features = append(features, capabilities.Pricing.Detail)
	return strings.Join(features, ", ")
}

func capabilityAvailable(availability CapabilityAvailability) bool {
	return availability.Known && availability.Available
}

func capabilityDetail(name string, available, known bool) string {
	if !known {
		return name + "=unknown"
	}
	return fmt.Sprintf("%s=%t", name, available)
}
