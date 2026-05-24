package gemini

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

type geminiDiagnosticSmokeFailure struct {
	Kind       providerdiag.SmokeFailureKind
	Feature    providerdiag.SmokeFailureFeature
	Message    string
	Detail     string
	Suggestion string
}

func classifyGeminiDiagnosticSmokeFailure(smoke DiagnosticSmokeResult, err error) geminiDiagnosticSmokeFailure {
	return classifyGeminiDiagnosticSmokeFailureWithEndpoint(smoke, err, geminiDiagnosticEndpointFailureContextFromEnv())
}

func classifyGeminiDiagnosticSmokeFailureWithEndpoint(smoke DiagnosticSmokeResult, err error, endpoint geminiDiagnosticEndpointFailureContext) geminiDiagnosticSmokeFailure {
	classificationCtx := providerdiag.BasicMultimodalSmokeFailureContext(
		providerdiag.SmokeFailureContextOptions{
			Provider:         "Gemini",
			AuthEnv:          geminiAPIKeyEnv,
			EndpointEnv:      geminiAPIURLEnv,
			DebugEnv:         "XELYON_DEBUG_GEMINI",
			EndpointOverride: endpoint.HasOverride(),
		},
		smoke,
		err,
	)
	return newGeminiDiagnosticSmokeFailure(providerdiag.ClassifySmokeFailure(classificationCtx))
}

func (r *DiagnosticReport) addGeminiDiagnosticSmokeFailureChecks(smoke DiagnosticSmokeResult, failure geminiDiagnosticSmokeFailure) {
	for _, check := range geminiDiagnosticSmokeFailureRequestChecks(smoke, failure) {
		r.addCheck(DiagnosticStatusFail, check.Name, check.Message, check.Detail, check.Suggestion)
	}
}

type geminiDiagnosticSmokeFailureRequestCheck struct {
	Name       string
	Message    string
	Detail     string
	Suggestion string
}

func geminiDiagnosticSmokeFailureRequestChecks(smoke DiagnosticSmokeResult, failure geminiDiagnosticSmokeFailure) []geminiDiagnosticSmokeFailureRequestCheck {
	requests := []struct {
		name    string
		find    func(DiagnosticSmokeResult) (DiagnosticSmokeRequestResult, bool)
		message func(DiagnosticSmokeRequestResult) string
	}{
		{name: "tool_smoke", find: failedGeminiToolSmokeRequest, message: geminiDiagnosticToolSmokeFailureMessage},
		{name: "image_smoke", find: failedGeminiImageSmokeRequest, message: geminiDiagnosticImageSmokeFailureMessage},
		{name: "web_search_smoke", find: failedGeminiWebSearchSmokeRequest, message: geminiDiagnosticWebSearchSmokeFailureMessage},
	}

	var checks []geminiDiagnosticSmokeFailureRequestCheck
	for _, requestCheck := range requests {
		request, ok := requestCheck.find(smoke)
		if !ok {
			continue
		}
		checks = append(checks, geminiDiagnosticSmokeFailureRequestCheck{
			Name:       requestCheck.name,
			Message:    requestCheck.message(request),
			Detail:     failure.Detail,
			Suggestion: failure.Suggestion,
		})
	}
	return checks
}

func geminiDiagnosticToolSmokeFailureMessage(request DiagnosticSmokeRequestResult) string {
	if strings.Contains(request.Error, "tool smoke response did not include") {
		return "Gemini tool smoke response did not include the diagnostic tool call"
	}
	return "Gemini tool smoke failed before proving function calling"
}

func geminiDiagnosticImageSmokeFailureMessage(request DiagnosticSmokeRequestResult) string {
	if strings.Contains(request.Error, "image smoke response content is empty") {
		return "Gemini image smoke response content is empty"
	}
	return "Gemini image smoke failed before proving image input"
}

func geminiDiagnosticWebSearchSmokeFailureMessage(request DiagnosticSmokeRequestResult) string {
	if strings.Contains(request.Error, "web search smoke response did not include") {
		return "Gemini web search smoke did not return summary or sources"
	}
	return "Gemini web search smoke failed before proving native web search"
}

func failedGeminiToolSmokeRequest(smoke DiagnosticSmokeResult) (DiagnosticSmokeRequestResult, bool) {
	return failedGeminiSmokeRequest(smoke, func(request DiagnosticSmokeRequestResult) bool { return request.ToolPayload })
}

func failedGeminiWebSearchSmokeRequest(smoke DiagnosticSmokeResult) (DiagnosticSmokeRequestResult, bool) {
	return failedGeminiSmokeRequest(smoke, func(request DiagnosticSmokeRequestResult) bool { return request.WebSearchPayload })
}

func failedGeminiImageSmokeRequest(smoke DiagnosticSmokeResult) (DiagnosticSmokeRequestResult, bool) {
	return failedGeminiSmokeRequest(smoke, func(request DiagnosticSmokeRequestResult) bool { return request.ImagePayload })
}

func failedGeminiSmokeRequest(smoke DiagnosticSmokeResult, matches func(DiagnosticSmokeRequestResult) bool) (DiagnosticSmokeRequestResult, bool) {
	for _, request := range smoke.Requests {
		if matches(request) && strings.TrimSpace(request.Error) != "" {
			return request, true
		}
	}
	return DiagnosticSmokeRequestResult{}, false
}
