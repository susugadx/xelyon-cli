package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrockdocument "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

func (p *Provider) chatWithConverseStream(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, req bedrockRequestContext) (string, error) {
	if api.ShouldSendToolPayload(ctx, p.IsFunctionCallingEnabled()) {
		if err := ensureBedrockConverseToolUseSupported(req); err != nil {
			return "", err
		}
	}
	if image != nil && image.Base64 != "" {
		return "", fmt.Errorf("bedrock ConverseStream route does not support image input yet: model=%q catalog_model=%q", req.model, req.catalogModel)
	}
	if api.MessagesHaveImage(history) {
		return "", fmt.Errorf("bedrock ConverseStream route does not support image input yet: model=%q catalog_model=%q", req.model, req.catalogModel)
	}
	if p.converseClient == nil {
		return "", fmt.Errorf("bedrock ConverseStream client is not configured")
	}
	if strings.TrimSpace(userMessage) != "" {
		history = append(history, api.Message{Role: "user", Content: userMessage})
	}

	input, err := p.buildConverseStreamInput(ctx, systemPrompt, history, req)
	if err != nil {
		return "", err
	}

	spinner := api.StartThinkingSpinner(ctx, false, "")
	p.clearLastRequestID()
	output, err := p.converseClient.ConverseStream(ctx, input)
	if err != nil {
		spinner.Stop()
		return "", fmt.Errorf("bedrock converse API error: %w", err)
	}
	p.captureRequestIDFromConverseOutput(output)

	return p.handleConverseStream(ctx, output, spinner)
}

func (p *Provider) buildConverseStreamInput(ctx context.Context, systemPrompt string, history []api.Message, req bedrockRequestContext) (*bedrockruntime.ConverseStreamInput, error) {
	messages, err := convertToConverseMessages(history)
	if err != nil {
		return nil, fmt.Errorf("bedrock converse request conversion failed: %w", err)
	}

	input := &bedrockruntime.ConverseStreamInput{
		ModelId:  aws.String(req.model),
		Messages: messages,
		System:   buildConverseSystemBlocks(systemPrompt, api.ActiveContextBlocksFromContext(ctx)),
	}
	if maxTokens := resolveConverseMaxTokens(req); maxTokens != nil {
		input.InferenceConfig = &types.InferenceConfiguration{MaxTokens: maxTokens}
	}
	if api.ShouldSendToolPayload(ctx, p.IsFunctionCallingEnabled()) {
		input.ToolConfig = buildConverseToolConfig(ctx, p.mcpTools)
	}
	return input, nil
}

func resolveConverseMaxTokens(req bedrockRequestContext) *int32 {
	policy := converseMaxOutputPolicy(req)
	if policy.available {
		return int32Ptr(policy.tokens)
	}
	return nil
}

type bedrockMaxOutputSource int

const (
	bedrockMaxOutputSourceMissing bedrockMaxOutputSource = iota
	bedrockMaxOutputSourceModelOverrides
	bedrockMaxOutputSourceCatalog
	bedrockMaxOutputSourceProviderDefault
)

type bedrockMaxOutputPolicy struct {
	tokens    int
	source    bedrockMaxOutputSource
	available bool
}

func converseMaxOutputPolicy(req bedrockRequestContext) bedrockMaxOutputPolicy {
	if req.cfg != nil {
		if override, ok := req.cfg.ModelOverrideForProvider("bedrock", req.model); ok && override.MaxOutputTokens > 0 {
			return bedrockMaxOutputPolicy{
				tokens:    override.MaxOutputTokens,
				source:    bedrockMaxOutputSourceModelOverrides,
				available: true,
			}
		}
	}
	if llmcatalog.IsKnownModelNameForProvider("bedrock", req.catalogModel) {
		if tokens, ok := llmcatalog.KnownMaxOutputTokens(req.catalogModel); ok {
			return bedrockMaxOutputPolicy{
				tokens:    tokens,
				source:    bedrockMaxOutputSourceCatalog,
				available: true,
			}
		}
	}
	if req.catalogModel != req.model && llmcatalog.IsKnownModelNameForProvider("bedrock", req.model) {
		if tokens, ok := llmcatalog.KnownMaxOutputTokens(req.model); ok {
			return bedrockMaxOutputPolicy{
				tokens:    tokens,
				source:    bedrockMaxOutputSourceCatalog,
				available: true,
			}
		}
	}
	return bedrockMaxOutputPolicy{source: bedrockMaxOutputSourceMissing}
}

func buildConverseSystemBlocks(systemPrompt string, activeContext []api.ActiveContextBlock) []types.SystemContentBlock {
	renderedActiveContext := api.RenderActiveContextBlocks(activeContext)
	if strings.TrimSpace(systemPrompt) == "" && renderedActiveContext == "" {
		return nil
	}
	blocks := make([]types.SystemContentBlock, 0, 2)
	if strings.TrimSpace(systemPrompt) != "" {
		blocks = append(blocks, &types.SystemContentBlockMemberText{Value: systemPrompt})
	}
	if renderedActiveContext != "" {
		blocks = append(blocks, &types.SystemContentBlockMemberText{Value: renderedActiveContext})
	}
	return blocks
}

func buildConverseToolConfig(ctx context.Context, additionalTools []api.ToolDefinition) *types.ToolConfiguration {
	defs := api.ToolDefinitionsWithAdditional(ctx, additionalTools)
	if len(defs) == 0 {
		return nil
	}

	tools := make([]types.Tool, 0, len(defs))
	for _, def := range defs {
		schema := def.Parameters
		if len(schema) == 0 {
			schema = map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			}
		}
		spec := types.ToolSpecification{
			Name: aws.String(def.Name),
			InputSchema: &types.ToolInputSchemaMemberJson{
				Value: bedrockdocument.NewLazyDocument(schema),
			},
		}
		if def.Description != "" {
			spec.Description = aws.String(def.Description)
		}
		if def.Strict {
			spec.Strict = aws.Bool(true)
		}
		tools = append(tools, &types.ToolMemberToolSpec{Value: spec})
	}

	return &types.ToolConfiguration{Tools: tools}
}

func convertToConverseMessages(history []api.Message) ([]types.Message, error) {
	messages := make([]types.Message, 0, len(history))
	lastWasToolResults := false

	for _, msg := range history {
		switch msg.Role {
		case "system":
			lastWasToolResults = false
			continue
		case "user":
			if msg.HasImage() {
				return nil, fmt.Errorf("bedrock ConverseStream route does not support image input yet")
			}
			content := converseTextContent(msg.Content)
			if len(content) == 0 {
				lastWasToolResults = false
				continue
			}
			messages = append(messages, types.Message{
				Role:    types.ConversationRoleUser,
				Content: content,
			})
			lastWasToolResults = false
		case "assistant":
			content, err := converseAssistantContent(msg)
			if err != nil {
				return nil, err
			}
			if len(content) == 0 {
				lastWasToolResults = false
				continue
			}
			messages = append(messages, types.Message{
				Role:    types.ConversationRoleAssistant,
				Content: content,
			})
			lastWasToolResults = false
		case "tool":
			block, err := converseToolResultBlock(msg)
			if err != nil {
				return nil, err
			}
			if lastWasToolResults && len(messages) > 0 {
				messages[len(messages)-1].Content = append(messages[len(messages)-1].Content, block)
			} else {
				messages = append(messages, types.Message{
					Role:    types.ConversationRoleUser,
					Content: []types.ContentBlock{block},
				})
			}
			lastWasToolResults = true
		default:
			lastWasToolResults = false
		}
	}

	return messages, nil
}

func converseTextContent(content string) []types.ContentBlock {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	return []types.ContentBlock{
		&types.ContentBlockMemberText{Value: content},
	}
}

func converseAssistantContent(msg api.Message) ([]types.ContentBlock, error) {
	content := converseTextContent(msg.Content)
	for i, toolCall := range msg.ToolCalls {
		if strings.TrimSpace(toolCall.ID) == "" {
			return nil, fmt.Errorf("tool call at index %d has no id", i)
		}
		if strings.TrimSpace(toolCall.Function.Name) == "" {
			return nil, fmt.Errorf("tool call %q has no function name", toolCall.ID)
		}
		args, err := parseConverseToolArguments(toolCall.Function.Arguments)
		if err != nil {
			return nil, fmt.Errorf("tool call %q arguments: %w", toolCall.ID, err)
		}
		content = append(content, &types.ContentBlockMemberToolUse{
			Value: types.ToolUseBlock{
				ToolUseId: aws.String(toolCall.ID),
				Name:      aws.String(toolCall.Function.Name),
				Input:     bedrockdocument.NewLazyDocument(args),
			},
		})
	}
	return content, nil
}

func converseToolResultBlock(msg api.Message) (types.ContentBlock, error) {
	if strings.TrimSpace(msg.ToolCallID) == "" {
		return nil, fmt.Errorf("tool result has no tool_call_id")
	}
	return &types.ContentBlockMemberToolResult{
		Value: types.ToolResultBlock{
			ToolUseId: aws.String(msg.ToolCallID),
			Content: []types.ToolResultContentBlock{
				&types.ToolResultContentBlockMemberText{Value: msg.Content},
			},
		},
	}, nil
}

func parseConverseToolArguments(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, err
	}
	if args == nil {
		return map[string]any{}, nil
	}
	return args, nil
}

func int32Ptr(value int) *int32 {
	if value <= 0 {
		return nil
	}
	v := int32(value)
	return &v
}
