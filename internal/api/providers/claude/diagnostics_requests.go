package claude

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

type claudeDiagnosticRequest struct {
	Name             string
	SystemPrompt     string
	UserContent      string
	ToolPayload      bool
	ImagePayload     bool
	ThinkingPayload  bool
	WebSearchPayload bool
}

type claudeDiagnosticRequestPlan struct {
	Requests []claudeDiagnosticRequest
}

const (
	claudeDiagnosticTextRequestName      = "text"
	claudeDiagnosticToolRequestName      = "tool"
	claudeDiagnosticImageRequestName     = "image"
	claudeDiagnosticThinkingRequestName  = "thinking"
	claudeDiagnosticWebSearchRequestName = "web_search"
)

func (r claudeDiagnosticRequest) skipped(functionCallingEnabled bool) bool {
	return r.ToolPayload && !functionCallingEnabled
}

func (r claudeDiagnosticRequest) smokeBase() DiagnosticSmokeRequestResult {
	return providerdiag.NewMultimodalSmokeRequestResult(r.multimodalSmokeRequest())
}

func (r claudeDiagnosticRequest) multimodalSmokeRequest() providerdiag.MultimodalSmokeRequest {
	return providerdiag.MultimodalSmokeRequest{
		Name:             r.Name,
		ToolPayload:      r.ToolPayload,
		ImagePayload:     r.ImagePayload,
		ThinkingPayload:  r.ThinkingPayload,
		WebSearchPayload: r.WebSearchPayload,
		Route:            claudeDiagnosticRequestRoute(r),
	}
}

func buildClaudeDiagnosticRequestPlan(options DiagnosticOptions, functionCallingEnabled bool) claudeDiagnosticRequestPlan {
	includeText := options.TextSmoke || (!options.ToolSmoke && !options.ImageSmoke && !options.ThinkingSmoke && !options.WebSearchSmoke)
	if options.ToolSmoke && !functionCallingEnabled {
		includeText = true
	}

	var requests []claudeDiagnosticRequest
	if includeText {
		requests = append(requests, claudeDiagnosticTextRequest())
	}
	if options.ToolSmoke {
		requests = append(requests, claudeDiagnosticToolRequest())
	}
	if options.ImageSmoke {
		requests = append(requests, claudeDiagnosticImageRequest())
	}
	if options.ThinkingSmoke {
		requests = append(requests, claudeDiagnosticThinkingRequest())
	}
	if options.WebSearchSmoke {
		requests = append(requests, claudeDiagnosticWebSearchRequest())
	}
	return claudeDiagnosticRequestPlan{Requests: requests}
}

func claudeDiagnosticRequestMaxOutputTokens(options DiagnosticOptions) int {
	if options.MaxOutputTokens > 0 {
		return options.MaxOutputTokens
	}
	return defaultClaudeDiagnosticSmokeMaxOutputTokens
}

func claudeDiagnosticTextRequest() claudeDiagnosticRequest {
	return claudeDiagnosticRequest{
		Name:         claudeDiagnosticTextRequestName,
		SystemPrompt: "Reply briefly.",
		UserContent:  "Reply with: xelyon claude doctor ok",
	}
}

func claudeDiagnosticToolRequest() claudeDiagnosticRequest {
	return claudeDiagnosticRequest{
		Name:         claudeDiagnosticToolRequestName,
		SystemPrompt: "Use the diagnostic tool.",
		UserContent:  fmt.Sprintf(`Call %s exactly once with {"value":"claude-tool-ok"} and do not answer in prose.`, claudeDiagnosticToolName),
		ToolPayload:  true,
	}
}

func claudeDiagnosticImageRequest() claudeDiagnosticRequest {
	return claudeDiagnosticRequest{
		Name:         claudeDiagnosticImageRequestName,
		SystemPrompt: "Reply briefly.",
		UserContent:  "Look at the attached tiny diagnostic image and reply with a short non-empty response.",
		ImagePayload: true,
	}
}

func claudeDiagnosticThinkingRequest() claudeDiagnosticRequest {
	return claudeDiagnosticRequest{
		Name:            claudeDiagnosticThinkingRequestName,
		SystemPrompt:    "Think briefly, then reply briefly.",
		UserContent:     "Reply with: xelyon claude doctor thinking ok",
		ThinkingPayload: true,
	}
}

func claudeDiagnosticWebSearchRequest() claudeDiagnosticRequest {
	return claudeDiagnosticRequest{
		Name:             claudeDiagnosticWebSearchRequestName,
		SystemPrompt:     "Use native web search and reply briefly.",
		UserContent:      "Anthropic Claude Messages API web search tool documentation",
		WebSearchPayload: true,
	}
}

func newClaudeDiagnosticRequestContext(ctx context.Context, cfg *config.Config, request claudeDiagnosticRequest, output io.Writer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if output == nil {
		output = io.Discard
	}
	requestCfg := config.CloneConfig(cfg)
	if request.ThinkingPayload {
		requestCfg.Thinking.Enabled = true
	}
	requestCtx := uiruntime.WithRuntime(ctx, uiruntime.NewRuntime(strings.NewReader(""), output, output))
	requestCtx = api.WithAssistantUpdateMode(requestCtx, api.AssistantUpdatesOff)
	requestCtx = config.WithContext(requestCtx, requestCfg)

	if request.ToolPayload {
		requestCtx = api.WithToolDefinitions(requestCtx, claudeDiagnosticSmokeToolDefinitions())
	} else {
		requestCtx = api.WithToolDefinitions(requestCtx, nil)
		requestCtx = api.WithToolUseDisabled(requestCtx)
	}
	return requestCtx
}

func claudeDiagnosticRequestRoute(request claudeDiagnosticRequest) string {
	if request.WebSearchPayload {
		return DiagnosticRouteClaudeWebSearch
	}
	return DiagnosticRouteClaudeMessages
}

func claudeDiagnosticSmokeToolDefinitions() []api.ToolDefinition {
	return providerdiag.NoopDiagnosticToolDefinitions(claudeDiagnosticToolName, "Claude")
}

func applyClaudeDiagnosticToolChoice(ctx context.Context, provider *Provider, request claudeDiagnosticRequest, catalogModel string) {
	if provider == nil {
		return
	}
	if request.ToolPayload && claudeDiagnosticForcedToolChoiceAllowed(ctx, catalogModel) {
		provider.SetToolChoice(claudeDiagnosticToolName)
		return
	}
	provider.ClearToolChoice()
}

func claudeDiagnosticForcedToolChoiceAllowed(ctx context.Context, catalogModel string) bool {
	if IsAlwaysOnThinkingModel(catalogModel) {
		return false
	}
	return !api.IsThinkingEnabled(ctx)
}

func newClaudeDiagnosticSkippedToolPreviewRequest(request claudeDiagnosticRequest) DiagnosticRequestPreviewRequest {
	return providerdiag.NewSkippedMultimodalPreviewRequest(request.multimodalSmokeRequest(), claudeDiagnosticDisabledToolSkipReason())
}

func newClaudeDiagnosticSkippedToolSmokeRequest(request claudeDiagnosticRequest) DiagnosticSmokeRequestResult {
	return providerdiag.NewSkippedMultimodalSmokeRequest(request.multimodalSmokeRequest(), claudeDiagnosticDisabledToolSkipReason())
}

func claudeDiagnosticDisabledToolSkipReason() string {
	return fmt.Sprintf("Claude function calling payloads are disabled (%s=0)", claudeFunctionCallEnv)
}
