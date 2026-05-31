package agent

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ledger"
	"github.com/susugadx/xelyon-cli/internal/token"
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
	return token.EstimateTokenCountForModel(
		a.CurrentModel,
		api.RenderActiveContextBlocks(a.providerFacingActiveContextBlocksForTokenBudget(context.Background())),
	)
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
	return a.providerActiveContextTransport() != api.ActiveContextTransportNone
}

func (a *Agent) activeContextRuntimeProviderName() string {
	if a == nil {
		return ""
	}
	if runtimeProvider := providerRuntimeNameFromProvider(a.CurrentProvider); runtimeProvider != "" {
		return runtimeProvider
	}
	return config.CanonicalProviderName(a.ProviderName)
}

func (a *Agent) clearResponseContextForActiveContextRequest() {
	if len(a.providerFacingActiveContextBlocks()) == 0 {
		return
	}
	a.clearProviderAndSavedResponseContext()
}
