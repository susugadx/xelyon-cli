package clidoctor

import (
	"bytes"
	"strings"
	"testing"

	azureprovider "github.com/susugadx/xelyon-cli/internal/api/providers/azure"
	bedrockprovider "github.com/susugadx/xelyon-cli/internal/api/providers/bedrock"
	claudeprovider "github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	deepseekprovider "github.com/susugadx/xelyon-cli/internal/api/providers/deepseek"
	geminiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/gemini"
	groqprovider "github.com/susugadx/xelyon-cli/internal/api/providers/groq"
	kimiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/kimi"
	ollamaprovider "github.com/susugadx/xelyon-cli/internal/api/providers/ollama"
	openaiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	openrouterprovider "github.com/susugadx/xelyon-cli/internal/api/providers/openrouter"
)

type doctorSmokeJSONContractCase struct {
	provider string
	name     string
	render   func(*bytes.Buffer) error
	want     doctorSmokeJSONContract
}

type doctorSmokeJSONContract struct {
	route                    string
	responseID               string
	requestID                string
	duration                 string
	retentionPayload         bool
	usageObserved            bool
	usage                    *doctorJSONSmokeUsage
	cost                     *doctorJSONSmokeCost
	cachedInputTokens        int
	webSearchCallCount       int
	webSearchCallFeeEstimate float64
	webSearchUsageObserved   bool
	searchResultTotalTokens  int
	requests                 []doctorSmokeJSONRequestContract
	outputOmitted            []string
}

type doctorSmokeJSONRequestContract struct {
	name                     string
	ran                      bool
	skipped                  bool
	skipReasonContains       string
	toolPayload              bool
	imagePayload             bool
	webSearchPayload         bool
	retentionPayload         bool
	thinkingPayload          bool
	route                    string
	responseID               string
	requestID                string
	previousResponseID       string
	usageObserved            bool
	usage                    *doctorJSONSmokeUsage
	cost                     *doctorJSONSmokeCost
	webSearchCallCount       int
	webSearchCallFeeEstimate float64
	webSearchUsageObserved   bool
	searchResultTotalTokens  int
}

func TestDoctorSmokeJSONProviderContractMatrix(t *testing.T) {
	for _, tc := range doctorSmokeJSONContractCases() {
		t.Run(tc.caseName(), func(t *testing.T) {
			var out bytes.Buffer
			if err := tc.render(&out); err != nil {
				t.Fatalf("render %s doctor JSON error = %v", tc.provider, err)
			}
			for _, value := range tc.want.outputOmitted {
				if strings.Contains(out.String(), value) {
					t.Fatalf("rendered %s doctor JSON contains omitted substring %q:\n%s", tc.provider, value, out.String())
				}
			}
			smoke := unmarshalDoctorJSONSmoke(t, &out)
			requireDoctorSmokeJSONContract(t, smoke, tc.want)
		})
	}
}

func (tc doctorSmokeJSONContractCase) caseName() string {
	if tc.name != "" {
		return tc.name
	}
	return tc.provider
}

func doctorSmokeJSONContractCases() []doctorSmokeJSONContractCase {
	commonUsage := doctorSmokeJSONUsage(10, 3, 4, 2, 1)
	commonCost := doctorSmokeJSONCost(0.00012345, false)
	return []doctorSmokeJSONContractCase{
		{
			provider: "deepseek",
			render: func(out *bytes.Buffer) error {
				return renderDeepSeekDoctorJSON(out, deepseekprovider.DiagnosticReport{
					Provider: "deepseek",
					Smoke: &deepseekprovider.DiagnosticSmokeResult{
						Ran:           true,
						Route:         deepseekprovider.DiagnosticRouteChatCompletions,
						Duration:      "2ms",
						UsageObserved: true,
						Usage:         deepseekprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
						Cost:          deepseekprovider.DiagnosticSmokeCost{USD: 0.00012345},
						Requests: []deepseekprovider.DiagnosticSmokeRequestResult{
							{Name: "text", Ran: true, Route: deepseekprovider.DiagnosticRouteChatCompletions, UsageObserved: true, Usage: deepseekprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1}, Cost: deepseekprovider.DiagnosticSmokeCost{USD: 0.00012345}},
							{Name: "tool", Skipped: true, SkipReason: "DeepSeek function calling payloads are disabled", ToolPayload: true, Route: deepseekprovider.DiagnosticRouteChatCompletions},
						},
					},
				})
			},
			want: doctorSmokeJSONContract{
				route:         "chat_completions",
				duration:      "2ms",
				usageObserved: true,
				usage:         commonUsage,
				cost:          commonCost,
				requests: []doctorSmokeJSONRequestContract{
					{name: "text", ran: true, route: "chat_completions", usageObserved: true, usage: commonUsage, cost: commonCost},
					{name: "tool", skipped: true, skipReasonContains: "function calling payloads are disabled", toolPayload: true, route: "chat_completions"},
				},
			},
		},
		{
			provider: "groq",
			render: func(out *bytes.Buffer) error {
				return renderGroqDoctorJSON(out, groqprovider.DiagnosticReport{
					Provider: "groq",
					Smoke: &groqprovider.DiagnosticSmokeResult{
						Ran:           true,
						Route:         groqprovider.DiagnosticRouteChatCompletions,
						Duration:      "2ms",
						UsageObserved: true,
						Usage:         groqprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
						Cost:          groqprovider.DiagnosticSmokeCost{USD: 0.00012345},
						Requests: []groqprovider.DiagnosticSmokeRequestResult{
							{Name: "text", Ran: true, Route: groqprovider.DiagnosticRouteChatCompletions, UsageObserved: true, Usage: groqprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1}, Cost: groqprovider.DiagnosticSmokeCost{USD: 0.00012345}},
							{Name: "tool", Skipped: true, SkipReason: "Groq function calling payloads are disabled", ToolPayload: true, Route: groqprovider.DiagnosticRouteChatCompletions},
						},
					},
				})
			},
			want: doctorSmokeJSONContract{
				route:         "chat_completions",
				duration:      "2ms",
				usageObserved: true,
				usage:         commonUsage,
				cost:          commonCost,
				requests: []doctorSmokeJSONRequestContract{
					{name: "text", ran: true, route: "chat_completions", usageObserved: true, usage: commonUsage, cost: commonCost},
					{name: "tool", skipped: true, skipReasonContains: "function calling payloads are disabled", toolPayload: true, route: "chat_completions"},
				},
			},
		},
		{
			provider: "ollama",
			render: func(out *bytes.Buffer) error {
				return renderOllamaDoctorJSON(out, ollamaprovider.DiagnosticReport{
					Provider: "ollama",
					Smoke: &ollamaprovider.DiagnosticSmokeResult{
						Ran:           true,
						Route:         ollamaprovider.DiagnosticRouteOllamaChat,
						Duration:      "2ms",
						UsageObserved: true,
						Usage:         ollamaprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
						Cost:          ollamaprovider.DiagnosticSmokeCost{USD: 0},
						Requests: []ollamaprovider.DiagnosticSmokeRequestResult{
							{Name: "text", Ran: true, Route: ollamaprovider.DiagnosticRouteOllamaChat, UsageObserved: true, Usage: ollamaprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1}, Cost: ollamaprovider.DiagnosticSmokeCost{USD: 0}},
							{Name: "tool", Skipped: true, SkipReason: "Ollama function calling payloads are disabled", ToolPayload: true, Route: ollamaprovider.DiagnosticRouteOllamaChat},
						},
					},
				})
			},
			want: doctorSmokeJSONContract{
				route:         "ollama_chat",
				duration:      "2ms",
				usageObserved: true,
				usage:         commonUsage,
				cost:          doctorSmokeJSONCost(0, false),
				requests: []doctorSmokeJSONRequestContract{
					{name: "text", ran: true, route: "ollama_chat", usageObserved: true, usage: commonUsage, cost: doctorSmokeJSONCost(0, false)},
					{name: "tool", skipped: true, skipReasonContains: "function calling payloads are disabled", toolPayload: true, route: "ollama_chat"},
				},
			},
		},
		{
			provider: "openrouter",
			render: func(out *bytes.Buffer) error {
				return renderOpenRouterDoctorJSON(out, openrouterprovider.DiagnosticReport{
					Provider: "openrouter",
					Smoke: &openrouterprovider.DiagnosticSmokeResult{
						Ran:           true,
						Route:         openrouterprovider.DiagnosticRouteAnthropicMessages,
						Duration:      "2ms",
						UsageObserved: true,
						Usage:         openrouterprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
						Cost:          openrouterprovider.DiagnosticSmokeCost{USD: 0.00012345},
						Requests: []openrouterprovider.DiagnosticSmokeRequestResult{
							{Name: "text", Ran: true, Route: openrouterprovider.DiagnosticRouteAnthropicMessages, UsageObserved: true, Usage: openrouterprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1}, Cost: openrouterprovider.DiagnosticSmokeCost{USD: 0.00012345}},
							{Name: "tool", Skipped: true, SkipReason: "OpenRouter function calling payloads are disabled", ToolPayload: true, Route: openrouterprovider.DiagnosticRouteAnthropicMessages},
						},
					},
				})
			},
			want: doctorSmokeJSONContract{
				route:         "anthropic_messages",
				duration:      "2ms",
				usageObserved: true,
				usage:         commonUsage,
				cost:          commonCost,
				requests: []doctorSmokeJSONRequestContract{
					{name: "text", ran: true, route: "anthropic_messages", usageObserved: true, usage: commonUsage, cost: commonCost},
					{name: "tool", skipped: true, skipReasonContains: "function calling payloads are disabled", toolPayload: true, route: "anthropic_messages"},
				},
			},
		},
		{
			provider: "openai",
			name:     "openai/responses_streaming",
			render: func(out *bytes.Buffer) error {
				return renderOpenAIDoctorJSON(out, openaiprovider.DiagnosticReport{
					Provider: "openai",
					Smoke: &openaiprovider.DiagnosticSmokeResult{
						Ran:              true,
						Route:            openaiprovider.DiagnosticRouteResponsesStreaming,
						ResponseID:       "resp_summary",
						Duration:         "2ms",
						UsageObserved:    true,
						RetentionPayload: true,
						Usage:            openaiprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
						Cost:             openaiprovider.DiagnosticSmokeCost{USD: 0.00012345},
						Requests: []openaiprovider.DiagnosticSmokeRequestResult{
							{Name: "retention_initial", Ran: true, RetentionPayload: true, Route: openaiprovider.DiagnosticRouteResponsesStreaming, ResponseID: "resp_initial", UsageObserved: true, Usage: openaiprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1}, Cost: openaiprovider.DiagnosticSmokeCost{USD: 0.00012345}},
							{Name: "tool", Skipped: true, SkipReason: "OpenAI function calling payloads are disabled", ToolPayload: true, Route: openaiprovider.DiagnosticRouteResponsesStreaming},
						},
					},
				})
			},
			want: doctorSmokeJSONContract{
				route:            "responses_streaming",
				responseID:       "resp_summary",
				duration:         "2ms",
				retentionPayload: true,
				usageObserved:    true,
				usage:            commonUsage,
				cost:             commonCost,
				requests: []doctorSmokeJSONRequestContract{
					{name: "retention_initial", ran: true, retentionPayload: true, route: "responses_streaming", responseID: "resp_initial", usageObserved: true, usage: commonUsage, cost: commonCost},
					{name: "tool", skipped: true, skipReasonContains: "function calling payloads are disabled", toolPayload: true, route: "responses_streaming"},
				},
			},
		},
		{
			provider: "openai",
			name:     "openai/retention_non_streaming",
			render: func(out *bytes.Buffer) error {
				return renderOpenAIDoctorJSON(out, openaiprovider.DiagnosticReport{
					Provider: "openai",
					Smoke: &openaiprovider.DiagnosticSmokeResult{
						Ran:              true,
						Route:            openaiprovider.DiagnosticRouteResponsesNonStreaming,
						ResponseID:       "resp_json",
						Duration:         "2ms",
						UsageObserved:    true,
						RetentionPayload: true,
						Usage:            openaiprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
						Cost:             openaiprovider.DiagnosticSmokeCost{USD: 0.00012345},
						Requests: []openaiprovider.DiagnosticSmokeRequestResult{
							{Name: "text", Ran: true, Route: openaiprovider.DiagnosticRouteResponsesNonStreaming, ResponseID: "resp_json", UsageObserved: true, Usage: openaiprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1}, Cost: openaiprovider.DiagnosticSmokeCost{USD: 0.00012345}},
							{Name: "retention_followup", Ran: true, RetentionPayload: true, Route: openaiprovider.DiagnosticRouteResponsesNonStreaming, ResponseID: "resp_retention_followup", PreviousResponseID: "resp_json", UsageObserved: true, Usage: openaiprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1}, Cost: openaiprovider.DiagnosticSmokeCost{USD: 0.00012345}},
							{Name: "tool", Skipped: true, SkipReason: "OpenAI function calling payloads are disabled", ToolPayload: true, Route: openaiprovider.DiagnosticRouteResponsesNonStreaming},
						},
					},
				})
			},
			want: doctorSmokeJSONContract{
				route:            "responses_non_streaming",
				responseID:       "resp_json",
				duration:         "2ms",
				retentionPayload: true,
				usageObserved:    true,
				usage:            commonUsage,
				cost:             commonCost,
				requests: []doctorSmokeJSONRequestContract{
					{name: "text", ran: true, route: "responses_non_streaming", responseID: "resp_json", usageObserved: true, usage: commonUsage, cost: commonCost},
					{name: "retention_followup", ran: true, retentionPayload: true, route: "responses_non_streaming", responseID: "resp_retention_followup", previousResponseID: "resp_json", usageObserved: true, usage: commonUsage, cost: commonCost},
					{name: "tool", skipped: true, skipReasonContains: "function calling payloads are disabled", toolPayload: true, route: "responses_non_streaming"},
				},
			},
		},
		{
			provider: "azure",
			render: func(out *bytes.Buffer) error {
				return renderAzureDoctorJSON(out, azureprovider.DiagnosticReport{
					Provider: "azure",
					Smoke: &azureprovider.DiagnosticSmokeResult{
						Ran:              true,
						ResponseID:       "az_resp_summary",
						Duration:         "2ms",
						UsageObserved:    true,
						RetentionPayload: true,
						Usage:            azureprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
						Cost:             azureprovider.DiagnosticSmokeCost{USD: 0.00012345},
						Requests: []azureprovider.DiagnosticSmokeRequestResult{
							{Name: "retention_followup", Ran: true, RetentionPayload: true, ResponseID: "az_resp_followup", PreviousResponseID: "az_resp_initial", UsageObserved: true, Usage: azureprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1}, Cost: azureprovider.DiagnosticSmokeCost{USD: 0.00012345}},
							{Name: "tool", Skipped: true, SkipReason: "Azure OpenAI function calling payloads are disabled", ToolPayload: true},
						},
					},
				})
			},
			want: doctorSmokeJSONContract{
				responseID:       "az_resp_summary",
				duration:         "2ms",
				retentionPayload: true,
				usageObserved:    true,
				usage:            commonUsage,
				cost:             commonCost,
				requests: []doctorSmokeJSONRequestContract{
					{name: "retention_followup", ran: true, retentionPayload: true, responseID: "az_resp_followup", previousResponseID: "az_resp_initial", usageObserved: true, usage: commonUsage, cost: commonCost},
					{name: "tool", skipped: true, skipReasonContains: "function calling payloads are disabled", toolPayload: true},
				},
			},
		},
		{
			provider: "kimi",
			render: func(out *bytes.Buffer) error {
				return renderKimiDoctorJSON(out, kimiprovider.DiagnosticReport{
					Provider: "kimi",
					Smoke: &kimiprovider.DiagnosticSmokeResult{
						Ran:                      true,
						WebSearchPayload:         true,
						Duration:                 "2ms",
						UsageObserved:            true,
						CachedInputTokens:        7,
						WebSearchCallCount:       2,
						WebSearchCallFeeEstimate: 0.0100,
						WebSearchUsageObserved:   true,
						SearchResultTotalTokens:  55,
						Requests: []kimiprovider.DiagnosticSmokeRequestResult{
							{Name: "web_search_smoke", Ran: true, WebSearchPayload: true, UsageObserved: true, WebSearchCallCount: 2, WebSearchCallFeeEstimate: 0.0100, WebSearchUsageObserved: true, SearchResultTotalTokens: 55, Usage: kimiprovider.DiagnosticUsageObservation{CachedInputTokens: 7, WebSearchCallCount: 2, WebSearchCallFeeEstimate: 0.0100, SearchResultTotalTokens: 55}},
							{Name: "tool_smoke", Skipped: true, SkipReason: "Kimi function calling payloads are disabled", ToolPayload: true},
						},
					},
				})
			},
			want: doctorSmokeJSONContract{
				duration:                 "2ms",
				usageObserved:            true,
				cachedInputTokens:        7,
				webSearchCallCount:       2,
				webSearchCallFeeEstimate: 0.0100,
				webSearchUsageObserved:   true,
				searchResultTotalTokens:  55,
				requests: []doctorSmokeJSONRequestContract{
					{name: "web_search_smoke", ran: true, webSearchPayload: true, usageObserved: true, webSearchCallCount: 2, webSearchCallFeeEstimate: 0.0100, webSearchUsageObserved: true, searchResultTotalTokens: 55},
					{name: "tool_smoke", skipped: true, skipReasonContains: "function calling payloads are disabled", toolPayload: true},
				},
			},
		},
		{
			provider: "gemini",
			render: func(out *bytes.Buffer) error {
				return renderGeminiDoctorJSON(out, geminiprovider.DiagnosticReport{
					Provider: "gemini",
					Smoke: &geminiprovider.DiagnosticSmokeResult{
						Ran:           true,
						Route:         geminiprovider.DiagnosticRouteGenerateContent,
						Duration:      "2ms",
						UsageObserved: true,
						Usage:         geminiprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1, BillingServiceTier: "standard"},
						Cost:          geminiprovider.DiagnosticSmokeCost{USD: 0.00012345},
						Requests: []geminiprovider.DiagnosticSmokeRequestResult{
							{Name: "web_search", Ran: true, WebSearchPayload: true, Route: geminiprovider.DiagnosticRouteGenerateContent, UsageObserved: true, Usage: geminiprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1, BillingServiceTier: "standard"}, Cost: geminiprovider.DiagnosticSmokeCost{USD: 0.00012345}},
						},
					},
				})
			},
			want: doctorSmokeJSONContract{
				route:         "generate_content",
				duration:      "2ms",
				usageObserved: true,
				usage:         doctorSmokeJSONUsageWithBillingTier(10, 3, 4, 2, 1, "standard"),
				cost:          commonCost,
				requests: []doctorSmokeJSONRequestContract{
					{name: "web_search", ran: true, webSearchPayload: true, route: "generate_content", usageObserved: true, usage: doctorSmokeJSONUsageWithBillingTier(10, 3, 4, 2, 1, "standard"), cost: commonCost},
				},
			},
		},
		{
			provider: "claude",
			render: func(out *bytes.Buffer) error {
				return renderClaudeDoctorJSON(out, claudeprovider.DiagnosticReport{
					Provider: "claude",
					Smoke: &claudeprovider.DiagnosticSmokeResult{
						Ran:              true,
						Route:            claudeprovider.DiagnosticRouteClaudeMessages,
						Duration:         "2ms",
						ThinkingPayload:  true,
						WebSearchPayload: true,
						UsageObserved:    false,
						Usage:            claudeprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
						Cost:             claudeprovider.DiagnosticSmokeCost{USD: 0.00012345},
						Requests: []claudeprovider.DiagnosticSmokeRequestResult{
							{Name: "thinking", Ran: true, ThinkingPayload: true, Route: claudeprovider.DiagnosticRouteClaudeMessages, UsageObserved: true, Usage: claudeprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1}, Cost: claudeprovider.DiagnosticSmokeCost{USD: 0.00012345}},
							{Name: "tool", Skipped: true, SkipReason: "Claude function calling payloads are disabled", ToolPayload: true, Route: claudeprovider.DiagnosticRouteClaudeMessages},
						},
					},
				})
			},
			want: doctorSmokeJSONContract{
				route:         "claude_messages",
				duration:      "2ms",
				usageObserved: false,
				usage:         commonUsage,
				cost:          commonCost,
				requests: []doctorSmokeJSONRequestContract{
					{name: "thinking", ran: true, thinkingPayload: true, route: "claude_messages", usageObserved: true, usage: commonUsage, cost: commonCost},
					{name: "tool", skipped: true, skipReasonContains: "function calling payloads are disabled", toolPayload: true, route: "claude_messages"},
				},
			},
		},
		{
			provider: "bedrock",
			render: func(out *bytes.Buffer) error {
				return renderBedrockDoctorJSON(out, bedrockprovider.DiagnosticReport{
					Provider: "bedrock",
					Smoke: &bedrockprovider.DiagnosticSmokeResult{
						Ran:           true,
						UsageObserved: false,
						Usage:         bedrockprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
						Cost:          bedrockprovider.DiagnosticSmokeCost{USD: 0.00012345},
						Requests: []bedrockprovider.DiagnosticSmokeRequestResult{
							{Name: "text", Ran: true, RequestID: "req_text", UsageObserved: true, Usage: bedrockprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1}, Cost: bedrockprovider.DiagnosticSmokeCost{USD: 0.00012345}},
							{Name: "image", Skipped: true, SkipReason: "unsupported route", ImagePayload: true},
						},
					},
				})
			},
			want: doctorSmokeJSONContract{
				usageObserved: false,
				usage:         commonUsage,
				cost:          commonCost,
				requests: []doctorSmokeJSONRequestContract{
					{name: "text", ran: true, requestID: "req_text", usageObserved: true, usage: commonUsage, cost: commonCost},
					{name: "image", skipped: true, skipReasonContains: "unsupported route", imagePayload: true},
				},
				outputOmitted: []string{"response_id"},
			},
		},
	}
}
