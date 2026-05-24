package api

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

// ActiveContextTransport は provider adapter が active context を request-local に送る形式を表す。
type ActiveContextTransport string

const (
	ActiveContextTransportNone                 ActiveContextTransport = "none"
	ActiveContextTransportNativeResponses      ActiveContextTransport = "native_responses"
	ActiveContextTransportEphemeralSystem      ActiveContextTransport = "ephemeral_system_message"
	ActiveContextTransportSystemPromptSuffix   ActiveContextTransport = "system_prompt_suffix"
	ActiveContextTransportEphemeralUserContent ActiveContextTransport = "ephemeral_user_content"
	ActiveContextTransportBedrockSystemBlock   ActiveContextTransport = "bedrock_system_block"
)

// ActiveContextTransportCapable は provider 固有 route に基づいて active context transport を返す。
type ActiveContextTransportCapable interface {
	ActiveContextTransport(ctx context.Context, model string) ActiveContextTransport
}

// ProviderActiveContextTransport は provider の active context transport を返す。
func ProviderActiveContextTransport(provider Provider, ctx context.Context, model string) ActiveContextTransport {
	return ProviderActiveContextTransportForRequest(provider, "", "", ctx, model)
}

// ProviderActiveContextTransportForRequest は runtime provider と catalog owner を明示して transport を返す。
func ProviderActiveContextTransportForRequest(provider Provider, runtimeProvider, catalogProvider string, ctx context.Context, model string) ActiveContextTransport {
	if provider != nil {
		if capable, ok := provider.(ActiveContextTransportCapable); ok {
			return normalizeActiveContextTransport(capable.ActiveContextTransport(ctx, model))
		}
		if strings.TrimSpace(runtimeProvider) == "" {
			runtimeProvider = provider.Name()
		}
	}
	return ProviderActiveContextTransportForProviderName(runtimeProvider, catalogProvider, ctx, model)
}

// ProviderActiveContextTransportForProviderName は provider 名から active context transport を返す。
func ProviderActiveContextTransportForProviderName(runtimeProvider, catalogProvider string, ctx context.Context, model string) ActiveContextTransport {
	runtimeProvider = config.CanonicalProviderName(runtimeProvider)
	if runtimeProvider == "" {
		return ActiveContextTransportNone
	}
	if strings.TrimSpace(catalogProvider) == "" {
		catalogProvider = runtimeProvider
	}
	cfg := config.FromContext(ctx)

	switch runtimeProvider {
	case "openai":
		if cfg.IsProviderResponsesAPIRequest("openai", catalogProvider, model) {
			return ActiveContextTransportNativeResponses
		}
		return ActiveContextTransportEphemeralSystem
	case "azure":
		return ActiveContextTransportNativeResponses
	case "claude", "anthropic":
		return ActiveContextTransportSystemPromptSuffix
	case "gemini":
		return ActiveContextTransportEphemeralUserContent
	case "deepseek", "groq", "kimi", "moonshot", "ollama":
		return ActiveContextTransportEphemeralSystem
	case "openrouter":
		return ActiveContextTransportEphemeralSystem
	case "bedrock":
		catalogModel := cfg.ModelCatalogName(catalogProvider, model)
		if llmcatalog.BedrockModelFamilyFor(model, catalogModel) == llmcatalog.BedrockModelFamilyClaude {
			return ActiveContextTransportSystemPromptSuffix
		}
		return ActiveContextTransportBedrockSystemBlock
	default:
		return ActiveContextTransportNone
	}
}

// ProviderCanConsumeActiveContext は provider request が active context を消費できるか返す。
func ProviderCanConsumeActiveContext(provider Provider, ctx context.Context, model string) bool {
	return ProviderActiveContextTransport(provider, ctx, model) != ActiveContextTransportNone
}

// ProviderCanConsumeActiveContextForRequest は runtime/catatalog owner を明示して active context 対応を判定する。
func ProviderCanConsumeActiveContextForRequest(provider Provider, runtimeProvider, catalogProvider string, ctx context.Context, model string) bool {
	return ProviderActiveContextTransportForRequest(provider, runtimeProvider, catalogProvider, ctx, model) != ActiveContextTransportNone
}

func normalizeActiveContextTransport(transport ActiveContextTransport) ActiveContextTransport {
	switch transport {
	case ActiveContextTransportNativeResponses,
		ActiveContextTransportEphemeralSystem,
		ActiveContextTransportSystemPromptSuffix,
		ActiveContextTransportEphemeralUserContent,
		ActiveContextTransportBedrockSystemBlock:
		return transport
	default:
		return ActiveContextTransportNone
	}
}

// RenderActiveContextBlocks は provider request に載せる active context の文字列表現を返す。
func RenderActiveContextBlocks(blocks []ActiveContextBlock) string {
	if len(blocks) == 0 {
		return ""
	}
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		content := strings.Trim(block.Content, "\n")
		if strings.TrimSpace(content) == "" {
			continue
		}
		parts = append(parts, content)
	}
	return strings.Join(parts, "\n\n")
}

// RenderActiveContextBlocksFromContext は request context の active context を描画する。
func RenderActiveContextBlocksFromContext(ctx context.Context) string {
	return RenderActiveContextBlocks(ActiveContextBlocksFromContext(ctx))
}

// SystemPromptWithActiveContext は active context を system prompt の dynamic suffix として追加する。
func SystemPromptWithActiveContext(systemPrompt string, blocks []ActiveContextBlock) string {
	rendered := RenderActiveContextBlocks(blocks)
	if strings.TrimSpace(rendered) == "" {
		return systemPrompt
	}
	layout := SplitSystemPromptLayout(systemPrompt)
	layout.AppendDynamic(rendered)
	return layout.Compose()
}

// SystemPromptWithActiveContextFromContext は request context の active context を dynamic suffix として追加する。
func SystemPromptWithActiveContextFromContext(ctx context.Context, systemPrompt string) string {
	return SystemPromptWithActiveContext(systemPrompt, ActiveContextBlocksFromContext(ctx))
}
