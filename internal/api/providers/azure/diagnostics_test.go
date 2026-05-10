package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestDiagnose_FailsForMissingCredentialsAndPlaceholderDeployment(t *testing.T) {
	t.Setenv(baseURLEnv, "https://example.openai.azure.com/openai/v1")
	t.Setenv(apiKeyEnv, "")
	t.Setenv(authTokenEnv, "")
	t.Setenv(authTokenCommandEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{Config: config.DefaultConfig()})
	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want true: %#v", report.Checks)
	}
	if !hasDiagnosticCheck(report, "auth", DiagnosticStatusFail) {
		t.Fatalf("missing auth failure: %#v", report.Checks)
	}
	if !hasDiagnosticCheck(report, "deployment", DiagnosticStatusFail) {
		t.Fatalf("missing deployment failure: %#v", report.Checks)
	}
}

func TestDiagnose_AuthTokenCommandOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("token command test uses POSIX printf")
	}

	t.Setenv(baseURLEnv, "https://example.openai.azure.com/openai/v1")
	t.Setenv(apiKeyEnv, "")
	t.Setenv(authTokenEnv, "")
	t.Setenv(authTokenCommandEnv, "printf command-token")
	t.Setenv(authTokenCommandTimeoutEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55",
		CatalogModel: "gpt-5.5",
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.AuthMode != "entra_id_command" {
		t.Fatalf("AuthMode = %q, want entra_id_command", report.AuthMode)
	}
	if !hasDiagnosticCheck(report, "auth_token_command", DiagnosticStatusOK) {
		t.Fatalf("missing auth_token_command OK check: %#v", report.Checks)
	}
}

func TestDiagnose_AuthTokenWithRefreshCommandReportsCommandMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("token command test uses POSIX printf")
	}

	t.Setenv(baseURLEnv, "https://example.openai.azure.com/openai/v1")
	t.Setenv(apiKeyEnv, "")
	t.Setenv(authTokenEnv, "existing-token")
	t.Setenv(authTokenCommandEnv, "printf refreshed-token")
	t.Setenv(authTokenCommandTimeoutEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55",
		CatalogModel: "gpt-5.5",
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.AuthMode != "entra_id_command" {
		t.Fatalf("AuthMode = %q, want entra_id_command", report.AuthMode)
	}
	if !hasDiagnosticCheck(report, "auth_token_command", DiagnosticStatusOK) {
		t.Fatalf("missing auth_token_command OK check: %#v", report.Checks)
	}
}

func TestDiagnose_AuthTokenCommandFailureFailsWhenCommandIsOnlyCredential(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("token command test uses POSIX shell")
	}

	t.Setenv(baseURLEnv, "https://example.openai.azure.com/openai/v1")
	t.Setenv(apiKeyEnv, "")
	t.Setenv(authTokenEnv, "")
	t.Setenv(authTokenCommandEnv, "printf command-failed >&2; exit 2")
	t.Setenv(authTokenCommandTimeoutEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55",
		CatalogModel: "gpt-5.5",
	})

	if !hasDiagnosticCheck(report, "auth_token_command", DiagnosticStatusFail) {
		t.Fatalf("missing auth_token_command failure: %#v", report.Checks)
	}
}

func TestDiagnose_WarnsForAPIVersionQueryAndCatalogFallback(t *testing.T) {
	t.Setenv(baseURLEnv, "https://example.openai.azure.com/openai/v1?api-version=2025-04-01-preview")
	t.Setenv(apiKeyEnv, "azure-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:     config.DefaultConfig(),
		Deployment: "corp-gpt55-deployment",
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if !hasDiagnosticCheck(report, "api_version", DiagnosticStatusWarn) {
		t.Fatalf("missing api-version warning: %#v", report.Checks)
	}
	if !hasDiagnosticCheck(report, "catalog_model", DiagnosticStatusWarn) {
		t.Fatalf("missing catalog_model fallback warning: %#v", report.Checks)
	}
	if report.NormalizedBaseURL != "https://example.openai.azure.com/openai/v1" {
		t.Fatalf("NormalizedBaseURL = %q, want v1 URL without query", report.NormalizedBaseURL)
	}
}

func TestDiagnose_FailsForDeploymentScopedBaseURL(t *testing.T) {
	t.Setenv(baseURLEnv, "https://example.openai.azure.com/openai/deployments/corp-gpt55")
	t.Setenv(apiKeyEnv, "azure-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55",
		CatalogModel: "gpt-5.5",
	})

	if !hasDiagnosticCheck(report, "base_url", DiagnosticStatusFail) {
		t.Fatalf("missing deployment URL failure: %#v", report.Checks)
	}
}

func TestDiagnose_FailsForPublicOpenAIBaseURL(t *testing.T) {
	t.Setenv(baseURLEnv, "https://api.openai.com/v1")
	t.Setenv(apiKeyEnv, "azure-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55",
		CatalogModel: "gpt-5.5",
	})

	if !hasDiagnosticCheck(report, "base_url", DiagnosticStatusFail) {
		t.Fatalf("missing public OpenAI base URL failure: %#v", report.Checks)
	}
}

func TestDiagnose_WarnsForOpenAIKeyShape(t *testing.T) {
	t.Setenv(baseURLEnv, "https://example.openai.azure.com/openai/v1")
	t.Setenv(apiKeyEnv, "sk-public-openai-key")
	t.Setenv(authTokenEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55",
		CatalogModel: "gpt-5.5",
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if !hasDiagnosticCheck(report, "auth_key_shape", DiagnosticStatusWarn) {
		t.Fatalf("missing auth_key_shape warning: %#v", report.Checks)
	}
}

func TestDiagnose_WarnsWhenExplicitDeploymentLooksLikeCatalogModel(t *testing.T) {
	t.Setenv(baseURLEnv, "https://example.openai.azure.com/openai/v1")
	t.Setenv(apiKeyEnv, "azure-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:     config.DefaultConfig(),
		Deployment: "gpt-5.4",
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if !hasDiagnosticCheck(report, "deployment_catalog_mixup", DiagnosticStatusWarn) {
		t.Fatalf("missing deployment/catalog mixup warning: %#v", report.Checks)
	}
}

func TestDiagnose_WarnsWhenCatalogModelLooksLikeDeployment(t *testing.T) {
	t.Setenv(baseURLEnv, "https://example.openai.azure.com/openai/v1")
	t.Setenv(apiKeyEnv, "azure-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55",
		CatalogModel: "corp-gpt55",
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if !hasDiagnosticCheck(report, "catalog_model", DiagnosticStatusWarn) {
		t.Fatalf("missing catalog_model shape warning: %#v", report.Checks)
	}
}

func TestDiagnose_WarnsForAdvancedRetentionOverride(t *testing.T) {
	t.Setenv(baseURLEnv, "https://example.openai.azure.com/openai/v1")
	t.Setenv(apiKeyEnv, "azure-key")

	cfg := config.DefaultConfig()
	cfg.Responses.Store = false

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		Deployment:   "corp-gpt55",
		CatalogModel: "gpt-5.5",
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if !hasDiagnosticCheck(report, "responses_retention", DiagnosticStatusWarn) {
		t.Fatalf("missing responses_retention warning: %#v", report.Checks)
	}
}

func TestDiagnose_CatalogPolicyUsesCodexCatalogModel(t *testing.T) {
	t.Setenv(baseURLEnv, "https://example.openai.azure.com/openai/v1")
	t.Setenv(apiKeyEnv, "azure-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-codex-deployment",
		CatalogModel: "gpt-5.3-codex",
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	check, ok := diagnosticCheckByName(report, "catalog_policy")
	if !ok {
		t.Fatalf("missing catalog_policy check: %#v", report.Checks)
	}
	if check.Status != DiagnosticStatusOK {
		t.Fatalf("catalog_policy status = %s, want ok: %#v", check.Status, check)
	}
	for _, want := range []string{
		"catalog_model=gpt-5.3-codex",
		"context_window=400000",
		"max_output_tokens=128000",
		"responses_streaming=true",
		"input $1.75/M",
		"cached $0.175/M",
		"output $14.00/M",
	} {
		if !strings.Contains(check.Detail, want) {
			t.Fatalf("catalog_policy detail = %q, want substring %q", check.Detail, want)
		}
	}
}

func TestDiagnose_CatalogPolicyWarnsWhenMaxOutputFallsBackToProviderDefault(t *testing.T) {
	t.Setenv(baseURLEnv, "https://example.openai.azure.com/openai/v1")
	t.Setenv(apiKeyEnv, "azure-key")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt52-pro-deployment",
		CatalogModel: "gpt-5.2-pro",
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	check, ok := diagnosticCheckByName(report, "catalog_policy")
	if !ok {
		t.Fatalf("missing catalog_policy check: %#v", report.Checks)
	}
	if check.Status != DiagnosticStatusWarn {
		t.Fatalf("catalog_policy status = %s, want warn: %#v", check.Status, check)
	}
	if check.Message != "catalog_model is missing max output metadata" {
		t.Fatalf("catalog_policy message = %q, want max output metadata warning", check.Message)
	}
	for _, want := range []string{
		"catalog_model=gpt-5.2-pro",
		"context_window=400000",
		"max_output_tokens=missing",
		"runtime_fallback=16384",
		"input $21.00/M",
		"output $168.00/M",
	} {
		if !strings.Contains(check.Detail, want) {
			t.Fatalf("catalog_policy detail = %q, want substring %q", check.Detail, want)
		}
	}
}

func TestDiagnose_CatalogPolicyAllowsDeploymentMaxOutputOverride(t *testing.T) {
	t.Setenv(baseURLEnv, "https://example.openai.azure.com/openai/v1")
	t.Setenv(apiKeyEnv, "azure-key")

	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("azure", config.ProviderModelConfig{
		DefaultModel: "corp-gpt52-pro-deployment",
		CatalogModel: "gpt-5.2-pro",
		ModelOverrides: map[string]config.ModelOverride{
			"corp-gpt52-pro-deployment": {
				MaxOutputTokens: 64000,
			},
		},
	})

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       cfg,
		Deployment:   "corp-gpt52-pro-deployment",
		CatalogModel: "gpt-5.2-pro",
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	check, ok := diagnosticCheckByName(report, "catalog_policy")
	if !ok {
		t.Fatalf("missing catalog_policy check: %#v", report.Checks)
	}
	if check.Status != DiagnosticStatusOK {
		t.Fatalf("catalog_policy status = %s, want ok: %#v", check.Status, check)
	}
	if !strings.Contains(check.Detail, "max_output_tokens=64000 (model_overrides)") {
		t.Fatalf("catalog_policy detail = %q, want model override max output", check.Detail)
	}
}

func TestDiagnose_SmokeUsesConfiguredDeploymentAndStoreFalse(t *testing.T) {
	var received struct {
		Path   string
		APIKey string
		Body   map[string]any
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Path = r.URL.Path
		received.APIKey = r.Header.Get("api-key")
		if err := json.NewDecoder(r.Body).Decode(&received.Body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_doctor","output_text":"xelyon azure doctor ok","usage":{"input_tokens":10,"output_tokens":6,"input_tokens_details":{"cached_tokens":4},"output_tokens_details":{"reasoning_tokens":2}}}`))
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	t.Setenv(apiKeyEnv, "azure-key")
	t.Setenv(authTokenEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:          config.DefaultConfig(),
		Deployment:      "corp-gpt55-pro-deployment",
		CatalogModel:    "gpt-5.5-pro",
		RunSmoke:        true,
		MaxOutputTokens: 32,
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	policyCheck, ok := diagnosticCheckByName(report, "catalog_policy")
	if !ok {
		t.Fatalf("missing catalog_policy check: %#v", report.Checks)
	}
	if policyCheck.Status != DiagnosticStatusOK {
		t.Fatalf("catalog_policy status = %s, want ok: %#v", policyCheck.Status, policyCheck)
	}
	if !strings.Contains(policyCheck.Detail, "responses_streaming=false") {
		t.Fatalf("catalog_policy detail = %q, want non-streaming catalog capability", policyCheck.Detail)
	}
	if report.Smoke == nil || !report.Smoke.Ran {
		t.Fatalf("Smoke = %#v, want ran smoke", report.Smoke)
	}
	if report.Smoke.ResponseID != "resp_doctor" {
		t.Fatalf("Smoke.ResponseID = %q, want resp_doctor", report.Smoke.ResponseID)
	}
	if !report.Smoke.UsageObserved {
		t.Fatalf("Smoke.UsageObserved = false, want true: %#v", report.Smoke)
	}
	if report.Smoke.Usage.InputTokens != 10 ||
		report.Smoke.Usage.OutputTokens != 4 ||
		report.Smoke.Usage.CachedInputTokens != 4 ||
		report.Smoke.Usage.ThinkingTokens != 2 ||
		report.Smoke.Usage.CacheCreationTokens != 0 {
		t.Fatalf("Smoke.Usage = %+v, want input=10 output=4 cached=4 thinking=2 cache_creation=0", report.Smoke.Usage)
	}
	if report.Smoke.Cost.PricingUnavailable || report.Smoke.Cost.USD <= 0 {
		t.Fatalf("Smoke.Cost = %+v, want positive priced estimate", report.Smoke.Cost)
	}
	for _, want := range []struct {
		name   string
		status DiagnosticStatus
	}{
		{name: "response_id", status: DiagnosticStatusOK},
		{name: "usage", status: DiagnosticStatusOK},
		{name: "cost", status: DiagnosticStatusOK},
	} {
		if !hasDiagnosticCheck(report, want.name, want.status) {
			t.Fatalf("missing %s %s check: %#v", want.name, want.status, report.Checks)
		}
	}
	if !strings.Contains(report.Smoke.Content, "xelyon azure doctor ok") {
		t.Fatalf("Smoke.Content = %q, want smoke response", report.Smoke.Content)
	}
	if received.Path != "/openai/v1/responses" {
		t.Fatalf("path = %q, want /openai/v1/responses", received.Path)
	}
	if received.APIKey != "azure-key" {
		t.Fatalf("api-key = %q, want azure-key", received.APIKey)
	}
	if received.Body["model"] != "corp-gpt55-pro-deployment" {
		t.Fatalf("model = %#v, want deployment", received.Body["model"])
	}
	if received.Body["store"] != false {
		t.Fatalf("store = %#v, want false for doctor smoke", received.Body["store"])
	}
	if received.Body["stream"] == true {
		t.Fatalf("stream = true, want non-streaming for gpt-5.5-pro catalog model")
	}
	if got := int(received.Body["max_output_tokens"].(float64)); got != 32 {
		t.Fatalf("max_output_tokens = %d, want 32", got)
	}
	if _, ok := received.Body["tools"]; ok {
		t.Fatalf("tools should be omitted in doctor smoke: %#v", received.Body)
	}
}

func TestDiagnose_SmokeUsesCodexCatalogModelPolicy(t *testing.T) {
	var received struct {
		Path   string
		APIKey string
		Body   map[string]any
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Path = r.URL.Path
		received.APIKey = r.Header.Get("api-key")
		if err := json.NewDecoder(r.Body).Decode(&received.Body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"type":"response.created","response":{"id":"resp_codex"}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"response.output_text.delta","delta":"xelyon azure doctor ok"}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"response.completed","response":{"usage":{"input_tokens":13,"output_tokens":9,"input_tokens_details":{"cached_tokens":7},"output_tokens_details":{"reasoning_tokens":5}}}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: [DONE]`)
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	t.Setenv(apiKeyEnv, "azure-key")
	t.Setenv(authTokenEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:          config.DefaultConfig(),
		Deployment:      "corp-codex-deployment",
		CatalogModel:    "gpt-5.3-codex",
		RunSmoke:        true,
		MaxOutputTokens: 128000,
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.Smoke == nil || !report.Smoke.Ran {
		t.Fatalf("Smoke = %#v, want ran smoke", report.Smoke)
	}
	if report.Smoke.ResponseID != "resp_codex" {
		t.Fatalf("Smoke.ResponseID = %q, want resp_codex", report.Smoke.ResponseID)
	}
	if !report.Smoke.UsageObserved {
		t.Fatalf("Smoke.UsageObserved = false, want true: %#v", report.Smoke)
	}
	if report.Smoke.Usage.InputTokens != 13 ||
		report.Smoke.Usage.OutputTokens != 4 ||
		report.Smoke.Usage.CachedInputTokens != 7 ||
		report.Smoke.Usage.ThinkingTokens != 5 {
		t.Fatalf("Smoke.Usage = %+v, want input=13 output=4 cached=7 thinking=5", report.Smoke.Usage)
	}
	if report.Smoke.Cost.PricingUnavailable || report.Smoke.Cost.USD <= 0 {
		t.Fatalf("Smoke.Cost = %+v, want positive priced estimate", report.Smoke.Cost)
	}
	if !strings.Contains(report.Smoke.Content, "xelyon azure doctor ok") {
		t.Fatalf("Smoke.Content = %q, want smoke response", report.Smoke.Content)
	}
	if received.Path != "/openai/v1/responses" {
		t.Fatalf("path = %q, want /openai/v1/responses", received.Path)
	}
	if received.APIKey != "azure-key" {
		t.Fatalf("api-key = %q, want azure-key", received.APIKey)
	}
	if received.Body["model"] != "corp-codex-deployment" {
		t.Fatalf("model = %#v, want deployment", received.Body["model"])
	}
	if received.Body["store"] != false {
		t.Fatalf("store = %#v, want false for doctor smoke", received.Body["store"])
	}
	if received.Body["stream"] != true {
		t.Fatalf("stream = %#v, want true for gpt-5.3-codex catalog model", received.Body["stream"])
	}
	if got := int(received.Body["max_output_tokens"].(float64)); got != 128000 {
		t.Fatalf("max_output_tokens = %d, want 128000", got)
	}
	reasoning, ok := received.Body["reasoning"].(map[string]any)
	if !ok {
		t.Fatalf("reasoning = %#v, want reasoning object", received.Body["reasoning"])
	}
	if reasoning["effort"] != "low" {
		t.Fatalf("reasoning.effort = %#v, want low for codex catalog model", reasoning["effort"])
	}
}

func TestDiagnose_ToolSmokeIncludesToolPayloadWhenEnabled(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_tool_doctor","output":[{"type":"function_call","call_id":"call_probe","name":"xelyon_azure_doctor_probe","arguments":"{}"}],"usage":{"input_tokens":8,"output_tokens":4,"output_tokens_details":{"reasoning_tokens":1}}}`))
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	t.Setenv(apiKeyEnv, "azure-key")
	t.Setenv(authTokenEnv, "")
	t.Setenv("AZURE_OPENAI_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55-pro-deployment",
		CatalogModel: "gpt-5.5-pro",
		RunSmoke:     true,
		ToolSmoke:    true,
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.Smoke == nil || !report.Smoke.ToolPayload {
		t.Fatalf("Smoke = %#v, want tool payload smoke", report.Smoke)
	}
	if report.Smoke.ResponseID != "resp_tool_doctor" {
		t.Fatalf("Smoke.ResponseID = %q, want resp_tool_doctor", report.Smoke.ResponseID)
	}
	if !report.Smoke.UsageObserved {
		t.Fatalf("Smoke.UsageObserved = false, want true: %#v", report.Smoke)
	}
	if report.Smoke.Usage.InputTokens != 8 ||
		report.Smoke.Usage.OutputTokens != 3 ||
		report.Smoke.Usage.ThinkingTokens != 1 {
		t.Fatalf("Smoke.Usage = %+v, want input=8 output=3 thinking=1", report.Smoke.Usage)
	}
	if report.Smoke.Cost.PricingUnavailable || report.Smoke.Cost.USD <= 0 {
		t.Fatalf("Smoke.Cost = %+v, want positive priced estimate", report.Smoke.Cost)
	}
	if !strings.Contains(report.Smoke.Content, `"tool":"xelyon_azure_doctor_probe"`) {
		t.Fatalf("Smoke.Content = %q, want diagnostic tool call JSON", report.Smoke.Content)
	}
	tools, ok := received["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v, want one diagnostic tool", received["tools"])
	}
	tool, ok := tools[0].(map[string]any)
	if !ok || tool["name"] != "xelyon_azure_doctor_probe" {
		t.Fatalf("tool = %#v, want diagnostic probe", tools[0])
	}
	toolChoice, ok := received["tool_choice"].(map[string]any)
	if !ok {
		t.Fatalf("tool_choice = %#v, want forced function choice", received["tool_choice"])
	}
	if toolChoice["type"] != "function" || toolChoice["name"] != "xelyon_azure_doctor_probe" {
		t.Fatalf("tool_choice = %#v, want forced diagnostic function", toolChoice)
	}
	if !hasDiagnosticCheck(report, "tool_smoke", DiagnosticStatusOK) {
		t.Fatalf("missing tool_smoke OK check: %#v", report.Checks)
	}
}

func TestDiagnosticSmokeRequests_PreservesExplicitTextSmokeWithToolSmoke(t *testing.T) {
	requests := diagnosticSmokeRequests(DiagnosticOptions{
		TextSmoke: true,
		ToolSmoke: true,
	}, true)
	if len(requests) != 2 {
		t.Fatalf("len(requests) = %d, want text and tool requests: %#v", len(requests), requests)
	}
	if requests[0].Name != "text" || requests[1].Name != "tool" {
		t.Fatalf("request names = %q/%q, want text/tool", requests[0].Name, requests[1].Name)
	}
}

func TestDiagnose_RetentionSmokeChainsPreviousResponseIDAndForcesStore(t *testing.T) {
	var received []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		received = append(received, body)
		w.Header().Set("Content-Type", "application/json")
		switch len(received) {
		case 1:
			_, _ = w.Write([]byte(`{"id":"resp_retention_initial","output_text":"xelyon azure retention initial ok","usage":{"input_tokens":5,"output_tokens":2}}`))
		case 2:
			_, _ = w.Write([]byte(`{"id":"resp_retention_followup","output_text":"xelyon azure retention followup ok","usage":{"input_tokens":4,"output_tokens":2,"input_tokens_details":{"cached_tokens":1}}}`))
		default:
			t.Fatalf("unexpected smoke request count %d", len(received))
		}
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	t.Setenv(apiKeyEnv, "azure-key")
	t.Setenv(authTokenEnv, "")

	cfg := config.DefaultConfig()
	cfg.Responses.Store = false
	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         cfg,
		Deployment:     "corp-gpt55-pro-deployment",
		CatalogModel:   "gpt-5.5-pro",
		RunSmoke:       true,
		RetentionSmoke: true,
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.ResponsesStore {
		t.Fatal("ResponsesStore = true, want diagnostic report to preserve user config responses.store=false")
	}
	if !hasDiagnosticCheck(report, "responses_retention", DiagnosticStatusWarn) {
		t.Fatalf("missing responses_retention warning: %#v", report.Checks)
	}
	if !hasDiagnosticCheck(report, "retention_smoke", DiagnosticStatusOK) {
		t.Fatalf("missing retention_smoke OK check: %#v", report.Checks)
	}
	if len(received) != 2 {
		t.Fatalf("received %d smoke requests, want retention initial and followup", len(received))
	}
	if received[0]["store"] != true || received[1]["store"] != true {
		t.Fatalf("store values = %#v/%#v, want true for retention smoke", received[0]["store"], received[1]["store"])
	}
	if _, ok := received[0]["previous_response_id"]; ok {
		t.Fatalf("initial previous_response_id should be omitted: %#v", received[0])
	}
	if received[1]["previous_response_id"] != "resp_retention_initial" {
		t.Fatalf("followup previous_response_id = %#v, want resp_retention_initial", received[1]["previous_response_id"])
	}
	if report.Smoke == nil || !report.Smoke.RetentionPayload || len(report.Smoke.Requests) != 2 {
		t.Fatalf("Smoke = %#v, want two retention request results", report.Smoke)
	}
	if report.Smoke.Requests[0].Name != "retention_initial" || !report.Smoke.Requests[0].RetentionPayload || report.Smoke.Requests[0].PreviousResponseID != "" {
		t.Fatalf("initial request result = %#v, want retention initial without previous_response_id", report.Smoke.Requests[0])
	}
	if report.Smoke.Requests[1].Name != "retention_followup" || report.Smoke.Requests[1].PreviousResponseID != "resp_retention_initial" {
		t.Fatalf("followup request result = %#v, want chained previous_response_id", report.Smoke.Requests[1])
	}
	if !report.Smoke.UsageObserved || report.Smoke.Usage.InputTokens != 9 || report.Smoke.Usage.OutputTokens != 4 || report.Smoke.Usage.CachedInputTokens != 1 {
		t.Fatalf("Smoke usage = %+v observed=%t, want aggregate retention usage", report.Smoke.Usage, report.Smoke.UsageObserved)
	}
}

func TestDiagnose_RetentionSmokeDoesNotReportCachedInitialIDWhenFollowupOmitsID(t *testing.T) {
	var received []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		received = append(received, body)
		w.Header().Set("Content-Type", "application/json")
		switch len(received) {
		case 1:
			_, _ = w.Write([]byte(`{"id":"resp_retention_initial","output_text":"xelyon azure retention initial ok","usage":{"input_tokens":5,"output_tokens":2}}`))
		case 2:
			_, _ = w.Write([]byte(`{"output_text":"xelyon azure retention followup ok","usage":{"input_tokens":4,"output_tokens":2}}`))
		default:
			t.Fatalf("unexpected smoke request count %d", len(received))
		}
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	t.Setenv(apiKeyEnv, "azure-key")
	t.Setenv(authTokenEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         config.DefaultConfig(),
		Deployment:     "corp-gpt55-pro-deployment",
		CatalogModel:   "gpt-5.5-pro",
		RunSmoke:       true,
		RetentionSmoke: true,
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false when followup succeeds without response id: %#v", report.Checks)
	}
	if len(received) != 2 {
		t.Fatalf("received %d smoke requests, want retention initial and followup", len(received))
	}
	if received[1]["previous_response_id"] != "resp_retention_initial" {
		t.Fatalf("followup previous_response_id = %#v, want resp_retention_initial", received[1]["previous_response_id"])
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 2 {
		t.Fatalf("Smoke = %#v, want two retention request results", report.Smoke)
	}
	if report.Smoke.Requests[0].ResponseID != "resp_retention_initial" {
		t.Fatalf("initial response_id = %q, want resp_retention_initial", report.Smoke.Requests[0].ResponseID)
	}
	if report.Smoke.Requests[1].ResponseID != "" {
		t.Fatalf("followup response_id = %q, want empty when endpoint omits id", report.Smoke.Requests[1].ResponseID)
	}
	if report.Smoke.Requests[1].PreviousResponseID != "resp_retention_initial" {
		t.Fatalf("followup previous_response_id result = %q, want resp_retention_initial", report.Smoke.Requests[1].PreviousResponseID)
	}
}

func TestDiagnose_RetentionSmokeAllowsAuthRefreshRetryWithPreviousResponseID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("token command test uses POSIX printf")
	}

	var received []map[string]any
	var authorizations []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		received = append(received, body)
		authorizations = append(authorizations, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		switch len(received) {
		case 1:
			_, _ = w.Write([]byte(`{"id":"resp_retention_initial","output_text":"xelyon azure retention initial ok","usage":{"input_tokens":5,"output_tokens":2}}`))
		case 2:
			if body["previous_response_id"] != "resp_retention_initial" {
				t.Fatalf("second previous_response_id = %#v, want resp_retention_initial", body["previous_response_id"])
			}
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"message":"expired token"}}`))
		case 3:
			if body["previous_response_id"] != "resp_retention_initial" {
				t.Fatalf("third previous_response_id = %#v, want preserved resp_retention_initial", body["previous_response_id"])
			}
			_, _ = w.Write([]byte(`{"id":"resp_retention_followup","output_text":"xelyon azure retention followup ok","usage":{"input_tokens":4,"output_tokens":2}}`))
		default:
			t.Fatalf("unexpected smoke request count %d", len(received))
		}
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	t.Setenv(apiKeyEnv, "")
	t.Setenv(authTokenEnv, "expired-token")
	t.Setenv(authTokenCommandEnv, "printf refreshed-token")
	t.Setenv(authTokenCommandTimeoutEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         config.DefaultConfig(),
		Deployment:     "corp-gpt55-pro-deployment",
		CatalogModel:   "gpt-5.5-pro",
		RunSmoke:       true,
		RetentionSmoke: true,
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false for auth refresh retry: %#v", report.Checks)
	}
	if len(received) != 3 {
		t.Fatalf("received %d smoke requests, want initial + followup auth retry", len(received))
	}
	if authorizations[0] != "Bearer expired-token" ||
		authorizations[1] != "Bearer expired-token" ||
		authorizations[2] != "Bearer refreshed-token" {
		t.Fatalf("authorizations = %#v, want expired token then refreshed retry", authorizations)
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 2 || report.Smoke.Requests[1].PreviousResponseID != "resp_retention_initial" {
		t.Fatalf("Smoke = %#v, want successful retention followup after auth refresh", report.Smoke)
	}
	if !hasDiagnosticCheck(report, "retention_smoke", DiagnosticStatusOK) {
		t.Fatalf("missing retention_smoke OK check: %#v", report.Checks)
	}
}

func TestDiagnose_RetentionSmokeFailsWhenFollowupRetriesWithoutPreviousResponseID(t *testing.T) {
	var received []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		received = append(received, body)
		w.Header().Set("Content-Type", "application/json")
		switch len(received) {
		case 1:
			_, _ = w.Write([]byte(`{"id":"resp_retention_initial","output_text":"xelyon azure retention initial ok","usage":{"input_tokens":5,"output_tokens":2}}`))
		case 2:
			if body["previous_response_id"] != "resp_retention_initial" {
				t.Fatalf("second previous_response_id = %#v, want resp_retention_initial", body["previous_response_id"])
			}
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"invalid previous_response_id"}}`))
		case 3:
			if _, ok := body["previous_response_id"]; ok {
				t.Fatalf("retry request should omit previous_response_id after runtime recovery: %#v", body)
			}
			_, _ = w.Write([]byte(`{"id":"resp_retry","output_text":"retry without retention","usage":{"input_tokens":4,"output_tokens":2}}`))
		default:
			t.Fatalf("unexpected smoke request count %d", len(received))
		}
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	t.Setenv(apiKeyEnv, "azure-key")
	t.Setenv(authTokenEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:         config.DefaultConfig(),
		Deployment:     "corp-gpt55-pro-deployment",
		CatalogModel:   "gpt-5.5-pro",
		RunSmoke:       true,
		RetentionSmoke: true,
	})

	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want retention smoke failure: %#v", report.Checks)
	}
	if len(received) != 3 {
		t.Fatalf("received %d smoke requests, want retry path to be observed", len(received))
	}
	check, ok := diagnosticCheckByName(report, "smoke")
	if !ok || check.Status != DiagnosticStatusFail || !strings.Contains(check.Detail, "retry changed previous_response_id") {
		t.Fatalf("smoke check = %#v, %v; want retry failure detail", check, ok)
	}
	if report.Smoke == nil || len(report.Smoke.Requests) != 2 || !strings.Contains(report.Smoke.Requests[1].Error, "retry changed previous_response_id") {
		t.Fatalf("Smoke = %#v, want followup retry failure", report.Smoke)
	}
}

func TestDiagnose_SmokeWarnsForMissingResponseIDAndUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output_text":"xelyon azure doctor ok"}`))
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	t.Setenv(apiKeyEnv, "azure-key")
	t.Setenv(authTokenEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55-pro-deployment",
		CatalogModel: "gpt-5.5-pro",
		RunSmoke:     true,
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false for smoke observability warnings: %#v", report.Checks)
	}
	if report.Smoke == nil || !report.Smoke.Ran {
		t.Fatalf("Smoke = %#v, want ran smoke", report.Smoke)
	}
	if report.Smoke.ResponseID != "" {
		t.Fatalf("Smoke.ResponseID = %q, want empty when endpoint omits id", report.Smoke.ResponseID)
	}
	if report.Smoke.UsageObserved {
		t.Fatalf("Smoke.UsageObserved = true, want false: %#v", report.Smoke)
	}
	for _, name := range []string{"response_id", "usage", "cost"} {
		if !hasDiagnosticCheck(report, name, DiagnosticStatusWarn) {
			t.Fatalf("missing %s warn check: %#v", name, report.Checks)
		}
	}
}

func TestDiagnose_SmokeWarnsWhenPricingUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"type":"response.created","response":{"id":"resp_unknown_pricing"}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"response.output_text.delta","delta":"xelyon azure doctor ok"}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: {"type":"response.completed","response":{"usage":{"input_tokens":5,"output_tokens":2}}}`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `data: [DONE]`)
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	t.Setenv(apiKeyEnv, "azure-key")
	t.Setenv(authTokenEnv, "")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-unknown-deployment",
		CatalogModel: "gpt-unknown-model",
		RunSmoke:     true,
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false when only pricing is unavailable: %#v", report.Checks)
	}
	if report.Smoke == nil || !report.Smoke.UsageObserved {
		t.Fatalf("Smoke = %#v, want usage observed", report.Smoke)
	}
	if !report.Smoke.Cost.PricingUnavailable {
		t.Fatalf("Smoke.Cost = %+v, want pricing unavailable", report.Smoke.Cost)
	}
	if !hasDiagnosticCheck(report, "cost", DiagnosticStatusWarn) {
		t.Fatalf("missing cost warning: %#v", report.Checks)
	}
}

func TestDiagnose_ToolSmokeFailsWhenForcedToolCallIsMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_tool_doctor","output_text":"plain response"}`))
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	t.Setenv(apiKeyEnv, "azure-key")
	t.Setenv(authTokenEnv, "")
	t.Setenv("AZURE_OPENAI_FUNCTION_CALLING", "1")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55-pro-deployment",
		CatalogModel: "gpt-5.5-pro",
		RunSmoke:     true,
		ToolSmoke:    true,
	})

	if !hasDiagnosticCheck(report, "smoke", DiagnosticStatusFail) {
		t.Fatalf("missing smoke failure for absent tool call: %#v", report.Checks)
	}
}

func TestDiagnose_ToolSmokeSkippedWhenFunctionCallingDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var received map[string]any
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if _, ok := received["tools"]; ok {
			t.Fatalf("tools should be omitted when function calling is disabled: %#v", received)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_tool_disabled","output_text":"ok"}`))
	}))
	defer server.Close()

	t.Setenv(baseURLEnv, server.URL)
	t.Setenv(apiKeyEnv, "azure-key")
	t.Setenv(authTokenEnv, "")
	t.Setenv("AZURE_OPENAI_FUNCTION_CALLING", "0")

	report := Diagnose(context.Background(), DiagnosticOptions{
		Config:       config.DefaultConfig(),
		Deployment:   "corp-gpt55-pro-deployment",
		CatalogModel: "gpt-5.5-pro",
		RunSmoke:     true,
		ToolSmoke:    true,
	})

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %#v", report.Checks)
	}
	if report.Smoke == nil || report.Smoke.ToolPayload {
		t.Fatalf("Smoke = %#v, want basic smoke without tool payload", report.Smoke)
	}
	if !hasDiagnosticCheck(report, "tool_smoke", DiagnosticStatusWarn) {
		t.Fatalf("missing tool_smoke skip warning: %#v", report.Checks)
	}
}

func hasDiagnosticCheck(report DiagnosticReport, name string, status DiagnosticStatus) bool {
	for _, check := range report.Checks {
		if check.Name == name && check.Status == status {
			return true
		}
	}
	return false
}

func diagnosticCheckByName(report DiagnosticReport, name string) (DiagnosticCheck, bool) {
	for _, check := range report.Checks {
		if check.Name == name {
			return check, true
		}
	}
	return DiagnosticCheck{}, false
}
