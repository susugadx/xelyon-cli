package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	openaiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/openai"
)

func TestRunOpenAIDoctorInvocation_JSONReportsExplicitModelAndCatalogModel(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("OPENAI_API_URL", "")
	t.Setenv("OPENAI_RESPONSES_URL", "")

	cmd, out := newDoctorSubcommandTest(t, newOpenAIDoctorCommand)

	doctorOpenAIModelFlag = "corp-openai-deployment"
	doctorCatalogModelFlag = "gpt-5.4"
	doctorJSONFlag = true

	if err := runOpenAIDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runOpenAIDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	var report struct {
		Provider           string `json:"provider"`
		Model              string `json:"model"`
		ModelSource        string `json:"model_source"`
		CatalogModel       string `json:"catalog_model"`
		CatalogModelSource string `json:"catalog_model_source"`
		Route              string `json:"route"`
		RouteReason        string `json:"route_reason"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, out.String())
	}
	if report.Provider != "openai" {
		t.Fatalf("provider = %q, want openai", report.Provider)
	}
	if report.Model != "corp-openai-deployment" || report.ModelSource != "--model" {
		t.Fatalf("model = %q (%s), want explicit model", report.Model, report.ModelSource)
	}
	if report.CatalogModel != "gpt-5.4" || report.CatalogModelSource != "--catalog-model" {
		t.Fatalf("catalog_model = %q (%s), want explicit catalog model", report.CatalogModel, report.CatalogModelSource)
	}
	if report.Route != "responses_streaming" {
		t.Fatalf("route = %q, want responses_streaming", report.Route)
	}
	if !strings.Contains(report.RouteReason, "catalog_model=gpt-5.4 supports Responses streaming") {
		t.Fatalf("route_reason = %q, want catalog streaming reason", report.RouteReason)
	}
}

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

	var report struct {
		Smoke          any `json:"smoke"`
		RequestPreview struct {
			Requests []struct {
				Name               string `json:"name"`
				RetentionPayload   bool   `json:"retention_payload"`
				PreviousResponseID string `json:"previous_response_id"`
				Body               struct {
					Store              bool   `json:"store"`
					PreviousResponseID string `json:"previous_response_id"`
				} `json:"body"`
			} `json:"requests"`
		} `json:"request_preview"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, out.String())
	}
	if report.Smoke != nil {
		t.Fatalf("smoke = %#v, want omitted for --print-request", report.Smoke)
	}
	if len(report.RequestPreview.Requests) != 2 {
		t.Fatalf("request_preview = %#v, want two retention requests", report.RequestPreview)
	}
	followup := report.RequestPreview.Requests[1]
	if followup.Name != "retention_followup" || !followup.RetentionPayload {
		t.Fatalf("followup preview = %#v, want retention followup", followup)
	}
	if followup.PreviousResponseID == "" || followup.Body.PreviousResponseID != followup.PreviousResponseID || !followup.Body.Store {
		t.Fatalf("followup previous/store = %#v, want placeholder previous_response_id and store true", followup)
	}
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

	var report struct {
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
		Checks []struct {
			Name string `json:"name"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, out.String())
	}
	if report.Capabilities.Model != "corp-openai-deployment" ||
		report.Capabilities.CatalogModel != "gpt-5.4" ||
		!report.Capabilities.ResponsesAPI ||
		!report.Capabilities.ResponsesStreaming ||
		!report.Capabilities.Retention.PreviousResponseID ||
		!report.Capabilities.ServerCompaction.RequestPayload ||
		report.Capabilities.ServerCompaction.CompactThreshold <= 0 {
		t.Fatalf("capabilities = %+v, want resolved OpenAI capabilities", report.Capabilities)
	}
	for _, check := range report.Checks {
		if check.Name == "auth" {
			t.Fatalf("auth check should be skipped for capabilities-only report: %#v", report.Checks)
		}
	}
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
	for _, want := range []string{"--model", "--catalog-model", "--smoke", "--tool-smoke", "--retention-smoke", "--capabilities", "--print-request", "--timeout", "--json", "Diagnose OpenAI provider configuration"} {
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
	output := out.String()
	for _, want := range []string{
		"Route reason: model=gpt-5.4 uses Responses API; catalog_model=gpt-5.4 supports Responses streaming",
		"Smoke route: responses_streaming",
		"Smoke response ID: resp_text",
		"Smoke usage: input=10 cached=3 output=4 reasoning=2 cache_creation=1",
		"Smoke cost estimate: $0.00012345 USD",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
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
	output := out.String()
	for _, want := range []string{
		"Smoke request text: ok route=responses_streaming duration=1ms response_id=resp_text",
		"Smoke content text: xelyon openai doctor ok",
		"Smoke request tool: ok route=responses_streaming duration=2ms response_id=resp_tool",
		"Smoke usage tool: input=8 cached=0 output=4 reasoning=0 cache_creation=0",
		"Smoke total usage: input=18 cached=0 output=8 reasoning=0 cache_creation=0",
		"Smoke total cost estimate: $0.00012345 USD",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
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
	output := out.String()
	for _, want := range []string{
		"Smoke request retention_initial: ok route=responses_non_streaming duration=1ms response_id=resp_retention_initial previous_response_id=(not returned)",
		"Smoke request retention_followup: ok route=responses_non_streaming duration=2ms response_id=resp_retention_followup previous_response_id=resp_retention_initial",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output = %q, want substring %q", output, want)
		}
	}
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

	var got struct {
		Smoke struct {
			Route         string `json:"route"`
			ResponseID    string `json:"response_id"`
			UsageObserved bool   `json:"usage_observed"`
			Usage         struct {
				InputTokens         int `json:"input_tokens"`
				OutputTokens        int `json:"output_tokens"`
				ThinkingTokens      int `json:"thinking_tokens"`
				CachedInputTokens   int `json:"cached_input_tokens"`
				CacheCreationTokens int `json:"cache_creation_tokens"`
			} `json:"usage"`
			Cost struct {
				USD                float64 `json:"usd"`
				PricingUnavailable bool    `json:"pricing_unavailable"`
			} `json:"cost"`
			Requests []struct {
				Name               string `json:"name"`
				Ran                bool   `json:"ran"`
				RetentionPayload   bool   `json:"retention_payload"`
				Route              string `json:"route"`
				ResponseID         string `json:"response_id"`
				PreviousResponseID string `json:"previous_response_id"`
			} `json:"requests"`
		} `json:"smoke"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, out.String())
	}
	if got.Smoke.Route != "responses_non_streaming" || got.Smoke.ResponseID != "resp_json" || !got.Smoke.UsageObserved {
		t.Fatalf("smoke metadata = %#v, want route, response_id, and usage_observed", got.Smoke)
	}
	if got.Smoke.Usage.InputTokens != 10 ||
		got.Smoke.Usage.OutputTokens != 4 ||
		got.Smoke.Usage.ThinkingTokens != 2 ||
		got.Smoke.Usage.CachedInputTokens != 3 ||
		got.Smoke.Usage.CacheCreationTokens != 1 {
		t.Fatalf("smoke usage = %+v, want nested usage fields", got.Smoke.Usage)
	}
	if got.Smoke.Cost.USD != 0.00012345 || got.Smoke.Cost.PricingUnavailable {
		t.Fatalf("smoke cost = %+v, want nested cost fields", got.Smoke.Cost)
	}
	if len(got.Smoke.Requests) != 2 || got.Smoke.Requests[0].Name != "text" || got.Smoke.Requests[0].ResponseID != "resp_json" {
		t.Fatalf("smoke requests = %+v, want text and retention request metadata", got.Smoke.Requests)
	}
	if got.Smoke.Requests[1].Name != "retention_followup" ||
		!got.Smoke.Requests[1].RetentionPayload ||
		got.Smoke.Requests[1].PreviousResponseID != "resp_json" {
		t.Fatalf("retention request = %+v, want retention payload and previous_response_id", got.Smoke.Requests[1])
	}
}
