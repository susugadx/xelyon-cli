package gemini

import (
	"errors"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func TestClassifyGeminiDiagnosticSmokeFailure(t *testing.T) {
	tests := []struct {
		name                string
		endpointOverrideURL string
		smoke               DiagnosticSmokeResult
		err                 error
		wantKind            providerdiag.SmokeFailureKind
		wantFeature         providerdiag.SmokeFailureFeature
		wantMessage         string
		wantSuggestion      string
		notMessage          string
		notSuggestion       string
	}{
		{
			name:           "auth",
			smoke:          geminiDiagnosticFailedSmoke("text", DiagnosticRouteStreamGenerateContentSSE, `API error (401): {"error":{"message":"API key not valid"}}`),
			err:            errors.New(`API error (401): {"error":{"message":"API key not valid"}}`),
			wantKind:       providerdiag.SmokeFailureKindAuth,
			wantMessage:    "authentication or authorization",
			wantSuggestion: geminiAPIKeyEnv,
		},
		{
			name:           "quota",
			smoke:          geminiDiagnosticFailedSmoke("text", DiagnosticRouteStreamGenerateContentSSE, "rate limit exceeded (429). Please retry later"),
			err:            errors.New("rate limit exceeded (429). Please retry later"),
			wantKind:       providerdiag.SmokeFailureKindQuota,
			wantMessage:    "quota, rate limit, or capacity",
			wantSuggestion: "quota",
		},
		{
			name:           "capacity from service unavailable status",
			smoke:          geminiDiagnosticFailedSmoke("text", DiagnosticRouteStreamGenerateContentSSE, "API error (503): Service Unavailable: backend overloaded"),
			err:            errors.New("API error (503): Service Unavailable: backend overloaded"),
			wantKind:       providerdiag.SmokeFailureKindQuota,
			wantMessage:    "quota, rate limit, or capacity",
			wantSuggestion: "quota",
		},
		{
			name:                "endpoint mismatch with explicit url",
			endpointOverrideURL: "https://proxy.example/v1beta/models/foo:generateContent",
			smoke:               geminiDiagnosticFailedSmoke("text", DiagnosticRouteStreamGenerateContentSSE, `API error (404): {"error":{"message":"not found"}}`),
			err:                 errors.New(`API error (404): {"error":{"message":"not found"}}`),
			wantKind:            providerdiag.SmokeFailureKindEndpointMismatch,
			wantMessage:         "endpoint does not match",
			wantSuggestion:      "streamGenerateContent?alt=sse",
		},
		{
			name:                "endpoint mismatch from alt=sse query marker",
			endpointOverrideURL: "https://proxy.example/v1beta/models/foo:streamGenerateContent",
			smoke:               geminiDiagnosticFailedSmoke("text", DiagnosticRouteStreamGenerateContentSSE, "proxy rejected alt=sse"),
			err:                 errors.New("proxy rejected alt=sse"),
			wantKind:            providerdiag.SmokeFailureKindEndpointMismatch,
			wantMessage:         "endpoint does not match",
			wantSuggestion:      "streamGenerateContent?alt=sse",
		},
		{
			name:           "built in empty SSE response is not endpoint guidance",
			smoke:          geminiDiagnosticFailedSmoke("text", DiagnosticRouteStreamGenerateContentSSE, "no content in Gemini SSE response (stream ended without generating any text or function calls)"),
			err:            errors.New("no content in Gemini SSE response (stream ended without generating any text or function calls)"),
			wantKind:       providerdiag.SmokeFailureKindEmptyResponse,
			wantMessage:    "response was empty",
			wantSuggestion: "--model",
			notMessage:     "endpoint does not match",
			notSuggestion:  geminiAPIURLEnv,
		},
		{
			name:           "generated empty image response uses empty guidance",
			smoke:          geminiDiagnosticFailedSmoke("image", DiagnosticRouteStreamGenerateContentSSE, "image smoke response content is empty", geminiDiagnosticImageSmokePayload),
			err:            errors.New("image smoke response content is empty"),
			wantKind:       providerdiag.SmokeFailureKindEmptyResponse,
			wantFeature:    providerdiag.SmokeFailureFeatureImageInput,
			wantMessage:    "response was empty",
			wantSuggestion: "--model",
			notMessage:     "image smoke was not accepted",
			notSuggestion:  geminiAPIURLEnv,
		},
		{
			name:           "built in decode failure is not endpoint guidance",
			smoke:          geminiDiagnosticFailedSmoke("text", DiagnosticRouteStreamGenerateContentSSE, "failed to decode response: invalid character '<' looking for beginning of value"),
			err:            errors.New("failed to decode response: invalid character '<' looking for beginning of value"),
			wantKind:       providerdiag.SmokeFailureKindGeneric,
			wantMessage:    "live Gemini smoke request failed",
			wantSuggestion: "--print-request",
			notMessage:     "endpoint does not match",
			notSuggestion:  geminiAPIURLEnv,
		},
		{
			name:                "override empty SSE response keeps endpoint guidance",
			endpointOverrideURL: "https://proxy.example/v1beta/models/foo:streamGenerateContent?alt=sse",
			smoke:               geminiDiagnosticFailedSmoke("text", DiagnosticRouteStreamGenerateContentSSE, "no content in Gemini SSE response (stream ended without generating any text or function calls)"),
			err:                 errors.New("no content in Gemini SSE response (stream ended without generating any text or function calls)"),
			wantKind:            providerdiag.SmokeFailureKindEndpointMismatch,
			wantMessage:         "endpoint does not match",
			wantSuggestion:      geminiAPIURLEnv,
		},
		{
			name:           "model unavailable",
			smoke:          geminiDiagnosticFailedSmoke("text", DiagnosticRouteStreamGenerateContentSSE, `API error (404): {"error":{"message":"Model not found for API version"}}`),
			err:            errors.New(`API error (404): {"error":{"message":"Model not found for API version"}}`),
			wantKind:       providerdiag.SmokeFailureKindModelUnavailable,
			wantMessage:    "model is unavailable",
			wantSuggestion: "--model",
		},
		{
			name:           "web search model error mentioning generateContent still uses model guidance",
			smoke:          geminiDiagnosticFailedSmoke("web_search", DiagnosticRouteGenerateContent, `API error (404): models/missing is not found for API version v1beta, or is not supported for generateContent`, geminiDiagnosticWebSearchSmokePayload),
			err:            errors.New(`API error (404): models/missing is not found for API version v1beta, or is not supported for generateContent`),
			wantKind:       providerdiag.SmokeFailureKindModelUnavailable,
			wantMessage:    "model is unavailable",
			wantSuggestion: "--model",
		},
		{
			name:           "tool unsupported",
			smoke:          geminiDiagnosticFailedSmoke("tool", DiagnosticRouteStreamGenerateContentSSE, "function calling is unsupported for this model", geminiDiagnosticToolSmokePayload),
			err:            errors.New("function calling is unsupported for this model"),
			wantKind:       providerdiag.SmokeFailureKindFeatureUnsupported,
			wantFeature:    providerdiag.SmokeFailureFeatureFunctionCalling,
			wantMessage:    "tool smoke was not accepted",
			wantSuggestion: "function calling",
		},
		{
			name:           "web search unsupported",
			smoke:          geminiDiagnosticFailedSmoke("web_search", DiagnosticRouteGenerateContent, "google_search is unsupported for this model", geminiDiagnosticWebSearchSmokePayload),
			err:            errors.New("google_search is unsupported for this model"),
			wantKind:       providerdiag.SmokeFailureKindFeatureUnsupported,
			wantFeature:    providerdiag.SmokeFailureFeatureWebSearch,
			wantMessage:    "web search smoke was not accepted",
			wantSuggestion: "google_search",
		},
		{
			name:                "web search route error mentioning generateContent still uses endpoint guidance",
			endpointOverrideURL: "https://proxy.example/v1beta/models/foo:streamGenerateContent?alt=sse",
			smoke:               geminiDiagnosticFailedSmoke("web_search", DiagnosticRouteGenerateContent, `API error (404): generateContent route not found`, geminiDiagnosticWebSearchSmokePayload),
			err:                 errors.New(`API error (404): generateContent route not found`),
			wantKind:            providerdiag.SmokeFailureKindEndpointMismatch,
			wantMessage:         "endpoint does not match",
			wantSuggestion:      geminiAPIURLEnv,
		},
		{
			name:           "image unsupported",
			smoke:          geminiDiagnosticFailedSmoke("image", DiagnosticRouteStreamGenerateContentSSE, "inline_data image input is unsupported for this model", geminiDiagnosticImageSmokePayload),
			err:            errors.New("inline_data image input is unsupported for this model"),
			wantKind:       providerdiag.SmokeFailureKindFeatureUnsupported,
			wantFeature:    providerdiag.SmokeFailureFeatureImageInput,
			wantMessage:    "image smoke was not accepted",
			wantSuggestion: "inline_data",
		},
		{
			name:                "image route error mentioning GenerateContentRequest without image tokens uses endpoint guidance",
			endpointOverrideURL: "https://proxy.example/v1beta/models/foo:generateContent",
			smoke:               geminiDiagnosticFailedSmoke("image", DiagnosticRouteStreamGenerateContentSSE, "API error (404): GenerateContentRequest route not found", geminiDiagnosticImageSmokePayload),
			err:                 errors.New("API error (404): GenerateContentRequest route not found"),
			wantKind:            providerdiag.SmokeFailureKindEndpointMismatch,
			wantMessage:         "endpoint does not match",
			wantSuggestion:      geminiAPIURLEnv,
		},
		{
			name:                "image error mentioning GenerateContentRequest with explicit url still uses image guidance",
			endpointOverrideURL: "https://proxy.example/v1beta/models/foo:streamGenerateContent?alt=sse",
			smoke:               geminiDiagnosticFailedSmoke("image", DiagnosticRouteStreamGenerateContentSSE, "GenerateContentRequest.contents[0].parts[0].inline_data is unsupported", geminiDiagnosticImageSmokePayload),
			err:                 errors.New("GenerateContentRequest.contents[0].parts[0].inline_data is unsupported"),
			wantKind:            providerdiag.SmokeFailureKindFeatureUnsupported,
			wantFeature:         providerdiag.SmokeFailureFeatureImageInput,
			wantMessage:         "image smoke was not accepted",
			wantSuggestion:      "inline_data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyGeminiDiagnosticSmokeFailureWithEndpoint(tt.smoke, tt.err, geminiDiagnosticEndpointFailureContext{
				OverrideURL: tt.endpointOverrideURL,
			})
			if got.Kind != tt.wantKind {
				t.Fatalf("Kind = %q, want %q; failure = %#v", got.Kind, tt.wantKind, got)
			}
			if tt.wantFeature != "" && got.Feature != tt.wantFeature {
				t.Fatalf("Feature = %q, want %q; failure = %#v", got.Feature, tt.wantFeature, got)
			}
			if !strings.Contains(got.Message, tt.wantMessage) {
				t.Fatalf("Message = %q, want substring %q", got.Message, tt.wantMessage)
			}
			if !strings.Contains(got.Suggestion, tt.wantSuggestion) {
				t.Fatalf("Suggestion = %q, want substring %q", got.Suggestion, tt.wantSuggestion)
			}
			if tt.notMessage != "" && strings.Contains(got.Message, tt.notMessage) {
				t.Fatalf("Message = %q, want it not to contain %q", got.Message, tt.notMessage)
			}
			if tt.notSuggestion != "" && strings.Contains(got.Suggestion, tt.notSuggestion) {
				t.Fatalf("Suggestion = %q, want it not to contain %q", got.Suggestion, tt.notSuggestion)
			}
			if !strings.Contains(got.Detail, "request=") {
				t.Fatalf("Detail = %q, want request-level context", got.Detail)
			}
		})
	}
}

func geminiDiagnosticFailedSmoke(name, route, requestError string, options ...func(*DiagnosticSmokeRequestResult)) DiagnosticSmokeResult {
	request := DiagnosticSmokeRequestResult{
		Name:  name,
		Route: route,
		Error: requestError,
	}
	for _, option := range options {
		option(&request)
	}
	return DiagnosticSmokeResult{Requests: []DiagnosticSmokeRequestResult{request}}
}

func geminiDiagnosticToolSmokePayload(request *DiagnosticSmokeRequestResult) {
	request.ToolPayload = true
}

func geminiDiagnosticImageSmokePayload(request *DiagnosticSmokeRequestResult) {
	request.ImagePayload = true
}

func geminiDiagnosticWebSearchSmokePayload(request *DiagnosticSmokeRequestResult) {
	request.WebSearchPayload = true
}
