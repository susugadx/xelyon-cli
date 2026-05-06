package kimi

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
)

type kimiChatCompletionsBuild struct {
	Model          string
	Request        openaicompat.ChatCompletionsRequest
	ThinkingActive bool
	SpinnerSuffix  string
	PromptCacheKey string
}

func (p *Provider) buildChatCompletionsRequest(ctx context.Context, systemPrompt string, history []api.Message, model string) kimiChatCompletionsBuild {
	providerConfigKey := p.configLookupKey()
	requestedModel := api.GetDefaultModelWithContext(ctx, model, providerConfigKey, defaultKimiModel)
	extraFields, thinkingActive, spinnerSuffix := kimiThinkingConfig(ctx, providerConfigKey, requestedModel)
	promptCacheKey := buildKimiPromptCacheKey(ctx, requestedModel, systemPrompt)

	options := openaicompat.ChatCompletionsRequestOptions{
		Model:               requestedModel,
		Messages:            openaicompat.BuildChatMessages(systemPrompt, history),
		MaxCompletionTokens: api.GetMaxOutputTokens(ctx, providerConfigKey, requestedModel),
		Stream:              true,
		IncludeUsage:        true,
		PromptCacheKey:      promptCacheKey,
		ExtraFields:         extraFields,
	}

	if p.IsFunctionCallingEnabled() {
		toolChoicePolicy := openaicompat.AllowForcedToolChoicePolicy
		if thinkingActive {
			toolChoicePolicy = openaicompat.AutoToolChoicePolicy
		}
		options.FunctionCalling = &openaicompat.FunctionCallingOptions{
			Tools:            openai.GetCombinedOpenAIToolsWithContext(ctx, p.mcpTools),
			ToolName:         p.toolChoice,
			ToolChoicePolicy: toolChoicePolicy,
		}
	}

	return kimiChatCompletionsBuild{
		Model:          requestedModel,
		Request:        openaicompat.BuildChatCompletionsRequest(options),
		ThinkingActive: thinkingActive,
		SpinnerSuffix:  spinnerSuffix,
		PromptCacheKey: promptCacheKey,
	}
}
