package openai

import (
	"context"
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func resolveResponsesAPIURL() string {
	apiURL := os.Getenv("OPENAI_RESPONSES_URL")
	if apiURL == "" {
		apiURL = defaultOpenAIResponsesURL
	}
	return apiURL
}

func (p *Provider) buildChatResponsesRequest(ctx context.Context, systemPrompt string, history []api.Message, model string) ResponsesRequest {
	reqBody := p.newBaseResponsesRequest(ctx, systemPrompt, model)
	developerMsg := buildResponsesDeveloperMessage(systemPrompt)

	if p.lastResponseID != "" && len(history) > 0 {
		lastMsg := history[len(history)-1]
		if lastMsg.Role == "tool" {
			reqBody.PreviousResponseID = p.lastResponseID
			reqBody.Input = buildResponsesTrailingToolOutputs(history)
			return reqBody
		}

		reqBody.PreviousResponseID = p.lastResponseID
		reqBody.Input = []InputItem{{
			Type:    "message",
			Role:    lastMsg.Role,
			Content: lastMsg.Content,
		}}
		return reqBody
	}

	historyInput := convertHistoryToResponsesInput(history)
	reqBody.Input = append([]InputItem{developerMsg}, historyInput...)
	return reqBody
}

func (p *Provider) buildImageResponsesRequest(
	ctx context.Context,
	systemPrompt string,
	history []api.Message,
	userMessage string,
	image *api.ImageData,
	model string,
) ResponsesRequest {
	reqBody := p.newBaseResponsesRequest(ctx, systemPrompt, model)

	developerMsg := buildResponsesDeveloperMessage(systemPrompt)
	input := append([]InputItem{developerMsg}, convertHistoryToResponsesInput(history)...)
	dataURL := fmt.Sprintf("data:%s;base64,%s", image.MediaType, image.Base64)

	imageMessage := InputItem{
		Type: "message",
		Role: "user",
		Content: []InputContentPart{
			{
				Type:     "input_image",
				ImageURL: dataURL,
			},
			{
				Type: "input_text",
				Text: userMessage,
			},
		},
	}
	input = append(input, imageMessage)
	reqBody.Input = input
	return reqBody
}

func (p *Provider) newBaseResponsesRequest(ctx context.Context, systemPrompt, model string) ResponsesRequest {
	reqBody := ResponsesRequest{
		Model:                model,
		MaxOutputTokens:      api.GetMaxOutputTokens(ctx, "openai", model),
		Stream:               true,
		Store:                true,
		Tools:                GetResponsesToolDefinitionsWithContext(ctx, p.mcpTools),
		PromptCacheKey:       BuildPromptCacheKey(model, systemPrompt),
		PromptCacheRetention: "24h",
	}

	applyResponsesToolChoice(&reqBody, p.toolChoice)
	applyResponsesReasoning(ctx, model, &reqBody)
	return reqBody
}

func buildResponsesDeveloperMessage(systemPrompt string) InputItem {
	return InputItem{
		Type:    "message",
		Role:    "developer",
		Content: systemPrompt,
	}
}

func buildResponsesTrailingToolOutputs(history []api.Message) []InputItem {
	toolStart := len(history) - 1
	for toolStart >= 0 && history[toolStart].Role == "tool" {
		toolStart--
	}

	toolMessages := history[toolStart+1:]
	toolOutputs := make([]InputItem, 0, len(toolMessages))
	for _, msg := range toolMessages {
		toolOutputs = append(toolOutputs, InputItem{
			Type:   "function_call_output",
			CallID: msg.ToolCallID,
			Output: msg.Content,
		})
	}
	return toolOutputs
}

func applyResponsesToolChoice(reqBody *ResponsesRequest, toolChoice *string) {
	if toolChoice == nil {
		return
	}

	reqBody.ToolChoice = map[string]interface{}{
		"type": "function",
		"function": map[string]string{
			"name": *toolChoice,
		},
	}
}

func applyResponsesReasoning(ctx context.Context, model string, reqBody *ResponsesRequest) {
	cfg := config.FromContext(ctx)
	if api.IsThinkingEnabled(ctx) {
		reqBody.Reasoning = &ReasoningConfig{
			Effort: LevelToReasoningEffort(cfg.Thinking.Level),
		}
		return
	}

	if isCodexModel(model) {
		// Codexモデルは reasoning 必須のため "low" にフォールバック
		reqBody.Reasoning = &ReasoningConfig{Effort: "low"}
	}
}
