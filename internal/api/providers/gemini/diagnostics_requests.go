package gemini

import (
	"context"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type geminiDiagnosticRequest struct {
	Name             string
	SystemPrompt     string
	UserContent      string
	ToolPayload      bool
	ImagePayload     bool
	WebSearchPayload bool
}

func geminiDiagnosticRequests(options DiagnosticOptions) []geminiDiagnosticRequest {
	includeText := options.TextSmoke || (!options.ToolSmoke && !options.ImageSmoke && !options.WebSearchSmoke)

	var requests []geminiDiagnosticRequest
	if includeText {
		requests = append(requests, geminiDiagnosticTextRequest())
	}
	if options.ToolSmoke {
		requests = append(requests, geminiDiagnosticToolRequest())
	}
	if options.ImageSmoke {
		requests = append(requests, geminiDiagnosticImageRequest())
	}
	if options.WebSearchSmoke {
		requests = append(requests, geminiDiagnosticWebSearchRequest())
	}
	return requests
}

func geminiDiagnosticTextRequest() geminiDiagnosticRequest {
	return geminiDiagnosticRequest{
		Name:         "text",
		SystemPrompt: "Reply briefly.",
		UserContent:  "Reply with: xelyon gemini doctor ok",
	}
}

func geminiDiagnosticToolRequest() geminiDiagnosticRequest {
	return geminiDiagnosticRequest{
		Name:         "tool",
		SystemPrompt: "Use the diagnostic tool.",
		UserContent:  `Call xelyon_gemini_doctor_probe exactly once with {"value":"gemini-tool-ok"} and do not answer in prose.`,
		ToolPayload:  true,
	}
}

func geminiDiagnosticImageRequest() geminiDiagnosticRequest {
	return geminiDiagnosticRequest{
		Name:         "image",
		SystemPrompt: "Reply briefly.",
		UserContent:  "Look at the attached tiny diagnostic image and reply with a short non-empty response.",
		ImagePayload: true,
	}
}

func geminiDiagnosticWebSearchRequest() geminiDiagnosticRequest {
	return geminiDiagnosticRequest{
		Name:             "web_search",
		SystemPrompt:     "Use native web search and reply briefly.",
		UserContent:      "Gemini API web search grounding latest documentation",
		WebSearchPayload: true,
	}
}

func newGeminiDiagnosticRequestContext(ctx context.Context, cfg *config.Config, request geminiDiagnosticRequest, output io.Writer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		output = io.Discard
	}
	requestCtx := ui.WithRuntime(ctx, ui.NewRuntime(strings.NewReader(""), output, output))
	requestCtx = api.WithAssistantUpdateMode(requestCtx, api.AssistantUpdatesOff)
	requestCtx = config.WithContext(requestCtx, cfg)

	switch {
	case request.ToolPayload:
		requestCtx = api.WithToolDefinitions(requestCtx, geminiDiagnosticSmokeToolDefinitions())
		requestCtx = withGeminiFunctionCallingMode(requestCtx, "ANY")
	case request.ImagePayload || request.WebSearchPayload:
		requestCtx = api.WithToolDefinitions(requestCtx, nil)
		requestCtx = api.WithToolUseDisabled(requestCtx)
	default:
		requestCtx = api.WithToolDefinitions(requestCtx, nil)
		requestCtx = api.WithToolUseDisabled(requestCtx)
	}
	return requestCtx
}

func geminiDiagnosticRequestRoute(request geminiDiagnosticRequest) string {
	if request.WebSearchPayload {
		return DiagnosticRouteGenerateContent
	}
	return DiagnosticRouteStreamGenerateContentSSE
}

func geminiDiagnosticRequestURL(model string, request geminiDiagnosticRequest) string {
	if request.WebSearchPayload {
		return getGeminiFunctionCallingURL(model)
	}
	return getGeminiURL(model)
}

func geminiDiagnosticSmokeToolDefinitions() []api.ToolDefinition {
	return providerdiag.NoopDiagnosticToolDefinitions(geminiDiagnosticToolName, "Gemini")
}
