package clidoctor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	openaisubscription "github.com/susugadx/xelyon-cli/internal/api/providers/openai_subscription"
)

func TestRunOpenAISubscriptionDoctorInvocation_MissingAuthFailsWithLoginSuggestion(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_OPENAI_SUBSCRIPTION_AUTH_DIR", filepath.Join(t.TempDir(), "auth"))

	cmd, out := newDoctorSubcommandTest(t, newOpenAISubscriptionDoctorCommand)

	err := runOpenAISubscriptionDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatalf("runOpenAISubscriptionDoctorInvocation() error = nil, want login failure\noutput:\n%s", out.String())
	}
	output := out.String()
	for _, want := range []string{
		"OpenAI Subscription doctor",
		"Auth: login_required",
		"Run: xelyon auth openai-subscription login",
		"API cost: N/A",
		"previous_response_id: unsupported",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("doctor output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(strings.ToLower(output), "api key") {
		t.Fatalf("doctor output mentioned API key:\n%s", output)
	}
}

func TestRunOpenAISubscriptionDoctorInvocation_UnsafeAuthPermissionFails(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	authDir := filepath.Join(t.TempDir(), "auth")
	t.Setenv("XELYON_OPENAI_SUBSCRIPTION_AUTH_DIR", authDir)
	authConfig := openaisubscription.DefaultSubscriptionAuthConfig()
	if err := openaisubscription.SaveSubscriptionCredential(authConfig, openaisubscription.SubscriptionCredential{
		AccessToken:  "oauth-access-token",
		RefreshToken: "oauth-refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveSubscriptionCredential() error = %v", err)
	}
	if err := os.Chmod(filepath.Join(authDir, "openai_subscription.json"), 0o644); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}

	cmd, out := newDoctorSubcommandTest(t, newOpenAISubscriptionDoctorCommand)

	err := runOpenAISubscriptionDoctorInvocation(cmd, nil)
	if err == nil {
		t.Fatalf("runOpenAISubscriptionDoctorInvocation() error = nil, want unsafe permission failure\noutput:\n%s", out.String())
	}
	output := out.String()
	for _, want := range []string{
		"Status: FAIL",
		"Auth: permission_unsafe",
		"Fix openai_subscription auth file/directory permissions",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("doctor output missing %q:\n%s", want, output)
		}
	}
}

func TestRunOpenAISubscriptionDoctorInvocation_PrintRequestJSONRedactsAndShowsV2Shape(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_OPENAI_SUBSCRIPTION_AUTH_DIR", filepath.Join(t.TempDir(), "auth"))
	t.Setenv("XELYON_OPENAI_SUBSCRIPTION_ENDPOINT", "https://user-secret:pass-secret@proxy.example.test/backend-api/codex/responses?token=query-secret#frag-secret")

	cmd, out := newDoctorSubcommandTest(t, newOpenAISubscriptionDoctorCommand)

	doctorOpenAISubscriptionModelFlag = "gpt-5.4-mini"
	doctorCatalogModelFlag = "gpt-5.4-mini"
	doctorToolSmokeFlag = true
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runOpenAISubscriptionDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runOpenAISubscriptionDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[openAISubscriptionDoctorJSONReport](t, out)
	if report.Provider != "openai_subscription" || report.Billing != "ChatGPT subscription" || report.APICost != "N/A" || report.RuntimeMode != "full_payload" {
		t.Fatalf("subscription identity = %+v, want provider/billing/cost/runtime mode", report)
	}
	if report.Endpoint != "https://redacted@proxy.example.test/backend-api/codex/responses?redacted#redacted" {
		t.Fatalf("endpoint = %q, want sanitized endpoint", report.Endpoint)
	}
	requireNoDoctorJSONChecks(t, report.Checks, "auth")
	requireDoctorJSONCheckStatus(t, requireDoctorJSONCheck(t, report.Checks, "request_preview"), "ok")
	requireDoctorJSONRequestPreviewCount(t, report.RequestPreview, 1)
	request := requireDoctorJSONRequestPreviewAt(t, report.RequestPreview, 0, "tool")
	if request.URL != report.Endpoint {
		t.Fatalf("preview URL = %q, want sanitized report endpoint %q", request.URL, report.Endpoint)
	}
	requireDoctorJSONRequestPreviewHeader(t, request, "Authorization", "Bearer <redacted>")
	requireDoctorJSONRequestPreviewHeader(t, request, "originator", "xelyon")
	body := requireDoctorJSONRequestPreviewBody[openAISubscriptionDoctorPreviewBody](t, request)
	if body.Model != "gpt-5.4-mini" || !body.Stream || body.Store || body.ToolsCount <= 0 {
		t.Fatalf("preview body = %+v, want streaming store=false tool payload", body)
	}
	if body.PromptCacheKey != "present" || body.PromptCacheRetention != "omitted" || body.PreviousResponseIDPresent || body.ContextManagementCount != 0 {
		t.Fatalf("preview body cache/retention/chain = %+v, want prompt_cache_key only and no chain", body)
	}
	if body.MaxOutputTokens != "omitted" {
		t.Fatalf("preview body max_output_tokens = %#v, want omitted", body.MaxOutputTokens)
	}
	if body.ToolChoice == nil {
		t.Fatalf("preview body tool_choice = nil, want diagnostic tool choice")
	}
	output := out.String()
	for _, leaked := range []string{
		"Call xelyon_openai_subscription_doctor_probe exactly once",
		"Reply with: xelyon",
		"access_token",
		"refresh_token",
		"id_token",
		"acct_secret",
		"user-secret",
		"pass-secret",
		"query-secret",
		"frag-secret",
	} {
		if strings.Contains(output, leaked) {
			t.Fatalf("doctor JSON leaked %q:\n%s", leaked, output)
		}
	}
}

func TestRunOpenAISubscriptionDoctorInvocation_CapabilitiesAndRequiredCapabilityAreLocalOnly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_OPENAI_SUBSCRIPTION_AUTH_DIR", filepath.Join(t.TempDir(), "auth"))

	cmd, out := newDoctorSubcommandTest(t, newOpenAISubscriptionDoctorCommand)

	doctorOpenAISubscriptionModelFlag = "gpt-5.5"
	doctorCatalogModelFlag = "gpt-5.5"
	doctorCapabilitiesFlag = true
	doctorRequiredCapabilityFlags = []string{"responses_streaming", "function_calling"}
	doctorJSONFlag = true

	if err := runOpenAISubscriptionDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runOpenAISubscriptionDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[openAISubscriptionDoctorJSONReport](t, out)
	requireNoDoctorJSONChecks(t, report.Checks, "auth", "endpoint")
	requireDoctorJSONCheckStatus(t, requireDoctorJSONCheck(t, report.Checks, "capabilities"), "ok")
	required := requireDoctorJSONCheck(t, report.Checks, "required_capability")
	requireDoctorJSONCheckStatus(t, required, "ok")
	requireDoctorJSONCheckDetailContains(t, required, "responses_streaming=ok")
	requireDoctorJSONCheckDetailContains(t, required, "function_calling=ok")
	if !report.Capabilities.ResponsesAPI || !report.Capabilities.ResponsesStreaming || report.Capabilities.ChatCompletions {
		t.Fatalf("capabilities = %+v, want Responses streaming only", report.Capabilities)
	}
	if report.Capabilities.Retention.PreviousResponseID || report.Capabilities.ServerCompaction.RequestPayload {
		t.Fatalf("capabilities = %+v, want no response chain/server compaction", report.Capabilities)
	}
	if report.Capabilities.Pricing.Available {
		t.Fatalf("pricing capability = %+v, want subscription pricing unavailable", report.Capabilities.Pricing)
	}
}

type openAISubscriptionDoctorJSONReport struct {
	Provider       string                   `json:"provider"`
	Endpoint       string                   `json:"endpoint"`
	Billing        string                   `json:"billing"`
	APICost        string                   `json:"api_cost"`
	RuntimeMode    string                   `json:"runtime_mode"`
	Checks         []doctorJSONCheck        `json:"checks"`
	RequestPreview doctorJSONRequestPreview `json:"request_preview"`
	Capabilities   struct {
		ResponsesAPI       bool `json:"responses_api"`
		ResponsesStreaming bool `json:"responses_streaming"`
		ChatCompletions    bool `json:"chat_completions"`
		Retention          struct {
			PreviousResponseID bool `json:"previous_response_id"`
		} `json:"retention"`
		ServerCompaction struct {
			RequestPayload bool `json:"request_payload"`
		} `json:"server_compaction"`
		Pricing struct {
			Available bool `json:"available"`
		} `json:"pricing"`
	} `json:"capabilities"`
}

type openAISubscriptionDoctorPreviewBody struct {
	Model                     string          `json:"model"`
	Stream                    bool            `json:"stream"`
	Store                     bool            `json:"store"`
	ToolsCount                int             `json:"tools_count"`
	PromptCacheKey            string          `json:"prompt_cache_key"`
	PromptCacheRetention      string          `json:"prompt_cache_retention"`
	PreviousResponseIDPresent bool            `json:"previous_response_id_present"`
	ContextManagementCount    int             `json:"context_management_count"`
	MaxOutputTokens           string          `json:"max_output_tokens"`
	ToolChoice                json.RawMessage `json:"tool_choice"`
}
