package clidoctor

import (
	"encoding/json"
	"sort"
	"testing"
)

type doctorJSONBaselineContractCase struct {
	provider            string
	newCommand          func() *doctorTestCommand
	run                 func(*doctorTestCommand, []string) error
	setup               func(*testing.T, *doctorTestCommand) doctorJSONBaselineSetupContract
	requiredJSONFields  []string
	want                doctorJSONContractIdentity
	stringFields        map[string]string
	trueFields          []string
	routeReasonContains []string
	checks              []doctorJSONBaselineCheckContract
	checkDetails        []doctorJSONBaselineCheckDetailContract
}

type doctorJSONBaselineSetupContract struct {
	apiURL string
}

type doctorJSONBaselineCheckContract struct {
	name   string
	status string
}

type doctorJSONBaselineCheckDetailContract struct {
	name     string
	contains string
}

func TestDoctorJSONBaselineProviderContractMatrix(t *testing.T) {
	for _, tc := range doctorJSONBaselineContractCases() {
		t.Run(tc.provider, func(t *testing.T) {
			cmd, out := newDoctorSubcommandTest(t, tc.newCommand)
			setupContract := tc.setup(t, cmd)
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
			requireDoctorJSONBaselineSetupContract(t, report, setupContract)
			requireContainsAll(t, "route_reason", report.RouteReason, tc.routeReasonContains)
			requireDoctorJSONBaselineStringFields(t, report, tc.stringFields)
			requireDoctorJSONBaselineTrueFields(t, report, tc.trueFields)
			requireDoctorJSONBaselineChecks(t, report.Checks, tc.checks)
			requireDoctorJSONBaselineCheckDetails(t, report.Checks, tc.checkDetails)
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
			stringFields: map[string]string{
				"api_model":     "corp-deepseek-model",
				"thinking_type": "disabled",
			},
			trueFields: []string{"thinking_supported"},
			checks:     append(openAICompatibleChecks, doctorJSONBaselineCheckContract{name: "thinking", status: "ok"}),
			checkDetails: []doctorJSONBaselineCheckDetailContract{
				{name: "catalog_policy", contains: "max_output_tokens=384000"},
				{name: "thinking", contains: "thinking.type=disabled"},
			},
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
			checks:       openAICompatibleChecks,
			checkDetails: []doctorJSONBaselineCheckDetailContract{{name: "catalog_policy", contains: "max_output_tokens=8192"}},
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
			checks:       doctorJSONBaselineOKChecks("auth", "endpoint", "provider_registration", "model", "catalog_model", "installed_model", "route", "catalog_policy", "function_calling"),
			checkDetails: []doctorJSONBaselineCheckDetailContract{{name: "catalog_policy", contains: "pricing=input $0.00/M"}},
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
			trueFields: []string{"image_input_supported"},
			checks:     append(openAICompatibleChecks, doctorJSONBaselineCheckContract{name: "image_input", status: "ok"}),
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
			routeReasonContains: []string{"catalog_model=gpt-5.4 supports Responses streaming"},
			checks:              doctorJSONBaselineOKChecks("auth", "api_url", "responses_url", "provider_registration", "model", "route", "catalog_policy", "function_calling", "responses_retention"),
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
			routeReasonContains: []string{"catalog_model=gpt-5.3-codex supports Responses streaming"},
			checks:              doctorJSONBaselineOKChecks("base_url", "auth", "deployment", "catalog_model", "route", "catalog_policy", "function_calling", "responses_retention"),
			checkDetails:        []doctorJSONBaselineCheckDetailContract{{name: "catalog_policy", contains: "max_output_tokens=128000"}},
		},
		{
			provider:           "kimi",
			newCommand:         newKimiDoctorCommand,
			run:                runKimiDoctorInvocation,
			setup:              setupKimiJSONBaselineContract,
			requiredJSONFields: []string{"api_url", "model", "model_source", "catalog_model", "catalog_model_source", "route_reason", "max_output_tokens", "context_window_tokens", "function_calling_enabled", "unsupported_features", "prompt_cache_key_present"},
			want: doctorJSONContractIdentity{
				model:              "kimi-k2.5",
				modelSource:        "--model",
				catalogModel:       "kimi-k2.6",
				catalogModelSource: "--catalog-model",
				route:              "chat_completions",
				apiURLContains:     []string{"/v1/chat/completions"},
			},
			trueFields: []string{"prompt_cache_key_present"},
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
			requiredJSONFields: []string{"api_url", "model", "model_source", "catalog_model", "catalog_model_source", "route_reason", "max_output_tokens", "context_window_tokens", "function_calling_enabled", "image_input_supported", "web_search_supported", "context_caching_enabled", "thinking_enabled", "service_tier"},
			want: doctorJSONContractIdentity{
				model:              "corp-gemini-model",
				modelSource:        "--model",
				catalogModel:       "gemini-3.1-pro-preview-customtools",
				catalogModelSource: "--catalog-model",
				route:              "stream_generate_content_sse",
				apiURLContains:     []string{"models/corp-gemini-model:streamGenerateContent", "alt=sse"},
			},
			trueFields:   []string{"function_calling_enabled", "image_input_supported", "web_search_supported"},
			checks:       doctorJSONBaselineOKChecks("auth", "endpoint", "provider_registration", "model", "catalog_model", "route", "service_tier", "catalog_policy", "function_calling", "image_input", "thinking", "context_caching", "web_search"),
			checkDetails: []doctorJSONBaselineCheckDetailContract{{name: "catalog_policy", contains: "max_output_tokens=65536"}, {name: "service_tier", contains: "request_body=omitted"}},
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
			stringFields: map[string]string{"anthropic_version": "2023-06-01"},
			trueFields: []string{
				"function_calling_enabled",
				"image_input_supported",
				"web_search_supported",
				"context_management_enabled",
				"claude_compaction_supported",
			},
			checks:       doctorJSONBaselineOKChecks("auth", "endpoint", "provider_registration", "model", "catalog_model", "route", "catalog_policy", "function_calling", "image_input", "thinking", "context_management", "web_search"),
			checkDetails: []doctorJSONBaselineCheckDetailContract{{name: "catalog_policy", contains: "max_output_tokens=64000"}},
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
			checks: doctorJSONBaselineOKChecks("region", "auth", "provider_registration", "model", "catalog_model", "route", "catalog_policy", "function_calling"),
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
	if len(checks) != len(wants) {
		t.Fatalf("checks = %v, want exactly %v", doctorJSONCheckNameStatusList(checks), doctorJSONBaselineCheckNameStatusList(wants))
	}
	seen := make(map[string]doctorJSONCheck, len(checks))
	for _, check := range checks {
		if _, ok := seen[check.Name]; ok {
			t.Fatalf("duplicate %s check in %v", check.Name, doctorJSONCheckNameStatusList(checks))
		}
		seen[check.Name] = check
	}
	for _, want := range wants {
		check, ok := seen[want.name]
		if !ok {
			t.Fatalf("missing %s check in %v", want.name, doctorJSONCheckNameStatusList(checks))
		}
		requireDoctorJSONCheckStatus(t, check, want.status)
	}
}

func requireDoctorJSONBaselineCheckDetails(t *testing.T, checks []doctorJSONCheck, wants []doctorJSONBaselineCheckDetailContract) {
	t.Helper()
	for _, want := range wants {
		requireDoctorJSONCheckDetailContains(t, requireDoctorJSONCheck(t, checks, want.name), want.contains)
	}
}

func requireDoctorJSONBaselineSetupContract(t *testing.T, report doctorJSONContractReport, want doctorJSONBaselineSetupContract) {
	t.Helper()
	if want.apiURL != "" && report.APIURL != want.apiURL {
		t.Fatalf("api_url = %q, want setup URL %q", report.APIURL, want.apiURL)
	}
}

func requireDoctorJSONBaselineStringFields(t *testing.T, report doctorJSONContractReport, wants map[string]string) {
	t.Helper()
	for name, want := range wants {
		var got string
		switch name {
		case "api_model":
			got = report.APIModel
		case "anthropic_version":
			got = report.AnthropicVersion
		case "thinking_type":
			got = report.ThinkingType
		default:
			t.Fatalf("unknown string field contract %q", name)
		}
		if got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
}

func requireDoctorJSONBaselineTrueFields(t *testing.T, report doctorJSONContractReport, names []string) {
	t.Helper()
	for _, name := range names {
		var got bool
		switch name {
		case "claude_compaction_supported":
			got = report.ClaudeCompactionSupported
		case "context_management_enabled":
			got = report.ContextManagementEnabled
		case "function_calling_enabled":
			got = report.FunctionCallingEnabled
		case "image_input_supported":
			got = report.ImageInputSupported
		case "prompt_cache_key_present":
			got = report.PromptCacheKeyPresent
		case "thinking_supported":
			got = report.ThinkingSupported
		case "web_search_supported":
			got = report.WebSearchSupported
		default:
			t.Fatalf("unknown true field contract %q", name)
		}
		if !got {
			t.Fatalf("%s = false, want true", name)
		}
	}
}

func doctorJSONCheckNameStatusList(checks []doctorJSONCheck) []string {
	values := make([]string, 0, len(checks))
	for _, check := range checks {
		values = append(values, check.Name+"="+check.Status)
	}
	sort.Strings(values)
	return values
}

func doctorJSONBaselineCheckNameStatusList(checks []doctorJSONBaselineCheckContract) []string {
	values := make([]string, 0, len(checks))
	for _, check := range checks {
		values = append(values, check.name+"="+check.status)
	}
	sort.Strings(values)
	return values
}
