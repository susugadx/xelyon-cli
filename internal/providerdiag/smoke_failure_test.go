package providerdiag

import (
	"errors"
	"strings"
	"testing"
)

func TestClassifySmokeFailureCommonVocabulary(t *testing.T) {
	tests := []struct {
		name     string
		ctx      SmokeFailureContext
		want     SmokeFailureKind
		message  string
		suggest  string
		suggest2 string
		reject   string
	}{
		{
			name:    "auth",
			ctx:     SmokeFailureContext{SmokeFailureContextOptions: SmokeFailureContextOptions{Provider: "Claude", AuthEnv: "ANTHROPIC_API_KEY"}, Detail: `API error (401): invalid API key`},
			want:    SmokeFailureKindAuth,
			message: "authentication or authorization",
		},
		{
			name:    "quota",
			ctx:     SmokeFailureContext{SmokeFailureContextOptions: SmokeFailureContextOptions{Provider: "Kimi"}, Detail: "rate limit exceeded (429)"},
			want:    SmokeFailureKindQuota,
			message: "quota, rate limit, or capacity",
			suggest: "quota",
		},
		{
			name:    "quota provider error code",
			ctx:     SmokeFailureContext{SmokeFailureContextOptions: SmokeFailureContextOptions{Provider: "OpenAI"}, Detail: "OpenAI API error: rate_limit_exceeded"},
			want:    SmokeFailureKindQuota,
			message: "quota, rate limit, or capacity",
			suggest: "quota",
		},
		{
			name:    "model unavailable",
			ctx:     SmokeFailureContext{SmokeFailureContextOptions: SmokeFailureContextOptions{Provider: "OpenRouter"}, Detail: "model not found"},
			want:    SmokeFailureKindModelUnavailable,
			message: "model is unavailable",
		},
		{
			name:     "deployment unavailable with endpoint override",
			ctx:      SmokeFailureContext{SmokeFailureContextOptions: SmokeFailureContextOptions{Provider: "Azure OpenAI", EndpointEnv: "AZURE_OPENAI_BASE_URL", EndpointOverride: true, ModelFlag: "--deployment", ModelLabel: "deployment"}, Detail: "DeploymentNotFound: deployment does not exist"},
			want:     SmokeFailureKindModelUnavailable,
			message:  "model is unavailable",
			suggest:  "--deployment",
			suggest2: "AZURE_OPENAI_BASE_URL",
			reject:   "--model",
		},
		{
			name:    "endpoint mismatch",
			ctx:     SmokeFailureContext{SmokeFailureContextOptions: SmokeFailureContextOptions{Provider: "Gemini", EndpointEnv: "GEMINI_API_URL", EndpointOverride: true}, Detail: "API error (404): route not found"},
			want:    SmokeFailureKindEndpointMismatch,
			message: "endpoint does not match",
			suggest: "GEMINI_API_URL",
			reject:  "--model",
		},
		{
			name:    "endpoint override generic route does not exist",
			ctx:     SmokeFailureContext{SmokeFailureContextOptions: SmokeFailureContextOptions{Provider: "OpenRouter", EndpointEnv: "OPENROUTER_API_URL", EndpointOverride: true}, Detail: "route does not exist"},
			want:    SmokeFailureKindEndpointMismatch,
			message: "endpoint does not match",
			suggest: "OPENROUTER_API_URL",
			reject:  "--model",
		},
		{
			name:    "endpoint override generic resource not found",
			ctx:     SmokeFailureContext{SmokeFailureContextOptions: SmokeFailureContextOptions{Provider: "DeepSeek", EndpointEnv: "DEEPSEEK_API_URL", EndpointOverride: true}, Detail: "resource not found"},
			want:    SmokeFailureKindEndpointMismatch,
			message: "endpoint does not match",
			suggest: "DEEPSEEK_API_URL",
			reject:  "--model",
		},
		{
			name:    "endpoint mismatch from Gemini streaming query marker",
			ctx:     SmokeFailureContext{SmokeFailureContextOptions: SmokeFailureContextOptions{Provider: "Gemini", EndpointEnv: "GEMINI_API_URL", EndpointOverride: true}, Detail: "proxy rejected alt=sse"},
			want:    SmokeFailureKindEndpointMismatch,
			message: "endpoint does not match",
			suggest: "GEMINI_API_URL",
			reject:  "--model",
		},
		{
			name:    "feature unsupported",
			ctx:     SmokeFailureContext{SmokeFailureContextOptions: SmokeFailureContextOptions{Provider: "Bedrock"}, Feature: SmokeFailureFeatureThinking, Detail: "extended thinking is unsupported"},
			want:    SmokeFailureKindFeatureUnsupported,
			message: "thinking smoke was not accepted",
		},
		{
			name:    "empty",
			ctx:     SmokeFailureContext{SmokeFailureContextOptions: SmokeFailureContextOptions{Provider: "Gemini"}, Detail: "stream ended without generating content"},
			want:    SmokeFailureKindEmptyResponse,
			message: "response was empty",
		},
		{
			name:    "empty generated smoke content error",
			ctx:     SmokeFailureContext{SmokeFailureContextOptions: SmokeFailureContextOptions{Provider: "Claude"}, Feature: SmokeFailureFeatureImageInput, Detail: "image smoke response content is empty"},
			want:    SmokeFailureKindEmptyResponse,
			message: "response was empty",
			suggest: "--model",
		},
		{
			name:    "timeout",
			ctx:     SmokeFailureContext{SmokeFailureContextOptions: SmokeFailureContextOptions{Provider: "Gemini"}, Detail: "thinking timeout: no Gemini progress or actionable output received"},
			want:    SmokeFailureKindTimeout,
			message: "request timed out",
			suggest: "--timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifySmokeFailure(tt.ctx)
			if got.Kind != tt.want {
				t.Fatalf("Kind = %q, want %q; failure=%#v", got.Kind, tt.want, got)
			}
			if got.Feature != tt.ctx.Feature {
				t.Fatalf("Feature = %q, want %q; failure=%#v", got.Feature, tt.ctx.Feature, got)
			}
			if !strings.Contains(got.Message, tt.message) {
				t.Fatalf("Message = %q, want contains %q", got.Message, tt.message)
			}
			if strings.TrimSpace(got.Suggestion) == "" {
				t.Fatalf("Suggestion is empty for %#v", got)
			}
			if tt.suggest != "" && !strings.Contains(got.Suggestion, tt.suggest) {
				t.Fatalf("Suggestion = %q, want contains %q", got.Suggestion, tt.suggest)
			}
			if tt.suggest2 != "" && !strings.Contains(got.Suggestion, tt.suggest2) {
				t.Fatalf("Suggestion = %q, want contains %q", got.Suggestion, tt.suggest2)
			}
			if tt.reject != "" && strings.Contains(got.Suggestion, tt.reject) {
				t.Fatalf("Suggestion = %q, should not contain %q", got.Suggestion, tt.reject)
			}
		})
	}
}

func TestSmokeFailureContextsExtractFailedRequestFeature(t *testing.T) {
	tests := []struct {
		name        string
		ctx         SmokeFailureContext
		wantFeature SmokeFailureFeature
		wantDetail  []string
	}{
		{
			name: "text tool",
			ctx: TextToolSmokeFailureContext(
				SmokeFailureContextOptions{Provider: "Groq"},
				TextToolSmokeResult{Requests: []TextToolSmokeRequestResult{{
					Name:        "tool",
					Route:       "chat_completions",
					ToolPayload: true,
					Error:       "tool smoke response did not include diagnostic call",
				}}},
				errors.New("outer"),
			),
			wantFeature: SmokeFailureFeatureFunctionCalling,
			wantDetail:  []string{"request=tool", "route=chat_completions"},
		},
		{
			name: "multimodal",
			ctx: BasicMultimodalSmokeFailureContext(
				SmokeFailureContextOptions{Provider: "Gemini"},
				MultimodalSmokeResult{Requests: []MultimodalSmokeRequestResult{{
					Name:             "web_search",
					Route:            "generate_content",
					WebSearchPayload: true,
					Error:            "google_search is unsupported",
				}}},
				nil,
			),
			wantFeature: SmokeFailureFeatureWebSearch,
			wantDetail:  []string{"request=web_search", "route=generate_content"},
		},
		{
			name: "responses",
			ctx: ResponsesSmokeFailureContext(
				SmokeFailureContextOptions{Provider: "Azure OpenAI"},
				ResponsesSmokeResult{Requests: []ResponsesSmokeRequestResult{{
					Name:        "tool",
					ToolPayload: true,
					Error:       "tool smoke response did not include xelyon_azure_doctor_probe function_call",
				}}},
				nil,
			),
			wantFeature: SmokeFailureFeatureFunctionCalling,
			wantDetail:  []string{"request=tool"},
		},
		{
			name: "routed responses",
			ctx: RoutedResponsesSmokeFailureContext(
				SmokeFailureContextOptions{Provider: "OpenAI"},
				RoutedResponsesSmokeResult{Requests: []RoutedResponsesSmokeRequestResult{{
					Name:  "text",
					Route: "responses_streaming",
					Error: "rate limit exceeded",
				}}},
				nil,
			),
			wantDetail: []string{"request=text", "route=responses_streaming"},
		},
		{
			name: "kimi",
			ctx: KimiSmokeFailureContext(
				SmokeFailureContextOptions{Provider: "Kimi"},
				KimiSmokeResult{Requests: []KimiSmokeRequestResult{{
					Name:             "web_search",
					WebSearchPayload: true,
					Error:            "$web_search is unsupported",
				}}},
				nil,
			),
			wantFeature: SmokeFailureFeatureWebSearch,
			wantDetail:  []string{"request=web_search"},
		},
		{
			name: "invocation",
			ctx: InvocationSmokeFailureContext(
				SmokeFailureContextOptions{Provider: "Bedrock"},
				InvocationSmokeResult{Requests: []InvocationSmokeRequestResult{{
					Name:            "thinking",
					ThinkingEnabled: true,
					Error:           "thinking is not supported",
				}}},
				nil,
			),
			wantFeature: SmokeFailureFeatureThinking,
			wantDetail:  []string{"request=thinking"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSmokeFailureContext(t, tt.ctx, tt.wantFeature, tt.wantDetail)
		})
	}
}

func assertSmokeFailureContext(t *testing.T, got SmokeFailureContext, wantFeature SmokeFailureFeature, wantDetail []string) {
	t.Helper()

	if got.Feature != wantFeature {
		t.Fatalf("Feature = %q, want %q; context=%#v", got.Feature, wantFeature, got)
	}
	for _, detail := range wantDetail {
		if !strings.Contains(got.Detail, detail) {
			t.Fatalf("Detail = %q, want substring %q; context=%#v", got.Detail, detail, got)
		}
	}
}
