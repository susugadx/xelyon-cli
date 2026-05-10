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
	for _, want := range []string{"--model", "--catalog-model", "--smoke", "--tool-smoke", "--timeout", "--json", "Diagnose OpenAI provider configuration"} {
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

func TestRenderOpenAIDoctorJSONIncludesSmokeObservability(t *testing.T) {
	report := openaiprovider.DiagnosticReport{
		Provider: "openai",
		Smoke: &openaiprovider.DiagnosticSmokeResult{
			Ran:           true,
			Route:         openaiprovider.DiagnosticRouteResponsesNonStreaming,
			ResponseID:    "resp_json",
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
				USD:                0.00012345,
				PricingUnavailable: false,
			},
			Requests: []openaiprovider.DiagnosticSmokeRequestResult{{
				Name:          "text",
				Ran:           true,
				Route:         openaiprovider.DiagnosticRouteResponsesNonStreaming,
				ResponseID:    "resp_json",
				UsageObserved: true,
			}},
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
				Name       string `json:"name"`
				Ran        bool   `json:"ran"`
				Route      string `json:"route"`
				ResponseID string `json:"response_id"`
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
	if len(got.Smoke.Requests) != 1 || got.Smoke.Requests[0].Name != "text" || got.Smoke.Requests[0].ResponseID != "resp_json" {
		t.Fatalf("smoke requests = %+v, want text request metadata", got.Smoke.Requests)
	}
}
