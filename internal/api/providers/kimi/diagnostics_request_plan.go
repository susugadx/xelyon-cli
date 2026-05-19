package kimi

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

type kimiDiagnosticSmokeRequest struct {
	Name             string
	SystemPrompt     string
	UserContent      string
	Thinking         bool
	SessionID        string
	ToolPayload      bool
	ImagePayload     bool
	WebSearchPayload bool
}

type kimiDiagnosticRequestPlan struct {
	Requests     []kimiDiagnosticSmokeRequest
	RunTextSmoke bool
}

const (
	kimiDiagnosticSmokeCacheFirstName  = "thinking_off_cache_first"
	kimiDiagnosticSmokeCacheSecondName = "thinking_off_cache_second"
	kimiDiagnosticSmokeThinkingName    = "thinking_on"
	kimiDiagnosticSmokeImageName       = "image_smoke"
	kimiDiagnosticSmokeWebSearchName   = "web_search_smoke"
	kimiDiagnosticSmokeToolName        = "tool_smoke"
)

func buildKimiDiagnosticRequestPlan(options DiagnosticOptions) kimiDiagnosticRequestPlan {
	runTextSmoke := options.TextSmoke || options.ToolSmoke || (!options.ImageSmoke && !options.WebSearchSmoke)
	var requests []kimiDiagnosticSmokeRequest
	if runTextSmoke {
		requests = append(requests, kimiDiagnosticTextSmokeRequests()...)
	}
	if options.ImageSmoke {
		requests = append(requests, kimiDiagnosticImageSmokeRequest())
	}
	if options.WebSearchSmoke {
		requests = append(requests, kimiDiagnosticWebSearchSmokeRequest())
	}
	if options.ToolSmoke {
		requests = append(requests, kimiDiagnosticToolSmokeRequest())
	}
	return kimiDiagnosticRequestPlan{
		Requests:     requests,
		RunTextSmoke: runTextSmoke,
	}
}

func kimiDiagnosticTextSmokeRequests() []kimiDiagnosticSmokeRequest {
	return []kimiDiagnosticSmokeRequest{
		{
			Name:         kimiDiagnosticSmokeCacheFirstName,
			SystemPrompt: "Reply briefly.",
			UserContent:  "Reply with: xelyon kimi doctor cache one",
			Thinking:     false,
			SessionID:    "xelyon-kimi-doctor-cache",
		},
		{
			Name:         kimiDiagnosticSmokeCacheSecondName,
			SystemPrompt: "Reply briefly.",
			UserContent:  "Reply with: xelyon kimi doctor cache two",
			Thinking:     false,
			SessionID:    "xelyon-kimi-doctor-cache",
		},
		{
			Name:         kimiDiagnosticSmokeThinkingName,
			SystemPrompt: "Think briefly, then reply briefly.",
			UserContent:  "Reply with: xelyon kimi doctor thinking ok",
			Thinking:     true,
			SessionID:    "xelyon-kimi-doctor-thinking",
		},
	}
}

func kimiDiagnosticImageSmokeRequest() kimiDiagnosticSmokeRequest {
	return kimiDiagnosticSmokeRequest{
		Name:         kimiDiagnosticSmokeImageName,
		SystemPrompt: "Reply briefly.",
		UserContent:  "Look at the attached tiny diagnostic image and reply with a short non-empty response.",
		Thinking:     false,
		SessionID:    "xelyon-kimi-doctor-image",
		ImagePayload: true,
	}
}

func kimiDiagnosticWebSearchSmokeRequest() kimiDiagnosticSmokeRequest {
	return kimiDiagnosticSmokeRequest{
		Name:             kimiDiagnosticSmokeWebSearchName,
		SystemPrompt:     "Use web search and reply briefly.",
		UserContent:      "Search the web for Moonshot AI Kimi API web search pricing and reply with one short non-empty summary.",
		Thinking:         false,
		SessionID:        "xelyon-kimi-doctor-web-search",
		WebSearchPayload: true,
	}
}

func kimiDiagnosticToolSmokeRequest() kimiDiagnosticSmokeRequest {
	return kimiDiagnosticSmokeRequest{
		Name:         kimiDiagnosticSmokeToolName,
		SystemPrompt: "Use the diagnostic tool.",
		UserContent:  `Call xelyon_kimi_doctor_probe exactly once with {"value":"kimi-tool-ok"} and do not answer in prose.`,
		Thinking:     false,
		SessionID:    "xelyon-kimi-doctor-tool",
		ToolPayload:  true,
	}
}

func newKimiDiagnosticSkippedToolSmokeRequest(request kimiDiagnosticSmokeRequest) DiagnosticSmokeRequestResult {
	return providerdiag.NewSkippedKimiSmokeRequest(request.kimiSmokeRequest(), kimiDiagnosticDisabledToolSkipReason())
}

func newKimiDiagnosticSkippedToolPreviewRequest(request kimiDiagnosticSmokeRequest) DiagnosticRequestPreviewRequest {
	return providerdiag.NewSkippedKimiPreviewRequest(request.kimiSmokeRequest(), kimiDiagnosticDisabledToolSkipReason())
}

func kimiDiagnosticDisabledToolSkipReason() string {
	return fmt.Sprintf("Kimi function calling payloads are disabled (%s=0)", kimiFunctionCallingEnv)
}

func kimiDiagnosticRequestRoute(request kimiDiagnosticSmokeRequest) string {
	if request.WebSearchPayload {
		return DiagnosticRouteChatCompletionsWebSearch
	}
	return DiagnosticRouteChatCompletions
}

func (request kimiDiagnosticSmokeRequest) kimiSmokeRequest() providerdiag.KimiSmokeRequest {
	return providerdiag.KimiSmokeRequest{
		Name:             request.Name,
		ToolPayload:      request.ToolPayload,
		ImagePayload:     request.ImagePayload,
		WebSearchPayload: request.WebSearchPayload,
		Route:            kimiDiagnosticRequestRoute(request),
	}
}
