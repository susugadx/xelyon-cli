package planning

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func init() {
	// AskUserQuestion ツールを登録
	tools.DefaultRegistry.Register(&AskUserQuestionTool{})
}
