package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

type doctorJSONCheck struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Detail     string `json:"detail"`
	Suggestion string `json:"suggestion"`
}

type doctorJSONContractReport struct {
	Provider                  string                   `json:"provider"`
	APIURL                    string                   `json:"api_url"`
	ResponsesURL              string                   `json:"responses_url"`
	NormalizedBaseURL         string                   `json:"normalized_base_url"`
	Region                    string                   `json:"region"`
	Deployment                string                   `json:"deployment"`
	Model                     string                   `json:"model"`
	ModelSource               string                   `json:"model_source"`
	APIModel                  string                   `json:"api_model"`
	CatalogModel              string                   `json:"catalog_model"`
	CatalogModelSource        string                   `json:"catalog_model_source"`
	UpstreamProvider          string                   `json:"upstream_provider"`
	UpstreamModel             string                   `json:"upstream_model"`
	Route                     string                   `json:"route"`
	RouteReason               string                   `json:"route_reason"`
	MaxOutputTokens           int                      `json:"max_output_tokens"`
	ContextWindowTokens       int                      `json:"context_window_tokens"`
	FunctionCallingEnabled    bool                     `json:"function_calling_enabled"`
	ImageInputSupported       bool                     `json:"image_input_supported"`
	WebSearchSupported        bool                     `json:"web_search_supported"`
	ContextManagementEnabled  bool                     `json:"context_management_enabled"`
	ClaudeCompactionSupported bool                     `json:"claude_compaction_supported"`
	ContextCachingEnabled     bool                     `json:"context_caching_enabled"`
	ThinkingEnabled           bool                     `json:"thinking_enabled"`
	ThinkingSupported         bool                     `json:"thinking_supported"`
	ThinkingType              string                   `json:"thinking_type"`
	AnthropicVersion          string                   `json:"anthropic_version"`
	PromptCacheKeyPresent     bool                     `json:"prompt_cache_key_present"`
	Smoke                     any                      `json:"smoke"`
	RequestPreview            doctorJSONRequestPreview `json:"request_preview"`
	Checks                    []doctorJSONCheck        `json:"checks"`
}

type doctorJSONRequestPreview struct {
	Requests []doctorJSONRequestPreviewRequest `json:"requests"`
}

type doctorJSONRequestPreviewRequest struct {
	Name               string            `json:"name"`
	Skipped            bool              `json:"skipped"`
	SkipReason         string            `json:"skip_reason"`
	ToolPayload        bool              `json:"tool_payload"`
	ImagePayload       bool              `json:"image_payload"`
	WebSearchPayload   bool              `json:"web_search_payload"`
	RetentionPayload   bool              `json:"retention_payload"`
	ThinkingEnabled    bool              `json:"thinking_enabled"`
	Route              string            `json:"route"`
	Operation          string            `json:"operation"`
	ModelID            string            `json:"model_id"`
	Method             string            `json:"method"`
	URL                string            `json:"url"`
	PreviousResponseID string            `json:"previous_response_id"`
	Headers            map[string]string `json:"headers"`
	Body               json.RawMessage   `json:"body"`
}

func unmarshalDoctorJSON[T any](t *testing.T, out *bytes.Buffer) T {
	t.Helper()
	var report T
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal output: %v\n%s", err, out.String())
	}
	return report
}

func requireDoctorJSONPrintRequestOmittedSmoke(t *testing.T, smoke any) {
	t.Helper()
	if smoke != nil {
		t.Fatalf("smoke = %#v, want omitted for --print-request", smoke)
	}
}

func requireDoctorJSONRequestPreviewCount(t *testing.T, preview doctorJSONRequestPreview, want int) {
	t.Helper()
	if len(preview.Requests) != want {
		t.Fatalf("request_preview = %#v, want %d requests", preview, want)
	}
}

func requireDoctorJSONRequestPreviewAt(t *testing.T, preview doctorJSONRequestPreview, index int, name string) doctorJSONRequestPreviewRequest {
	t.Helper()
	if index >= len(preview.Requests) {
		t.Fatalf("missing request index %d in %#v", index, preview.Requests)
	}
	request := preview.Requests[index]
	if request.Name != name {
		t.Fatalf("request[%d] = %+v, want name=%q", index, request, name)
	}
	return request
}

func requireDoctorJSONRequestPreviewHeader(t *testing.T, request doctorJSONRequestPreviewRequest, name, want string) {
	t.Helper()
	if request.Headers[name] != want {
		t.Fatalf("%s preview = %q, want %q", name, request.Headers[name], want)
	}
}

func requireDoctorJSONRequestPreviewBody[T any](t *testing.T, request doctorJSONRequestPreviewRequest) T {
	t.Helper()
	if len(request.Body) == 0 {
		t.Fatalf("request body is empty: %+v", request)
	}
	var body T
	if err := json.Unmarshal(request.Body, &body); err != nil {
		t.Fatalf("unmarshal request body: %v\n%s", err, string(request.Body))
	}
	return body
}

func requireDoctorJSONRequestPreviewBodyMap(t *testing.T, request doctorJSONRequestPreviewRequest) map[string]any {
	t.Helper()
	return requireDoctorJSONRequestPreviewBody[map[string]any](t, request)
}

func requireDoctorJSONPreviewBodyContains(t *testing.T, body any, wants ...string) {
	t.Helper()
	rendered := renderedDoctorContractValue(t, body)
	for _, want := range wants {
		if !strings.Contains(rendered, want) {
			t.Fatalf("body = %#v, want substring %q", body, want)
		}
	}
}

func requireDoctorJSONPreviewBodyAbsent(t *testing.T, body map[string]any, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if _, ok := body[key]; ok {
			t.Fatalf("body should not contain %q: %#v", key, body)
		}
	}
}

func requireDoctorJSONCheck(t *testing.T, checks []doctorJSONCheck, name string) doctorJSONCheck {
	t.Helper()
	for _, check := range checks {
		if check.Name == name {
			return check
		}
	}
	t.Fatalf("missing %s check: %#v", name, checks)
	return doctorJSONCheck{}
}

func requireNoDoctorJSONChecks(t *testing.T, checks []doctorJSONCheck, names ...string) {
	t.Helper()
	for _, check := range checks {
		for _, name := range names {
			if check.Name == name {
				t.Fatalf("%s check should be skipped: %#v", check.Name, checks)
			}
		}
	}
}

func requireDoctorJSONCheckStatus(t *testing.T, check doctorJSONCheck, want string) {
	t.Helper()
	if check.Status != want {
		t.Fatalf("%s check status = %q, want %q: %#v", check.Name, check.Status, want, check)
	}
}

func requireDoctorJSONCheckDetailContains(t *testing.T, check doctorJSONCheck, want string) {
	t.Helper()
	if !strings.Contains(check.Detail, want) {
		t.Fatalf("%s check detail = %q, want substring %q", check.Name, check.Detail, want)
	}
}

func requireDoctorJSONCheckSuggestionContains(t *testing.T, check doctorJSONCheck, want string) {
	t.Helper()
	if !strings.Contains(check.Suggestion, want) {
		t.Fatalf("%s check suggestion = %q, want substring %q", check.Name, check.Suggestion, want)
	}
}
