package cmd

import (
	"bytes"
	"net/url"
	"strings"
	"testing"

	bedrockprovider "github.com/susugadx/xelyon-cli/internal/api/providers/bedrock"
)

type bedrockDoctorJSONContractReport struct {
	Provider               string            `json:"provider"`
	Region                 string            `json:"region"`
	Model                  string            `json:"model"`
	ModelSource            string            `json:"model_source"`
	CatalogModel           string            `json:"catalog_model"`
	CatalogModelSource     string            `json:"catalog_model_source"`
	Route                  string            `json:"route"`
	FunctionCallingEnabled bool              `json:"function_calling_enabled"`
	Checks                 []doctorJSONCheck `json:"checks"`
	RequestPreview         struct {
		Requests []bedrockDoctorJSONPreviewRequest `json:"requests"`
	} `json:"request_preview"`
	Smoke any `json:"smoke"`
}

type bedrockDoctorJSONPreviewRequest struct {
	Name            string            `json:"name"`
	Skipped         bool              `json:"skipped"`
	SkipReason      string            `json:"skip_reason"`
	ToolPayload     bool              `json:"tool_payload"`
	ImagePayload    bool              `json:"image_payload"`
	ThinkingEnabled bool              `json:"thinking_enabled"`
	Route           string            `json:"route"`
	Operation       string            `json:"operation"`
	ModelID         string            `json:"model_id"`
	Method          string            `json:"method"`
	URL             string            `json:"url"`
	Headers         map[string]string `json:"headers"`
	Body            map[string]any    `json:"body"`
}

func TestRunBedrockDoctorInvocation_JSONContractPrintRequestClaudeAllShapes(t *testing.T) {
	setBedrockDoctorCommandTestEnv(t)
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")

	const runtimeModel = "corp-bedrock-sonnet"
	cmd, out := newDoctorSubcommandTest(t, newBedrockDoctorCommand)
	doctorBedrockModelFlag = runtimeModel
	doctorCatalogModelFlag = bedrockDoctorCatalogModelForTest
	doctorSmokeFlag = true
	doctorToolSmokeFlag = true
	doctorBedrockImageSmokeFlag = true
	doctorBedrockThinkingSmokeFlag = true
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runBedrockDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runBedrockDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[bedrockDoctorJSONContractReport](t, out)
	if report.Provider != "bedrock" ||
		report.Region != "us-east-1" ||
		report.Model != runtimeModel ||
		report.ModelSource != "--model" ||
		report.CatalogModel != bedrockDoctorCatalogModelForTest ||
		report.CatalogModelSource != "--catalog-model" ||
		report.Route != "claude_messages" {
		t.Fatalf("Bedrock doctor JSON identity fields = %+v", report)
	}
	if !report.FunctionCallingEnabled {
		t.Fatalf("function_calling_enabled = false, want true")
	}
	if report.Smoke != nil {
		t.Fatalf("smoke = %#v, want omitted for --print-request", report.Smoke)
	}
	requireNoDoctorJSONChecks(t, report.Checks, "auth")
	for _, check := range []string{
		"region",
		"provider_registration",
		"model",
		"catalog_model",
		"route",
		"catalog_policy",
		"function_calling",
		"request_preview",
	} {
		requireDoctorJSONCheckStatus(t, requireDoctorJSONCheck(t, report.Checks, check), "ok")
	}

	if got, want := len(report.RequestPreview.Requests), 4; got != want {
		t.Fatalf("request_preview.requests length = %d, want %d", got, want)
	}
	text := requireBedrockDoctorJSONPreviewRequest(t, report, 0, "text", "claude_messages", "invoke_model_with_response_stream", runtimeModel, "invoke-with-response-stream")
	requireBedrockDoctorPreviewBodyContains(t, text.Body, "Reply with: xelyon bedrock doctor ok")
	requireBedrockDoctorPreviewBodyAbsent(t, text.Body, "tools", "thinking")

	tool := requireBedrockDoctorJSONPreviewRequest(t, report, 1, "tool", "claude_messages", "invoke_model_with_response_stream", runtimeModel, "invoke-with-response-stream")
	if !tool.ToolPayload {
		t.Fatalf("tool request = %+v, want tool_payload", tool)
	}
	requireBedrockDoctorPreviewBodyContains(t, tool.Body, "xelyon_bedrock_doctor_probe")

	image := requireBedrockDoctorJSONPreviewRequest(t, report, 2, "image", "claude_messages", "invoke_model_with_response_stream", runtimeModel, "invoke-with-response-stream")
	if !image.ImagePayload {
		t.Fatalf("image request = %+v, want image_payload", image)
	}
	requireBedrockDoctorPreviewBodyContains(t, image.Body, "image/png")
	requireBedrockDoctorPreviewBodyAbsent(t, image.Body, "tools")

	thinking := requireBedrockDoctorJSONPreviewRequest(t, report, 3, "thinking", "claude_messages", "invoke_model_with_response_stream", runtimeModel, "invoke-with-response-stream")
	if !thinking.ThinkingEnabled {
		t.Fatalf("thinking request = %+v, want thinking_enabled", thinking)
	}
	requireBedrockDoctorPreviewBodyContains(t, thinking.Body, "thinking")
}

func TestRunBedrockDoctorInvocation_JSONContractPrintRequestConverseSkippedShapes(t *testing.T) {
	setBedrockDoctorCommandTestEnv(t)
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_SESSION_TOKEN", "")

	const model = "amazon.nova-pro-v1:0"
	cmd, out := newDoctorSubcommandTest(t, newBedrockDoctorCommand)
	doctorBedrockModelFlag = model
	doctorCatalogModelFlag = model
	doctorSmokeFlag = true
	doctorToolSmokeFlag = true
	doctorBedrockImageSmokeFlag = true
	doctorBedrockThinkingSmokeFlag = true
	doctorPrintRequestFlag = true
	doctorJSONFlag = true

	if err := runBedrockDoctorInvocation(cmd, nil); err != nil {
		t.Fatalf("runBedrockDoctorInvocation() error = %v\noutput:\n%s", err, out.String())
	}

	report := unmarshalDoctorJSON[bedrockDoctorJSONContractReport](t, out)
	if report.Provider != "bedrock" ||
		report.Model != model ||
		report.CatalogModel != model ||
		report.Route != "converse_stream" {
		t.Fatalf("Bedrock Converse doctor JSON identity fields = %+v", report)
	}
	if report.Smoke != nil {
		t.Fatalf("smoke = %#v, want omitted for --print-request", report.Smoke)
	}
	requireNoDoctorJSONChecks(t, report.Checks, "auth")
	requireDoctorJSONCheckStatus(t, requireDoctorJSONCheck(t, report.Checks, "route"), "ok")
	requireDoctorJSONCheckStatus(t, requireDoctorJSONCheck(t, report.Checks, "request_preview"), "ok")

	if got, want := len(report.RequestPreview.Requests), 4; got != want {
		t.Fatalf("request_preview.requests length = %d, want %d", got, want)
	}
	text := requireBedrockDoctorJSONPreviewRequest(t, report, 0, "text", "converse_stream", "converse_stream", model, "converse-stream")
	requireBedrockDoctorPreviewBodyContains(t, text.Body, "messages", "inferenceConfig")
	requireBedrockDoctorPreviewBodyAbsent(t, text.Body, "toolConfig")

	tool := requireBedrockDoctorJSONPreviewRequest(t, report, 1, "tool", "converse_stream", "converse_stream", model, "converse-stream")
	if !tool.ToolPayload {
		t.Fatalf("tool request = %+v, want tool_payload", tool)
	}
	requireBedrockDoctorPreviewBodyContains(t, tool.Body, "toolConfig", "xelyon_bedrock_doctor_probe")

	requireBedrockDoctorJSONSkippedPreviewRequest(t, report, 2, "image", "converse_stream")
	requireBedrockDoctorJSONSkippedPreviewRequest(t, report, 3, "thinking", "converse_stream")
}

func TestRenderBedrockDoctorTextContractWithPreviewAndMultipleSmokeRequests(t *testing.T) {
	report := bedrockprovider.DiagnosticReport{
		Provider:           "bedrock",
		Region:             "us-east-1",
		Model:              "global.anthropic.claude-sonnet-4-6",
		ModelSource:        "test",
		CatalogModel:       "global.anthropic.claude-sonnet-4-6",
		CatalogModelSource: "test",
		Route:              "claude_messages",
		Checks: []bedrockprovider.DiagnosticCheck{
			{Name: "request_preview", Status: bedrockprovider.DiagnosticStatusOK, Message: "Bedrock request preview was built without sending a live request"},
			{Name: "smoke", Status: bedrockprovider.DiagnosticStatusOK, Message: "live Bedrock smoke requests completed"},
		},
		RequestPreview: &bedrockprovider.DiagnosticRequestPreview{
			Requests: []bedrockprovider.DiagnosticRequestPreviewRequest{{
				Name:      "text",
				Route:     "claude_messages",
				Operation: "invoke_model_with_response_stream",
				ModelID:   "global.anthropic.claude-sonnet-4-6",
				Method:    "POST",
				URL:       "https://bedrock-runtime.us-east-1.amazonaws.com/model/global.anthropic.claude-sonnet-4-6/invoke-with-response-stream",
				Headers:   map[string]string{"Authorization": "<redacted: AWS SigV4>"},
				Body:      map[string]any{"anthropic_version": "bedrock-2023-05-31"},
			}},
		},
		Smoke: &bedrockprovider.DiagnosticSmokeResult{
			Ran:           true,
			UsageObserved: true,
			Usage: bedrockprovider.DiagnosticSmokeUsage{
				InputTokens:         22,
				CachedInputTokens:   6,
				OutputTokens:        10,
				ThinkingTokens:      3,
				CacheCreationTokens: 8,
			},
			Cost: bedrockprovider.DiagnosticSmokeCost{USD: 0.00003},
			Requests: []bedrockprovider.DiagnosticSmokeRequestResult{
				{
					Name:          "text",
					Ran:           true,
					RequestID:     "req_text",
					Duration:      "1ms",
					Content:       "xelyon bedrock doctor ok",
					UsageObserved: true,
					Usage: bedrockprovider.DiagnosticSmokeUsage{
						InputTokens:         10,
						CachedInputTokens:   2,
						OutputTokens:        4,
						ThinkingTokens:      1,
						CacheCreationTokens: 3,
					},
					Cost: bedrockprovider.DiagnosticSmokeCost{USD: 0.00001},
				},
				{
					Name:          "tool",
					Ran:           true,
					RequestID:     "req_tool",
					Duration:      "2ms",
					UsageObserved: true,
					Usage: bedrockprovider.DiagnosticSmokeUsage{
						InputTokens:         12,
						CachedInputTokens:   4,
						OutputTokens:        6,
						ThinkingTokens:      2,
						CacheCreationTokens: 5,
					},
					Cost: bedrockprovider.DiagnosticSmokeCost{USD: 0.00002},
				},
				{Name: "image", Skipped: true, SkipReason: "unsupported route"},
			},
		},
	}

	var out bytes.Buffer
	renderBedrockDoctorText(&out, report)
	requireDoctorContractTextContainsAll(t, out.String(), []string{
		"Bedrock doctor",
		"Status: OK",
		"Request preview:",
		`"operation": "invoke_model_with_response_stream"`,
		`"Authorization": "<redacted: AWS SigV4>"`,
		"Smoke request text: ok duration=1ms request_id=req_text",
		"Smoke content text: xelyon bedrock doctor ok",
		"Smoke usage text: input=10 cached=2 output=4 reasoning=1 cache_creation=3",
		"Smoke cost estimate text: $0.00001000 USD",
		"Smoke request tool: ok duration=2ms request_id=req_tool",
		"Smoke usage tool: input=12 cached=4 output=6 reasoning=2 cache_creation=5",
		"Smoke cost estimate tool: $0.00002000 USD",
		"Smoke request image: skipped (unsupported route)",
		"Smoke total usage: input=22 cached=6 output=10 reasoning=3 cache_creation=8",
		"Smoke total cost estimate: $0.00003000 USD",
	})
}

func requireBedrockDoctorJSONPreviewRequest(t *testing.T, report bedrockDoctorJSONContractReport, index int, name, route, operation, modelID, urlPath string) bedrockDoctorJSONPreviewRequest {
	t.Helper()
	if index >= len(report.RequestPreview.Requests) {
		t.Fatalf("missing request index %d in %#v", index, report.RequestPreview.Requests)
	}
	request := report.RequestPreview.Requests[index]
	if request.Name != name || request.Route != route || request.Operation != operation || request.Method != "POST" {
		t.Fatalf("request[%d] = %+v, want name=%s route=%s operation=%s method=POST", index, request, name, route, operation)
	}
	if request.ModelID != modelID {
		t.Fatalf("request[%d] model_id = %q, want runtime model %q", index, request.ModelID, modelID)
	}
	if request.URL == "" {
		t.Fatalf("request[%d] url is empty, want Bedrock runtime target", index)
	}
	if request.Headers["Content-Type"] != "application/json" ||
		request.Headers["Accept"] != "application/json" ||
		request.Headers["Authorization"] != "<redacted: AWS SigV4>" {
		t.Fatalf("request[%d] headers = %#v, want redacted Bedrock JSON headers", index, request.Headers)
	}
	modelPath := "/model/" + url.PathEscape(modelID) + "/"
	if !strings.Contains(request.URL, modelPath) || !strings.Contains(request.URL, "/"+urlPath) {
		t.Fatalf("request[%d] url = %q, want runtime model path %q and operation path %q", index, request.URL, modelPath, urlPath)
	}
	return request
}

func requireBedrockDoctorJSONSkippedPreviewRequest(t *testing.T, report bedrockDoctorJSONContractReport, index int, name, route string) {
	t.Helper()
	if index >= len(report.RequestPreview.Requests) {
		t.Fatalf("missing request index %d in %#v", index, report.RequestPreview.Requests)
	}
	request := report.RequestPreview.Requests[index]
	if request.Name != name || request.Route != route || !request.Skipped {
		t.Fatalf("request[%d] = %+v, want skipped %s route=%s", index, request, name, route)
	}
	if !strings.Contains(request.SkipReason, "ConverseStream route does not support image or thinking smoke") {
		t.Fatalf("request[%d] skip_reason = %q, want ConverseStream unsupported shape", index, request.SkipReason)
	}
	if request.Operation != "" || request.Method != "" || request.URL != "" || request.Body != nil || request.Headers != nil {
		t.Fatalf("skipped request[%d] should not include send fields: %+v", index, request)
	}
}

func requireBedrockDoctorPreviewBodyContains(t *testing.T, body map[string]any, wants ...string) {
	t.Helper()
	rendered := renderedDoctorContractValue(t, body)
	for _, want := range wants {
		if !strings.Contains(rendered, want) {
			t.Fatalf("body = %#v, want substring %q", body, want)
		}
	}
}

func requireBedrockDoctorPreviewBodyAbsent(t *testing.T, body map[string]any, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if _, ok := body[key]; ok {
			t.Fatalf("body should not contain %q: %#v", key, body)
		}
	}
}
