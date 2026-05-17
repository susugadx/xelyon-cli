package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	openaiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/openai"
)

func TestRunOpenAIDoctorInvocation_PrintRequestJSONDoesNotRequireAPIKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_API_URL", "")
	t.Setenv("OPENAI_RESPONSES_URL", "")

	cmd, out := newDoctorSubcommandTest(t, newOpenAIDoctorCommand)

	doctorOpenAIModelFlag = "gpt-5.5-pro"
	doctorCatalogModelFlag = "gpt-5.5-pro"
	doctorOpenAIRetentionSmokeFlag = true
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runOpenAIDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runOpenAIDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	requireDoctorJSONPrintRequestOmittedSmoke(t, report.Smoke)
	requireDoctorJSONRequestPreviewCount(t, report.RequestPreview, 2)
	followup := requireDoctorJSONRequestPreviewAt(t, report.RequestPreview, 1, "retention_followup")
	if followup.Name != "retention_followup" || !followup.RetentionPayload {
		t.Fatalf("followup preview = %#v, want retention followup", followup)
	}
	body := requireDoctorJSONRequestPreviewBody[struct {
		Store              bool   `json:"store"`
		PreviousResponseID string `json:"previous_response_id"`
	}](t, followup)
	if followup.PreviousResponseID == "" || body.PreviousResponseID != followup.PreviousResponseID || !body.Store {
		t.Fatalf("followup previous/store = body:%#v request:%#v, want placeholder previous_response_id and store true", body, followup)
	}
}

func TestRunOpenAIDoctorInvocation_PrintRequestJSONReportsChatProxyEndpointWarning(t *testing.T) {
	proxyURL := "https://openai.example/proxy/chat"
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_API_URL", proxyURL)
	t.Setenv("OPENAI_RESPONSES_URL", "")
	t.Setenv("OPENAI_FUNCTION_CALLING", "1")

	cmd, out := newDoctorSubcommandTest(t, newOpenAIDoctorCommand)

	doctorOpenAIModelFlag = "gpt-4"
	doctorCatalogModelFlag = "gpt-4"
	doctorToolSmokeFlag = true
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runOpenAIDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runOpenAIDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	if report.APIURL != proxyURL {
		t.Fatalf("api_url = %q, want configured proxy URL", report.APIURL)
	}
	requireDoctorJSONProxyWarning(t, report.Checks, "api_url_path", "api_url", proxyURL)
	requireDoctorJSONPrintRequestSkippedAuth(t, report.Checks)
	requireDoctorJSONRequestPreviewRouteAndURL(t, report.RequestPreview, 1, string(openaiprovider.DiagnosticRouteChatCompletions), proxyURL)
}

func TestRunOpenAIDoctorInvocation_PrintRequestJSONReportsResponsesProxyEndpointWarning(t *testing.T) {
	proxyURL := "https://openai.example/proxy/responses"
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_API_URL", "")
	t.Setenv("OPENAI_RESPONSES_URL", proxyURL)

	cmd, out := newDoctorSubcommandTest(t, newOpenAIDoctorCommand)

	doctorOpenAIModelFlag = "gpt-5.5-pro"
	doctorCatalogModelFlag = "gpt-5.5-pro"
	doctorOpenAIRetentionSmokeFlag = true
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runOpenAIDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runOpenAIDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	if report.ResponsesURL != proxyURL {
		t.Fatalf("responses_url = %q, want configured proxy URL", report.ResponsesURL)
	}
	requireDoctorJSONProxyWarning(t, report.Checks, "responses_url_path", "responses_url", proxyURL)
	requireDoctorJSONPrintRequestSkippedAuth(t, report.Checks)
	requireDoctorJSONRequestPreviewRouteAndURL(t, report.RequestPreview, 2, string(openaiprovider.DiagnosticRouteResponsesNonStreaming), proxyURL)
}

func TestRunOpenAIDoctorInvocation_CapabilitiesJSONDoesNotRequireAPIKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_API_URL", "")
	t.Setenv("OPENAI_RESPONSES_URL", "")

	cmd, out := newDoctorSubcommandTest(t, newOpenAIDoctorCommand)

	doctorOpenAIModelFlag = "corp-openai-deployment"
	doctorCatalogModelFlag = "gpt-5.4"
	doctorCapabilitiesFlag = true
	doctorJSONFlag = true

	if err := runOpenAIDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runOpenAIDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[struct {
		Capabilities struct {
			Model              string `json:"model"`
			CatalogModel       string `json:"catalog_model"`
			ResponsesAPI       bool   `json:"responses_api"`
			ResponsesStreaming bool   `json:"responses_streaming"`
			Retention          struct {
				PreviousResponseID bool `json:"previous_response_id"`
			} `json:"retention"`
			ServerCompaction struct {
				RequestPayload   bool `json:"request_payload"`
				CompactThreshold int  `json:"compact_threshold"`
			} `json:"server_compaction"`
		} `json:"capabilities"`
		Checks []doctorJSONCheck `json:"checks"`
	}](t, out)
	if report.Capabilities.Model != "corp-openai-deployment" ||
		report.Capabilities.CatalogModel != "gpt-5.4" ||
		!report.Capabilities.ResponsesAPI ||
		!report.Capabilities.ResponsesStreaming ||
		!report.Capabilities.Retention.PreviousResponseID ||
		!report.Capabilities.ServerCompaction.RequestPayload ||
		report.Capabilities.ServerCompaction.CompactThreshold <= 0 {
		t.Fatalf("capabilities = %+v, want resolved OpenAI capabilities", report.Capabilities)
	}
	requireNoDoctorJSONChecks(t, report.Checks, "auth")
}

func TestRunOpenAIDoctorInvocation_RequireCapabilityFailsWithoutAPIKeyCheck(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_API_URL", "")
	t.Setenv("OPENAI_RESPONSES_URL", "")

	cmd, out := newDoctorSubcommandTest(t, newOpenAIDoctorCommand)

	doctorOpenAIModelFlag = "gpt-5.5-pro"
	doctorCatalogModelFlag = "gpt-5.5-pro"
	doctorRequiredCapabilityFlags = []string{"responses_streaming"}
	doctorJSONFlag = true

	err := runOpenAIDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatalf("runOpenAIDoctorInvocation() error = nil, want required capability failure\noutput:\n%s", out.String())
	}

	report := unmarshalDoctorJSON[struct {
		Checks []doctorJSONCheck `json:"checks"`
	}](t, out)
	requireNoDoctorJSONChecks(t, report.Checks, "auth")
	check := requireDoctorJSONCheck(t, report.Checks, "required_capability")
	requireDoctorJSONCheckStatus(t, check, "fail")
	requireDoctorJSONCheckDetailContains(t, check, "responses_streaming=missing")
}

func TestRunOpenAIDoctorInvocation_RequireStreamingCapabilityFailsWithoutCatalogModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_API_URL", "")
	t.Setenv("OPENAI_RESPONSES_URL", "")
	configDir := filepath.Join(home, ".xelyon")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("openai:\n  responses_api_models:\n    - corp-gpt55-pro-alias\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cmd, out := newDoctorSubcommandTest(t, newOpenAIDoctorCommand)

	doctorOpenAIModelFlag = "corp-gpt55-pro-alias"
	doctorRequiredCapabilityFlags = []string{"responses_streaming"}
	doctorJSONFlag = true

	err := runOpenAIDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatalf("runOpenAIDoctorInvocation() error = nil, want unknown required capability failure\noutput:\n%s", out.String())
	}

	report := unmarshalDoctorJSON[struct {
		Checks []doctorJSONCheck `json:"checks"`
	}](t, out)
	requireNoDoctorJSONChecks(t, report.Checks, "auth")
	check := requireDoctorJSONCheck(t, report.Checks, "required_capability")
	requireDoctorJSONCheckStatus(t, check, "fail")
	requireDoctorJSONCheckDetailContains(t, check, "responses_streaming=unknown")
	requireDoctorJSONCheckSuggestionContains(t, check, "--catalog-model")
}

func TestRootCommand_OpenAIDoctorCommandParsesFlags(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("OPENAI_API_URL", "")
	t.Setenv("OPENAI_RESPONSES_URL", "")

	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "openai", "--model", "corp-openai-deployment", "--catalog-model", "gpt-5.4", "--json"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"model": "corp-openai-deployment"`) {
		t.Fatalf("output = %q, want parsed OpenAI model", out.String())
	}
	if !strings.Contains(out.String(), `"catalog_model": "gpt-5.4"`) {
		t.Fatalf("output = %q, want parsed OpenAI catalog model", out.String())
	}
}

func TestRootCommand_OpenAIDoctorHelpShowsDoctorFlags(t *testing.T) {
	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "openai", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	for _, want := range []string{"--model", "--catalog-model", "--smoke", "--tool-smoke", "--retention-smoke", "--capabilities", "--require-capability", "--print-request", "--timeout", "--json", "Diagnose OpenAI provider configuration"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output = %q, want OpenAI doctor help substring %q", out.String(), want)
		}
	}
	for _, unwanted := range []string{"--print-config", "--image-smoke", "--web-search-smoke"} {
		if strings.Contains(out.String(), unwanted) {
			t.Fatalf("output = %q, should not contain %s", out.String(), unwanted)
		}
	}
}

func TestRenderOpenAIDoctorTextIncludesSmokeObservability(t *testing.T) {
	report := openaiprovider.DiagnosticReport{
		Provider:           "openai",
		APIURL:             "https://api.openai.com/v1/chat/completions",
		ResponsesURL:       "https://api.openai.com/v1/responses",
		Model:              "gpt-5.4",
		ModelSource:        "test",
		CatalogModel:       "gpt-5.4",
		CatalogModelSource: "test",
		Route:              openaiprovider.DiagnosticRouteResponsesStreaming,
		RouteReason:        "model=gpt-5.4 uses Responses API; catalog_model=gpt-5.4 supports Responses streaming",
		Checks: []openaiprovider.DiagnosticCheck{
			{Name: "smoke", Status: openaiprovider.DiagnosticStatusOK, Message: "live OpenAI smoke request succeeded"},
		},
		Smoke: &openaiprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         openaiprovider.DiagnosticRouteResponsesStreaming,
			Content:       "xelyon openai doctor ok",
			ResponseID:    "resp_text",
			Duration:      "1ms",
			UsageObserved: true,
			Usage: openaiprovider.DiagnosticSmokeUsage{
				InputTokens:         10,
				OutputTokens:        4,
				ThinkingTokens:      2,
				CachedInputTokens:   3,
				CacheCreationTokens: 1,
			},
			Cost: openaiprovider.DiagnosticSmokeCost{
				USD: 0.00012345,
			},
		},
	}

	var out bytes.Buffer
	renderOpenAIDoctorText(&out, report)
	requireDoctorContractTextContainsAll(t, out.String(), []string{
		"Route reason: model=gpt-5.4 uses Responses API; catalog_model=gpt-5.4 supports Responses streaming",
		"Smoke route: responses_streaming",
		"Smoke response ID: resp_text",
		"Smoke usage: input=10 cached=3 output=4 reasoning=2 cache_creation=1",
		"Smoke cost estimate: $0.00012345 USD",
	})
}

func TestRenderOpenAIDoctorTextIncludesRequestPreview(t *testing.T) {
	report := openaiprovider.DiagnosticReport{
		Provider:           "openai",
		APIURL:             "https://api.openai.com/v1/chat/completions",
		ResponsesURL:       "https://api.openai.com/v1/responses",
		Model:              "gpt-5.4",
		ModelSource:        "test",
		CatalogModel:       "gpt-5.4",
		CatalogModelSource: "test",
		Route:              openaiprovider.DiagnosticRouteResponsesStreaming,
		RequestPreview: &openaiprovider.DiagnosticRequestPreview{
			Requests: []openaiprovider.DiagnosticRequestPreviewRequest{{
				Name:    "text",
				Route:   openaiprovider.DiagnosticRouteResponsesStreaming,
				Method:  "POST",
				URL:     "https://api.openai.com/v1/responses",
				Headers: map[string]string{"Authorization": "Bearer <redacted>"},
				Body:    map[string]any{"model": "gpt-5.4", "store": false},
			}},
		},
	}

	var out bytes.Buffer
	renderOpenAIDoctorText(&out, report)
	output := out.String()
	for _, want := range []string{
		"Request preview:",
		`"Authorization": "Bearer <redacted>"`,
		`"model": "gpt-5.4"`,
		`"store": false`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
}

func TestRenderOpenAIDoctorTextIncludesCapabilities(t *testing.T) {
	report := openaiprovider.DiagnosticReport{
		Provider:           "openai",
		APIURL:             "https://api.openai.com/v1/chat/completions",
		ResponsesURL:       "https://api.openai.com/v1/responses",
		Model:              "gpt-5.4",
		ModelSource:        "test",
		CatalogModel:       "gpt-5.4",
		CatalogModelSource: "test",
		Route:              openaiprovider.DiagnosticRouteResponsesStreaming,
		Capabilities: &openaiprovider.DiagnosticCapabilities{
			Model:              "gpt-5.4",
			CatalogModel:       "gpt-5.4",
			Route:              openaiprovider.DiagnosticRouteResponsesStreaming,
			ResponsesAPI:       true,
			ResponsesStreaming: true,
			Retention: openaiprovider.DiagnosticRetentionCapability{
				PreviousResponseID: true,
			},
			ServerCompaction: openaiprovider.DiagnosticServerCompactionCapability{
				RequestPayload:   true,
				CompactThreshold: 800000,
			},
		},
	}

	var out bytes.Buffer
	renderOpenAIDoctorText(&out, report)
	output := out.String()
	for _, want := range []string{
		"Capabilities:",
		`"responses_api": true`,
		`"previous_response_id": true`,
		`"compact_threshold": 800000`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
}

func TestRenderOpenAIDoctorTextIncludesSmokeRequests(t *testing.T) {
	report := openaiprovider.DiagnosticReport{
		Provider:           "openai",
		APIURL:             "https://api.openai.com/v1/chat/completions",
		ResponsesURL:       "https://api.openai.com/v1/responses",
		Model:              "gpt-5.4",
		ModelSource:        "test",
		CatalogModel:       "gpt-5.4",
		CatalogModelSource: "test",
		Route:              openaiprovider.DiagnosticRouteResponsesStreaming,
		Smoke: &openaiprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         openaiprovider.DiagnosticRouteResponsesStreaming,
			UsageObserved: true,
			Usage:         openaiprovider.DiagnosticSmokeUsage{InputTokens: 18, OutputTokens: 8},
			Cost:          openaiprovider.DiagnosticSmokeCost{USD: 0.00012345},
			Requests: []openaiprovider.DiagnosticSmokeRequestResult{
				{
					Name:          "text",
					Ran:           true,
					Route:         openaiprovider.DiagnosticRouteResponsesStreaming,
					Content:       "xelyon openai doctor ok",
					ResponseID:    "resp_text",
					Duration:      "1ms",
					UsageObserved: true,
					Usage:         openaiprovider.DiagnosticSmokeUsage{InputTokens: 10, OutputTokens: 4},
					Cost:          openaiprovider.DiagnosticSmokeCost{USD: 0.00010000},
				},
				{
					Name:          "tool",
					Ran:           true,
					ToolPayload:   true,
					Route:         openaiprovider.DiagnosticRouteResponsesStreaming,
					Content:       `{"tool":"xelyon_openai_doctor_probe"}`,
					ResponseID:    "resp_tool",
					Duration:      "2ms",
					UsageObserved: true,
					Usage:         openaiprovider.DiagnosticSmokeUsage{InputTokens: 8, OutputTokens: 4},
					Cost:          openaiprovider.DiagnosticSmokeCost{USD: 0.00002345},
				},
			},
		},
	}

	var out bytes.Buffer
	renderOpenAIDoctorText(&out, report)
	requireDoctorContractTextContainsAll(t, out.String(), []string{
		"Smoke request text: ok route=responses_streaming duration=1ms response_id=resp_text",
		"Smoke content text: xelyon openai doctor ok",
		"Smoke request tool: ok route=responses_streaming duration=2ms response_id=resp_tool",
		"Smoke usage tool: input=8 cached=0 output=4 reasoning=0 cache_creation=0",
		"Smoke total usage: input=18 cached=0 output=8 reasoning=0 cache_creation=0",
		"Smoke total cost estimate: $0.00012345 USD",
	})
}

func TestRenderOpenAIDoctorTextIncludesRetentionSmokePreviousResponseID(t *testing.T) {
	report := openaiprovider.DiagnosticReport{
		Provider:           "openai",
		APIURL:             "https://api.openai.com/v1/chat/completions",
		ResponsesURL:       "https://api.openai.com/v1/responses",
		Model:              "gpt-5.5-pro",
		ModelSource:        "test",
		CatalogModel:       "gpt-5.5-pro",
		CatalogModelSource: "test",
		Route:              openaiprovider.DiagnosticRouteResponsesNonStreaming,
		Smoke: &openaiprovider.DiagnosticSmokeResult{
			Ran:              true,
			Route:            openaiprovider.DiagnosticRouteResponsesNonStreaming,
			RetentionPayload: true,
			UsageObserved:    true,
			Requests: []openaiprovider.DiagnosticSmokeRequestResult{
				{
					Name:             "retention_initial",
					Ran:              true,
					RetentionPayload: true,
					Route:            openaiprovider.DiagnosticRouteResponsesNonStreaming,
					ResponseID:       "resp_retention_initial",
					Duration:         "1ms",
					UsageObserved:    true,
				},
				{
					Name:               "retention_followup",
					Ran:                true,
					RetentionPayload:   true,
					Route:              openaiprovider.DiagnosticRouteResponsesNonStreaming,
					ResponseID:         "resp_retention_followup",
					PreviousResponseID: "resp_retention_initial",
					Duration:           "2ms",
					UsageObserved:      true,
				},
			},
		},
	}

	var out bytes.Buffer
	renderOpenAIDoctorText(&out, report)
	requireDoctorContractTextContainsAll(t, out.String(), []string{
		"Smoke request retention_initial: ok route=responses_non_streaming duration=1ms response_id=resp_retention_initial previous_response_id=(not returned)",
		"Smoke request retention_followup: ok route=responses_non_streaming duration=2ms response_id=resp_retention_followup previous_response_id=resp_retention_initial",
	})
}

func TestRenderOpenAIDoctorJSONIncludesSmokeObservability(t *testing.T) {
	report := openaiprovider.DiagnosticReport{
		Provider: "openai",
		Smoke: &openaiprovider.DiagnosticSmokeResult{
			Ran:              true,
			Route:            openaiprovider.DiagnosticRouteResponsesNonStreaming,
			ResponseID:       "resp_json",
			Duration:         "1ms",
			RetentionPayload: true,
			UsageObserved:    true,
			Usage: openaiprovider.DiagnosticSmokeUsage{
				InputTokens:         10,
				OutputTokens:        4,
				ThinkingTokens:      2,
				CachedInputTokens:   3,
				CacheCreationTokens: 1,
			},
			Cost: openaiprovider.DiagnosticSmokeCost{
				USD:                0.00012345,
				PricingUnavailable: false,
			},
			Requests: []openaiprovider.DiagnosticSmokeRequestResult{
				{
					Name:          "text",
					Ran:           true,
					Route:         openaiprovider.DiagnosticRouteResponsesNonStreaming,
					ResponseID:    "resp_json",
					UsageObserved: true,
				},
				{
					Name:               "retention_followup",
					Ran:                true,
					RetentionPayload:   true,
					Route:              openaiprovider.DiagnosticRouteResponsesNonStreaming,
					ResponseID:         "resp_retention_followup",
					PreviousResponseID: "resp_json",
					UsageObserved:      true,
				},
			},
		},
	}

	var out bytes.Buffer
	if err := renderOpenAIDoctorJSON(&out, report); err != nil {
		t.Fatalf("renderOpenAIDoctorJSON() error = %v", err)
	}

	got := unmarshalDoctorJSONSmoke(t, &out)
	if got.Route != "responses_non_streaming" || got.ResponseID != "resp_json" || !got.UsageObserved {
		t.Fatalf("smoke metadata = %#v, want route, response_id, and usage_observed", got)
	}
	requireDoctorJSONSmokeUsage(t, got.Usage, doctorJSONSmokeUsage{
		InputTokens:         10,
		OutputTokens:        4,
		ThinkingTokens:      2,
		CachedInputTokens:   3,
		CacheCreationTokens: 1,
	})
	requireDoctorJSONSmokeCost(t, got.Cost, 0.00012345, false)
	requireDoctorJSONSmokeRequestCount(t, got, 2)
	text := requireDoctorJSONSmokeRequestAt(t, got, 0, "text")
	if text.ResponseID != "resp_json" {
		t.Fatalf("text request = %+v, want response_id", text)
	}
	retention := requireDoctorJSONSmokeRequestAt(t, got, 1, "retention_followup")
	if !retention.RetentionPayload || retention.PreviousResponseID != "resp_json" {
		t.Fatalf("retention request = %+v, want retention payload and previous_response_id", retention)
	}
}
