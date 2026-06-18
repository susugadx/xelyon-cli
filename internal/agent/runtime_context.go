package agent

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/token"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func (a *Agent) requestContext(ctx context.Context) context.Context {
	ctx = tools.WithRegistry(ctx, a.registry())
	ctx = tools.WithConfig(ctx, a.cfg())
	ctx = uiruntime.WithRuntime(ctx, a.ui())
	ctx = api.WithAssistantUpdateMode(ctx, a.assistantUpdateMode())
	if a != nil && a.session != nil && strings.TrimSpace(a.session.ID) != "" {
		ctx = api.WithPromptCacheScope(ctx, api.PromptCacheScope{SessionID: a.session.ID})
	}
	inputPlan := a.modelInputAssemblyPlan()
	if len(inputPlan.CompactedInput) > 0 {
		ctx = api.WithCompactedInputItems(ctx, inputPlan.CompactedInput)
	}
	if blocks := inputPlan.ActiveContextBlocks; len(blocks) > 0 {
		ctx = api.WithActiveContextBlocks(ctx, blocks)
	} else {
		ctx = api.WithoutActiveContextBlocks(ctx)
	}
	return ctx
}

func (a *Agent) requestContextWithoutActiveContext(ctx context.Context) context.Context {
	return api.WithoutActiveContextBlocks(a.requestContext(ctx))
}

func (a *Agent) parseToolCalls(response string) []*tools.ToolCall {
	return tools.ParseToolCallsWithRegistry(response, a.registry(), a.ui().ErrorOutput())
}

func (a *Agent) estimateToolDefinitionTokens() int {
	total := 0
	for _, def := range a.registry().GetToolDefinitions() {
		total += token.EstimateStructuredValueTokenCountForModel(a.CurrentModel, def)
	}
	return total
}

func (a *Agent) countToolsByType() (builtin, mcp int) {
	for _, def := range a.registry().GetToolDefinitions() {
		if strings.HasPrefix(def.Name, "mcp_") {
			mcp++
		} else {
			builtin++
		}
	}
	return
}
