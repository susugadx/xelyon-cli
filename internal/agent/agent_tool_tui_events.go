package agent

import (
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func (a *Agent) emitTUIToolRunning(toolCall *tools.ToolCall) {
	if toolCall == nil {
		return
	}
	a.emitTUIToolInfo(tools.ToolResultInfo{
		ToolName:  toolCall.Tool,
		Args:      toolCall.Args,
		ID:        toolCall.DisplayID(),
		Status:    tools.ToolStatusRunning,
		StartedAt: time.Now(),
	})
}

func (a *Agent) emitTUIToolInfo(info tools.ToolResultInfo) {
	if a == nil || a.tuiToolResultCh == nil || a.tuiToolResultClosed.Load() {
		return
	}
	info.Args = tools.CloneToolArgs(info.Args)
	select {
	case a.tuiToolResultCh <- info:
	default:
	}
}
