package commandcatalog

// CommandInfo はコマンド情報
type CommandInfo struct {
	Name           string           // "/config"
	Aliases        []string         // []string{"/quit", "/q"}
	Args           string           // "[id]", "<provider> [model]"
	Description    string           // "Edit global config.yaml settings"
	DescriptionJP  string           // "対話式設定変更"
	SubCommands    []SubCommand     // サブコマンド
	Surfaces       []CommandSurface // 省略時は TUI primary のみ
	Owner          CommandOwner     // 省略時は agent dispatcher
	TUILocalArgs   TUILocalArgPolicy
	TUILocalAction TUILocalAction
	TUILocalWhen   TUILocalWhen
	Lifecycle      CommandLifecycle // 省略時は stable
	Category       CommandCategory  // 省略時は other
	Discoverable   bool             // true の command だけを候補表示に出す
	HiddenFromHelp bool             // true の command は互換実行だけ残し help には出さない
	SortWeight     int              // 小さいほど候補で上に出す
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

func tuiOnlySurfaces() []CommandSurface {
	return []CommandSurface{CommandSurfaceTUI}
}

// CommandOwner は command 実行責務の owner を表す。
type CommandOwner string

const (
	// CommandOwnerAgent は agent command registry が実行する command。
	CommandOwnerAgent CommandOwner = "agent"
	// CommandOwnerTUIRouter は TUI commandrouter が bare command を処理する command。
	CommandOwnerTUIRouter CommandOwner = "tui_router"
)

// TUILocalArgPolicy は TUI ローカル処理時の引数許可ポリシーを表す。
type TUILocalArgPolicy string

const (
	// TUILocalArgBareOnly は引数なし（bare）のみ TUI ローカル処理する。
	TUILocalArgBareOnly TUILocalArgPolicy = "bare_only"
	// TUILocalArgAllowAny は引数付きでも TUI ローカル処理する。
	TUILocalArgAllowAny TUILocalArgPolicy = "allow_any"
)

// TUILocalAction は TUI ローカル command が引き起こす処理種別を表す。
type TUILocalAction string

const (
	TUILocalActionNone               TUILocalAction = ""
	TUILocalActionCopyMouseSelection TUILocalAction = "copy_mouse_selection"
	TUILocalActionManageAttachments  TUILocalAction = "manage_attachments"
	TUILocalActionQuit               TUILocalAction = "quit"
	TUILocalActionOpenConfig         TUILocalAction = "open_config"
	TUILocalActionOpenReview         TUILocalAction = "open_review"
	TUILocalActionOpenProject        TUILocalAction = "open_project"
	TUILocalActionOpenProviderPicker TUILocalAction = "open_provider_picker"
	TUILocalActionOpenModelPicker    TUILocalAction = "open_model_picker"
)

// TUILocalWhen は TUI ローカル action の実行前提条件を表す。
type TUILocalWhen string

const (
	TUILocalWhenNone              TUILocalWhen = ""
	TUILocalWhenHasMouseSelection TUILocalWhen = "has_mouse_selection"
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
