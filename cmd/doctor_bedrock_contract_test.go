package cmd

import (
	"bytes"
	"net/url"
	"strings"
	"testing"

	bedrockprovider "github.com/susugadx/xelyon-cli/internal/api/providers/bedrock"
)

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

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
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
	requireDoctorJSONPrintRequestOmittedSmoke(t, report.Smoke)
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

	requireDoctorJSONRequestPreviewCount(t, report.RequestPreview, 4)
	text := requireBedrockDoctorJSONPreviewRequest(t, report, 0, "text", "claude_messages", "invoke_model_with_response_stream", runtimeModel, "invoke-with-response-stream")
	textBody := requireDoctorJSONRequestPreviewBodyMap(t, text)
	requireDoctorJSONPreviewBodyContains(t, textBody, "Reply with: xelyon bedrock doctor ok")
	requireDoctorJSONPreviewBodyAbsent(t, textBody, "tools", "thinking")

	tool := requireBedrockDoctorJSONPreviewRequest(t, report, 1, "tool", "claude_messages", "invoke_model_with_response_stream", runtimeModel, "invoke-with-response-stream")
	if !tool.ToolPayload {
		t.Fatalf("tool request = %+v, want tool_payload", tool)
	}
	requireDoctorJSONPreviewBodyContains(t, requireDoctorJSONRequestPreviewBodyMap(t, tool), "xelyon_bedrock_doctor_probe")

	image := requireBedrockDoctorJSONPreviewRequest(t, report, 2, "image", "claude_messages", "invoke_model_with_response_stream", runtimeModel, "invoke-with-response-stream")
	if !image.ImagePayload {
		t.Fatalf("image request = %+v, want image_payload", image)
	}
	imageBody := requireDoctorJSONRequestPreviewBodyMap(t, image)
	requireDoctorJSONPreviewBodyContains(t, imageBody, "image/png")
	requireDoctorJSONPreviewBodyAbsent(t, imageBody, "tools")

	thinking := requireBedrockDoctorJSONPreviewRequest(t, report, 3, "thinking", "claude_messages", "invoke_model_with_response_stream", runtimeModel, "invoke-with-response-stream")
	if !thinking.ThinkingEnabled {
		t.Fatalf("thinking request = %+v, want thinking_enabled", thinking)
	}
	requireDoctorJSONPreviewBodyContains(t, requireDoctorJSONRequestPreviewBodyMap(t, thinking), "thinking")
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

	report := unmarshalDoctorJSON[doctorJSONContractReport](t, out)
	if report.Provider != "bedrock" ||
		report.Model != model ||
		report.CatalogModel != model ||
		report.Route != "converse_stream" {
		t.Fatalf("Bedrock Converse doctor JSON identity fields = %+v", report)
	}
	requireDoctorJSONPrintRequestOmittedSmoke(t, report.Smoke)
	requireNoDoctorJSONChecks(t, report.Checks, "auth")
	requireDoctorJSONCheckStatus(t, requireDoctorJSONCheck(t, report.Checks, "route"), "ok")
	requireDoctorJSONCheckStatus(t, requireDoctorJSONCheck(t, report.Checks, "request_preview"), "ok")

	requireDoctorJSONRequestPreviewCount(t, report.RequestPreview, 4)
	text := requireBedrockDoctorJSONPreviewRequest(t, report, 0, "text", "converse_stream", "converse_stream", model, "converse-stream")
	textBody := requireDoctorJSONRequestPreviewBodyMap(t, text)
	requireDoctorJSONPreviewBodyContains(t, textBody, "messages", "inferenceConfig")
	requireDoctorJSONPreviewBodyAbsent(t, textBody, "toolConfig")

	tool := requireBedrockDoctorJSONPreviewRequest(t, report, 1, "tool", "converse_stream", "converse_stream", model, "converse-stream")
	if !tool.ToolPayload {
		t.Fatalf("tool request = %+v, want tool_payload", tool)
	}
	requireDoctorJSONPreviewBodyContains(t, requireDoctorJSONRequestPreviewBodyMap(t, tool), "toolConfig", "xelyon_bedrock_doctor_probe")

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

func requireBedrockDoctorJSONPreviewRequest(t *testing.T, report doctorJSONContractReport, index int, name, route, operation, modelID, urlPath string) doctorJSONRequestPreviewRequest {
	t.Helper()
	request := requireDoctorJSONRequestPreviewAt(t, report.RequestPreview, index, name)
	if request.Name != name || request.Route != route || request.Operation != operation || request.Method != "POST" {
		t.Fatalf("request[%d] = %+v, want name=%s route=%s operation=%s method=POST", index, request, name, route, operation)
	}
	if request.ModelID != modelID {
		t.Fatalf("request[%d] model_id = %q, want runtime model %q", index, request.ModelID, modelID)
	}
	if request.URL == "" {
		t.Fatalf("request[%d] url is empty, want Bedrock runtime target", index)
	}
	requireDoctorJSONRequestPreviewHeader(t, request, "Content-Type", "application/json")
	requireDoctorJSONRequestPreviewHeader(t, request, "Accept", "application/json")
	requireDoctorJSONRequestPreviewHeader(t, request, "Authorization", "<redacted: AWS SigV4>")
	modelPath := "/model/" + url.PathEscape(modelID) + "/"
	if !strings.Contains(request.URL, modelPath) || !strings.Contains(request.URL, "/"+urlPath) {
		t.Fatalf("request[%d] url = %q, want runtime model path %q and operation path %q", index, request.URL, modelPath, urlPath)
	}
	return request
}

func requireBedrockDoctorJSONSkippedPreviewRequest(t *testing.T, report doctorJSONContractReport, index int, name, route string) {
	t.Helper()
	request := requireDoctorJSONRequestPreviewAt(t, report.RequestPreview, index, name)
	if request.Name != name || request.Route != route || !request.Skipped {
		t.Fatalf("request[%d] = %+v, want skipped %s route=%s", index, request, name, route)
	}
	if !strings.Contains(request.SkipReason, "ConverseStream route does not support image or thinking smoke") {
		t.Fatalf("request[%d] skip_reason = %q, want ConverseStream unsupported shape", index, request.SkipReason)
	}
	if request.Operation != "" || request.Method != "" || request.URL != "" || len(request.Body) != 0 || request.Headers != nil {
		t.Fatalf("skipped request[%d] should not include send fields: %+v", index, request)
	}
}
