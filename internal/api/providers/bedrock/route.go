package bedrock

import (
	"context"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

type bedrockRoute string

const (
	bedrockRouteClaudeMessages bedrockRoute = "claude_messages"
	bedrockRouteConverseStream bedrockRoute = "converse_stream"
)

type bedrockRequestContext struct {
	model          string
	catalogModel   string
	route          bedrockRoute
	cfg            *config.Config
	providerConfig config.ProviderModelConfig
}

func (p *Provider) resolveBedrockRequestContext(ctx context.Context, model string) bedrockRequestContext {
	model = api.ResolveProviderRequestModel(ctx, model, "bedrock")

	cfg := config.ResolveContext(ctx, p.effectiveConfig())
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	catalogModel := cfg.ModelCatalogName("bedrock", model)
	pCfg, _ := cfg.GetProviderModelConfig("bedrock")

	return bedrockRequestContext{
		model:          model,
		catalogModel:   catalogModel,
		route:          resolveBedrockRoute(model, catalogModel),
		cfg:            cfg,
		providerConfig: pCfg,
	}
}

func resolveBedrockRoute(model, catalogModel string) bedrockRoute {
	if llmcatalog.BedrockModelFamilyFor(model, catalogModel) == llmcatalog.BedrockModelFamilyClaude {
		return bedrockRouteClaudeMessages
	}
	return bedrockRouteConverseStream
}

func ensureBedrockClaudeMessagesRoute(req bedrockRequestContext) error {
	if req.route == bedrockRouteClaudeMessages {
		return nil
	}
	return fmt.Errorf("bedrock Claude Messages route requires an Anthropic Claude model: model=%q catalog_model=%q route=%q", req.model, req.catalogModel, req.route)
}

func ensureBedrockConverseToolUseSupported(req bedrockRequestContext) error {
	if req.route != bedrockRouteConverseStream || llmcatalog.BedrockConverseToolUseSupported(req.model, req.catalogModel) {
		return nil
	}
	return fmt.Errorf("bedrock ConverseStream route requires a model with streaming tool use support: model=%q catalog_model=%q", req.model, req.catalogModel)
}
