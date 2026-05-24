package claude

import (
	"context"
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type claudeMessagesRequestBuild struct {
	Model   string
	Request Request
}

type claudeMultimodalRequestBuild struct {
	Model   string
	Request MultimodalRequest
}

type claudeRequestFeatures struct {
	System            interface{}
	CacheControl      *api.CacheControl
	MaxTokens         int
	Thinking          *ThinkingConfig
	OutputConfig      *OutputConfig
	Tools             []ClaudeTool
	ToolChoice        *ClaudeToolChoice
	ContextManagement *ContextManagement
}

func (p *Provider) buildMessagesRequest(ctx context.Context, systemPrompt string, history []api.Message, model string) claudeMessagesRequestBuild {
	model = api.GetDefaultModelWithContext(ctx, model, "claude", defaultClaudeModel)
	messages := ConvertToAnthropicMessagesWithThinking(history, api.IsThinkingEnabled(ctx))
	logClaudeRequestConversionDebug(ctx, history, messages)

	cfg := config.ResolveContext(ctx, p.effectiveConfig())
	catalogModel := cfg.ModelCatalogName(p.configLookupKey(), model)
	features := p.buildRequestFeatures(ctx, cfg, systemPrompt, model, catalogModel)

	return claudeMessagesRequestBuild{
		Model: model,
		Request: Request{
			Model:             model,
			Messages:          messages,
			System:            features.System,
			CacheControl:      features.CacheControl,
			MaxTokens:         features.MaxTokens,
			Stream:            true,
			Thinking:          features.Thinking,
			OutputConfig:      features.OutputConfig,
			Tools:             features.Tools,
			ToolChoice:        features.ToolChoice,
			ContextManagement: features.ContextManagement,
		},
	}
}

func (p *Provider) buildMultimodalRequest(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) claudeMultimodalRequestBuild {
	model = api.GetDefaultModelWithContext(ctx, model, "claude", defaultClaudeModel)
	converted := ConvertToAnthropicMessagesWithThinking(history, api.IsThinkingEnabled(ctx))

	cfg := config.ResolveContext(ctx, p.effectiveConfig())
	catalogModel := cfg.ModelCatalogName(p.configLookupKey(), model)
	features := p.buildRequestFeatures(ctx, cfg, systemPrompt, model, catalogModel)

	messages := make([]interface{}, 0, len(converted)+1)
	for _, msg := range converted {
		messages = append(messages, msg)
	}
	messages = append(messages, claudeImageMessage(userMessage, image))

	return claudeMultimodalRequestBuild{
		Model: model,
		Request: MultimodalRequest{
			Model:             model,
			Messages:          messages,
			System:            features.System,
			CacheControl:      features.CacheControl,
			MaxTokens:         features.MaxTokens,
			Stream:            true,
			Thinking:          features.Thinking,
			OutputConfig:      features.OutputConfig,
			Tools:             features.Tools,
			ToolChoice:        features.ToolChoice,
			ContextManagement: features.ContextManagement,
		},
	}
}

func (p *Provider) buildRequestFeatures(ctx context.Context, cfg *config.Config, systemPrompt, model, catalogModel string) claudeRequestFeatures {
	cfg = config.ResolveContext(ctx, cfg)
	systemPrompt = api.SystemPromptWithActiveContextFromContext(ctx, systemPrompt)
	features := claudeRequestFeatures{
		System:            api.BuildSystemFieldWithConfig(systemPrompt, cfg),
		MaxTokens:         p.maxOutputTokens(ctx, model),
		ContextManagement: buildContextManagementForModel(catalogModel, cfg.Compression),
	}
	if cfg.PromptCache.Enabled {
		features.CacheControl = api.NewCacheControlWithConfig(cfg)
	}
	if api.IsThinkingEnabled(ctx) {
		if IsAdaptiveThinkingModel(catalogModel) {
			features.Thinking = &ThinkingConfig{Type: "adaptive"}
			features.OutputConfig = &OutputConfig{Effort: LevelToEffort(cfg.Thinking.Level, catalogModel)}
		} else {
			features.Thinking = &ThinkingConfig{
				Type:         "enabled",
				BudgetTokens: LevelToBudgetTokens(cfg.Thinking.Level),
			}
		}
	}
	if api.ShouldSendToolPayload(ctx, p.IsFunctionCallingEnabled()) {
		features.Tools = GetCombinedClaudeToolsWithContext(ctx, p.mcpTools)
		features.ToolChoice = buildClaudeToolChoice(p.toolChoice)
	}
	return features
}

func buildClaudeToolChoice(toolChoice *string) *ClaudeToolChoice {
	if toolChoice == nil || *toolChoice == "" {
		return nil
	}
	return &ClaudeToolChoice{
		Type: "tool",
		Name: *toolChoice,
	}
}

func claudeImageMessage(userMessage string, image *api.ImageData) MultimodalMessage {
	return MultimodalMessage{
		Role: "user",
		Content: []ContentPart{
			{
				Type: "image",
				Source: &ImageSource{
					Type:      "base64",
					MediaType: image.MediaType,
					Data:      image.Base64,
				},
			},
			{
				Type: "text",
				Text: userMessage,
			},
		},
	}
}

func logClaudeRequestConversionDebug(ctx context.Context, history []api.Message, messages []AnthropicMessage) {
	if os.Getenv("XELYON_DEBUG_CLAUDE") != "1" {
		return
	}
	errOut := api.ErrorWriterFromContext(ctx)
	fmt.Fprintf(errOut, "[DEBUG Claude] === History (%d messages) ===\n", len(history))
	for i, m := range history {
		tcIDs := make([]string, len(m.ToolCalls))
		for j, tc := range m.ToolCalls {
			tcIDs[j] = tc.ID
		}
		if len(tcIDs) > 0 {
			fmt.Fprintf(errOut, "[DEBUG Claude] history[%d] role=%s tool_calls=%v\n", i, m.Role, tcIDs)
		} else if m.ToolCallID != "" {
			fmt.Fprintf(errOut, "[DEBUG Claude] history[%d] role=%s tool_call_id=%s\n", i, m.Role, m.ToolCallID)
		} else {
			fmt.Fprintf(errOut, "[DEBUG Claude] history[%d] role=%s content_len=%d\n", i, m.Role, len(m.Content))
		}
	}
	fmt.Fprintf(errOut, "[DEBUG Claude] === Converted (%d messages) ===\n", len(messages))
	for i, m := range messages {
		var types []string
		for _, b := range m.Content {
			switch b.Type {
			case "tool_use":
				types = append(types, "tool_use:"+b.ID)
			case "tool_result":
				types = append(types, "tool_result:"+b.ToolUseID)
			default:
				types = append(types, b.Type)
			}
		}
		fmt.Fprintf(errOut, "[DEBUG Claude] messages[%d] role=%s content=%v\n", i, m.Role, types)
	}
	validateAnthropicToolPairs(messages, errOut)
}
