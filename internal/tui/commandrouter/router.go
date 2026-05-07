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
	// ActionQuit は TUI を終了する処理を表す。
	ActionQuit
	// ActionOpenConfig は TUI config screen を開く処理を表す。
	ActionOpenConfig
	// ActionOpenReview は TUI review screen を開く、または即時 review 実行する処理を表す。
	ActionOpenReview
	// ActionOpenProject は TUI project config screen を開く処理を表す。
	ActionOpenProject
)

// Context は command routing に必要な TUI 状態を保持する。
type Context struct {
	HasMouseSelection bool
}

// Route は slash command を TUI ローカル処理または agent 委譲に分類する。
func Route(command slash.Command, ctx Context) Action {
	if command.IsBare("/copy") && ctx.HasMouseSelection {
		return ActionCopyMouseSelection
	}
	if command.Matches("/exit") || command.Matches("/quit") {
		return ActionQuit
	}
	if command.IsBare("/config") {
		return ActionOpenConfig
	}
	if action, ok := routeCatalogTUILocalCommand(command); ok {
		return action
	}
	return ActionDispatchAgent
}

func routeCatalogTUILocalCommand(command slash.Command) (Action, bool) {
	cmdInfo, ok := commandcatalog.Find(command.ResolvedName)
	if !ok || cmdInfo.EffectiveOwner() != commandcatalog.CommandOwnerTUIRouter {
		return ActionDispatchAgent, false
	}

	switch cmdInfo.Name {
	case "/review":
		return ActionOpenReview, true
	case "/project":
		if command.ArgCount != 1 {
			return ActionDispatchAgent, false
		}
		return ActionOpenProject, true
	default:
		if command.ArgCount != 1 {
			return ActionDispatchAgent, false
		}
		return ActionDispatchAgent, false
	}
}
