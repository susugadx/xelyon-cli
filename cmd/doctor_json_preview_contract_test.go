package cmd

import (
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
)

type doctorJSONPreviewContractCase struct {
	provider           string
	newCommand         func() *cobra.Command
	run                func(*cobra.Command, []string) error
	setup              func(*testing.T, *cobra.Command)
	requiredJSONFields []string
	want               doctorJSONPreviewContractIdentity
	previewCount       int
	previewRequests    []doctorJSONPreviewRequestContract
}

type doctorJSONPreviewContractIdentity struct {
	model                 string
	modelSource           string
	deployment            string
	catalogModel          string
	catalogModelSource    string
	route                 string
	apiURLContains        []string
	responsesURLContains  []string
	normalizedURLContains []string
	region                string
	requiredChecks        []string
	omittedChecks         []string
}

type doctorJSONPreviewRequestContract struct {
	name                 string
	route                string
	operation            string
	modelID              string
	method               string
	toolPayload          bool
	imagePayload         bool
	webSearchPayload     bool
	retentionPayload     bool
	thinkingEnabled      bool
	previousResponseID   bool
	urlContains          []string
	headers              map[string]string
	bodyContains         []string
	bodyOmittedTopFields []string
}

func TestDoctorJSONPrintRequestProviderContractMatrix(t *testing.T) {
	for _, tc := range doctorJSONPreviewContractCases() {
		t.Run(tc.provider, func(t *testing.T) {
			cmd, out := newDoctorSubcommandTest(t, tc.newCommand)
			tc.setup(t, cmd)
			setDoctorCommandFlag(t, cmd, "print-request", "true")
			setDoctorCommandFlag(t, cmd, "json", "true")

			if err := tc.run(cmd, nil); err != nil {
				t.Fatalf("run doctor %s --json --print-request error = %v\noutput:\n%s", tc.provider, err, out.String())
			}

			raw := unmarshalDoctorJSON[map[string]json.RawMessage](t, out)
			report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
			requireDoctorJSONFields(t, raw, append([]string{
				"provider",
				"route",
				"checks",
				"request_preview",
			}, tc.requiredJSONFields...)...)
			requireDoctorJSONFieldsOmitted(t, raw, "smoke")
			requireDoctorJSONPreviewIdentity(t, report, tc.want)
			requireDoctorJSONPrintRequestOmittedSmoke(t, report.Smoke)
			for _, check := range tc.want.requiredChecks {
				requireDoctorJSONCheckStatus(t, requireDoctorJSONCheck(t, report.Checks, check), "ok")
			}
			requireNoDoctorJSONChecks(t, report.Checks, tc.want.omittedChecks...)
			requireDoctorJSONPreviewRequests(t, report.RequestPreview, tc.previewCount, tc.previewRequests)
		})
	}
}

func doctorJSONPreviewContractCases() []doctorJSONPreviewContractCase {
	openAICompatibleRequiredFields := []string{
		"api_url",
		"model",
		"model_source",
		"catalog_model",
		"catalog_model_source",
		"route_reason",
		"max_output_tokens",
		"function_calling_enabled",
	}
	openAICompatibleChecks := []string{
		"provider_registration",
		"model",
		"catalog_model",
		"route",
		"catalog_policy",
		"function_calling",
		"request_preview",
	}
	redactedBearerHeaders := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer <redacted>",
	}
	return []doctorJSONPreviewContractCase{
		{
			provider:           "deepseek",
			newCommand:         newDeepSeekDoctorCommand,
			run:                runDeepSeekDoctorInvocation,
			setup:              setupDeepSeekJSONPreviewContract,
			requiredJSONFields: append([]string{"api_model", "thinking_supported", "thinking_enabled"}, openAICompatibleRequiredFields...),
			want: doctorJSONPreviewContractIdentity{
				model:              "deepseek-v4-flash",
				modelSource:        "--model",
				catalogModel:       "deepseek-v4-flash",
				catalogModelSource: "--catalog-model",
				route:              "chat_completions",
				apiURLContains:     []string{"/chat/completions"},
				requiredChecks:     openAICompatibleChecks,
				omittedChecks:      []string{"auth"},
			},
			previewCount: 1,
			previewRequests: []doctorJSONPreviewRequestContract{{
				name:         "tool",
				route:        "chat_completions",
				method:       "POST",
				toolPayload:  true,
				urlContains:  []string{"/chat/completions"},
				headers:      redactedBearerHeaders,
				bodyContains: []string{`"model":"deepseek-v4-flash"`, "xelyon_deepseek_doctor_probe", `"thinking":{"type":"disabled"}`},
			}},
		},
		{
			provider:           "groq",
			newCommand:         newGroqDoctorCommand,
			run:                runGroqDoctorInvocation,
			setup:              setupGroqJSONPreviewContract,
			requiredJSONFields: append([]string{"context_window_tokens"}, openAICompatibleRequiredFields...),
			want: doctorJSONPreviewContractIdentity{
				model:              "corp-groq-model",
				modelSource:        "--model",
				catalogModel:       "meta-llama/llama-4-scout-17b-16e-instruct",
				catalogModelSource: "--catalog-model",
				route:              "chat_completions",
				apiURLContains:     []string{"/openai/v1/chat/completions"},
				requiredChecks:     openAICompatibleChecks,
				omittedChecks:      []string{"auth"},
			},
			previewCount: 1,
			previewRequests: []doctorJSONPreviewRequestContract{{
				name:        "tool",
				route:       "chat_completions",
				method:      "POST",
				toolPayload: true,
				urlContains: []string{"/openai/v1/chat/completions"},
				headers:     redactedBearerHeaders,
				bodyContains: []string{
					`"model":"corp-groq-model"`,
					"xelyon_groq_doctor_probe",
					`"max_tokens":64`,
				},
			}},
		},
		{
			provider:           "ollama",
			newCommand:         newOllamaDoctorCommand,
			run:                runOllamaDoctorInvocation,
			setup:              setupOllamaJSONPreviewContract,
			requiredJSONFields: []string{"api_url", "model", "model_source", "catalog_model", "catalog_model_source", "route_reason", "function_calling_enabled"},
			want: doctorJSONPreviewContractIdentity{
				model:              "qwen2.5-coder:7b",
				modelSource:        "--model",
				catalogModel:       "qwen2.5-coder:7b",
				catalogModelSource: "--catalog-model",
				route:              "ollama_chat",
				apiURLContains:     []string{"http://127.0.0.1:11434"},
				requiredChecks:     []string{"auth", "endpoint", "provider_registration", "catalog_policy", "function_calling", "request_preview"},
			},
			previewCount: 1,
			previewRequests: []doctorJSONPreviewRequestContract{{
				name:        "tool",
				route:       "ollama_chat",
				method:      "POST",
				toolPayload: true,
				urlContains: []string{"http://127.0.0.1:11434/api/chat"},
				headers:     map[string]string{"Content-Type": "application/json"},
				bodyContains: []string{
					`"model":"qwen2.5-coder:7b"`,
					`"stream":true`,
					"xelyon_ollama_doctor_probe",
				},
			}},
		},
		{
			provider:           "openrouter",
			newCommand:         newOpenRouterDoctorCommand,
			run:                runOpenRouterDoctorInvocation,
			setup:              setupOpenRouterJSONPreviewContract,
			requiredJSONFields: append([]string{"upstream_provider", "upstream_model"}, openAICompatibleRequiredFields...),
			want: doctorJSONPreviewContractIdentity{
				model:              "anthropic/claude-sonnet-4.6",
				modelSource:        "--model",
				catalogModel:       "anthropic/claude-sonnet-4.6",
				catalogModelSource: "--catalog-model",
				route:              "anthropic_messages",
				apiURLContains:     []string{"/api/v1/messages"},
				requiredChecks:     append(openAICompatibleChecks, "image_input"),
				omittedChecks:      []string{"auth"},
			},
			previewCount: 1,
			previewRequests: []doctorJSONPreviewRequestContract{{
				name:        "tool",
				route:       "anthropic_messages",
				method:      "POST",
				toolPayload: true,
				urlContains: []string{"/api/v1/messages"},
				headers: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer <redacted>",
					"X-Title":       "XELYON CLI",
				},
				bodyContains: []string{
					`"model":"anthropic/claude-sonnet-4.6"`,
					"xelyon_openrouter_doctor_probe",
					`"context_management"`,
				},
			}},
		},
		{
			provider:           "openai",
			newCommand:         newOpenAIDoctorCommand,
			run:                runOpenAIDoctorInvocation,
			setup:              setupOpenAIJSONPreviewContract,
			requiredJSONFields: []string{"api_url", "responses_url", "model", "model_source", "catalog_model", "catalog_model_source", "route_reason", "max_output_tokens", "function_calling_enabled"},
			want: doctorJSONPreviewContractIdentity{
				model:                "corp-openai-responses",
				modelSource:          "--model",
				catalogModel:         "gpt-5.4",
				catalogModelSource:   "--catalog-model",
				route:                "responses_streaming",
				responsesURLContains: []string{"/v1/responses"},
				requiredChecks:       []string{"api_url", "responses_url", "provider_registration", "model", "route", "catalog_policy", "function_calling", "request_preview"},
				omittedChecks:        []string{"auth"},
			},
			previewCount: 1,
			previewRequests: []doctorJSONPreviewRequestContract{{
				name:        "tool",
				route:       "responses_streaming",
				method:      "POST",
				toolPayload: true,
				urlContains: []string{"/v1/responses"},
				headers:     redactedBearerHeaders,
				bodyContains: []string{
					`"model":"corp-openai-responses"`,
					"xelyon_openai_doctor_probe",
					`"max_output_tokens":64`,
				},
			}},
		},
		{
			provider:           "azure",
			newCommand:         newAzureDoctorCommand,
			run:                runAzureDoctorInvocation,
			setup:              setupAzureJSONPreviewContract,
			requiredJSONFields: []string{"normalized_base_url", "auth_mode", "deployment", "deployment_source", "catalog_model", "catalog_model_source", "route_reason", "function_calling_enabled"},
			want: doctorJSONPreviewContractIdentity{
				deployment:            "corp-azure-gpt55",
				catalogModel:          "gpt-5.5-pro",
				catalogModelSource:    "--catalog-model",
				route:                 "responses_non_streaming",
				normalizedURLContains: []string{"https://example.openai.azure.com/openai/v1"},
				requiredChecks:        []string{"base_url", "deployment", "catalog_model", "route", "catalog_policy", "function_calling", "request_preview"},
				omittedChecks:         []string{"auth"},
			},
			previewCount: 2,
			previewRequests: []doctorJSONPreviewRequestContract{{
				name:               "retention_followup",
				route:              "responses_non_streaming",
				method:             "POST",
				retentionPayload:   true,
				previousResponseID: true,
				urlContains:        []string{"https://example.openai.azure.com/openai/v1/responses"},
				headers:            map[string]string{"Content-Type": "application/json", "api-key": "<redacted>"},
				bodyContains: []string{
					`"model":"corp-azure-gpt55"`,
					`"store":true`,
					`"previous_response_id":"${retention_initial.response_id}"`,
				},
			}},
		},
		{
			provider:           "kimi",
			newCommand:         newKimiDoctorCommand,
			run:                runKimiDoctorInvocation,
			setup:              setupKimiJSONPreviewContract,
			requiredJSONFields: []string{"api_url", "model", "model_source", "catalog_model", "catalog_model_source", "route_reason", "max_output_tokens", "context_window_tokens", "function_calling_enabled", "prompt_cache_key_present"},
			want: doctorJSONPreviewContractIdentity{
				model:              "corp-kimi-model",
				modelSource:        "--model",
				catalogModel:       "kimi-k2.6",
				catalogModelSource: "--catalog-model",
				route:              "chat_completions",
				apiURLContains:     []string{"/v1/chat/completions"},
				requiredChecks:     []string{"api_url", "provider_registration", "model", "catalog_model", "route", "catalog_policy", "function_calling", "image_input", "request_preview"},
				omittedChecks:      []string{"auth"},
			},
			previewCount: 4,
			previewRequests: []doctorJSONPreviewRequestContract{{
				name:        "tool_smoke",
				route:       "chat_completions",
				method:      "POST",
				toolPayload: true,
				urlContains: []string{"/v1/chat/completions"},
				headers:     redactedBearerHeaders,
				bodyContains: []string{
					`"model":"corp-kimi-model"`,
					"xelyon_kimi_doctor_probe",
					`"tool_choice"`,
				},
			}},
		},
		{
			provider:           "gemini",
			newCommand:         newGeminiDoctorCommand,
			run:                runGeminiDoctorInvocation,
			setup:              setupGeminiJSONPreviewContract,
			requiredJSONFields: []string{"api_url", "model", "model_source", "catalog_model", "catalog_model_source", "route_reason", "max_output_tokens", "context_window_tokens", "function_calling_enabled", "image_input_supported", "web_search_supported", "context_caching_enabled", "thinking_enabled"},
			want: doctorJSONPreviewContractIdentity{
				model:              "corp-gemini-model",
				modelSource:        "--model",
				catalogModel:       "gemini-3.1-pro-preview-customtools",
				catalogModelSource: "--catalog-model",
				route:              "stream_generate_content_sse",
				apiURLContains:     []string{":streamGenerateContent", "alt=sse"},
				requiredChecks:     []string{"endpoint", "provider_registration", "model", "catalog_model", "route", "catalog_policy", "function_calling", "image_input", "thinking", "context_caching", "web_search", "request_preview"},
				omittedChecks:      []string{"smoke"},
			},
			previewCount: 1,
			previewRequests: []doctorJSONPreviewRequestContract{{
				name:        "tool",
				route:       "stream_generate_content_sse",
				method:      "POST",
				toolPayload: true,
				urlContains: []string{"models/corp-gemini-model:streamGenerateContent", "alt=sse"},
				headers:     map[string]string{"Content-Type": "application/json", "x-goog-api-key": "<redacted>"},
				bodyContains: []string{
					"xelyon_gemini_doctor_probe",
					`"function_calling_config":{"mode":"ANY"}`,
				},
			}},
		},
		{
			provider:           "claude",
			newCommand:         newClaudeDoctorCommand,
			run:                runClaudeDoctorInvocation,
			setup:              setupClaudeJSONPreviewContract,
			requiredJSONFields: []string{"api_url", "model", "model_source", "catalog_model", "catalog_model_source", "route_reason", "max_output_tokens", "context_window_tokens", "function_calling_enabled", "image_input_supported", "web_search_supported", "context_management_enabled", "claude_compaction_supported", "thinking_enabled", "anthropic_version"},
			want: doctorJSONPreviewContractIdentity{
				model:              "corp-claude-model",
				modelSource:        "--model",
				catalogModel:       "claude-sonnet-4-6",
				catalogModelSource: "--catalog-model",
				route:              "claude_messages",
				apiURLContains:     []string{"/v1/messages"},
				requiredChecks:     []string{"endpoint", "provider_registration", "model", "catalog_model", "route", "catalog_policy", "function_calling", "image_input", "thinking", "context_management", "web_search", "request_preview"},
			},
			previewCount: 1,
			previewRequests: []doctorJSONPreviewRequestContract{{
				name:        "tool",
				route:       "claude_messages",
				method:      "POST",
				toolPayload: true,
				urlContains: []string{"/v1/messages"},
				headers:     map[string]string{"Content-Type": "application/json", "x-api-key": "<redacted>", "anthropic-version": "2023-06-01"},
				bodyContains: []string{
					`"model":"corp-claude-model"`,
					"xelyon_claude_doctor_probe",
					`"tool_choice"`,
				},
			}},
		},
		{
			provider:           "bedrock",
			newCommand:         newBedrockDoctorCommand,
			run:                runBedrockDoctorInvocation,
			setup:              setupBedrockJSONPreviewContract,
			requiredJSONFields: []string{"region", "model", "model_source", "catalog_model", "catalog_model_source", "function_calling_enabled"},
			want: doctorJSONPreviewContractIdentity{
				model:              "corp-bedrock-sonnet",
				modelSource:        "--model",
				catalogModel:       bedrockDoctorCatalogModelForTest,
				catalogModelSource: "--catalog-model",
				route:              "claude_messages",
				region:             "us-east-1",
				requiredChecks:     []string{"region", "provider_registration", "model", "catalog_model", "route", "catalog_policy", "function_calling", "request_preview"},
				omittedChecks:      []string{"auth"},
			},
			previewCount: 1,
			previewRequests: []doctorJSONPreviewRequestContract{{
				name:        "tool",
				route:       "claude_messages",
				operation:   "invoke_model_with_response_stream",
				modelID:     "corp-bedrock-sonnet",
				method:      "POST",
				toolPayload: true,
				urlContains: []string{"/model/corp-bedrock-sonnet/", "/invoke-with-response-stream"},
				headers:     map[string]string{"Content-Type": "application/json", "Accept": "application/json", "Authorization": "<redacted: AWS SigV4>"},
				bodyContains: []string{
					"xelyon_bedrock_doctor_probe",
					`"anthropic_version":"bedrock-2023-05-31"`,
				},
			}},
		},
	}
}

func setupDeepSeekJSONPreviewContract(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	setCommonDoctorJSONPreviewEnv(t)
	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	t.Setenv("DEEPSEEK_API_URL", "")
	t.Setenv("DEEPSEEK_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "model", "deepseek-v4-flash")
	setDoctorCommandFlag(t, cmd, "catalog-model", "deepseek-v4-flash")
	setDoctorCommandFlag(t, cmd, "tool-smoke", "true")
}

func setupGroqJSONPreviewContract(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	setCommonDoctorJSONPreviewEnv(t)
	t.Setenv("GROQ_API_KEY", "gsk-test")
	t.Setenv("GROQ_API_URL", "")
	t.Setenv("GROQ_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "model", "corp-groq-model")
	setDoctorCommandFlag(t, cmd, "catalog-model", "meta-llama/llama-4-scout-17b-16e-instruct")
	setDoctorCommandFlag(t, cmd, "tool-smoke", "true")
}

func setupOllamaJSONPreviewContract(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	setCommonDoctorJSONPreviewEnv(t)
	t.Setenv("OLLAMA_BASE_URL", "http://127.0.0.1:11434")
	t.Setenv("OLLAMA_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "model", "qwen2.5-coder:7b")
	setDoctorCommandFlag(t, cmd, "catalog-model", "qwen2.5-coder:7b")
	setDoctorCommandFlag(t, cmd, "tool-smoke", "true")
}

func setupOpenRouterJSONPreviewContract(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	setCommonDoctorJSONPreviewEnv(t)
	t.Setenv("OPENROUTER_API_KEY", "sk-or-test")
	t.Setenv("OPENROUTER_API_URL", "")
	t.Setenv("OPENROUTER_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "model", "anthropic/claude-sonnet-4.6")
	setDoctorCommandFlag(t, cmd, "catalog-model", "anthropic/claude-sonnet-4.6")
	setDoctorCommandFlag(t, cmd, "tool-smoke", "true")
}

func setupOpenAIJSONPreviewContract(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	setCommonDoctorJSONPreviewEnv(t)
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("OPENAI_API_URL", "")
	t.Setenv("OPENAI_RESPONSES_URL", "")
	t.Setenv("OPENAI_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "model", "corp-openai-responses")
	setDoctorCommandFlag(t, cmd, "catalog-model", "gpt-5.4")
	setDoctorCommandFlag(t, cmd, "tool-smoke", "true")
}

func setupAzureJSONPreviewContract(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	setCommonDoctorJSONPreviewEnv(t)
	t.Setenv("AZURE_OPENAI_BASE_URL", "https://example.openai.azure.com/openai/v1")
	t.Setenv("AZURE_OPENAI_API_KEY", "azure-key")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN", "")
	t.Setenv("AZURE_OPENAI_AUTH_TOKEN_COMMAND", "")
	t.Setenv("AZURE_OPENAI_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "deployment", "corp-azure-gpt55")
	setDoctorCommandFlag(t, cmd, "catalog-model", "gpt-5.5-pro")
	setDoctorCommandFlag(t, cmd, "retention-smoke", "true")
}

func setupKimiJSONPreviewContract(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	setCommonDoctorJSONPreviewEnv(t)
	t.Setenv("MOONSHOT_API_KEY", "moonshot-key")
	t.Setenv("KIMI_API_URL", "")
	t.Setenv("KIMI_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "model", "corp-kimi-model")
	setDoctorCommandFlag(t, cmd, "catalog-model", "kimi-k2.6")
	setDoctorCommandFlag(t, cmd, "tool-smoke", "true")
}

func setupGeminiJSONPreviewContract(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	setCommonDoctorJSONPreviewEnv(t)
	t.Setenv("GEMINI_API_KEY", "gemini-key")
	t.Setenv("GEMINI_API_URL", "")
	t.Setenv("GEMINI_CONTEXT_CACHING", "")
	t.Setenv("GEMINI_FC_MODE", "")
	setDoctorCommandFlag(t, cmd, "model", "corp-gemini-model")
	setDoctorCommandFlag(t, cmd, "catalog-model", "gemini-3.1-pro-preview-customtools")
	setDoctorCommandFlag(t, cmd, "tool-smoke", "true")
}

func setupClaudeJSONPreviewContract(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	setCommonDoctorJSONPreviewEnv(t)
	t.Setenv("ANTHROPIC_API_KEY", "claude-key")
	t.Setenv("ANTHROPIC_API_URL", "")
	t.Setenv("CLAUDE_FUNCTION_CALLING", "1")
	setDoctorCommandFlag(t, cmd, "model", "corp-claude-model")
	setDoctorCommandFlag(t, cmd, "catalog-model", "claude-sonnet-4-6")
	setDoctorCommandFlag(t, cmd, "tool-smoke", "true")
}

func setupBedrockJSONPreviewContract(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	setCommonDoctorJSONPreviewEnv(t)
	setBedrockDoctorCommandTestEnv(t)
	t.Setenv("BEDROCK_FUNCTION_CALLING", "1")
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")
	setDoctorCommandFlag(t, cmd, "model", "corp-bedrock-sonnet")
	setDoctorCommandFlag(t, cmd, "catalog-model", bedrockDoctorCatalogModelForTest)
	setDoctorCommandFlag(t, cmd, "tool-smoke", "true")
}

func setCommonDoctorJSONPreviewEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_MODEL", "")
}
