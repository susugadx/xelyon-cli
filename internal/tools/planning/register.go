package planning

import (
	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func init() {
	// AskUserQuestion ツールを登録
	tools.DefaultRegistry.Register(&AskUserQuestionTool{})

	// Storage を必要とするツールを登録
	storage, err := plan.NewPlanStorage()
	if err != nil {
		return
	}

	// Phase 1
	tools.DefaultRegistry.Register(NewCreatePlanTool(storage))

	// Phase 2
	tools.DefaultRegistry.Register(NewUpdatePlanTool(storage))
}
