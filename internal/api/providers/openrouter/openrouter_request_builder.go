package openrouter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/claude"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// request builder 層: provider の I/O や streaming 分岐を持たず、
// OpenRouter 向け request payload 構築だけを担当する。

type openRouterClaudeToolChoice struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func (p *Provider) buildOpenAITextChatPayload(ctx context.Context, systemPrompt string, history []api.Message, model string) ([]byte, error) {
	options := openaicompat.ChatCompletionsRequestOptions{
		Model:            model,
		SystemPrompt:     systemPrompt,
		ActiveContext:    api.ActiveContextBlocksFromContext(ctx),
		History:          history,
		MaxTokens:        api.GetMaxOutputTokens(ctx, "openrouter", model),
		Stream:           true,
		IncludeUsage:     true,
		ImagePayloadMode: openaicompat.ImagePayloadMultimodal,
	}

	if api.ShouldSendToolPayload(ctx, p.IsFunctionCallingEnabled()) {
		options.FunctionCalling = &openaicompat.FunctionCallingOptions{
			Tools:    openai.GetCombinedOpenAIToolsWithContext(ctx, p.mcpTools),
			ToolName: p.toolChoice,
		}
	}

	reqBody := openaicompat.BuildChatCompletionsRequest(options)
	return json.Marshal(reqBody)
}

func (p *Provider) buildOpenAIImageChatPayload(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) ([]byte, error) {
	imageURL := fmt.Sprintf("data:%s;base64,%s", image.MediaType, image.Base64)

	content := []ContentPart{
		{Type: "text", Text: userMessage},
		{Type: "image_url", ImageURL: &ImageURL{URL: imageURL}},
	}

	messages := openaicompat.BuildChatMessageInterfacesWithActiveContextAndImagePayloadMode(
		systemPrompt,
		api.ActiveContextBlocksFromContext(ctx),
		history,
		nil,
		openaicompat.ImagePayloadMultimodal,
	)
	messages = append(messages, MultimodalMessage{
		Role:    "user",
		Content: content,
	})

	reqBody := struct {
		Model         string             `json:"model"`
		Messages      []interface{}      `json:"messages"`
		MaxTokens     int                `json:"max_tokens"`
		Stream        bool               `json:"stream"`
		StreamOptions *api.StreamOptions `json:"stream_options,omitempty"`
	}{
		Model:         model,
		Messages:      messages,
		MaxTokens:     api.GetMaxOutputTokens(ctx, "openrouter", model),
		Stream:        true,
		StreamOptions: &api.StreamOptions{IncludeUsage: true},
	}

	return json.Marshal(reqBody)
}

func (p *Provider) buildClaudeChatPayload(ctx context.Context, systemPrompt string, history []api.Message, userMessage, model string, image *api.ImageData) ([]byte, error) {
	anthropicMessages := claude.ConvertToAnthropicMessages(history)
	cfg := config.ResolveContext(ctx, p.effectiveConfig())

	var messages []interface{}
	for _, msg := range anthropicMessages {
		messages = append(messages, msg)
	}
	if image != nil && image.Base64 != "" {
		messages = append(messages, claude.MultimodalMessage{
			Role: "user",
			Content: []claude.ContentPart{
				{
					Type: "image",
					Source: &claude.ImageSource{
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
		})
	}

	reqBody := struct {
		Model             string                      `json:"model"`
		AnthropicVersion  string                      `json:"anthropic_version"`
		AnthropicBeta     []string                    `json:"anthropic_beta,omitempty"`
		CacheControl      *api.CacheControl           `json:"cache_control,omitempty"`
		MaxTokens         int                         `json:"max_tokens"`
		System            interface{}                 `json:"system,omitempty"`
		Messages          []interface{}               `json:"messages"`
		Tools             []claude.ClaudeTool         `json:"tools,omitempty"`
		ToolChoice        *openRouterClaudeToolChoice `json:"tool_choice,omitempty"`
		Stream            bool                        `json:"stream"`
		ContextManagement *claude.ContextManagement   `json:"context_management,omitempty"`
	}{
		Model:            model,
		AnthropicVersion: "2023-06-01",
		MaxTokens:        api.GetMaxOutputTokens(ctx, "openrouter", model),
		System:           api.BuildSystemFieldWithConfig(api.SystemPromptWithActiveContextFromContext(ctx, systemPrompt), cfg),
		Messages:         messages,
		Stream:           true,
	}

	if cfg != nil && cfg.PromptCache.Enabled {
		reqBody.CacheControl = api.NewCacheControlWithConfig(cfg)
	}
	if api.ShouldSendToolPayload(ctx, p.IsFunctionCallingEnabled()) {
		reqBody.Tools = claude.GetCombinedClaudeToolsWithContext(ctx, p.mcpTools)
		reqBody.ToolChoice = buildOpenRouterClaudeToolChoice(p.toolChoice)
	}
	reqBody.ContextManagement, reqBody.AnthropicBeta = buildOpenRouterClaudeContextManagement(model, cfg.Compression, reqBody.AnthropicBeta)

	return json.Marshal(reqBody)
}

func buildOpenRouterClaudeToolChoice(toolChoice *string) *openRouterClaudeToolChoice {
	if toolChoice == nil || *toolChoice == "" {
		return nil
	}
	return &openRouterClaudeToolChoice{
		Type: "tool",
		Name: *toolChoice,
	}
}
