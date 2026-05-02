package commandcatalog

// CommandInfo はコマンド情報
type CommandInfo struct {
	Name          string           // "/config"
	Aliases       []string         // []string{"/quit", "/q"}
	Args          string           // "[id]", "<provider> [model]"
	Description   string           // "Edit global config.yaml settings"
	DescriptionJP string           // "対話式設定変更"
	SubCommands   []SubCommand     // サブコマンド
	Surfaces      []CommandSurface // 省略時は TUI primary のみ
	Owner         CommandOwner     // 省略時は agent dispatcher
	Lifecycle     CommandLifecycle // 省略時は stable
	Category      CommandCategory  // 省略時は other
	Discoverable  bool             // true の command だけを候補表示に出す
	SortWeight    int              // 小さいほど候補で上に出す
}

// SubCommand はサブコマンド情報
type SubCommand struct {
	Name        string // "/config show"
	Description string // "Show all settings with diff from defaults"
}

// CommandSurface は command を表示・実行できる UI surface を表す。
type CommandSurface string

const (
	// CommandSurfaceTUI は primary interactive surface。
	CommandSurfaceTUI CommandSurface = "tui"
	// CommandSurfaceClassic は --no-tui 用の legacy surface。
	CommandSurfaceClassic CommandSurface = "classic"
)

// legacyFallbackSurfaces は classic REPL に残す既存互換 command だけに使う。
// 新しい対話型 command は TUI を primary surface とし、ここには追加しない。
func legacyFallbackSurfaces() []CommandSurface {
	return []CommandSurface{CommandSurfaceTUI, CommandSurfaceClassic}
}

// classicOnlySurfaces は TUI primary では公開しない legacy/debug command に使う。
func classicOnlySurfaces() []CommandSurface {
	return []CommandSurface{CommandSurfaceClassic}
}

// CommandOwner は command 実行責務の owner を表す。
type CommandOwner string

const (
	// CommandOwnerAgent は agent command registry が実行する command。
	CommandOwnerAgent CommandOwner = "agent"
	// CommandOwnerTUIRouter は TUI commandrouter が bare command を処理する command。
	CommandOwnerTUIRouter CommandOwner = "tui_router"
)

// CommandLifecycle は command の公開段階を表す。
type CommandLifecycle string

const (
	CommandLifecycleStable  CommandLifecycle = "stable"
	CommandLifecyclePreview CommandLifecycle = "preview"
)

// CommandCategory は command discovery 用の大まかな分類を表す。
type CommandCategory string

const (
	CommandCategoryOther   CommandCategory = "other"
	CommandCategoryReview  CommandCategory = "review"
	CommandCategoryModel   CommandCategory = "model"
	CommandCategoryConfig  CommandCategory = "config"
	CommandCategorySession CommandCategory = "session"
	CommandCategoryContext CommandCategory = "context"
	CommandCategorySystem  CommandCategory = "system"
	CommandCategoryDev     CommandCategory = "dev"
)

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
