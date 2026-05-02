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

// Commands はコマンド一覧
var Commands = []CommandInfo{
	{
		Name:           "/exit",
		Aliases:        []string{"/quit", "/q"},
		Description:    "Exit the CLI",
		DescriptionJP:  "CLIを終了",
		Surfaces:       legacyFallbackSurfaces(),
		Owner:          CommandOwnerAgent,
		TUILocalArgs:   TUILocalArgAllowAny,
		TUILocalAction: TUILocalActionQuit,
		Category:       CommandCategorySystem,
		Discoverable:   true,
		SortWeight:     900,
	},
	{
		Name:          "/clear",
		Description:   "Clear conversation history",
		DescriptionJP: "会話履歴をクリア",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategorySession,
		Discoverable:  true,
		SortWeight:    160,
	},
	{
		Name:          "/history",
		Description:   "Show conversation history",
		DescriptionJP: "会話履歴を表示",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategorySession,
		Discoverable:  true,
		SortWeight:    170,
	},
	{
		Name:          "/save",
		Description:   "Save current session",
		DescriptionJP: "セッションを保存",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategorySession,
		Discoverable:  true,
		SortWeight:    130,
	},
	{
		Name:          "/load",
		Args:          "[id]",
		Description:   "Load session (or last if no ID)",
		DescriptionJP: "セッションを読み込み",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategorySession,
		Discoverable:  true,
		SortWeight:    140,
	},
	{
		Name:          "/sessions",
		Description:   "List recent sessions",
		DescriptionJP: "最近のセッション一覧",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategorySession,
		Discoverable:  true,
		SortWeight:    150,
	},
	{
		Name:          "/status",
		Aliases:       []string{"/stats"},
		Description:   "Show current state, last request, and session statistics",
		DescriptionJP: "現在状態、直近リクエスト、セッション統計を表示",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategorySystem,
		Discoverable:  true,
		SortWeight:    50,
	},
	{
		Name:          "/tokens",
		Description:   "Show token usage and context window status",
		DescriptionJP: "トークン使用量を表示",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategoryContext,
		Discoverable:  true,
		SortWeight:    60,
	},
	{
		Name:           "/copy",
		Args:           "[code] [-n N]",
		Description:    "Copy last AI output to clipboard (code=code blocks only, -n=N-th last output)",
		DescriptionJP:  "AI出力をクリップボードにコピー",
		Surfaces:       legacyFallbackSurfaces(),
		Owner:          CommandOwnerAgent,
		TUILocalArgs:   TUILocalArgBareOnly,
		TUILocalAction: TUILocalActionCopyMouseSelection,
		TUILocalWhen:   TUILocalWhenHasMouseSelection,
		Category:       CommandCategorySession,
		Discoverable:   true,
		SortWeight:     100,
	},
	{
		Name:           "/attach",
		Args:           "<path>",
		Description:    "Attach a file or image to the current composer draft (combined limit: up to 12 attachments per draft)",
		DescriptionJP:  "現在の入力ドラフトにファイル/画像を添付",
		Surfaces:       []CommandSurface{CommandSurfaceTUI},
		Owner:          CommandOwnerTUIRouter,
		TUILocalArgs:   TUILocalArgAllowAny,
		TUILocalAction: TUILocalActionManageAttachments,
		Category:       CommandCategorySession,
		Discoverable:   true,
		SortWeight:     101,
	},
	{
		Name:           "/detach",
		Args:           "<index>",
		Description:    "Detach one attachment by index",
		DescriptionJP:  "指定番号の添付を1件外す",
		Surfaces:       []CommandSurface{CommandSurfaceTUI},
		Owner:          CommandOwnerTUIRouter,
		TUILocalArgs:   TUILocalArgAllowAny,
		TUILocalAction: TUILocalActionManageAttachments,
		Category:       CommandCategorySession,
		Discoverable:   true,
		SortWeight:     102,
	},
	{
		Name:           "/detach-all",
		Description:    "Detach all attachments from the current draft",
		DescriptionJP:  "現在の入力ドラフトの添付をすべて外す",
		Surfaces:       []CommandSurface{CommandSurfaceTUI},
		Owner:          CommandOwnerTUIRouter,
		TUILocalArgs:   TUILocalArgAllowAny,
		TUILocalAction: TUILocalActionManageAttachments,
		Category:       CommandCategorySession,
		Discoverable:   true,
		SortWeight:     103,
	},
	{
		Name:           "/review",
		Description:    "Review current changes and find issues",
		DescriptionJP:  "現在の変更をレビュー",
		Surfaces:       []CommandSurface{CommandSurfaceTUI},
		Owner:          CommandOwnerTUIRouter,
		TUILocalArgs:   TUILocalArgBareOnly,
		TUILocalAction: TUILocalActionOpenReview,
		Lifecycle:      CommandLifecyclePreview,
		Category:       CommandCategoryReview,
		Discoverable:   true,
		SortWeight:     70,
	},
	{
		Name:          "/compress",
		Args:          "[N] [-c]",
		Description:   "Compress history (keep recent N, -c: use OpenAI Compact API)",
		DescriptionJP: "履歴を圧縮",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategoryContext,
		Discoverable:  true,
		SortWeight:    110,
	},
	{
		Name:          "/use",
		Args:          "<provider> [model]",
		Description:   "Switch provider and optionally model (e.g., /use gemini gemini-2.0-flash-exp)",
		DescriptionJP: "プロバイダーを切り替え",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategoryModel,
		Discoverable:  true,
		SortWeight:    20,
	},
	{
		Name:          "/providers",
		Description:   "List available providers and their API key status",
		DescriptionJP: "利用可能なプロバイダー一覧",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategoryModel,
		Discoverable:  true,
		SortWeight:    30,
	},
	{
		Name:           "/config",
		Description:    "Edit global config.yaml settings",
		DescriptionJP:  "対話式設定変更",
		Surfaces:       legacyFallbackSurfaces(),
		Owner:          CommandOwnerAgent,
		TUILocalArgs:   TUILocalArgBareOnly,
		TUILocalAction: TUILocalActionOpenConfig,
		Category:       CommandCategoryConfig,
		Discoverable:   true,
		SortWeight:     90,
		SubCommands: []SubCommand{
			{Name: "/config show", Description: "Show all settings with diff from defaults"},
			{Name: "/config model <name>", Description: "Change default model"},
		},
	},
	{
		Name:          "/model",
		Args:          "[name]",
		Description:   "Show current model or switch model without restart",
		DescriptionJP: "モデルを表示/切り替え",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategoryModel,
		Discoverable:  true,
		SortWeight:    10,
	},
	{
		Name:          "/init",
		Description:   "Create xelyon.yaml project template",
		DescriptionJP: "xelyon.yamlテンプレートを作成",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategoryConfig,
		Discoverable:  true,
		SortWeight:    180,
	},
	{
		Name:           "/project",
		Description:    "Edit project xelyon.yaml interactively",
		DescriptionJP:  "xelyon.yamlを対話式で編集",
		Surfaces:       []CommandSurface{CommandSurfaceTUI},
		Owner:          CommandOwnerTUIRouter,
		TUILocalArgs:   TUILocalArgBareOnly,
		TUILocalAction: TUILocalActionOpenProject,
		Category:       CommandCategoryConfig,
		Discoverable:   true,
		SortWeight:     80,
	},
	{
		Name:          "/plan",
		Args:          "[on|off]",
		Description:   "Toggle Plan Mode (investigation -> plan -> approval; implementation happens on next normal turn)",
		DescriptionJP: "Plan Modeを切り替え",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategoryDev,
		Discoverable:  true,
		SortWeight:    120,
	},
	{
		Name:          "/think",
		Args:          "[on|off|level]",
		Description:   "Toggle Extended Thinking mode (level: low/medium/high/xhigh)",
		DescriptionJP: "Extended Thinkingを切り替え",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategoryModel,
		Discoverable:  true,
		SortWeight:    40,
	},
	{
		Name:          "/lsp",
		Args:          "[status]",
		Description:   "Show LSP server status (running/not started/disabled)",
		DescriptionJP: "LSPサーバー状態を表示",
		Surfaces:      classicOnlySurfaces(),
		Category:      CommandCategoryDev,
		SortWeight:    190,
	},
	{
		Name:          "/version",
		Description:   "Show version information",
		DescriptionJP: "バージョンを表示",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategorySystem,
		SortWeight:    920,
	},
	{
		Name:          "/help",
		Description:   "Show this help",
		DescriptionJP: "ヘルプを表示",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategorySystem,
		SortWeight:    910,
	},
}

// Tips はTips一覧
var Tips = []string{
	"Just describe what you want in natural language",
	"AI will ask confirmation for dangerous operations",
	"Use Ctrl+C to cancel current operation",
	"Use git to revert file changes when needed",
}
