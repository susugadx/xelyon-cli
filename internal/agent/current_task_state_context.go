package agent

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ledger"
)

const currentTaskStateActiveContextName = "current_task_state"

func (a *Agent) buildActiveContextBlocks() []api.ActiveContextBlock {
	block, ok := a.buildCurrentTaskStateBlock()
	if !ok {
		return nil
	}
	return []api.ActiveContextBlock{block}
}

func (a *Agent) buildCurrentTaskStateBlock() (api.ActiveContextBlock, bool) {
	if a == nil || a.Runtime == nil {
		return api.ActiveContextBlock{}, false
	}
	if !a.Runtime.Options.EnableCurrentTaskStateContext || a.Runtime.TaskLedger == nil {
		return api.ActiveContextBlock{}, false
	}

	state := a.Runtime.TaskLedger.Snapshot()
	if state.IsEmpty() {
		return api.ActiveContextBlock{}, false
	}

	content := ledger.RenderCurrentTaskStateSnapshot(state, ledger.DefaultSnapshotRenderOptions())
	if strings.TrimSpace(content) == "" {
		return api.ActiveContextBlock{}, false
	}
	return api.ActiveContextBlock{
		Name:    currentTaskStateActiveContextName,
		Content: content,
	}, true
}

func (a *Agent) estimateActiveContextTokens() int {
	total := 0
	for _, block := range a.providerFacingActiveContextBlocksForTokenBudget(context.Background()) {
		total += token.EstimateTokenCountForModel(a.CurrentModel, block.Content)
	}
	return total
}

func (a *Agent) providerFacingActiveContextBlocks() []api.ActiveContextBlock {
	return activeContextInputPolicy{}.Blocks(a)
}

func (a *Agent) shouldSendActiveContextToProvider() bool {
	if a == nil || a.Runtime == nil || !a.Runtime.Options.EnableCurrentTaskStateContext {
		return false
	}
	return a.providerCanConsumeActiveContext()
}

func (a *Agent) providerCanConsumeActiveContext() bool {
	if a == nil {
		return false
	}
	cfg := a.cfg()
	runtimeProvider := providerRuntimeNameFromProvider(a.CurrentProvider)
	if runtimeProvider == "" {
		runtimeProvider = config.CanonicalProviderName(a.ProviderName)
	}
	catalogProvider := a.activeModelProviderConfigKey(cfg)
	return cfg.IsProviderResponsesAPIRequest(runtimeProvider, catalogProvider, a.CurrentModel)
}

func (a *Agent) clearResponseContextForActiveContextRequest() {
	if len(a.providerFacingActiveContextBlocks()) == 0 {
		return
	}
	a.clearProviderAndSavedResponseContext()
}
