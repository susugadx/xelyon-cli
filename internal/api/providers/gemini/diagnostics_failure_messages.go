package gemini

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

type geminiDiagnosticEndpointFailureContext struct {
	OverrideURL string
}

func geminiDiagnosticEndpointFailureContextFromEnv() geminiDiagnosticEndpointFailureContext {
	return geminiDiagnosticEndpointFailureContext{OverrideURL: strings.TrimSpace(os.Getenv(geminiAPIURLEnv))}
}

func (c geminiDiagnosticEndpointFailureContext) HasOverride() bool {
	return strings.TrimSpace(c.OverrideURL) != ""
}

// 共通分類は providerdiag が持ち、ここでは Gemini 固有の表示文言だけを上書きする。
func newGeminiDiagnosticSmokeFailure(failure providerdiag.SmokeFailure) geminiDiagnosticSmokeFailure {
	switch {
	case failure.Kind == providerdiag.SmokeFailureKindAuth:
		return geminiDiagnosticSmokeFailure{
			Kind:       failure.Kind,
			Feature:    failure.Feature,
			Message:    "live Gemini smoke authentication or authorization failed",
			Detail:     failure.Detail,
			Suggestion: fmt.Sprintf("Check %s, API key permissions, and that the key belongs to a project with Gemini API access", geminiAPIKeyEnv),
		}
	case failure.Kind == providerdiag.SmokeFailureKindQuota:
		return geminiDiagnosticSmokeFailure{
			Kind:       failure.Kind,
			Feature:    failure.Feature,
			Message:    "live Gemini smoke hit quota, rate limit, or capacity",
			Detail:     failure.Detail,
			Suggestion: "Check Gemini API quota and rate limits for the selected model, honor Retry-After if present, or rerun later",
		}
	case failure.Kind == providerdiag.SmokeFailureKindModelUnavailable:
		return geminiDiagnosticSmokeFailure{
			Kind:       failure.Kind,
			Feature:    failure.Feature,
			Message:    "live Gemini smoke model is unavailable",
			Detail:     failure.Detail,
			Suggestion: "Use --model with an available Gemini model; keep --catalog-model for alias token/pricing policy",
		}
	case failure.Kind == providerdiag.SmokeFailureKindFeatureUnsupported && failure.Feature == providerdiag.SmokeFailureFeatureFunctionCalling:
		return geminiDiagnosticSmokeFailure{
			Kind:       failure.Kind,
			Feature:    failure.Feature,
			Message:    "Gemini tool smoke was not accepted by the selected model or endpoint",
			Detail:     failure.Detail,
			Suggestion: "Use a Gemini model and endpoint with function calling support, then rerun --tool-smoke --print-request to inspect the diagnostic ANY payload",
		}
	case failure.Kind == providerdiag.SmokeFailureKindFeatureUnsupported && failure.Feature == providerdiag.SmokeFailureFeatureImageInput:
		return geminiDiagnosticSmokeFailure{
			Kind:       failure.Kind,
			Feature:    failure.Feature,
			Message:    "Gemini image smoke was not accepted by the selected model or endpoint",
			Detail:     failure.Detail,
			Suggestion: "Use a Gemini model and endpoint with inline_data image input support, or rerun without --image-smoke",
		}
	case failure.Kind == providerdiag.SmokeFailureKindFeatureUnsupported && failure.Feature == providerdiag.SmokeFailureFeatureWebSearch:
		return geminiDiagnosticSmokeFailure{
			Kind:       failure.Kind,
			Feature:    failure.Feature,
			Message:    "Gemini native web search smoke was not accepted by the selected model or endpoint",
			Detail:     failure.Detail,
			Suggestion: "Use a Gemini model and endpoint that support native google_search on generateContent, or rerun without --web-search-smoke",
		}
	case failure.Kind == providerdiag.SmokeFailureKindEmptyResponse:
		return geminiDiagnosticSmokeFailure{
			Kind:       failure.Kind,
			Feature:    failure.Feature,
			Message:    "live Gemini smoke response was empty",
			Detail:     failure.Detail,
			Suggestion: "The selected Gemini model returned an SSE stream without text or tool calls; retry, change --model if it repeats, or inspect --print-request for the request shape",
		}
	case failure.Kind == providerdiag.SmokeFailureKindTimeout:
		return geminiDiagnosticSmokeFailure{
			Kind:       failure.Kind,
			Feature:    failure.Feature,
			Message:    "live Gemini smoke request timed out",
			Detail:     failure.Detail,
			Suggestion: "Rerun with a larger --timeout, increase streaming.thinking_timeout_seconds for long-thinking Gemini requests, or use XELYON_DEBUG_GEMINI=1 to inspect SSE progress",
		}
	case failure.Kind == providerdiag.SmokeFailureKindEndpointMismatch:
		return geminiDiagnosticSmokeFailure{
			Kind:       failure.Kind,
			Feature:    failure.Feature,
			Message:    "live Gemini smoke endpoint does not match the selected request route",
			Detail:     failure.Detail,
			Suggestion: fmt.Sprintf("Run --print-request and check %s: text/tool/image need streamGenerateContent?alt=sse; web search needs generateContent or a proxy that accepts both shapes", geminiAPIURLEnv),
		}
	default:
		return geminiDiagnosticSmokeFailure{
			Kind:       failure.Kind,
			Feature:    failure.Feature,
			Message:    failure.Message,
			Detail:     failure.Detail,
			Suggestion: failure.Suggestion,
		}
	}
}
