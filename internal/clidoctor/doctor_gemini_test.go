package clidoctor

import (
	"bytes"
	"testing"

	geminiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/gemini"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func TestRenderGeminiDoctorTextIncludesRequestPreviewAndSmokeObservability(t *testing.T) {
	report := geminiprovider.DiagnosticReport{
		Provider:               "gemini",
		APIURL:                 "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.1-pro-preview-customtools:streamGenerateContent?alt=sse",
		Model:                  "gemini-3.1-pro-preview-customtools",
		ModelSource:            "test",
		CatalogModel:           "gemini-3.1-pro-preview-customtools",
		CatalogModelSource:     "test",
		Route:                  geminiprovider.DiagnosticRouteStreamGenerateContentSSE,
		RouteReason:            "Gemini text, tool, and image requests use streamGenerateContent?alt=sse",
		FunctionCallingEnabled: true,
		ImageInputSupported:    true,
		WebSearchSupported:     true,
		ContextCachingEnabled:  true,
		ThinkingEnabled:        false,
		ServiceTier: providerdiag.GeminiServiceTierSnapshot{
			ConfiguredTier:       "priority",
			RequestBodyTier:      "priority",
			PricingFamily:        "gemini_priority",
			BillingTier:          "standard",
			BillingPricingFamily: "gemini",
		},
		Checks: []geminiprovider.DiagnosticCheck{
			{Name: "smoke", Status: geminiprovider.DiagnosticStatusOK, Message: "live Gemini smoke request succeeded"},
		},
		RequestPreview: &geminiprovider.DiagnosticRequestPreview{
			Requests: []geminiprovider.DiagnosticRequestPreviewRequest{{
				Name:    "text",
				Route:   geminiprovider.DiagnosticRouteStreamGenerateContentSSE,
				Method:  "POST",
				URL:     "https://example.test/gemini",
				Headers: map[string]string{"x-goog-api-key": "<redacted>"},
				Body:    map[string]any{"model": "gemini-3.1-pro-preview-customtools"},
			}},
		},
		Smoke: &geminiprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         geminiprovider.DiagnosticRouteStreamGenerateContentSSE,
			Content:       "xelyon gemini doctor ok",
			Duration:      "1ms",
			UsageObserved: true,
			Usage: geminiprovider.DiagnosticSmokeUsage{
				InputTokens:         10,
				OutputTokens:        4,
				ThinkingTokens:      1,
				CachedInputTokens:   3,
				CacheCreationTokens: 0,
				BillingServiceTier:  "standard",
			},
			Cost: geminiprovider.DiagnosticSmokeCost{
				USD: 0.00012345,
			},
		},
	}

	var out bytes.Buffer
	renderGeminiDoctorText(&out, report)
	requireDoctorContractTextContainsAll(t, out.String(), []string{
		"Gemini doctor",
		"Route: stream_generate_content_sse",
		"Capabilities: function_calling=true image_input=true web_search=true context_caching=true thinking=false",
		"Service tier: configured=priority, request_body=priority, pricing_family=gemini_priority, billing=standard, billing_pricing_family=gemini",
		"Request preview:",
		`"x-goog-api-key": "<redacted>"`,
		"Smoke route: stream_generate_content_sse",
		"Smoke usage: input=10 cached=3 output=4 reasoning=1 cache_creation=0 billing_tier=standard",
		"Smoke cost estimate: $0.00012345 USD",
	})
}
