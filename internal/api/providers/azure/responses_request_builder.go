package azure

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type modelIdentity = openairesponses.ModelIdentity
type responsesRequest = openairesponses.Request

func newModelIdentity(requestModel, catalogModel string) modelIdentity {
	return openairesponses.NewModelIdentity(requestModel, catalogModel)
}

func (p *Provider) buildChatResponsesRequest(ctx context.Context, systemPrompt string, history []api.Message, model string) responsesRequest {
	modelIdentity := azureModelIdentity(ctx, model)
	return openairesponses.BuildChatRequest(openairesponses.ChatRequestOptions{
		Base:               p.newBaseResponsesRequestOptions(ctx, modelIdentity),
		SystemPrompt:       systemPrompt,
		History:            history,
		PreviousResponseID: p.GetResponseID(),
	})
}

func (p *Provider) buildImageResponsesRequest(
	ctx context.Context,
	systemPrompt string,
	history []api.Message,
	userMessage string,
	image *api.ImageData,
	model string,
) responsesRequest {
	modelIdentity := azureModelIdentity(ctx, model)
	return openairesponses.BuildImageRequest(openairesponses.ImageRequestOptions{
		Base:         p.newBaseResponsesRequestOptions(ctx, modelIdentity),
		SystemPrompt: systemPrompt,
		History:      history,
		UserMessage:  userMessage,
		Image:        image,
	})
}

func (p *Provider) newBaseResponsesRequestOptions(ctx context.Context, model modelIdentity) openairesponses.BaseRequestOptions {
	options := openairesponses.BaseRequestOptions{
		Model:           model,
		MaxOutputTokens: api.GetMaxOutputTokens(ctx, "azure", model.RequestName()),
		Stream:          openai.ShouldStreamResponses(model.CatalogName()),
		Store:           true,
	}
	if p.IsFunctionCallingEnabled() {
		options.Tools = openai.GetResponsesToolDefinitionsWithContext(ctx, p.mcpTools)
		options.ToolChoice = openairesponses.BuildFunctionToolChoice(p.toolChoice)
	}

	options.Reasoning = azureResponsesReasoningConfig(ctx, model)
	return options
}

func azureResponsesReasoningConfig(ctx context.Context, model modelIdentity) *openairesponses.ReasoningConfig {
	cfg := config.FromContext(ctx)
	if api.IsThinkingEnabled(ctx) {
		return &openairesponses.ReasoningConfig{
			Effort: openai.LevelToReasoningEffort(cfg.Thinking.Level),
		}
	}

	if isCodexModel(model.CatalogName()) {
		return &openairesponses.ReasoningConfig{Effort: "low"}
	}
	return nil
}

func isCodexModel(model string) bool {
	return strings.Contains(strings.ToLower(model), "codex")
}
