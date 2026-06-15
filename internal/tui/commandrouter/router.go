package commandrouter

import (
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/tui/slash"
)

// Action は TUI が slash command に対して実行する処理種別を表す。
type Action string

const (
	// ActionDispatchAgent は agent 側 command handler へ委譲する処理を表す。
	ActionDispatchAgent Action = ""
	// ActionCopyMouseSelection は mouse selection をコピーする TUI ローカル処理を表す。
	ActionCopyMouseSelection Action = Action(commandcatalog.TUILocalActionCopyMouseSelection)
	// ActionManageAttachments は composer 添付を操作する TUI ローカル処理を表す。
	ActionManageAttachments Action = Action(commandcatalog.TUILocalActionManageAttachments)
	// ActionQuit は TUI を終了する処理を表す。
	ActionQuit Action = Action(commandcatalog.TUILocalActionQuit)
	// ActionOpenConfig は TUI config screen を開く処理を表す。
	ActionOpenConfig Action = Action(commandcatalog.TUILocalActionOpenConfig)
	// ActionOpenReview は TUI review screen を開く、または即時 review 実行する処理を表す。
	ActionOpenReview Action = Action(commandcatalog.TUILocalActionOpenReview)
	// ActionOpenProject は TUI project config screen を開く処理を表す。
	ActionOpenProject Action = Action(commandcatalog.TUILocalActionOpenProject)
	// ActionOpenProviderPicker は provider/model picker を開く処理を表す。
	ActionOpenProviderPicker Action = Action(commandcatalog.TUILocalActionOpenProviderPicker)
	// ActionOpenModelPicker は current provider の model picker を開く処理を表す。
	ActionOpenModelPicker Action = Action(commandcatalog.TUILocalActionOpenModelPicker)
	// ActionNewSession は TUI 表示状態を考慮して新規 session を開始する処理を表す。
	ActionNewSession Action = Action(commandcatalog.TUILocalActionNewSession)
	// ActionOpenSessionPicker は resume session picker を開く処理を表す。
	ActionOpenSessionPicker Action = Action(commandcatalog.TUILocalActionOpenSessionPicker)
)

// Context は command routing に必要な TUI 状態を保持する。
type Context struct {
	HasMouseSelection bool
}

// Route は parse 済み slash command を TUI ローカル処理または agent 委譲に分類する。
func Route(command slash.Command, ctx Context) Action {
	if action, ok := routeCatalogTUILocalCommand(command, ctx); ok {
		return action
	}
	return ActionDispatchAgent
}

// UsesRawTUILocalArgs は command token 後の入力を quote 解析せず処理する TUI local command かを返す。
func UsesRawTUILocalArgs(command slash.Command, ctx Context) bool {
	cmdInfo, ok := commandcatalog.Find(command.ResolvedName)
	if !ok || !cmdInfo.UsesRawTUILocalArgs() {
		return false
	}
	if !cmdInfo.SupportsSurface(commandcatalog.CommandSurfaceTUI) {
		return false
	}
	if !cmdInfo.AcceptsTUILocalContext(commandcatalog.TUILocalContext{
		HasMouseSelection: ctx.HasMouseSelection,
	}) {
		return false
	}
	return isKnownLocalAction(Action(cmdInfo.EffectiveTUILocalAction()))
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

	action := Action(cmdInfo.EffectiveTUILocalAction())
	if !isKnownLocalAction(action) {
		return ActionDispatchAgent, false
	}
	return action, true
}

func isKnownLocalAction(action Action) bool {
	switch action {
	case ActionCopyMouseSelection,
		ActionManageAttachments,
		ActionQuit,
		ActionOpenConfig,
		ActionOpenReview,
		ActionOpenProject,
		ActionOpenProviderPicker,
		ActionOpenModelPicker,
		ActionNewSession,
		ActionOpenSessionPicker:
		return true
	default:
		return false
	}
}
