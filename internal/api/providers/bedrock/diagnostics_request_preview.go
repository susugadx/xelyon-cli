package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

const (
	bedrockDiagnosticOperationInvokeModelWithResponseStream = "invoke_model_with_response_stream"
	bedrockDiagnosticOperationConverseStream                = "converse_stream"
)

func (r *DiagnosticReport) addRequestPreview(ctx context.Context, cfg *config.Config, options DiagnosticOptions, requestPlan bedrockDiagnosticRequestPlan) {
	preview, err := buildBedrockDiagnosticRequestPreview(ctx, cfg, *r, options, requestPlan)
	r.RequestPreview = &preview
	if err != nil {
		r.addCheck(DiagnosticStatusFail, "request_preview", "Bedrock request preview could not be built", err.Error(), "")
		return
	}
	r.addCheck(
		DiagnosticStatusOK,
		"request_preview",
		"Bedrock request preview was built without sending a live request",
		fmt.Sprintf("requests=%d", len(preview.Requests)),
		"",
	)
}

func buildBedrockDiagnosticRequestPreview(
	ctx context.Context,
	cfg *config.Config,
	report DiagnosticReport,
	options DiagnosticOptions,
	requestPlan bedrockDiagnosticRequestPlan,
) (DiagnosticRequestPreview, error) {
	maxOutputTokens := bedrockDiagnosticRequestMaxOutputTokens(options)
	previewCfg := bedrockDiagnosticSmokeConfig(cfg, report, maxOutputTokens)
	provider := &Provider{
		region:        report.Region,
		runtimeConfig: previewCfg,
	}
	provider.SetMCPTools(nil)

	preview := DiagnosticRequestPreview{}
	for _, request := range requestPlan.Requests {
		if skipReason, ok := bedrockDiagnosticRequestSkipReason(report, request); ok {
			preview.Requests = append(preview.Requests, newBedrockDiagnosticSkippedPreviewRequest(request, report.Route, skipReason))
			continue
		}

		requestCfg := bedrockDiagnosticRequestConfig(previewCfg, request)
		requestCtx := newBedrockDiagnosticSmokeRequestContext(ctx, requestCfg, request, io.Discard)
		previewRequest, err := buildBedrockDiagnosticRequestPreviewRequest(requestCtx, provider, report, request)
		if err != nil {
			return preview, err
		}
		preview.Requests = append(preview.Requests, previewRequest)
	}
	return preview, nil
}

func buildBedrockDiagnosticRequestPreviewRequest(
	ctx context.Context,
	provider *Provider,
	report DiagnosticReport,
	request bedrockDiagnosticSmokeRequest,
) (DiagnosticRequestPreviewRequest, error) {
	req := provider.resolveBedrockRequestContext(ctx, report.Model)
	transport := providerdiag.RequestPreviewTransport{
		Method:  "POST",
		Headers: providerdiag.RedactedSigV4Headers(),
	}

	switch req.route {
	case bedrockRouteClaudeMessages:
		transport.URL = bedrockDiagnosticRequestPreviewURL(report.Region, req.model, "invoke-with-response-stream")
		if request.ImagePayload {
			transport.Body = provider.buildBedrockClaudeImageRequest(ctx, request.SystemPrompt, nil, request.UserContent, bedrockDiagnosticImage(), req)
		} else {
			transport.Body = provider.buildBedrockClaudeMessagesRequest(ctx, request.SystemPrompt, []api.Message{{Role: "user", Content: request.UserContent}}, req)
		}
		return providerdiag.NewInvocationPreviewRequest(
			request.invocationSmokeRequest(),
			string(req.route),
			bedrockDiagnosticOperationInvokeModelWithResponseStream,
			req.model,
			transport,
		), nil
	case bedrockRouteConverseStream:
		transport.URL = bedrockDiagnosticRequestPreviewURL(report.Region, req.model, "converse-stream")
		input, err := provider.buildConverseStreamInput(ctx, request.SystemPrompt, []api.Message{{Role: "user", Content: request.UserContent}}, req)
		if err != nil {
			return DiagnosticRequestPreviewRequest{}, err
		}
		body, err := normalizeBedrockConverseStreamPreviewBody(input)
		if err != nil {
			return DiagnosticRequestPreviewRequest{}, err
		}
		transport.Body = body
		return providerdiag.NewInvocationPreviewRequest(
			request.invocationSmokeRequest(),
			string(req.route),
			bedrockDiagnosticOperationConverseStream,
			req.model,
			transport,
		), nil
	default:
		return DiagnosticRequestPreviewRequest{}, fmt.Errorf("unsupported bedrock route %q for request preview", req.route)
	}
}

func newBedrockDiagnosticSkippedPreviewRequest(request bedrockDiagnosticSmokeRequest, route, skipReason string) DiagnosticRequestPreviewRequest {
	return providerdiag.NewSkippedInvocationPreviewRequest(request.invocationSmokeRequest(), route, skipReason)
}

func bedrockDiagnosticRequestPreviewURL(region, modelID, operationPath string) string {
	if region == "" {
		region = defaultRegion
	}
	return fmt.Sprintf(
		"https://bedrock-runtime.%s.amazonaws.com/model/%s/%s",
		region,
		url.PathEscape(modelID),
		operationPath,
	)
}

func normalizeBedrockConverseStreamPreviewBody(input *bedrockruntime.ConverseStreamInput) (map[string]any, error) {
	if input == nil {
		return nil, fmt.Errorf("bedrock ConverseStream input is nil")
	}

	body := map[string]any{}
	system, err := normalizeBedrockConverseSystem(input.System)
	if err != nil {
		return nil, err
	}
	if len(system) > 0 {
		body["system"] = system
	}

	messages, err := normalizeBedrockConverseMessages(input.Messages)
	if err != nil {
		return nil, err
	}
	if len(messages) > 0 {
		body["messages"] = messages
	}

	if input.InferenceConfig != nil {
		inferenceConfig := map[string]any{}
		if input.InferenceConfig.MaxTokens != nil {
			inferenceConfig["maxTokens"] = aws.ToInt32(input.InferenceConfig.MaxTokens)
		}
		if len(input.InferenceConfig.StopSequences) > 0 {
			inferenceConfig["stopSequences"] = input.InferenceConfig.StopSequences
		}
		if input.InferenceConfig.Temperature != nil {
			inferenceConfig["temperature"] = aws.ToFloat32(input.InferenceConfig.Temperature)
		}
		if input.InferenceConfig.TopP != nil {
			inferenceConfig["topP"] = aws.ToFloat32(input.InferenceConfig.TopP)
		}
		if len(inferenceConfig) > 0 {
			body["inferenceConfig"] = inferenceConfig
		}
	}

	if input.AdditionalModelRequestFields != nil {
		fields, err := bedrockDiagnosticDocumentValue(input.AdditionalModelRequestFields)
		if err != nil {
			return nil, err
		}
		body["additionalModelRequestFields"] = fields
	}
	if len(input.AdditionalModelResponseFieldPaths) > 0 {
		body["additionalModelResponseFieldPaths"] = input.AdditionalModelResponseFieldPaths
	}
	if len(input.RequestMetadata) > 0 {
		body["requestMetadata"] = input.RequestMetadata
	}

	if input.ToolConfig != nil {
		toolConfig, err := normalizeBedrockConverseToolConfig(input.ToolConfig)
		if err != nil {
			return nil, err
		}
		if len(toolConfig) > 0 {
			body["toolConfig"] = toolConfig
		}
	}

	return body, nil
}

func normalizeBedrockConverseSystem(blocks []bedrocktypes.SystemContentBlock) ([]map[string]any, error) {
	normalized := make([]map[string]any, 0, len(blocks))
	for _, block := range blocks {
		switch value := block.(type) {
		case *bedrocktypes.SystemContentBlockMemberText:
			normalized = append(normalized, map[string]any{"text": value.Value})
		default:
			return nil, fmt.Errorf("unsupported Bedrock Converse system block %T in request preview", block)
		}
	}
	return normalized, nil
}

func normalizeBedrockConverseMessages(messages []bedrocktypes.Message) ([]map[string]any, error) {
	normalized := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		content, err := normalizeBedrockConverseContent(message.Content)
		if err != nil {
			return nil, err
		}
		normalized = append(normalized, map[string]any{
			"role":    string(message.Role),
			"content": content,
		})
	}
	return normalized, nil
}

func normalizeBedrockConverseContent(blocks []bedrocktypes.ContentBlock) ([]map[string]any, error) {
	normalized := make([]map[string]any, 0, len(blocks))
	for _, block := range blocks {
		switch value := block.(type) {
		case *bedrocktypes.ContentBlockMemberText:
			normalized = append(normalized, map[string]any{"text": value.Value})
		case *bedrocktypes.ContentBlockMemberToolUse:
			input, err := bedrockDiagnosticDocumentValue(value.Value.Input)
			if err != nil {
				return nil, err
			}
			normalized = append(normalized, map[string]any{
				"toolUse": map[string]any{
					"toolUseId": aws.ToString(value.Value.ToolUseId),
					"name":      aws.ToString(value.Value.Name),
					"input":     input,
				},
			})
		case *bedrocktypes.ContentBlockMemberToolResult:
			content, err := normalizeBedrockConverseToolResultContent(value.Value.Content)
			if err != nil {
				return nil, err
			}
			normalized = append(normalized, map[string]any{
				"toolResult": map[string]any{
					"toolUseId": aws.ToString(value.Value.ToolUseId),
					"content":   content,
				},
			})
		default:
			return nil, fmt.Errorf("unsupported Bedrock Converse content block %T in request preview", block)
		}
	}
	return normalized, nil
}

func normalizeBedrockConverseToolResultContent(blocks []bedrocktypes.ToolResultContentBlock) ([]map[string]any, error) {
	normalized := make([]map[string]any, 0, len(blocks))
	for _, block := range blocks {
		switch value := block.(type) {
		case *bedrocktypes.ToolResultContentBlockMemberText:
			normalized = append(normalized, map[string]any{"text": value.Value})
		case *bedrocktypes.ToolResultContentBlockMemberJson:
			jsonValue, err := bedrockDiagnosticDocumentValue(value.Value)
			if err != nil {
				return nil, err
			}
			normalized = append(normalized, map[string]any{"json": jsonValue})
		default:
			return nil, fmt.Errorf("unsupported Bedrock Converse tool result content block %T in request preview", block)
		}
	}
	return normalized, nil
}

func normalizeBedrockConverseToolConfig(config *bedrocktypes.ToolConfiguration) (map[string]any, error) {
	result := map[string]any{}
	tools := make([]map[string]any, 0, len(config.Tools))
	for _, tool := range config.Tools {
		switch value := tool.(type) {
		case *bedrocktypes.ToolMemberToolSpec:
			spec, err := normalizeBedrockConverseToolSpec(value.Value)
			if err != nil {
				return nil, err
			}
			tools = append(tools, map[string]any{"toolSpec": spec})
		default:
			return nil, fmt.Errorf("unsupported Bedrock Converse tool config entry %T in request preview", tool)
		}
	}
	if len(tools) > 0 {
		result["tools"] = tools
	}
	if config.ToolChoice != nil {
		choice, err := normalizeBedrockConverseToolChoice(config.ToolChoice)
		if err != nil {
			return nil, err
		}
		result["toolChoice"] = choice
	}
	return result, nil
}

func normalizeBedrockConverseToolSpec(spec bedrocktypes.ToolSpecification) (map[string]any, error) {
	result := map[string]any{
		"name": aws.ToString(spec.Name),
	}
	if description := aws.ToString(spec.Description); description != "" {
		result["description"] = description
	}
	if spec.Strict != nil {
		result["strict"] = aws.ToBool(spec.Strict)
	}
	if spec.InputSchema != nil {
		schema, err := normalizeBedrockConverseToolInputSchema(spec.InputSchema)
		if err != nil {
			return nil, err
		}
		result["inputSchema"] = schema
	}
	return result, nil
}

func normalizeBedrockConverseToolInputSchema(schema bedrocktypes.ToolInputSchema) (map[string]any, error) {
	switch value := schema.(type) {
	case *bedrocktypes.ToolInputSchemaMemberJson:
		documentValue, err := bedrockDiagnosticDocumentValue(value.Value)
		if err != nil {
			return nil, err
		}
		return map[string]any{"json": documentValue}, nil
	default:
		return nil, fmt.Errorf("unsupported Bedrock Converse tool input schema %T in request preview", schema)
	}
}

func normalizeBedrockConverseToolChoice(choice bedrocktypes.ToolChoice) (map[string]any, error) {
	switch value := choice.(type) {
	case *bedrocktypes.ToolChoiceMemberAny:
		return map[string]any{"any": map[string]any{}}, nil
	case *bedrocktypes.ToolChoiceMemberAuto:
		return map[string]any{"auto": map[string]any{}}, nil
	case *bedrocktypes.ToolChoiceMemberTool:
		return map[string]any{"tool": map[string]any{"name": aws.ToString(value.Value.Name)}}, nil
	default:
		return nil, fmt.Errorf("unsupported Bedrock Converse tool choice %T in request preview", choice)
	}
}

type bedrockDiagnosticDocumentMarshaler interface {
	MarshalSmithyDocument() ([]byte, error)
}

func bedrockDiagnosticDocumentValue(document bedrockDiagnosticDocumentMarshaler) (any, error) {
	if document == nil {
		return map[string]any{}, nil
	}
	payload, err := document.MarshalSmithyDocument()
	if err != nil {
		return nil, err
	}
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		return nil, err
	}
	if value == nil {
		return map[string]any{}, nil
	}
	return value, nil
}
