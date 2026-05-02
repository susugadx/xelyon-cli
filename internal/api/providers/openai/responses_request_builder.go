package openai

import (
	"context"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
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
	modelIdentity := newOpenAIResponsesModelIdentity(ctx, model)
	previousResponseID := previousResponseIDForRequest(ctx, p.lastResponseID)
	serverCompactionDecision := openairesponses.ResolveServerCompactionDecision(ctx, "openai", modelIdentity, previousResponseID)
	return openairesponses.BuildChatRequest(openairesponses.ChatRequestOptions{
		Base:               p.newBaseResponsesRequestOptions(ctx, systemPrompt, modelIdentity, serverCompactionDecision),
		SystemPrompt:       systemPrompt,
		CompactedInput:     api.CompactedInputItemsFromContext(ctx),
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
) ResponsesRequest {
	modelIdentity := newOpenAIResponsesModelIdentity(ctx, model)
	return openairesponses.BuildImageRequest(openairesponses.ImageRequestOptions{
		Base:           p.newBaseResponsesRequestOptions(ctx, systemPrompt, modelIdentity, openairesponses.ServerCompactionDecision{}),
		SystemPrompt:   systemPrompt,
		CompactedInput: api.CompactedInputItemsFromContext(ctx),
		History:        history,
		UserMessage:    userMessage,
		Image:          image,
	})
}

func newOpenAIResponsesModelIdentity(ctx context.Context, model string) openairesponses.ModelIdentity {
	cfg := config.FromContext(ctx)
	return openairesponses.NewModelIdentity(model, cfg.ModelCatalogName("openai", model))
}

func (p *Provider) newBaseResponsesRequestOptions(
	ctx context.Context,
	systemPrompt string,
	model openairesponses.ModelIdentity,
	serverCompactionDecision openairesponses.ServerCompactionDecision,
) openairesponses.BaseRequestOptions {
	cfg := config.FromContext(ctx)
	options := openairesponses.BaseRequestOptions{
		Model:                                 model,
		MaxOutputTokens:                       api.GetMaxOutputTokens(ctx, "openai", model.RequestName()),
		Stream:                                shouldStreamResponses(model.CatalogName()),
		Store:                                 cfg.ResponsesStoreEnabled(),
		Tools:                                 GetResponsesToolDefinitionsWithContext(ctx, p.mcpTools),
		ToolChoice:                            openairesponses.BuildFunctionToolChoice(p.toolChoice),
		PromptCacheKey:                        BuildPromptCacheKey(model.RequestName(), systemPrompt),
		PromptCacheRetention:                  "24h",
		ContextManagement:                     serverCompactionDecision.ContextManagement,
		SkipLocalAutoCompressionAfterResponse: serverCompactionDecision.ShouldSkipLocalAutoCompression,
	}

	options.Reasoning = responsesReasoningConfig(ctx, model)
	return options
}

func previousResponseIDForRequest(ctx context.Context, responseID string) string {
	if !config.FromContext(ctx).ResponsesStoreEnabled() {
		return ""
	}
	return responseID
}

func responsesReasoningConfig(ctx context.Context, model openairesponses.ModelIdentity) *ReasoningConfig {
	cfg := config.FromContext(ctx)
	if api.IsThinkingEnabled(ctx) {
		return &ReasoningConfig{
			Effort: LevelToReasoningEffort(cfg.Thinking.Level),
		}
	}

	if isCodexModel(model.CatalogName()) {
		// Codexモデルは reasoning 必須のため "low" にフォールバック
		return &ReasoningConfig{Effort: "low"}
	}
	return nil
}
