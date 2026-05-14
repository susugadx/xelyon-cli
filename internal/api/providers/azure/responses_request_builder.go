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
	activeContext := openairesponses.ActiveContextFromContext(ctx)
	previousResponseID := azurePreviousResponseIDForRequest(ctx, p.GetResponseID())
	previousResponseID = openairesponses.PreviousResponseIDForActiveContext(previousResponseID, activeContext)
	serverCompactionDecision := openairesponses.ResolveServerCompactionDecision(ctx, "azure", modelIdentity, previousResponseID)
	return openairesponses.BuildChatRequest(openairesponses.ChatRequestOptions{
		Base:               p.newBaseResponsesRequestOptions(ctx, modelIdentity, serverCompactionDecision),
		SystemPrompt:       systemPrompt,
		CompactedInput:     api.CompactedInputItemsFromContext(ctx),
		ActiveContext:      activeContext,
		History:            history,
		PreviousResponseID: previousResponseID,
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
		Base:           p.newBaseResponsesRequestOptions(ctx, modelIdentity, openairesponses.ServerCompactionDecision{}),
		SystemPrompt:   systemPrompt,
		CompactedInput: api.CompactedInputItemsFromContext(ctx),
		ActiveContext:  openairesponses.ActiveContextFromContext(ctx),
		History:        history,
		UserMessage:    userMessage,
		Image:          image,
	})
}

func (p *Provider) newBaseResponsesRequestOptions(
	ctx context.Context,
	model modelIdentity,
	serverCompactionDecision openairesponses.ServerCompactionDecision,
) openairesponses.BaseRequestOptions {
	cfg := config.FromContext(ctx)
	options := openairesponses.BaseRequestOptions{
		Model:                                 model,
		MaxOutputTokens:                       api.GetMaxOutputTokens(ctx, "azure", model.RequestName()),
		Stream:                                openai.ShouldStreamResponses(model.CatalogName()),
		Store:                                 cfg.ResponsesStoreEnabled(),
		ContextManagement:                     serverCompactionDecision.ContextManagement,
		SkipLocalAutoCompressionAfterResponse: serverCompactionDecision.ShouldSkipLocalAutoCompression,
	}
	if api.ShouldSendToolPayload(ctx, p.IsFunctionCallingEnabled()) {
		options.Tools = openai.GetResponsesToolDefinitionsWithContext(ctx, p.mcpTools)
		options.ToolChoice = openairesponses.BuildFunctionToolChoice(p.toolChoice)
	}

	options.Reasoning = azureResponsesReasoningConfig(ctx, model)
	return options
}

func azurePreviousResponseIDForRequest(ctx context.Context, responseID string) string {
	if !config.FromContext(ctx).ResponsesStoreEnabled() {
		return ""
	}
	return responseID
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
