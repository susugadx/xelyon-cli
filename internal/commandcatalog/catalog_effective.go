package commandcatalog

// SupportsSurface は command が指定 surface で利用可能かを返す。
func (cmd CommandInfo) SupportsSurface(surface CommandSurface) bool {
	for _, candidate := range cmd.effectiveSurfaces() {
		if candidate == surface {
			return true
		}
	}
	return false
}

// EffectiveOwner は command 実行責務の owner を返す。
func (cmd CommandInfo) EffectiveOwner() CommandOwner {
	if cmd.Owner == "" {
		return CommandOwnerAgent
	}
	return cmd.Owner
}

// EffectiveTUILocalArgPolicy は TUI ローカル処理時の引数許可ポリシーを返す。
func (cmd CommandInfo) EffectiveTUILocalArgPolicy() TUILocalArgPolicy {
	if cmd.TUILocalArgs == "" {
		return TUILocalArgBareOnly
	}
	return cmd.TUILocalArgs
}

// AcceptsTUILocalArgs は TUI ローカル処理として受け付ける引数数かを返す。
func (cmd CommandInfo) AcceptsTUILocalArgs(args []string) bool {
	switch cmd.EffectiveTUILocalArgPolicy() {
	case TUILocalArgAllowAny:
		return true
	default:
		return len(args) == 0
	}
}

// TUILocalContext は TUI ローカル action 判定に必要な実行時コンテキスト。
type TUILocalContext struct {
	HasMouseSelection bool
}

// EffectiveTUILocalWhen は TUI ローカル action の実行条件を返す。
func (cmd CommandInfo) EffectiveTUILocalWhen() TUILocalWhen {
	return cmd.TUILocalWhen
}

// AcceptsTUILocalContext は TUI ローカル action の実行条件を満たすかを返す。
func (cmd CommandInfo) AcceptsTUILocalContext(ctx TUILocalContext) bool {
	switch cmd.EffectiveTUILocalWhen() {
	case TUILocalWhenHasMouseSelection:
		return ctx.HasMouseSelection
	default:
		return true
	}
}

// EffectiveTUILocalAction は TUI ローカル command の処理種別を返す。
func (cmd CommandInfo) EffectiveTUILocalAction() TUILocalAction {
	return cmd.TUILocalAction
}

// EffectiveLifecycle は command の公開段階を返す。
func (cmd CommandInfo) EffectiveLifecycle() CommandLifecycle {
	if cmd.Lifecycle == "" {
		return CommandLifecycleStable
	}
	return cmd.Lifecycle
}

// EffectiveCategory は command の分類を返す。
func (cmd CommandInfo) EffectiveCategory() CommandCategory {
	if cmd.Category == "" {
		return CommandCategoryOther
	}
	return cmd.Category
}

// EffectiveSortWeight は command 候補の並び順を返す。
func (cmd CommandInfo) EffectiveSortWeight() int {
	if cmd.SortWeight == 0 {
		return 1000
	}
	return cmd.SortWeight
}

func (cmd CommandInfo) effectiveSurfaces() []CommandSurface {
	if len(cmd.Surfaces) == 0 {
		return []CommandSurface{CommandSurfaceTUI}
	}
	return cmd.Surfaces
}
