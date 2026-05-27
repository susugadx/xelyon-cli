package ollama

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
)

const ollamaChatEndpointPath = "/api/chat"

type ollamaChatRequestBuild struct {
	Model   string
	URL     string
	Request OllamaRequest
}

func (p *Provider) buildChatRequest(ctx context.Context, systemPrompt string, history []api.Message, model string) ollamaChatRequestBuild {
	model = api.ResolveProviderRequestModel(ctx, model, "ollama")

	reqBody := OllamaRequest{
		Model:    model,
		Messages: buildOllamaChatMessages(systemPrompt, api.ActiveContextBlocksFromContext(ctx), history),
		Stream:   true,
		Options: &OllamaOptions{
			NumPredict: api.GetMaxOutputTokens(ctx, "ollama", model),
		},
	}

	if api.ShouldSendToolPayload(ctx, p.IsFunctionCallingEnabled()) {
		reqBody.Tools = openai.GetCombinedOpenAIToolsWithContext(ctx, p.mcpTools)
		reqBody.ToolChoice = "auto"
		if p.toolChoice != nil {
			reqBody.ToolChoice = *p.toolChoice
		}
	}

	return ollamaChatRequestBuild{
		Model:   model,
		URL:     p.chatEndpointURL(),
		Request: reqBody,
	}
}

func buildOllamaChatMessages(systemPrompt string, activeContext []api.ActiveContextBlock, history []api.Message) []api.Message {
	messages := []api.Message{
		{Role: "system", Content: systemPrompt},
	}
	if content := api.RenderActiveContextBlocks(activeContext); content != "" {
		messages = append(messages, api.Message{Role: "system", Content: content})
	}
	messages = append(messages, history...)
	return messages
}

func (p *Provider) chatEndpointURL() string {
	return p.baseURL + ollamaChatEndpointPath
}
