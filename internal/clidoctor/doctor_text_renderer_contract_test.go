package clidoctor

import "testing"

type doctorTextRendererContractCase struct {
	provider string
	render   func() string
	wants    []string
	omits    []string
}

func TestDoctorTextRendererProviderContractMatrix(t *testing.T) {
	for _, tc := range doctorTextRendererContractCases() {
		t.Run(tc.provider, func(t *testing.T) {
			output := tc.render()
			requireDoctorContractTextContainsAll(t, output, tc.wants)
			requireDoctorContractTextOmitsAll(t, output, tc.omits)
		})
	}
}

func doctorTextRendererContractCases() []doctorTextRendererContractCase {
	commonSmoke := []string{
		"Request preview:",
		"Smoke usage: input=10 cached=3 output=4 reasoning=2 cache_creation=1",
		"Smoke cost estimate: $0.00012345 USD",
	}
	return []doctorTextRendererContractCase{
		{
			provider: "deepseek",
			render:   renderDeepSeekDoctorTextContractFixture,
			wants: append([]string{
				"DeepSeek doctor",
				"Status: OK",
				"Model: deepseek-v4-flash (test)",
				"API model: deepseek-v4-flash",
				"Catalog model: deepseek-v4-flash (test)",
				"Route: chat_completions",
				"Thinking: supported=true enabled=true type=enabled",
				`"Authorization": "Bearer <redacted>"`,
				"Smoke route: chat_completions",
			}, commonSmoke...),
			omits: []string{"sk-test"},
		},
		{
			provider: "groq",
			render:   renderGroqDoctorTextContractFixture,
			wants: append([]string{
				"Groq doctor",
				"Status: OK",
				"Model: meta-llama/llama-4-scout-17b-16e-instruct (test)",
				"Catalog model: meta-llama/llama-4-scout-17b-16e-instruct (test)",
				"Route: chat_completions",
				`"Authorization": "Bearer <redacted>"`,
				"Smoke route: chat_completions",
			}, commonSmoke...),
			omits: []string{"gsk-test"},
		},
		{
			provider: "ollama",
			render:   renderOllamaDoctorTextContractFixture,
			wants: append([]string{
				"Ollama doctor",
				"Status: OK",
				"Model: qwen2.5-coder:7b (test)",
				"Catalog model: qwen2.5-coder:7b (test)",
				"Route: ollama_chat",
				`"Content-Type": "application/json"`,
				"Smoke route: ollama_chat",
			}, commonSmoke...),
		},
		{
			provider: "openrouter",
			render:   renderOpenRouterDoctorTextContractFixture,
			wants: append([]string{
				"OpenRouter doctor",
				"Status: OK",
				"Model: anthropic/claude-sonnet-4.6 (test)",
				"Catalog model: anthropic/claude-sonnet-4.6 (test)",
				"Upstream model: anthropic/claude-sonnet-4.6",
				"Route: anthropic_messages",
				`"Authorization": "Bearer <redacted>"`,
				"Smoke route: anthropic_messages",
			}, commonSmoke...),
			omits: []string{"sk-or-test"},
		},
		{
			provider: "openai",
			render:   renderOpenAIDoctorTextContractFixture,
			wants: append([]string{
				"OpenAI doctor",
				"Status: OK",
				"Model: gpt-5.4 (test)",
				"Catalog model: gpt-5.4 (test)",
				"Route: responses_streaming",
				"Responses URL: https://api.openai.com/v1/responses",
				`"Authorization": "Bearer <redacted>"`,
				"Smoke response ID: resp_text",
				"Smoke route: responses_streaming",
			}, commonSmoke...),
			omits: []string{"sk-test"},
		},
		{
			provider: "azure",
			render:   renderAzureDoctorTextContractFixture,
			wants: append([]string{
				"Azure OpenAI doctor",
				"Status: OK",
				"Route: responses_non_streaming",
				`"api-key": "<redacted>"`,
				"Smoke response ID: az_resp_text",
			}, []string{
				"Request preview:",
				"Smoke usage: input=10 cached=3 output=4 reasoning=2 cache_creation=1",
				"Smoke cost estimate: $0.00012345 USD",
			}...),
			omits: []string{"azure-key"},
		},
		{
			provider: "kimi",
			render:   renderKimiDoctorTextContractFixture,
			wants: []string{
				"Kimi doctor",
				"Status: OK",
				"Model: kimi-k2.6 (test)",
				"Catalog model: kimi-k2.6 (test)",
				"Route: chat_completions",
				"Request preview:",
				`"Authorization": "Bearer <redacted>"`,
				"Smoke duration: 1ms",
				"Smoke request web_search_smoke: ok duration=1ms",
				"Smoke request tool_smoke: skipped (Kimi function calling payloads are disabled)",
				"Cached input tokens observed: 7",
				"Web search call count: 2",
				"Web search call fee estimate: $0.0100 USD",
				"Web search usage observed: true",
				"Search result total tokens observed: 55",
			},
			omits: []string{"moonshot-key"},
		},
		{
			provider: "gemini",
			render:   renderGeminiDoctorTextContractFixture,
			wants: append([]string{
				"Gemini doctor",
				"Status: OK",
				"Model: gemini-3.1-pro-preview-customtools (test)",
				"Catalog model: gemini-3.1-pro-preview-customtools (test)",
				"Route: stream_generate_content_sse",
				"Capabilities: function_calling=true image_input=true web_search=true context_caching=true thinking=false",
				`"x-goog-api-key": "<redacted>"`,
				"Smoke route: stream_generate_content_sse",
			}, commonSmoke...),
			omits: []string{"gemini-key"},
		},
		{
			provider: "claude",
			render:   renderClaudeDoctorTextContractFixture,
			wants: append([]string{
				"Claude doctor",
				"Status: OK",
				"Model: claude-sonnet-4-6 (test)",
				"Catalog model: claude-sonnet-4-6 (test)",
				"Route: claude_messages",
				"Capabilities: function_calling=true image_input=true web_search=true thinking=true context_management=true claude_compaction=true thinking_type=adaptive",
				"Anthropic version: 2023-06-01",
				`"x-api-key": "<redacted>"`,
				"Smoke route: claude_messages",
			}, commonSmoke...),
			omits: []string{"claude-key"},
		},
		{
			provider: "bedrock",
			render:   renderBedrockDoctorTextContractFixture,
			wants: []string{
				"Bedrock doctor",
				"Status: OK",
				"Region: us-east-1",
				"Model: global.anthropic.claude-sonnet-4-6 (test)",
				"Catalog model: global.anthropic.claude-sonnet-4-6 (test)",
				"Route: claude_messages",
				"Request preview:",
				`"Authorization": "<redacted: AWS SigV4>"`,
				"Smoke request text: ok duration=1ms request_id=req_text",
				"Smoke usage text: input=10 cached=3 output=4 reasoning=2 cache_creation=1",
				"Smoke cost estimate text: $0.00012345 USD",
			},
			omits: []string{"test-secret-key"},
		},
	}
}
