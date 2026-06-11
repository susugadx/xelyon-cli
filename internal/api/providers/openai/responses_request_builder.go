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
	return buildChatResponsesRequestWithProfile(ctx, openAIResponsesRuntimeProfile(), responsesRequestBuilderInputs{
		SystemPrompt:    systemPrompt,
		History:         history,
		Model:           model,
		ResponseID:      p.lastResponseID,
		MCPTools:        p.mcpTools,
		ToolChoice:      p.toolChoice,
		FunctionCalling: p.IsFunctionCallingEnabled(),
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
	return buildImageResponsesRequestWithProfile(ctx, openAIResponsesRuntimeProfile(), responsesImageRequestBuilderInputs{
		SystemPrompt:    systemPrompt,
		History:         history,
		UserMessage:     userMessage,
		Image:           image,
		Model:           model,
		MCPTools:        p.mcpTools,
		ToolChoice:      p.toolChoice,
		FunctionCalling: p.IsFunctionCallingEnabled(),
	})
}

func newOpenAIResponsesModelIdentity(ctx context.Context, model string) openairesponses.ModelIdentity {
	return openAIResponsesRuntimeProfile().modelIdentity(ctx, model)
}

type responsesRequestBuilderInputs struct {
	SystemPrompt    string
	History         []api.Message
	Model           string
	ResponseID      string
	MCPTools        []api.ToolDefinition
	ToolChoice      *string
	FunctionCalling bool
}

type responsesImageRequestBuilderInputs struct {
	SystemPrompt    string
	History         []api.Message
	UserMessage     string
	Image           *api.ImageData
	Model           string
	MCPTools        []api.ToolDefinition
	ToolChoice      *string
	FunctionCalling bool
}

func buildChatResponsesRequestWithProfile(ctx context.Context, profile responsesRuntimeProfile, inputs responsesRequestBuilderInputs) ResponsesRequest {
	modelIdentity := profile.modelIdentity(ctx, inputs.Model)
	activeContext := openairesponses.ActiveContextFromContext(ctx)
	previousResponseID := profile.previousResponseID(ctx, inputs.ResponseID, activeContext)
	serverCompactionDecision := profile.serverCompactionDecision(ctx, modelIdentity, previousResponseID)
	return openairesponses.BuildChatRequest(openairesponses.ChatRequestOptions{
		Base:               newBaseResponsesRequestOptions(ctx, profile, inputs.SystemPrompt, modelIdentity, serverCompactionDecision, inputs.MCPTools, inputs.ToolChoice, inputs.FunctionCalling),
		RequestContext:     ctx,
		SystemPrompt:       inputs.SystemPrompt,
		CompactedInput:     api.CompactedInputItemsFromContext(ctx),
		ActiveContext:      activeContext,
		History:            inputs.History,
		PreviousResponseID: previousResponseID,
	})
}

func buildImageResponsesRequestWithProfile(ctx context.Context, profile responsesRuntimeProfile, inputs responsesImageRequestBuilderInputs) ResponsesRequest {
	modelIdentity := profile.modelIdentity(ctx, inputs.Model)
	return openairesponses.BuildImageRequest(openairesponses.ImageRequestOptions{
		Base:           newBaseResponsesRequestOptions(ctx, profile, inputs.SystemPrompt, modelIdentity, openairesponses.ServerCompactionDecision{}, inputs.MCPTools, inputs.ToolChoice, inputs.FunctionCalling),
		SystemPrompt:   inputs.SystemPrompt,
		CompactedInput: api.CompactedInputItemsFromContext(ctx),
		ActiveContext:  openairesponses.ActiveContextFromContext(ctx),
		History:        inputs.History,
		UserMessage:    inputs.UserMessage,
		Image:          inputs.Image,
	})
}

func newBaseResponsesRequestOptions(
	ctx context.Context,
	profile responsesRuntimeProfile,
	systemPrompt string,
	model openairesponses.ModelIdentity,
	serverCompactionDecision openairesponses.ServerCompactionDecision,
	mcpTools []api.ToolDefinition,
	toolChoice *string,
	functionCallingEnabled bool,
) openairesponses.BaseRequestOptions {
	options := openairesponses.BaseRequestOptions{
		Model:                                 model,
		MaxOutputTokens:                       profile.maxOutputTokens(ctx, model),
		Stream:                                profile.stream(model),
		Store:                                 profile.store(ctx),
		PromptCacheKey:                        BuildPromptCacheKey(model.RequestName(), systemPrompt),
		PromptCacheRetention:                  profile.promptCacheRetention(),
		ContextManagement:                     serverCompactionDecision.ContextManagement,
		SkipLocalAutoCompressionAfterResponse: serverCompactionDecision.ShouldSkipLocalAutoCompression,
	}
	if profile.IncludeInstructions {
		options.Instructions = systemPrompt
	}
	if api.ShouldSendToolPayload(ctx, functionCallingEnabled) {
		options.Tools = openairesponses.BuildToolDefinitionsWithContext(ctx, mcpTools)
		options.ToolChoice = openairesponses.BuildFunctionToolChoice(toolChoice)
	}

	options.Reasoning = responsesReasoningConfig(ctx, model)
	return options
}

func previousResponseIDForRequest(ctx context.Context, responseID string) string {
	if !openairesponses.ResponseIDChainReusable(ctx) {
		return ""
	}
	return responseID
}

func responsesReasoningConfig(ctx context.Context, model openairesponses.ModelIdentity) *ReasoningConfig {
	cfg := config.FromContext(ctx)
	if api.IsThinkingEnabled(ctx) {
		return &ReasoningConfig{
			Effort: openairesponses.ReasoningEffortFromThinkingLevel(cfg.Thinking.Level),
		}
	}

	if isCodexModel(model.CatalogName()) {
		// Codexモデルは reasoning 必須のため "low" にフォールバック
		return &ReasoningConfig{Effort: "low"}
	}
	return nil
}
