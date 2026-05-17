package cmd

import (
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
)

type doctorJSONBaselineContractCase struct {
	provider           string
	newCommand         func() *cobra.Command
	run                func(*cobra.Command, []string) error
	setup              func(*testing.T, *cobra.Command)
	requiredJSONFields []string
	want               doctorJSONContractIdentity
	checks             []doctorJSONBaselineCheckContract
}

type doctorJSONBaselineCheckContract struct {
	name   string
	status string
}

func TestDoctorJSONBaselineProviderContractMatrix(t *testing.T) {
	for _, tc := range doctorJSONBaselineContractCases() {
		t.Run(tc.provider, func(t *testing.T) {
			cmd, out := newDoctorSubcommandTest(t, tc.newCommand)
			tc.setup(t, cmd)
			setDoctorCommandFlag(t, cmd, "json", "true")

			if err := tc.run(cmd, nil); err != nil {
				t.Fatalf("run doctor %s --json error = %v\noutput:\n%s", tc.provider, err, out.String())
			}

			raw := unmarshalDoctorJSON[map[string]json.RawMessage](t, out)
			report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
			requireDoctorJSONFields(t, raw, append([]string{
				"provider",
				"route",
				"checks",
			}, tc.requiredJSONFields...)...)
			requireDoctorJSONFieldsOmitted(t, raw, "request_preview", "smoke", "capabilities")
			if report.Provider != tc.provider {
				t.Fatalf("provider = %q, want %q", report.Provider, tc.provider)
			}
			requireDoctorJSONContractIdentity(t, report, tc.want)
			requireDoctorJSONBaselineChecks(t, report.Checks, tc.checks)
			requireNoDoctorJSONChecks(t, report.Checks, "request_preview", "smoke", "tool_smoke", "image_smoke", "web_search_smoke", "retention_smoke")
		})
	}
}

func doctorJSONBaselineContractCases() []doctorJSONBaselineContractCase {
	openAICompatibleFields := []string{
		"api_url",
		"model",
		"model_source",
		"catalog_model",
		"catalog_model_source",
		"route_reason",
		"max_output_tokens",
		"context_window_tokens",
		"function_calling_enabled",
	}
	openAICompatibleChecks := doctorJSONBaselineOKChecks(
		"auth",
		"endpoint",
		"provider_registration",
		"model",
		"catalog_model",
		"route",
		"catalog_policy",
		"function_calling",
	)
	return []doctorJSONBaselineContractCase{
		{
			provider:           "deepseek",
			newCommand:         newDeepSeekDoctorCommand,
			run:                runDeepSeekDoctorInvocation,
			setup:              setupDeepSeekJSONBaselineContract,
			requiredJSONFields: append([]string{"api_model", "thinking_supported", "thinking_enabled", "thinking_type"}, openAICompatibleFields...),
			want: doctorJSONContractIdentity{
				model:              "corp-deepseek-model",
				modelSource:        "--model",
				catalogModel:       "deepseek-v4-flash",
				catalogModelSource: "--catalog-model",
				route:              "chat_completions",
				apiURLContains:     []string{"/chat/completions"},
			},
			checks: append(openAICompatibleChecks, doctorJSONBaselineCheckContract{name: "thinking", status: "ok"}),
		},
		{
			provider:           "groq",
			newCommand:         newGroqDoctorCommand,
			run:                runGroqDoctorInvocation,
			setup:              setupGroqJSONBaselineContract,
			requiredJSONFields: openAICompatibleFields,
			want: doctorJSONContractIdentity{
				model:              "corp-groq-model",
				modelSource:        "--model",
				catalogModel:       "meta-llama/llama-4-scout-17b-16e-instruct",
				catalogModelSource: "--catalog-model",
				route:              "chat_completions",
				apiURLContains:     []string{"/openai/v1/chat/completions"},
			},
			checks: openAICompatibleChecks,
		},
		{
			provider:           "ollama",
			newCommand:         newOllamaDoctorCommand,
			run:                runOllamaDoctorInvocation,
			setup:              setupOllamaJSONBaselineContract,
			requiredJSONFields: []string{"api_url", "model", "model_source", "catalog_model", "catalog_model_source", "route_reason", "max_output_tokens", "context_window_tokens", "function_calling_enabled"},
			want: doctorJSONContractIdentity{
				model:              "qwen2.5-coder:7b",
				modelSource:        "--model",
				catalogModel:       "qwen2.5-coder:7b",
				catalogModelSource: "--catalog-model",
				route:              "ollama_chat",
			},
			checks: doctorJSONBaselineOKChecks("auth", "endpoint", "provider_registration", "model", "catalog_model", "installed_model", "route", "catalog_policy", "function_calling"),
		},
		{
			provider:           "openrouter",
			newCommand:         newOpenRouterDoctorCommand,
			run:                runOpenRouterDoctorInvocation,
			setup:              setupOpenRouterJSONBaselineContract,
			requiredJSONFields: append([]string{"upstream_provider", "upstream_model", "image_input_supported"}, openAICompatibleFields...),
			want: doctorJSONContractIdentity{
				model:              "anthropic/claude-sonnet-4.6",
				modelSource:        "--model",
				catalogModel:       "anthropic/claude-sonnet-4.6",
				catalogModelSource: "--catalog-model",
				route:              "anthropic_messages",
				apiURLContains:     []string{"/api/v1/messages"},
			},
			checks: append(openAICompatibleChecks, doctorJSONBaselineCheckContract{name: "image_input", status: "ok"}),
		},
		{
			provider:           "openai",
			newCommand:         newOpenAIDoctorCommand,
			run:                runOpenAIDoctorInvocation,
			setup:              setupOpenAIJSONBaselineContract,
			requiredJSONFields: []string{"api_url", "responses_url", "model", "model_source", "catalog_model", "catalog_model_source", "route_reason", "max_output_tokens", "context_window_tokens", "function_calling_enabled", "responses_store", "responses_persist_response_id"},
			want: doctorJSONContractIdentity{
				model:                "corp-openai-deployment",
				modelSource:          "--model",
				catalogModel:         "gpt-5.4",
				catalogModelSource:   "--catalog-model",
				route:                "responses_streaming",
				apiURLContains:       []string{"/v1/chat/completions"},
				responsesURLContains: []string{"/v1/responses"},
			},
			checks: doctorJSONBaselineOKChecks("auth", "api_url", "responses_url", "provider_registration", "model", "route", "catalog_policy", "function_calling", "responses_retention"),
		},
		{
			provider:           "azure",
			newCommand:         newAzureDoctorCommand,
			run:                runAzureDoctorInvocation,
			setup:              setupAzureJSONBaselineContract,
			requiredJSONFields: []string{"base_url", "normalized_base_url", "auth_mode", "deployment", "deployment_source", "catalog_model", "catalog_model_source", "route_reason", "function_calling_enabled", "responses_store", "responses_persist_response_id"},
			want: doctorJSONContractIdentity{
				deployment:            "corp-codex-deployment",
				catalogModel:          "gpt-5.3-codex",
				catalogModelSource:    "--catalog-model",
				route:                 "responses_streaming",
				normalizedURLContains: []string{"https://example.openai.azure.com/openai/v1"},
			},
			checks: doctorJSONBaselineOKChecks("base_url", "auth", "deployment", "catalog_model", "route", "catalog_policy", "function_calling", "responses_retention"),
		},
		{
			provider:           "kimi",
			newCommand:         newKimiDoctorCommand,
			run:                runKimiDoctorInvocation,
			setup:              setupKimiJSONBaselineContract,
			requiredJSONFields: []string{"api_url", "model", "model_source", "catalog_model", "catalog_model_source", "route_reason", "max_output_tokens", "context_window_tokens", "function_calling_enabled", "unsupported_features", "prompt_cache_key_present"},
			want: doctorJSONContractIdentity{
				model:              "corp-kimi-model",
				modelSource:        "--model",
				catalogModel:       "kimi-k2.6",
				catalogModelSource: "--catalog-model",
				route:              "chat_completions",
				apiURLContains:     []string{"/v1/chat/completions"},
			},
			checks: append(
				doctorJSONBaselineOKChecks("api_url", "auth", "provider_registration", "model", "catalog_model", "route", "catalog_policy", "function_calling", "image_input", "prompt_cache_key"),
				doctorJSONBaselineCheckContract{name: "unsupported_features", status: "info"},
			),
		},
		{
			provider:           "gemini",
			newCommand:         newGeminiDoctorCommand,
			run:                runGeminiDoctorInvocation,
			setup:              setupGeminiJSONBaselineContract,
			requiredJSONFields: []string{"api_url", "model", "model_source", "catalog_model", "catalog_model_source", "route_reason", "max_output_tokens", "context_window_tokens", "function_calling_enabled", "image_input_supported", "web_search_supported", "context_caching_enabled", "thinking_enabled"},
			want: doctorJSONContractIdentity{
				model:              "corp-gemini-model",
				modelSource:        "--model",
				catalogModel:       "gemini-3.1-pro-preview-customtools",
				catalogModelSource: "--catalog-model",
				route:              "stream_generate_content_sse",
				apiURLContains:     []string{"models/corp-gemini-model:streamGenerateContent", "alt=sse"},
			},
			checks: doctorJSONBaselineOKChecks("auth", "endpoint", "provider_registration", "model", "catalog_model", "route", "catalog_policy", "function_calling", "image_input", "thinking", "context_caching", "web_search"),
		},
		{
			provider:           "claude",
			newCommand:         newClaudeDoctorCommand,
			run:                runClaudeDoctorInvocation,
			setup:              setupClaudeJSONBaselineContract,
			requiredJSONFields: []string{"api_url", "model", "model_source", "catalog_model", "catalog_model_source", "route_reason", "max_output_tokens", "context_window_tokens", "function_calling_enabled", "image_input_supported", "web_search_supported", "context_management_enabled", "claude_compaction_supported", "thinking_enabled", "anthropic_version"},
			want: doctorJSONContractIdentity{
				model:              "corp-claude-model",
				modelSource:        "--model",
				catalogModel:       "claude-sonnet-4-6",
				catalogModelSource: "--catalog-model",
				route:              "claude_messages",
				apiURLContains:     []string{"/v1/messages"},
			},
			checks: doctorJSONBaselineOKChecks("auth", "endpoint", "provider_registration", "model", "catalog_model", "route", "catalog_policy", "function_calling", "image_input", "thinking", "context_management", "web_search"),
		},
		{
			provider:           "bedrock",
			newCommand:         newBedrockDoctorCommand,
			run:                runBedrockDoctorInvocation,
			setup:              setupBedrockJSONBaselineContract,
			requiredJSONFields: []string{"region", "model", "model_source", "catalog_model", "catalog_model_source", "function_calling_enabled"},
			want: doctorJSONContractIdentity{
				model:              "corp-bedrock-sonnet",
				modelSource:        "--model",
				catalogModel:       bedrockDoctorCatalogModelForTest,
				catalogModelSource: "--catalog-model",
				route:              "claude_messages",
				region:             "us-east-1",
			},
			checks: doctorJSONBaselineOKChecks("region", "auth", "provider_registration", "model", "catalog_model", "route", "function_calling"),
		},
	}
}

func doctorJSONBaselineOKChecks(names ...string) []doctorJSONBaselineCheckContract {
	checks := make([]doctorJSONBaselineCheckContract, 0, len(names))
	for _, name := range names {
		checks = append(checks, doctorJSONBaselineCheckContract{name: name, status: "ok"})
	}
	return checks
}

func requireDoctorJSONBaselineChecks(t *testing.T, checks []doctorJSONCheck, wants []doctorJSONBaselineCheckContract) {
	t.Helper()
	for _, want := range wants {
		requireDoctorJSONCheckStatus(t, requireDoctorJSONCheck(t, checks, want.name), want.status)
	}
}
