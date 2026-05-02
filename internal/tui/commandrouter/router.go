package commandrouter

import (
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/tui/slash"
)

// Action は TUI が slash command に対して実行する処理種別を表す。
type Action int

const (
	// ActionDispatchAgent は agent 側 command handler へ委譲する処理を表す。
	ActionDispatchAgent Action = iota
	// ActionCopyMouseSelection は mouse selection をコピーする TUI ローカル処理を表す。
	ActionCopyMouseSelection
	// ActionManageAttachments は composer 添付を操作する TUI ローカル処理を表す。
	ActionManageAttachments
	// ActionQuit は TUI を終了する処理を表す。
	ActionQuit
	// ActionOpenConfig は TUI config screen を開く処理を表す。
	ActionOpenConfig
	// ActionOpenReview は TUI review preset screen を開く処理を表す。
	ActionOpenReview
	// ActionOpenProject は TUI project config screen を開く処理を表す。
	ActionOpenProject
)

// Context は command routing に必要な TUI 状態を保持する。
type Context struct {
	HasMouseSelection bool
}

var catalogActionToRouterAction = map[commandcatalog.TUILocalAction]Action{
	commandcatalog.TUILocalActionCopyMouseSelection: ActionCopyMouseSelection,
	commandcatalog.TUILocalActionManageAttachments:  ActionManageAttachments,
	commandcatalog.TUILocalActionQuit:               ActionQuit,
	commandcatalog.TUILocalActionOpenConfig:         ActionOpenConfig,
	commandcatalog.TUILocalActionOpenReview:         ActionOpenReview,
	commandcatalog.TUILocalActionOpenProject:        ActionOpenProject,
}

// Route は parse 済み slash command を TUI ローカル処理または agent 委譲に分類する。
func Route(command slash.Command, ctx Context) Action {
	if action, ok := routeCatalogTUILocalCommand(command, ctx); ok {
		return action
	}
	return ActionDispatchAgent
}

func routeCatalogTUILocalCommand(command slash.Command, ctx Context) (Action, bool) {
	cmdInfo, ok := commandcatalog.Find(command.ResolvedName)
	if !ok || cmdInfo.EffectiveTUILocalAction() == commandcatalog.TUILocalActionNone {
		return ActionDispatchAgent, false
	}
	if !cmdInfo.SupportsSurface(commandcatalog.CommandSurfaceTUI) {
		return ActionDispatchAgent, false
	}
	if !cmdInfo.AcceptsTUILocalArgs(command.Args) {
		return ActionDispatchAgent, false
	}
	if !cmdInfo.AcceptsTUILocalContext(commandcatalog.TUILocalContext{
		HasMouseSelection: ctx.HasMouseSelection,
	}) {
		return ActionDispatchAgent, false
	}

	action, ok := catalogActionToRouterAction[cmdInfo.EffectiveTUILocalAction()]
	if !ok {
		return ActionDispatchAgent, false
	}
	return action, true
}
