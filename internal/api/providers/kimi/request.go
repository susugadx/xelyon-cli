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

// kimiChatCompletionsRequestOptions は Kimi 固有の request field を messages から分離して保持する。
type kimiChatCompletionsRequestOptions struct {
	model               string
	maxCompletionTokens int
	stream              bool
	includeUsage        bool
	promptCacheKey      string
	extraFields         map[string]any
	functionCalling     *openaicompat.FunctionCallingOptions
}

func (p *Provider) buildChatCompletionsRequest(ctx context.Context, systemPrompt string, history []api.Message, model string) kimiChatCompletionsBuild {
	options, thinkingActive, spinnerSuffix := p.buildChatCompletionsRequestOptions(ctx, systemPrompt, model)
	messages := openaicompat.BuildChatMessages(systemPrompt, history)

	return kimiChatCompletionsBuild{
		Model:          options.model,
		Request:        buildKimiTextChatCompletionsRequest(options, messages),
		ThinkingActive: thinkingActive,
		SpinnerSuffix:  spinnerSuffix,
		PromptCacheKey: options.promptCacheKey,
	}
}

func (p *Provider) buildChatCompletionsRequestOptions(ctx context.Context, systemPrompt string, model string) (kimiChatCompletionsRequestOptions, bool, string) {
	providerConfigKey := p.configLookupKey()
	requestedModel := api.GetDefaultModelWithContext(ctx, model, providerConfigKey, defaultKimiModel)
	extraFields, thinkingActive, spinnerSuffix := kimiThinkingConfig(ctx, providerConfigKey, requestedModel)
	promptCacheKey := buildKimiPromptCacheKey(ctx, requestedModel, systemPrompt)

	return kimiChatCompletionsRequestOptions{
		model:               requestedModel,
		maxCompletionTokens: api.GetMaxOutputTokens(ctx, providerConfigKey, requestedModel),
		stream:              true,
		includeUsage:        true,
		promptCacheKey:      promptCacheKey,
		extraFields:         extraFields,
		functionCalling:     p.buildFunctionCallingOptions(ctx, thinkingActive),
	}, thinkingActive, spinnerSuffix
}

func buildKimiTextChatCompletionsRequest(options kimiChatCompletionsRequestOptions, messages []api.Message) openaicompat.ChatCompletionsRequest {
	return openaicompat.BuildChatCompletionsRequest(options.openAICompatOptions(messages))
}

func (o kimiChatCompletionsRequestOptions) openAICompatOptions(messages []api.Message) openaicompat.ChatCompletionsRequestOptions {
	return openaicompat.ChatCompletionsRequestOptions{
		Model:               o.model,
		Messages:            messages,
		MaxCompletionTokens: o.maxCompletionTokens,
		Stream:              o.stream,
		IncludeUsage:        o.includeUsage,
		PromptCacheKey:      o.promptCacheKey,
		ExtraFields:         o.extraFields,
		FunctionCalling:     o.functionCalling,
	}
}

func (p *Provider) buildFunctionCallingOptions(ctx context.Context, thinkingActive bool) *openaicompat.FunctionCallingOptions {
	if !p.IsFunctionCallingEnabled() {
		return nil
	}
	return &openaicompat.FunctionCallingOptions{
		Tools:            openai.GetCombinedOpenAIToolsWithContext(ctx, p.mcpTools),
		ToolName:         p.toolChoice,
		ToolChoicePolicy: kimiToolChoicePolicy(thinkingActive),
	}
}

func kimiToolChoicePolicy(thinkingActive bool) openaicompat.ToolChoicePolicy {
	if thinkingActive {
		return openaicompat.AutoToolChoicePolicy
	}
	return openaicompat.AllowForcedToolChoicePolicy
}
