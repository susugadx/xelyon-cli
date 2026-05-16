package bedrock

const defaultBedrockDiagnosticSmokeMaxOutputTokens = 64

const (
	bedrockDiagnosticTextRequestName     = "text"
	bedrockDiagnosticToolRequestName     = "tool"
	bedrockDiagnosticImageRequestName    = "image"
	bedrockDiagnosticThinkingRequestName = "thinking"
)

type bedrockDiagnosticRequestPlan struct {
	Requests []bedrockDiagnosticSmokeRequest
}

type bedrockDiagnosticSmokeRequest struct {
	Name            string
	SystemPrompt    string
	UserContent     string
	ToolPayload     bool
	ImagePayload    bool
	ThinkingEnabled bool
}

func buildBedrockDiagnosticRequestPlan(options DiagnosticOptions) bedrockDiagnosticRequestPlan {
	var requests []bedrockDiagnosticSmokeRequest
	if options.TextSmoke {
		requests = append(requests, bedrockDiagnosticSmokeRequest{
			Name:         bedrockDiagnosticTextRequestName,
			SystemPrompt: "Reply briefly.",
			UserContent:  "Reply with: xelyon bedrock doctor ok",
		})
	}
	if options.ToolSmoke {
		requests = append(requests, bedrockDiagnosticSmokeRequest{
			Name:         bedrockDiagnosticToolRequestName,
			SystemPrompt: "Use the diagnostic tool.",
			UserContent:  `Call xelyon_bedrock_doctor_probe exactly once with {"value":"bedrock-tool-ok"} and do not answer in prose.`,
			ToolPayload:  true,
		})
	}
	if options.ImageSmoke {
		requests = append(requests, bedrockDiagnosticSmokeRequest{
			Name:         bedrockDiagnosticImageRequestName,
			SystemPrompt: "Reply briefly.",
			UserContent:  "Look at the attached tiny diagnostic image and reply with a short non-empty response.",
			ImagePayload: true,
		})
	}
	if options.ThinkingSmoke {
		requests = append(requests, bedrockDiagnosticSmokeRequest{
			Name:            bedrockDiagnosticThinkingRequestName,
			SystemPrompt:    "Think briefly, then reply briefly.",
			UserContent:     "Reply with: xelyon bedrock thinking ok",
			ThinkingEnabled: true,
		})
	}
	return bedrockDiagnosticRequestPlan{Requests: requests}
}

func (p bedrockDiagnosticRequestPlan) UsesToolPayload() bool {
	for _, request := range p.Requests {
		if request.ToolPayload {
			return true
		}
	}
	return false
}

func bedrockDiagnosticRequestMaxOutputTokens(options DiagnosticOptions) int {
	if options.MaxOutputTokens > 0 {
		return options.MaxOutputTokens
	}
	return defaultBedrockDiagnosticSmokeMaxOutputTokens
}

func bedrockDiagnosticSmokeSkipReason(report DiagnosticReport, request bedrockDiagnosticSmokeRequest) (string, bool) {
	if request.ToolPayload && !report.FunctionCallingEnabled {
		return "Bedrock function calling payloads are disabled (BEDROCK_FUNCTION_CALLING=0)", true
	}
	if report.Route == string(bedrockRouteConverseStream) && (request.ImagePayload || request.ThinkingEnabled) {
		return "Bedrock ConverseStream route does not support image or thinking smoke", true
	}
	return "", false
}

func newBedrockDiagnosticSkippedSmokeRequest(request bedrockDiagnosticSmokeRequest, skipReason string) DiagnosticSmokeRequestResult {
	return DiagnosticSmokeRequestResult{
		Name:            request.Name,
		Skipped:         true,
		SkipReason:      skipReason,
		ToolPayload:     request.ToolPayload,
		ImagePayload:    request.ImagePayload,
		ThinkingEnabled: request.ThinkingEnabled,
	}
}
