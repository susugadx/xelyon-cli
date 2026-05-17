package agent

import (
	"github.com/susugadx/xelyon-cli/internal/api"
)

type modelInputAssemblyPlan struct {
	CompactedInput      []api.InputItem
	ActiveContextBlocks []api.ActiveContextBlock
}

func (a *Agent) modelInputAssemblyPlan() modelInputAssemblyPlan {
	return modelInputAssemblyPlan{
		CompactedInput:      a.compactedInputForModelRequest(),
		ActiveContextBlocks: activeContextInputPolicy{}.Blocks(a),
	}
}

func (a *Agent) compactedInputForModelRequest() []api.InputItem {
	if a == nil || !a.isCompactedMode || len(a.compactedItems) == 0 {
		return nil
	}
	return api.CloneInputItems(a.compactedItems)
}

type activeContextInputPolicy struct{}

func (activeContextInputPolicy) Blocks(a *Agent) []api.ActiveContextBlock {
	if a == nil || !a.shouldSendActiveContextToProvider() {
		return nil
	}
	return a.buildActiveContextBlocks()
}
