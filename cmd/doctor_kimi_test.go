package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	kimiprovider "github.com/susugadx/xelyon-cli/internal/api/providers/kimi"
)

func TestRunKimiDoctorInvocation_JSONReportsExplicitModel(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MOONSHOT_API_KEY", "moonshot-key")
	t.Setenv("KIMI_API_URL", "")
	t.Setenv("XELYON_MODEL", "kimi-k2.6")

	cmd, out := newDoctorSubcommandTest(t, newKimiDoctorCommand)

	if err := cmd.Flags().Set("model", "kimi-k2.5"); err != nil {
		t.Fatalf("set model flag: %v", err)
	}
	doctorJSONFlag = true

	if err := runKimiDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runKimiDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	var report struct {
		Provider              string `json:"provider"`
		Model                 string `json:"model"`
		PromptCacheKeyPresent bool   `json:"prompt_cache_key_present"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, out.String())
	}
	if report.Provider != "kimi" {
		t.Fatalf("provider = %q, want kimi", report.Provider)
	}
	if report.Model != "kimi-k2.5" {
		t.Fatalf("model = %q, want kimi-k2.5", report.Model)
	}
	if !report.PromptCacheKeyPresent {
		t.Fatal("prompt_cache_key_present = false, want true")
	}
}

func TestRenderKimiDoctorText_WebSearchSmokeObservation(t *testing.T) {
	report := kimiprovider.DiagnosticReport{
		Provider:    "kimi",
		Model:       "kimi-k2.6",
		ModelSource: "test",
		APIURL:      "https://api.moonshot.ai/v1/chat/completions",
		Smoke: &kimiprovider.DiagnosticSmokeResult{
			Ran:                      true,
			WebSearchPayload:         true,
			Content:                  "ok",
			Duration:                 "10ms",
			UsageObserved:            true,
			CachedInputTokens:        3,
			WebSearchCallCount:       1,
			WebSearchCallFeeEstimate: 0.005,
			WebSearchUsageObserved:   true,
			SearchResultTotalTokens:  42,
		},
	}

	var out bytes.Buffer
	renderKimiDoctorText(&out, report)

	for _, want := range []string{
		"Web search call count: 1",
		"Web search call fee estimate: $0.0050 USD",
		"Web search usage observed: true",
		"Search result total tokens observed: 42",
		"call fee is separate from token cost",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("renderKimiDoctorText() output missing %q:\n%s", want, out.String())
		}
	}
}

func TestRenderKimiDoctorJSON_WebSearchSmokeObservation(t *testing.T) {
	report := kimiprovider.DiagnosticReport{
		Provider:    "kimi",
		Model:       "kimi-k2.6",
		ModelSource: "test",
		Smoke: &kimiprovider.DiagnosticSmokeResult{
			Ran:                      true,
			WebSearchPayload:         true,
			WebSearchCallCount:       2,
			WebSearchCallFeeEstimate: 0.010,
			WebSearchUsageObserved:   true,
			SearchResultTotalTokens:  55,
			CachedInputTokens:        7,
		},
	}

	var out bytes.Buffer
	if err := renderKimiDoctorJSON(&out, report); err != nil {
		t.Fatalf("renderKimiDoctorJSON() error = %v", err)
	}

	var parsed struct {
		Smoke struct {
			WebSearchCallCount       int     `json:"web_search_call_count"`
			WebSearchCallFeeEstimate float64 `json:"web_search_call_fee_estimate"`
			WebSearchUsageObserved   bool    `json:"web_search_usage_observed"`
			CachedInputTokens        int     `json:"cached_input_tokens"`
			SearchResultTotalTokens  int     `json:"search_result_total_tokens"`
		} `json:"smoke"`
	}
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, out.String())
	}
	if parsed.Smoke.WebSearchCallCount != 2 || parsed.Smoke.WebSearchCallFeeEstimate != 0.010 || !parsed.Smoke.WebSearchUsageObserved || parsed.Smoke.CachedInputTokens != 7 || parsed.Smoke.SearchResultTotalTokens != 55 {
		t.Fatalf("smoke JSON = %+v, want web search observation", parsed.Smoke)
	}
}

func TestRunKimiDoctorInvocation_UsesConfiguredModelWhenFlagOmitted(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MOONSHOT_API_KEY", "moonshot-key")
	t.Setenv("KIMI_API_URL", "")
	t.Setenv("XELYON_MODEL", "kimi-k2.5")

	cmd, out := newDoctorSubcommandTest(t, newKimiDoctorCommand)

	doctorJSONFlag = true

	if err := runKimiDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runKimiDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	var report struct {
		Model       string `json:"model"`
		ModelSource string `json:"model_source"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, out.String())
	}
	if report.Model != "kimi-k2.5" {
		t.Fatalf("model = %q, want XELYON_MODEL value kimi-k2.5", report.Model)
	}
	if report.ModelSource != "XELYON_MODEL" {
		t.Fatalf("model_source = %q, want XELYON_MODEL", report.ModelSource)
	}
}

func TestRunKimiDoctorInvocation_FailsForMissingKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MOONSHOT_API_KEY", "")
	t.Setenv("KIMI_API_URL", "")
	t.Setenv("XELYON_MODEL", "")

	cmd, out := newDoctorSubcommandTest(t, newKimiDoctorCommand)

	err := runKimiDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatalf("runKimiDoctorInvocation() error = nil, want diagnostics failure\noutput:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "MOONSHOT_API_KEY") {
		t.Fatalf("output = %q, want MOONSHOT_API_KEY failure", out.String())
	}
}

func TestRootCommand_KimiDoctorCommandParsesFlags(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("MOONSHOT_API_KEY", "moonshot-key")
	t.Setenv("KIMI_API_URL", "")
	t.Setenv("XELYON_MODEL", "")

	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "kimi", "--model", "kimi-k2.5", "--json"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), `"model": "kimi-k2.5"`) {
		t.Fatalf("output = %q, want parsed Kimi model", out.String())
	}
}

func TestRootCommand_KimiDoctorHelpShowsDoctorFlags(t *testing.T) {
	out := newRootCommandExecutionTest(t)
	rootCmd.SetArgs([]string{"doctor", "kimi", "--help"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root Execute() error = %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "--model") {
		t.Fatalf("output = %q, want Kimi doctor model flag", out.String())
	}
	if !strings.Contains(out.String(), "--tool-smoke") {
		t.Fatalf("output = %q, want Kimi doctor tool smoke flag", out.String())
	}
	if !strings.Contains(out.String(), "--image-smoke") {
		t.Fatalf("output = %q, want Kimi doctor image smoke flag", out.String())
	}
	if !strings.Contains(out.String(), "--web-search-smoke") {
		t.Fatalf("output = %q, want Kimi doctor web search smoke flag", out.String())
	}
	if !strings.Contains(out.String(), "Diagnose Kimi native provider configuration") {
		t.Fatalf("output = %q, want Kimi doctor help", out.String())
	}
}
