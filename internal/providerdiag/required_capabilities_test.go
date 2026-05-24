package providerdiag

import (
	"strings"
	"testing"
)

func TestEvaluateRequiredCapabilities(t *testing.T) {
	snapshot := CapabilitySnapshot{
		ResponsesAPI:                   true,
		ResponsesStreaming:             false,
		ResponsesStreamingAvailability: KnownCapabilityAvailability(false),
		FunctionCalling:                true,
		ImageInput:                     KnownCapabilityAvailability(true),
		Retention:                      NewRetentionSnapshot(true, true, true),
		ServerCompaction: ServerCompactionSnapshot{
			RequestPayload: true,
		},
	}

	check := EvaluateRequiredCapabilities(snapshot, []string{
		"responses-api",
		"function_calling",
		"responses_api",
		"responses_streaming",
		"unknown_capability",
	})
	if check.Satisfied() {
		t.Fatalf("Satisfied() = true, want false for missing/unknown capability: %+v", check.Results)
	}
	if !check.HasUnknown() {
		t.Fatalf("HasUnknown() = false, want true: %+v", check.Results)
	}
	if got, want := check.Detail(), "responses_api=ok, function_calling=ok, responses_streaming=missing, unknown_capability=unknown"; got != want {
		t.Fatalf("Detail() = %q, want %q", got, want)
	}
}

func TestSupportedRequiredCapabilitiesSharedV11Vocabulary(t *testing.T) {
	got := strings.Join(SupportedRequiredCapabilities(), ",")
	want := strings.Join([]string{
		RequiredCapabilityResponsesAPI,
		RequiredCapabilityResponsesStreaming,
		RequiredCapabilityChatCompletions,
		RequiredCapabilityFunctionCalling,
		RequiredCapabilityImageInput,
		RequiredCapabilityWebSearch,
		RequiredCapabilityThinking,
		RequiredCapabilityPreviousResponseID,
		RequiredCapabilitySessionPersistence,
		RequiredCapabilityServerCompaction,
		RequiredCapabilityLocalModelAvailable,
	}, ",")
	if got != want {
		t.Fatalf("SupportedRequiredCapabilities() = %q, want %q", got, want)
	}
}

func TestHasRequiredCapability(t *testing.T) {
	values := []string{" function-calling, local-model-available ", "web_search"}

	for _, capability := range []string{
		RequiredCapabilityFunctionCalling,
		RequiredCapabilityLocalModelAvailable,
		"local-model-available",
		RequiredCapabilityWebSearch,
	} {
		if !HasRequiredCapability(values, capability) {
			t.Fatalf("HasRequiredCapability(%q) = false, want true", capability)
		}
	}
	if HasRequiredCapability(values, RequiredCapabilityThinking) {
		t.Fatalf("HasRequiredCapability(%q) = true, want false", RequiredCapabilityThinking)
	}
}

func TestLocalCapabilityRequestCheckPolicy(t *testing.T) {
	tests := []struct {
		name              string
		request           LocalCapabilityRequest
		wantOnly          bool
		wantAuth          bool
		wantExternalSetup bool
	}{
		{
			name:              "capabilities flag only",
			request:           LocalCapabilityRequest{Capabilities: true},
			wantOnly:          true,
			wantAuth:          false,
			wantExternalSetup: false,
		},
		{
			name: "required capability only",
			request: LocalCapabilityRequest{
				RequiredCapabilities: []string{RequiredCapabilityFunctionCalling},
			},
			wantOnly:          true,
			wantAuth:          false,
			wantExternalSetup: false,
		},
		{
			name: "empty required capability value",
			request: LocalCapabilityRequest{
				RequiredCapabilities: []string{" ", ","},
			},
			wantOnly:          false,
			wantAuth:          true,
			wantExternalSetup: true,
		},
		{
			name: "smoke keeps remote checks",
			request: LocalCapabilityRequest{
				Capabilities: true,
				RunSmoke:     true,
			},
			wantOnly:          false,
			wantAuth:          true,
			wantExternalSetup: true,
		},
		{
			name: "print request skips auth but keeps external setup checks",
			request: LocalCapabilityRequest{
				Capabilities: true,
				PrintRequest: true,
			},
			wantOnly:          false,
			wantAuth:          false,
			wantExternalSetup: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLocalCapabilityOnlyRequest(tt.request); got != tt.wantOnly {
				t.Fatalf("IsLocalCapabilityOnlyRequest() = %t, want %t", got, tt.wantOnly)
			}
			if got := tt.request.IsOnly(); got != tt.wantOnly {
				t.Fatalf("IsOnly() = %t, want %t", got, tt.wantOnly)
			}
			if got := tt.request.RequiresAuthCheck(); got != tt.wantAuth {
				t.Fatalf("RequiresAuthCheck() = %t, want %t", got, tt.wantAuth)
			}
			if got := tt.request.RequiresExternalSetupCheck(); got != tt.wantExternalSetup {
				t.Fatalf("RequiresExternalSetupCheck() = %t, want %t", got, tt.wantExternalSetup)
			}
		})
	}
}

func TestEvaluateRequiredCapabilitiesSatisfied(t *testing.T) {
	snapshot := CapabilitySnapshot{
		ResponsesAPI:                   true,
		ResponsesStreaming:             true,
		ResponsesStreamingAvailability: KnownCapabilityAvailability(true),
		ChatCompletions:                true,
		FunctionCalling:                true,
		ImageInput:                     KnownCapabilityAvailability(true),
		WebSearch:                      KnownCapabilityAvailability(true),
		Thinking:                       KnownCapabilityAvailability(true),
		LocalModelAvailable:            KnownCapabilityAvailability(true),
		Retention:                      NewRetentionSnapshot(true, true, true),
		ServerCompaction: ServerCompactionSnapshot{
			RequestPayload: true,
		},
	}

	check := EvaluateRequiredCapabilities(snapshot, SupportedRequiredCapabilities())
	if !check.Satisfied() {
		t.Fatalf("Satisfied() = false, want true: detail=%q results=%+v", check.Detail(), check.Results)
	}
}

func TestEvaluateRequiredCapabilitiesUnknownAvailability(t *testing.T) {
	snapshot := CapabilitySnapshot{
		ResponsesAPI:                   true,
		ResponsesStreaming:             true,
		ResponsesStreamingAvailability: UnknownCapabilityAvailability(),
	}

	check := EvaluateRequiredCapabilities(snapshot, []string{RequiredCapabilityResponsesStreaming})
	if check.Satisfied() {
		t.Fatalf("Satisfied() = true, want false for unknown availability: %+v", check.Results)
	}
	if !check.HasUnknownAvailability() {
		t.Fatalf("HasUnknownAvailability() = false, want true: %+v", check.Results)
	}
	if got, want := check.Detail(), "responses_streaming=unknown"; got != want {
		t.Fatalf("Detail() = %q, want %q", got, want)
	}
}

func TestDiagnosticCapabilitiesFromSnapshotPreservesTriStateAvailability(t *testing.T) {
	capabilities := DiagnosticCapabilitiesFromSnapshot(CapabilitySnapshot{
		ResponsesStreaming:             true,
		ResponsesStreamingAvailability: UnknownCapabilityAvailability(),
		ImageInput:                     UnknownCapabilityAvailability(),
		WebSearch:                      KnownCapabilityAvailability(false),
		Thinking:                       KnownCapabilityAvailability(true),
		LocalModelAvailable:            UnknownCapabilityAvailability(),
	})

	if !capabilities.ResponsesStreaming || capabilities.ResponsesStreamingKnown {
		t.Fatalf("ResponsesStreaming = %t known=%t, want available route with unknown availability", capabilities.ResponsesStreaming, capabilities.ResponsesStreamingKnown)
	}
	if capabilities.ImageInput || capabilities.ImageInputKnown {
		t.Fatalf("ImageInput = %t known=%t, want unknown", capabilities.ImageInput, capabilities.ImageInputKnown)
	}
	if capabilities.WebSearch || !capabilities.WebSearchKnown {
		t.Fatalf("WebSearch = %t known=%t, want known missing", capabilities.WebSearch, capabilities.WebSearchKnown)
	}
	if !capabilities.Thinking || !capabilities.ThinkingKnown {
		t.Fatalf("Thinking = %t known=%t, want known available", capabilities.Thinking, capabilities.ThinkingKnown)
	}
	if capabilities.LocalModelAvailable || capabilities.LocalModelAvailableKnown {
		t.Fatalf("LocalModelAvailable = %t known=%t, want unknown without endpoint discovery", capabilities.LocalModelAvailable, capabilities.LocalModelAvailableKnown)
	}

	detail := DiagnosticCapabilitiesDetail(capabilities)
	for _, want := range []string{
		"responses_streaming=unknown",
		"image_input=unknown",
		"web_search=false",
		"thinking=true",
		"local_model_available=unknown",
	} {
		if !strings.Contains(detail, want) {
			t.Fatalf("DiagnosticCapabilitiesDetail() = %q, want %q", detail, want)
		}
	}
}

func TestNewRequiredCapabilityDiagnostic(t *testing.T) {
	snapshot := CapabilitySnapshot{
		FunctionCalling: true,
		ImageInput:      KnownCapabilityAvailability(true),
	}

	empty := NewRequiredCapabilityDiagnostic(snapshot, nil, RequiredCapabilityDiagnosticOptions{ProviderName: "Groq"})
	if empty.Requested {
		t.Fatalf("Requested = true, want false for empty required capability request: %+v", empty)
	}

	ok := NewRequiredCapabilityDiagnostic(snapshot, []string{"function_calling"}, RequiredCapabilityDiagnosticOptions{ProviderName: "Groq"})
	if !ok.Requested || !ok.Satisfied || ok.Name != RequiredCapabilityCheckName ||
		ok.Message != "required Groq capabilities are available" ||
		ok.Detail != "function_calling=ok" ||
		ok.Suggestion != "" {
		t.Fatalf("diagnostic ok = %+v, want Groq success check", ok)
	}

	missing := NewRequiredCapabilityDiagnostic(snapshot, []string{"web_search"}, RequiredCapabilityDiagnosticOptions{
		ProviderName:  "Groq",
		MissingTarget: "Groq model/configuration",
	})
	if !missing.Requested || missing.Satisfied ||
		missing.Message != "required Groq capabilities are missing" ||
		missing.Detail != "web_search=unknown" ||
		missing.Suggestion != "Choose a Groq model/configuration that provides the missing capability, or remove --require-capability" {
		t.Fatalf("diagnostic missing = %+v, want Groq missing check", missing)
	}
}

func TestResponsesStreamingCapabilityAvailability(t *testing.T) {
	knownPolicy := CatalogPolicy{ContextWindowKnown: true}
	unknownPolicy := CatalogPolicy{ContextWindowKnown: false}

	for _, tt := range []struct {
		name               string
		responsesStreaming bool
		policy             CatalogPolicy
		want               CapabilityAvailability
	}{
		{
			name:               "streaming route with known catalog",
			responsesStreaming: true,
			policy:             knownPolicy,
			want:               KnownCapabilityAvailability(true),
		},
		{
			name:               "streaming route with unknown catalog",
			responsesStreaming: true,
			policy:             unknownPolicy,
			want:               UnknownCapabilityAvailability(),
		},
		{
			name:               "non streaming route with unknown catalog",
			responsesStreaming: false,
			policy:             unknownPolicy,
			want:               KnownCapabilityAvailability(false),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := ResponsesStreamingCapabilityAvailability(tt.responsesStreaming, tt.policy)
			if got != tt.want {
				t.Fatalf("ResponsesStreamingCapabilityAvailability() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRequiredCapabilityFailureSuggestion(t *testing.T) {
	missing := RequiredCapabilityCheck{Results: []RequiredCapabilityResult{{
		Name:   RequiredCapabilityResponsesStreaming,
		Status: RequiredCapabilityStatusMissing,
	}}}
	if got, want := RequiredCapabilityFailureSuggestion(missing, "model/configuration", ""), "Choose a model/configuration that provides the missing capability, or remove --require-capability"; got != want {
		t.Fatalf("RequiredCapabilityFailureSuggestion() = %q, want %q", got, want)
	}

	unknown := RequiredCapabilityCheck{Results: []RequiredCapabilityResult{{
		Name:   "unknown_capability",
		Status: RequiredCapabilityStatusUnknownName,
	}}}
	if got, want := RequiredCapabilityFailureSuggestion(unknown, "model/configuration", ""), "Use one of: "+SupportedRequiredCapabilitiesText(); got != want {
		t.Fatalf("RequiredCapabilityFailureSuggestion() = %q, want %q", got, want)
	}

	unknownAvailability := RequiredCapabilityCheck{Results: []RequiredCapabilityResult{{
		Name:   RequiredCapabilityResponsesStreaming,
		Status: RequiredCapabilityStatusUnknownAvailability,
	}}}
	if got, want := RequiredCapabilityFailureSuggestion(unknownAvailability, "model/configuration", "Set --catalog-model"), "Set --catalog-model"; got != want {
		t.Fatalf("RequiredCapabilityFailureSuggestion() = %q, want %q", got, want)
	}
}
