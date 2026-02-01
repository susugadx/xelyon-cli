package planning

import (
	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func init() {
	// AskUserQuestion ツールを登録
	tools.DefaultRegistry.Register(&AskUserQuestionTool{})

	// CreatePlan ツールを登録（Storage が必要）
	storage, err := plan.NewPlanStorage()
	if err == nil {
		tools.DefaultRegistry.Register(NewCreatePlanTool(storage))
	}
}
