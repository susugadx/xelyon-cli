package gemini

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// request builder 層: Gemini の HTTP 送信や SSE 解釈を持たず、
// GenerateContent 向け request payload 構築だけを担当する。

func buildGeminiTextRequest(ctx context.Context, systemPrompt string, messages []api.Message, model, cacheName string, cfg *config.Config) GeminiRequest {
	reqBody := GeminiRequest{
		CachedContent:     cacheName,
		SystemInstruction: geminiSystemInstructionIfUncached(systemPrompt, cacheName),
		Contents:          geminiTextContentsFromMessages(geminiMessagesWithActiveContext(ctx, messages)),
		ServiceTier:       config.GeminiRequestServiceTier(cfg),
	}
	reqBody.GenerationConfig = getThinkingConfigForModel(ctx, model, cfg)
	return reqBody
}

func buildGeminiMultimodalRequest(
	ctx context.Context,
	systemPrompt string,
	history []api.Message,
	userMessage string,
	image *api.ImageData,
	model string,
	mcpTools []api.ToolDefinition,
	functionCallingEnabled bool,
	cfg *config.Config,
) GeminiMultimodalRequest {
	reqBody := GeminiMultimodalRequest{
		SystemInstruction: newGeminiSystemInstruction(systemPrompt),
		Contents:          geminiMultimodalContentsFromMessages(ctx, history, userMessage, image),
		ServiceTier:       config.GeminiRequestServiceTier(cfg),
	}

	if api.ShouldSendToolPayload(ctx, functionCallingEnabled) {
		reqBody.Tools = GetCombinedToolDefinitionsWithContext(ctx, mcpTools)
		reqBody.ToolConfig = newGeminiToolConfig(geminiFunctionCallingMode(ctx))
	}

	reqBody.GenerationConfig = getThinkingConfigForModel(ctx, model, cfg)
	return reqBody
}

func buildGeminiFunctionCallingRequest(
	ctx context.Context,
	systemPrompt string,
	messages []api.Message,
	model string,
	cacheName string,
	toolDefs []api.GeminiToolConfig,
	toolCfg *GeminiToolConfigWrapper,
	cfg *config.Config,
) GeminiRequestWithTools {
	reqBody := GeminiRequestWithTools{
		Contents:    geminiFunctionCallingContentsFromMessages(geminiMessagesWithActiveContext(ctx, messages)),
		ServiceTier: config.GeminiRequestServiceTier(cfg),
	}
	if cacheName != "" {
		reqBody.CachedContent = cacheName
	} else {
		reqBody.SystemInstruction = newGeminiSystemInstruction(systemPrompt)
		reqBody.Tools = toolDefs
		reqBody.ToolConfig = toolCfg
	}
	reqBody.GenerationConfig = getThinkingConfigForModel(ctx, model, cfg)
	return reqBody
}

func newGeminiSystemInstruction(systemPrompt string) *GeminiSystemInstruction {
	if systemPrompt == "" {
		return nil
	}
	return &GeminiSystemInstruction{
		Parts: []GeminiPart{{Text: systemPrompt}},
	}
}

func geminiSystemInstructionIfUncached(systemPrompt, cacheName string) *GeminiSystemInstruction {
	if cacheName != "" {
		return nil
	}
	return newGeminiSystemInstruction(systemPrompt)
}

func geminiMessagesWithActiveContext(ctx context.Context, messages []api.Message) []api.Message {
	message, ok := geminiActiveContextMessage(ctx)
	if !ok {
		return messages
	}
	insertAt := len(messages)
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			insertAt = i
			break
		}
	}
	withActiveContext := make([]api.Message, 0, len(messages)+1)
	withActiveContext = append(withActiveContext, messages[:insertAt]...)
	withActiveContext = append(withActiveContext, message)
	withActiveContext = append(withActiveContext, messages[insertAt:]...)
	return withActiveContext
}

func geminiActiveContextMessage(ctx context.Context) (api.Message, bool) {
	content := api.RenderActiveContextBlocksFromContext(ctx)
	if content == "" {
		return api.Message{}, false
	}
	return api.Message{Role: "user", Content: content}, true
}

func geminiTextContentsFromMessages(messages []api.Message) []GeminiContent {
	var contents []GeminiContent
	for _, msg := range messages {
		contents = append(contents, geminiContentFromMessage(msg))
	}
	return contents
}

func geminiMultimodalContentsFromMessages(ctx context.Context, history []api.Message, userMessage string, image *api.ImageData) []interface{} {
	contents := make([]interface{}, 0, len(history)+2)
	for _, msg := range history {
		contents = append(contents, geminiContentFromMessage(msg))
	}
	if message, ok := geminiActiveContextMessage(ctx); ok {
		contents = append(contents, geminiContentFromMessage(message))
	}
	contents = append(contents, GeminiMultimodalContent{
		Role: "user",
		Parts: []GeminiMultimodalPart{
			{
				InlineData: &GeminiInlineData{
					MimeType: image.MediaType,
					Data:     image.Base64,
				},
			},
			{
				Text: userMessage,
			},
		},
	})
	return contents
}

func geminiContentFromMessage(msg api.Message) GeminiContent {
	return GeminiContent{
		Parts: []GeminiPart{{Text: msg.Content}},
		Role:  geminiContentRole(msg.Role),
	}
}

func geminiFunctionCallingContentsFromMessages(messages []api.Message) []interface{} {
	var contents []interface{}
	for _, msg := range messages {
		switch {
		case msg.Role == "assistant" && len(msg.ToolCalls) > 0:
			contents = append(contents, GeminiGenericContent{
				Parts: geminiFunctionCallPartsFromMessage(msg),
				Role:  "model",
			})
		case msg.Role == "tool" && msg.ToolCallID != "":
			contents = append(contents, geminiFunctionResponseContentFromMessage(msg))
		default:
			contents = append(contents, geminiContentFromMessage(msg))
		}
	}
	return contents
}

func geminiFunctionCallPartsFromMessage(msg api.Message) []interface{} {
	parts := make([]interface{}, 0, len(msg.ToolCalls)+1)
	if msg.Content != "" {
		parts = append(parts, GeminiPart{Text: msg.Content})
	}
	if len(msg.ToolCalls) > 0 {
		parts = append(parts, geminiThoughtPartsFromToolCall(msg.ToolCalls[0])...)
	}
	for _, tc := range msg.ToolCalls {
		var args map[string]any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		parts = append(parts, GeminiFunctionCallPart{
			FunctionCall: GeminiFunctionCallData{
				Name: tc.Function.Name,
				Args: args,
			},
			ThoughtSignature: tc.ThoughtSignature,
		})
	}
	return parts
}

func geminiThoughtPartsFromToolCall(toolCall api.OpenAIToolCall) []interface{} {
	parts := make([]interface{}, 0, len(toolCall.ThoughtParts))
	for _, tp := range toolCall.ThoughtParts {
		geminiPart := make(map[string]any)
		if text, ok := tp["text"].(string); ok && text != "" {
			geminiPart["text"] = text
		}
		if thought, ok := tp["thought"].(bool); ok && thought {
			geminiPart["thought"] = true
		}
		if sig, ok := tp["thought_signature"].(string); ok && sig != "" {
			geminiPart["thoughtSignature"] = sig
		}
		if len(geminiPart) > 0 {
			parts = append(parts, geminiPart)
		}
	}
	return parts
}

func geminiFunctionResponseContentFromMessage(msg api.Message) GeminiGenericContent {
	toolName := msg.ToolName
	if toolName == "" {
		toolName = extractToolNameFromContent(msg.Content)
	}
	return GeminiGenericContent{
		Parts: []interface{}{
			GeminiFunctionResponsePart{
				FunctionResponse: GeminiFunctionResponseData{
					Name: toolName,
					Response: map[string]any{
						"result": msg.Content,
					},
				},
			},
		},
		Role: "user",
	}
}

func geminiContentRole(role string) string {
	if role == "assistant" {
		return "model"
	}
	return "user"
}

type geminiFunctionCallingModeContextKey struct{}

func withGeminiFunctionCallingMode(ctx context.Context, mode string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return ctx
	}
	return context.WithValue(ctx, geminiFunctionCallingModeContextKey{}, mode)
}

func geminiFunctionCallingMode(ctx context.Context) string {
	if ctx != nil {
		if mode, _ := ctx.Value(geminiFunctionCallingModeContextKey{}).(string); strings.TrimSpace(mode) != "" {
			return strings.TrimSpace(mode)
		}
	}
	if mode := os.Getenv("GEMINI_FC_MODE"); mode != "" {
		return mode
	}
	return "AUTO"
}

func newGeminiToolConfig(mode string) *GeminiToolConfigWrapper {
	return &GeminiToolConfigWrapper{
		FunctionCallingConfig: GeminiFunctionCallingConfig{Mode: mode},
	}
}
