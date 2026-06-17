package search

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/websearch"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func webSearchRequestContext(execCtx tools.ExecutionContext, cfg *config.Config, searchProvider, searchModel string) context.Context {
	requestCtx := execCtx.EffectiveContext()
	requestCtx = tools.WithRegistry(requestCtx, execCtx.EffectiveRegistry())
	requestCtx = tools.WithConfig(requestCtx, cfg)
	requestCtx = api.WithAssistantUpdateMode(requestCtx, api.AssistantUpdatesOff)
	return websearch.WithUsageCallback(requestCtx, webSearchUsageCallback(execCtx, searchProvider, searchModel))
}

func webSearchAPIRequestContext(ctx context.Context, cfg *config.Config, legacy api.UsageCallback, attribution tools.UsageAttributionCallback, provider, model string) context.Context {
	requestCtx := tools.WithConfig(ctx, cfg)
	requestCtx = api.WithAssistantUpdateMode(requestCtx, api.AssistantUpdatesOff)
	if callback := webSearchProviderUsageCallback(legacy, attribution, provider, model); callback != nil {
		requestCtx = websearch.WithUsageCallback(requestCtx, callback)
	}
	return requestCtx
}

func webSearchUsageCallback(execCtx tools.ExecutionContext, provider, model string) api.UsageCallback {
	return webSearchProviderUsageCallback(nil, execCtx.UsageAttribution, provider, model)
}

func webSearchProviderUsageCallback(legacy api.UsageCallback, attribution tools.UsageAttributionCallback, provider, model string) api.UsageCallback {
	if legacy == nil && attribution == nil {
		return nil
	}
	return func(usage api.Usage) {
		if legacy != nil {
			legacy(usage)
		}
		if attribution != nil {
			attribution(provider, model, usage)
		}
	}
}
