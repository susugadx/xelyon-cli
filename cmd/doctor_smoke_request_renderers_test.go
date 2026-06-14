package cmd

import (
	"bytes"
	"strings"
	"testing"

	deepseekprovider "github.com/susugadx/xelyon-cli/internal/api/providers/deepseek"
	groqprovider "github.com/susugadx/xelyon-cli/internal/api/providers/groq"
	ollamaprovider "github.com/susugadx/xelyon-cli/internal/api/providers/ollama"
	openaisubscription "github.com/susugadx/xelyon-cli/internal/api/providers/openai_subscription"
	openrouterprovider "github.com/susugadx/xelyon-cli/internal/api/providers/openrouter"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func TestOpenAICompatDoctorSmokeRequestRenderers(t *testing.T) {
	request := providerdiag.TextToolSmokeRequestResult{
		Name:          "tool",
		Ran:           true,
		ToolPayload:   true,
		Route:         "chat_completions",
		Content:       "xelyon doctor ok",
		Duration:      "12ms",
		UsageObserved: true,
		Usage: providerdiag.SmokeUsage{
			InputTokens:         10,
			CachedInputTokens:   3,
			OutputTokens:        4,
			ThinkingTokens:      1,
			CacheCreationTokens: 2,
			BillingServiceTier:  "priority",
		},
		Cost: providerdiag.SmokeCost{USD: 0.00012345},
	}
	tests := []struct {
		name   string
		render func(*bytes.Buffer)
	}{
		{
			name: "deepseek",
			render: func(out *bytes.Buffer) {
				renderDeepSeekDoctorSmokeRequest(out, deepseekprovider.DiagnosticSmokeRequestResult(request))
			},
		},
		{
			name: "groq",
			render: func(out *bytes.Buffer) {
				renderGroqDoctorSmokeRequest(out, groqprovider.DiagnosticSmokeRequestResult(request))
			},
		},
		{
			name: "ollama",
			render: func(out *bytes.Buffer) {
				renderOllamaDoctorSmokeRequest(out, ollamaprovider.DiagnosticSmokeRequestResult(request))
			},
		},
		{
			name: "openrouter",
			render: func(out *bytes.Buffer) {
				renderOpenRouterDoctorSmokeRequest(out, openrouterprovider.DiagnosticSmokeRequestResult(request))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			tt.render(&out)
			got := out.String()
			for _, want := range []string{
				"Smoke request tool: ok route=chat_completions duration=12ms",
				"Smoke content tool: xelyon doctor ok",
				"Smoke usage tool: input=10 cached=3 output=4 reasoning=1 cache_creation=2 billing_tier=priority",
				"Smoke cost estimate tool: $0.00012345 USD",
			} {
				if !strings.Contains(got, want) {
					t.Fatalf("%s output missing %q:\n%s", tt.name, want, got)
				}
			}
		})
	}
}

func TestOpenAICompatDoctorSmokeRequestRenderersPrintSkippedRequests(t *testing.T) {
	request := providerdiag.TextToolSmokeRequestResult{
		Name:        "tool",
		Skipped:     true,
		SkipReason:  "function calling disabled",
		ToolPayload: true,
		Route:       "chat_completions",
	}
	tests := []struct {
		name   string
		render func(*bytes.Buffer)
	}{
		{
			name: "deepseek",
			render: func(out *bytes.Buffer) {
				renderDeepSeekDoctorSmokeRequest(out, deepseekprovider.DiagnosticSmokeRequestResult(request))
			},
		},
		{
			name: "groq",
			render: func(out *bytes.Buffer) {
				renderGroqDoctorSmokeRequest(out, groqprovider.DiagnosticSmokeRequestResult(request))
			},
		},
		{
			name: "ollama",
			render: func(out *bytes.Buffer) {
				renderOllamaDoctorSmokeRequest(out, ollamaprovider.DiagnosticSmokeRequestResult(request))
			},
		},
		{
			name: "openrouter",
			render: func(out *bytes.Buffer) {
				renderOpenRouterDoctorSmokeRequest(out, openrouterprovider.DiagnosticSmokeRequestResult(request))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			tt.render(&out)
			if got, want := out.String(), "Smoke request tool: skipped (function calling disabled)\n"; got != want {
				t.Fatalf("%s output = %q, want %q", tt.name, got, want)
			}
		})
	}
}

func TestOpenAISubscriptionDoctorSmokeRequestRendererPrintsRetentionAndErrors(t *testing.T) {
	request := openaisubscription.SubscriptionDiagnosticSmokeRequestResult{
		Name:             "retention_followup",
		Ran:              true,
		RetentionPayload: true,
		Route:            "responses_streaming",
		Duration:         "8ms",
		Error:            "invalid previous_response_id",
		UsageObserved:    true,
		Usage: providerdiag.SmokeUsage{
			InputTokens:       5,
			OutputTokens:      2,
			CachedInputTokens: 1,
		},
		Cost: providerdiag.SmokeCost{PricingUnavailable: true},
	}

	var out bytes.Buffer
	renderOpenAISubscriptionDoctorSmokeRequest(&out, request)
	got := out.String()
	for _, want := range []string{
		"Smoke request retention_followup: fail route=responses_streaming duration=8ms",
		"Smoke error retention_followup: invalid previous_response_id",
		"Smoke usage retention_followup: input=5 cached=1 output=2 reasoning=0 cache_creation=0",
		"Smoke cost estimate retention_followup: N/A (pricing unavailable)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("OpenAI subscription output missing %q:\n%s", want, got)
		}
	}
}

func TestOpenAISubscriptionDoctorSmokeUsageProjectsBillingTier(t *testing.T) {
	got := openAISubscriptionDoctorSmokeUsage(providerdiag.SmokeUsage{
		InputTokens:         1,
		CachedInputTokens:   2,
		OutputTokens:        3,
		ThinkingTokens:      4,
		CacheCreationTokens: 5,
		BillingServiceTier:  "priority",
	})
	if got.InputTokens != 1 ||
		got.CachedInputTokens != 2 ||
		got.OutputTokens != 3 ||
		got.ThinkingTokens != 4 ||
		got.CacheCreationTokens != 5 ||
		got.BillingServiceTier != "priority" {
		t.Fatalf("openAISubscriptionDoctorSmokeUsage() = %#v, want projected usage", got)
	}
}
