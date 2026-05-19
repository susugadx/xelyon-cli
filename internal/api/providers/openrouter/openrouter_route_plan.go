package openrouter

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

const (
	openRouterRouteChatCompletions   = "chat_completions"
	openRouterRouteAnthropicMessages = "anthropic_messages"

	openRouterChatCompletionsEndpointPath   = "/v1/chat/completions"
	openRouterAnthropicMessagesEndpointPath = "/v1/messages"
	openRouterChatCompletionsPathSuffix     = "/chat/completions"
	openRouterAnthropicMessagesPathSuffix   = "/messages"

	DiagnosticRouteChatCompletions   = openRouterRouteChatCompletions
	DiagnosticRouteAnthropicMessages = openRouterRouteAnthropicMessages
)

type openRouterRoutePlan struct {
	Route  string
	Reason string
	APIURL string
}

func (p *Provider) routePlanForRequest(cfg *config.Config, model string) openRouterRoutePlan {
	return resolveOpenRouterRoutePlan(cfg, p.APIURL, model)
}

func resolveOpenRouterRoutePlan(cfg *config.Config, configuredAPIURL, model string) openRouterRoutePlan {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	model = strings.TrimSpace(model)
	if shouldUseOpenRouterClaudeAPI(model, cfg.Compression) {
		return openRouterRoutePlan{
			Route: DiagnosticRouteAnthropicMessages,
			Reason: fmt.Sprintf(
				"request model %s enables OpenRouter Anthropic Skin context management; %s is selected",
				model,
				openRouterAnthropicMessagesEndpointPath,
			),
			APIURL: getAnthropicSkinURL(configuredAPIURL),
		}
	}

	reason := "request model does not enable OpenRouter Claude context management; OpenAI-compatible Chat Completions is selected"
	if isClaudeModel(model) {
		reason = "request model is Claude but OpenRouter Claude context management is disabled; OpenAI-compatible Chat Completions is selected"
	}
	return openRouterRoutePlan{
		Route:  DiagnosticRouteChatCompletions,
		Reason: reason,
		APIURL: configuredAPIURL,
	}
}

func (p openRouterRoutePlan) usesAnthropicMessages() bool {
	return p.Route == DiagnosticRouteAnthropicMessages
}

// getAnthropicSkinURL は OpenAI 互換 URL から Anthropic Skin URL を導出する。
func getAnthropicSkinURL(openaiURL string) string {
	if idx := strings.Index(openaiURL, openRouterChatCompletionsEndpointPath); idx >= 0 {
		return openaiURL[:idx] + openRouterAnthropicMessagesEndpointPath
	}
	return strings.TrimSuffix(openaiURL, openRouterChatCompletionsPathSuffix) + openRouterAnthropicMessagesPathSuffix
}
