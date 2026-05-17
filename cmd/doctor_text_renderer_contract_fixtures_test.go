package cmd

import (
	"bytes"

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

func renderDeepSeekDoctorTextContractFixture() string {
	report := deepseekprovider.DiagnosticReport{
		Provider:           "deepseek",
		APIURL:             "https://api.deepseek.com/chat/completions",
		Model:              "deepseek-v4-flash",
		ModelSource:        "test",
		APIModel:           "deepseek-v4-flash",
		CatalogModel:       "deepseek-v4-flash",
		CatalogModelSource: "test",
		Route:              deepseekprovider.DiagnosticRouteChatCompletions,
		RouteReason:        "contract fixture",
		ThinkingSupported:  true,
		ThinkingEnabled:    true,
		ThinkingType:       "enabled",
		Checks:             []deepseekprovider.DiagnosticCheck{{Name: "smoke", Status: deepseekprovider.DiagnosticStatusOK, Message: "ok"}},
		RequestPreview: &deepseekprovider.DiagnosticRequestPreview{Requests: []deepseekprovider.DiagnosticRequestPreviewRequest{{
			Name:    "text",
			Route:   deepseekprovider.DiagnosticRouteChatCompletions,
			Method:  "POST",
			URL:     "https://api.deepseek.com/chat/completions",
			Headers: map[string]string{"Authorization": "Bearer <redacted>"},
			Body:    map[string]any{"model": "deepseek-v4-flash"},
		}}},
		Smoke: &deepseekprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         deepseekprovider.DiagnosticRouteChatCompletions,
			Content:       "ok",
			Duration:      "1ms",
			UsageObserved: true,
			Usage:         deepseekprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
			Cost:          deepseekprovider.DiagnosticSmokeCost{USD: 0.00012345},
		},
	}
	var out bytes.Buffer
	renderDeepSeekDoctorText(&out, report)
	return out.String()
}

func renderGroqDoctorTextContractFixture() string {
	report := groqprovider.DiagnosticReport{
		Provider:           "groq",
		APIURL:             "https://api.groq.com/openai/v1/chat/completions",
		Model:              "meta-llama/llama-4-scout-17b-16e-instruct",
		ModelSource:        "test",
		CatalogModel:       "meta-llama/llama-4-scout-17b-16e-instruct",
		CatalogModelSource: "test",
		Route:              groqprovider.DiagnosticRouteChatCompletions,
		RouteReason:        "contract fixture",
		Checks:             []groqprovider.DiagnosticCheck{{Name: "smoke", Status: groqprovider.DiagnosticStatusOK, Message: "ok"}},
		RequestPreview: &groqprovider.DiagnosticRequestPreview{Requests: []groqprovider.DiagnosticRequestPreviewRequest{{
			Name:    "text",
			Route:   groqprovider.DiagnosticRouteChatCompletions,
			Method:  "POST",
			URL:     "https://api.groq.com/openai/v1/chat/completions",
			Headers: map[string]string{"Authorization": "Bearer <redacted>"},
			Body:    map[string]any{"model": "meta-llama/llama-4-scout-17b-16e-instruct"},
		}}},
		Smoke: &groqprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         groqprovider.DiagnosticRouteChatCompletions,
			Content:       "ok",
			Duration:      "1ms",
			UsageObserved: true,
			Usage:         groqprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
			Cost:          groqprovider.DiagnosticSmokeCost{USD: 0.00012345},
		},
	}
	var out bytes.Buffer
	renderGroqDoctorText(&out, report)
	return out.String()
}

func renderOllamaDoctorTextContractFixture() string {
	report := ollamaprovider.DiagnosticReport{
		Provider:           "ollama",
		APIURL:             "http://localhost:11434",
		Model:              "qwen2.5-coder:7b",
		ModelSource:        "test",
		CatalogModel:       "qwen2.5-coder:7b",
		CatalogModelSource: "test",
		Route:              ollamaprovider.DiagnosticRouteOllamaChat,
		RouteReason:        "contract fixture",
		Checks:             []ollamaprovider.DiagnosticCheck{{Name: "smoke", Status: ollamaprovider.DiagnosticStatusOK, Message: "ok"}},
		RequestPreview: &ollamaprovider.DiagnosticRequestPreview{Requests: []ollamaprovider.DiagnosticRequestPreviewRequest{{
			Name:    "text",
			Route:   ollamaprovider.DiagnosticRouteOllamaChat,
			Method:  "POST",
			URL:     "http://localhost:11434/api/chat",
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    map[string]any{"model": "qwen2.5-coder:7b"},
		}}},
		Smoke: &ollamaprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         ollamaprovider.DiagnosticRouteOllamaChat,
			Content:       "ok",
			Duration:      "1ms",
			UsageObserved: true,
			Usage:         ollamaprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
			Cost:          ollamaprovider.DiagnosticSmokeCost{USD: 0.00012345},
		},
	}
	var out bytes.Buffer
	renderOllamaDoctorText(&out, report)
	return out.String()
}

func renderOpenRouterDoctorTextContractFixture() string {
	report := openrouterprovider.DiagnosticReport{
		Provider:           "openrouter",
		APIURL:             "https://openrouter.ai/api/v1/messages",
		Model:              "anthropic/claude-sonnet-4.6",
		ModelSource:        "test",
		CatalogModel:       "anthropic/claude-sonnet-4.6",
		CatalogModelSource: "test",
		UpstreamProvider:   "anthropic",
		UpstreamModel:      "claude-sonnet-4.6",
		Route:              openrouterprovider.DiagnosticRouteAnthropicMessages,
		RouteReason:        "contract fixture",
		Checks:             []openrouterprovider.DiagnosticCheck{{Name: "smoke", Status: openrouterprovider.DiagnosticStatusOK, Message: "ok"}},
		RequestPreview: &openrouterprovider.DiagnosticRequestPreview{Requests: []openrouterprovider.DiagnosticRequestPreviewRequest{{
			Name:    "text",
			Route:   openrouterprovider.DiagnosticRouteAnthropicMessages,
			Method:  "POST",
			URL:     "https://openrouter.ai/api/v1/messages",
			Headers: map[string]string{"Authorization": "Bearer <redacted>"},
			Body:    map[string]any{"model": "anthropic/claude-sonnet-4.6"},
		}}},
		Smoke: &openrouterprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         openrouterprovider.DiagnosticRouteAnthropicMessages,
			Content:       "ok",
			Duration:      "1ms",
			UsageObserved: true,
			Usage:         openrouterprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
			Cost:          openrouterprovider.DiagnosticSmokeCost{USD: 0.00012345},
		},
	}
	var out bytes.Buffer
	renderOpenRouterDoctorText(&out, report)
	return out.String()
}

func renderOpenAIDoctorTextContractFixture() string {
	report := openaiprovider.DiagnosticReport{
		Provider:           "openai",
		APIURL:             "https://api.openai.com/v1/chat/completions",
		ResponsesURL:       "https://api.openai.com/v1/responses",
		Model:              "gpt-5.4",
		ModelSource:        "test",
		CatalogModel:       "gpt-5.4",
		CatalogModelSource: "test",
		Route:              openaiprovider.DiagnosticRouteResponsesStreaming,
		RouteReason:        "contract fixture",
		Checks:             []openaiprovider.DiagnosticCheck{{Name: "smoke", Status: openaiprovider.DiagnosticStatusOK, Message: "ok"}},
		RequestPreview: &openaiprovider.DiagnosticRequestPreview{Requests: []openaiprovider.DiagnosticRequestPreviewRequest{{
			Name:    "text",
			Route:   openaiprovider.DiagnosticRouteResponsesStreaming,
			Method:  "POST",
			URL:     "https://api.openai.com/v1/responses",
			Headers: map[string]string{"Authorization": "Bearer <redacted>"},
			Body:    map[string]any{"model": "gpt-5.4"},
		}}},
		Smoke: &openaiprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         openaiprovider.DiagnosticRouteResponsesStreaming,
			ResponseID:    "resp_text",
			Content:       "ok",
			Duration:      "1ms",
			UsageObserved: true,
			Usage:         openaiprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
			Cost:          openaiprovider.DiagnosticSmokeCost{USD: 0.00012345},
		},
	}
	var out bytes.Buffer
	renderOpenAIDoctorText(&out, report)
	return out.String()
}

func renderAzureDoctorTextContractFixture() string {
	report := azureprovider.DiagnosticReport{
		Provider:          "azure",
		NormalizedBaseURL: "https://example.openai.azure.com/openai/v1",
		AuthMode:          "api_key",
		Deployment:        "corp-azure-gpt55",
		CatalogModel:      "gpt-5.5-pro",
		Route:             azureprovider.DiagnosticRouteResponsesNonStreaming,
		RouteReason:       "contract fixture",
		Checks:            []azureprovider.DiagnosticCheck{{Name: "smoke", Status: azureprovider.DiagnosticStatusOK, Message: "ok"}},
		RequestPreview: &azureprovider.DiagnosticRequestPreview{Requests: []azureprovider.DiagnosticRequestPreviewRequest{{
			Name:    "text",
			Route:   azureprovider.DiagnosticRouteResponsesNonStreaming,
			Method:  "POST",
			URL:     "https://example.openai.azure.com/openai/v1/responses",
			Headers: map[string]string{"api-key": "<redacted>"},
			Body:    map[string]any{"model": "corp-azure-gpt55"},
		}}},
		Smoke: &azureprovider.DiagnosticSmokeResult{
			Ran:           true,
			ResponseID:    "az_resp_text",
			Content:       "ok",
			Duration:      "1ms",
			UsageObserved: true,
			Usage:         azureprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
			Cost:          azureprovider.DiagnosticSmokeCost{USD: 0.00012345},
		},
	}
	var out bytes.Buffer
	renderAzureDoctorText(&out, report)
	return out.String()
}

func renderKimiDoctorTextContractFixture() string {
	report := kimiprovider.DiagnosticReport{
		Provider:           "kimi",
		APIURL:             "https://api.moonshot.ai/v1/chat/completions",
		Model:              "kimi-k2.6",
		ModelSource:        "test",
		CatalogModel:       "kimi-k2.6",
		CatalogModelSource: "test",
		Route:              "chat_completions",
		RouteReason:        "contract fixture",
		Checks:             []kimiprovider.DiagnosticCheck{{Name: "smoke", Status: kimiprovider.DiagnosticStatusOK, Message: "ok"}},
		RequestPreview: &kimiprovider.DiagnosticRequestPreview{Requests: []kimiprovider.DiagnosticRequestPreviewRequest{{
			Name:    "web_search_smoke",
			Route:   "chat_completions_web_search",
			Method:  "POST",
			URL:     "https://api.moonshot.ai/v1/chat/completions",
			Headers: map[string]string{"Authorization": "Bearer <redacted>"},
			Body:    map[string]any{"model": "kimi-k2.6"},
		}}},
		Smoke: &kimiprovider.DiagnosticSmokeResult{
			Ran:                      true,
			WebSearchPayload:         true,
			Content:                  "ok",
			Duration:                 "1ms",
			UsageObserved:            true,
			CachedInputTokens:        7,
			WebSearchCallCount:       2,
			WebSearchCallFeeEstimate: 0.0100,
			WebSearchUsageObserved:   true,
			SearchResultTotalTokens:  55,
			Requests: []kimiprovider.DiagnosticSmokeRequestResult{
				{Name: "web_search_smoke", Ran: true, WebSearchPayload: true, Duration: "1ms"},
				{Name: "tool_smoke", Skipped: true, SkipReason: "Kimi function calling payloads are disabled", ToolPayload: true},
			},
		},
	}
	var out bytes.Buffer
	renderKimiDoctorText(&out, report)
	return out.String()
}

func renderGeminiDoctorTextContractFixture() string {
	report := geminiprovider.DiagnosticReport{
		Provider:               "gemini",
		APIURL:                 "https://generativelanguage.googleapis.com/v1beta/models/gemini-3.1-pro-preview-customtools:streamGenerateContent?alt=sse",
		Model:                  "gemini-3.1-pro-preview-customtools",
		ModelSource:            "test",
		CatalogModel:           "gemini-3.1-pro-preview-customtools",
		CatalogModelSource:     "test",
		Route:                  geminiprovider.DiagnosticRouteStreamGenerateContentSSE,
		RouteReason:            "contract fixture",
		FunctionCallingEnabled: true,
		ImageInputSupported:    true,
		WebSearchSupported:     true,
		ContextCachingEnabled:  true,
		Checks:                 []geminiprovider.DiagnosticCheck{{Name: "smoke", Status: geminiprovider.DiagnosticStatusOK, Message: "ok"}},
		RequestPreview: &geminiprovider.DiagnosticRequestPreview{Requests: []geminiprovider.DiagnosticRequestPreviewRequest{{
			Name:    "text",
			Route:   geminiprovider.DiagnosticRouteStreamGenerateContentSSE,
			Method:  "POST",
			URL:     "https://example.test/gemini",
			Headers: map[string]string{"x-goog-api-key": "<redacted>"},
			Body:    map[string]any{"model": "gemini-3.1-pro-preview-customtools"},
		}}},
		Smoke: &geminiprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         geminiprovider.DiagnosticRouteStreamGenerateContentSSE,
			Content:       "ok",
			Duration:      "1ms",
			UsageObserved: true,
			Usage:         geminiprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
			Cost:          geminiprovider.DiagnosticSmokeCost{USD: 0.00012345},
		},
	}
	var out bytes.Buffer
	renderGeminiDoctorText(&out, report)
	return out.String()
}

func renderClaudeDoctorTextContractFixture() string {
	report := claudeprovider.DiagnosticReport{
		Provider:                  "claude",
		APIURL:                    "https://api.anthropic.com/v1/messages",
		Model:                     "claude-sonnet-4-6",
		ModelSource:               "test",
		CatalogModel:              "claude-sonnet-4-6",
		CatalogModelSource:        "test",
		Route:                     claudeprovider.DiagnosticRouteClaudeMessages,
		RouteReason:               "contract fixture",
		FunctionCallingEnabled:    true,
		ImageInputSupported:       true,
		WebSearchSupported:        true,
		ThinkingEnabled:           true,
		ThinkingType:              "adaptive",
		ContextManagementEnabled:  true,
		ClaudeCompactionSupported: true,
		AnthropicVersion:          "2023-06-01",
		Checks:                    []claudeprovider.DiagnosticCheck{{Name: "smoke", Status: claudeprovider.DiagnosticStatusOK, Message: "ok"}},
		RequestPreview: &claudeprovider.DiagnosticRequestPreview{Requests: []claudeprovider.DiagnosticRequestPreviewRequest{{
			Name:    "text",
			Route:   claudeprovider.DiagnosticRouteClaudeMessages,
			Method:  "POST",
			URL:     "https://api.anthropic.com/v1/messages",
			Headers: map[string]string{"x-api-key": "<redacted>"},
			Body:    map[string]any{"model": "claude-sonnet-4-6"},
		}}},
		Smoke: &claudeprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         claudeprovider.DiagnosticRouteClaudeMessages,
			Content:       "ok",
			Duration:      "1ms",
			UsageObserved: true,
			Usage:         claudeprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
			Cost:          claudeprovider.DiagnosticSmokeCost{USD: 0.00012345},
		},
	}
	var out bytes.Buffer
	renderClaudeDoctorText(&out, report)
	return out.String()
}

func renderBedrockDoctorTextContractFixture() string {
	report := bedrockprovider.DiagnosticReport{
		Provider:           "bedrock",
		Region:             "us-east-1",
		Model:              bedrockDoctorCatalogModelForTest,
		ModelSource:        "test",
		CatalogModel:       bedrockDoctorCatalogModelForTest,
		CatalogModelSource: "test",
		Route:              "claude_messages",
		Checks:             []bedrockprovider.DiagnosticCheck{{Name: "smoke", Status: bedrockprovider.DiagnosticStatusOK, Message: "ok"}},
		RequestPreview: &bedrockprovider.DiagnosticRequestPreview{Requests: []bedrockprovider.DiagnosticRequestPreviewRequest{{
			Name:      "text",
			Route:     "claude_messages",
			Operation: "invoke_model_with_response_stream",
			ModelID:   bedrockDoctorCatalogModelForTest,
			Method:    "POST",
			URL:       "https://bedrock-runtime.us-east-1.amazonaws.com/model/global.anthropic.claude-sonnet-4-6/invoke-with-response-stream",
			Headers:   map[string]string{"Authorization": "<redacted: AWS SigV4>"},
			Body:      map[string]any{"anthropic_version": "bedrock-2023-05-31"},
		}}},
		Smoke: &bedrockprovider.DiagnosticSmokeResult{
			Ran: true,
			Requests: []bedrockprovider.DiagnosticSmokeRequestResult{{
				Name:          "text",
				Ran:           true,
				RequestID:     "req_text",
				Content:       "ok",
				Duration:      "1ms",
				UsageObserved: true,
				Usage:         bedrockprovider.DiagnosticSmokeUsage{InputTokens: 10, CachedInputTokens: 3, OutputTokens: 4, ThinkingTokens: 2, CacheCreationTokens: 1},
				Cost:          bedrockprovider.DiagnosticSmokeCost{USD: 0.00012345},
			}},
		},
	}
	var out bytes.Buffer
	renderBedrockDoctorText(&out, report)
	return out.String()
}
