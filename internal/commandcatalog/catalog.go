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

func newCommandInfo(name, args, description, descriptionJP string, surfaces []CommandSurface, category CommandCategory, sortWeight int, discoverable bool) CommandInfo {
	return CommandInfo{
		Name:          name,
		Args:          args,
		Description:   description,
		DescriptionJP: descriptionJP,
		Surfaces:      surfaces,
		Category:      category,
		Discoverable:  discoverable,
		SortWeight:    sortWeight,
	}
}

func legacyDiscoverableCommand(name, args, description, descriptionJP string, category CommandCategory, sortWeight int, aliases ...string) CommandInfo {
	cmd := newCommandInfo(name, args, description, descriptionJP, legacyFallbackSurfaces(), category, sortWeight, true)
	cmd.Aliases = aliases
	return cmd
}

func legacyHiddenCommand(name, args, description, descriptionJP string, category CommandCategory, sortWeight int) CommandInfo {
	return newCommandInfo(name, args, description, descriptionJP, legacyFallbackSurfaces(), category, sortWeight, false)
}

func classicOnlyHiddenCommand(name, args, description, descriptionJP string, category CommandCategory, sortWeight int) CommandInfo {
	return newCommandInfo(name, args, description, descriptionJP, classicOnlySurfaces(), category, sortWeight, false)
}

func legacyAgentTUILocalCommand(
	name, args, description, descriptionJP string,
	action TUILocalAction,
	argPolicy TUILocalArgPolicy,
	when TUILocalWhen,
	category CommandCategory,
	sortWeight int,
	aliases []string,
	subCommands []SubCommand,
) CommandInfo {
	cmd := newCommandInfo(name, args, description, descriptionJP, legacyFallbackSurfaces(), category, sortWeight, true)
	cmd.Aliases = aliases
	cmd.SubCommands = subCommands
	cmd.Owner = CommandOwnerAgent
	cmd.TUILocalArgs = argPolicy
	cmd.TUILocalAction = action
	cmd.TUILocalWhen = when
	return cmd
}

func attachmentTUILocalCommand(name, args, description, descriptionJP string, sortWeight int) CommandInfo {
	cmd := newCommandInfo(name, args, description, descriptionJP, tuiOnlySurfaces(), CommandCategorySession, sortWeight, true)
	cmd.Owner = CommandOwnerTUIRouter
	cmd.TUILocalArgs = TUILocalArgAllowAny
	cmd.TUILocalAction = TUILocalActionManageAttachments
	return cmd
}

func tuiRouterBareLocalCommand(name, description, descriptionJP string, action TUILocalAction, category CommandCategory, sortWeight int, lifecycle CommandLifecycle) CommandInfo {
	cmd := newCommandInfo(name, "", description, descriptionJP, tuiOnlySurfaces(), category, sortWeight, true)
	cmd.Owner = CommandOwnerTUIRouter
	cmd.TUILocalArgs = TUILocalArgBareOnly
	cmd.TUILocalAction = action
	cmd.Lifecycle = lifecycle
	return cmd
}

// Commands はコマンド一覧
var Commands = []CommandInfo{
	commandExit(),
	commandClear(),
	commandHistory(),
	commandSave(),
	commandLoad(),
	commandSessions(),
	commandStatus(),
	commandTokens(),
	commandCopy(),
	commandAttach(),
	commandDetach(),
	commandDetachAll(),
	commandReview(),
	commandCompress(),
	commandUse(),
	commandProviders(),
	commandConfig(),
	commandModel(),
	commandInit(),
	commandProject(),
	commandPlan(),
	commandThink(),
	commandLSP(),
	commandVersion(),
	commandHelp(),
}

func commandExit() CommandInfo {
	return legacyAgentTUILocalCommand(
		"/exit",
		"",
		"Exit the CLI",
		"CLIを終了",
		TUILocalActionQuit,
		TUILocalArgAllowAny,
		TUILocalWhenNone,
		CommandCategorySystem,
		900,
		[]string{"/quit", "/q"},
		nil,
	)
}

func commandClear() CommandInfo {
	return legacyDiscoverableCommand("/clear", "", "Clear conversation history", "会話履歴をクリア", CommandCategorySession, 160)
}

func commandHistory() CommandInfo {
	return legacyDiscoverableCommand("/history", "", "Show conversation history", "会話履歴を表示", CommandCategorySession, 170)
}

func commandSave() CommandInfo {
	return legacyDiscoverableCommand("/save", "", "Save current session", "セッションを保存", CommandCategorySession, 130)
}

func commandLoad() CommandInfo {
	return legacyDiscoverableCommand("/load", "[id]", "Load session (or last if no ID)", "セッションを読み込み", CommandCategorySession, 140)
}

func commandSessions() CommandInfo {
	return legacyDiscoverableCommand("/sessions", "", "List recent sessions", "最近のセッション一覧", CommandCategorySession, 150)
}

func commandStatus() CommandInfo {
	return legacyDiscoverableCommand(
		"/status",
		"",
		"Show current state, last request, and session statistics",
		"現在状態、直近リクエスト、セッション統計を表示",
		CommandCategorySystem,
		50,
		"/stats",
	)
}

func commandTokens() CommandInfo {
	return legacyDiscoverableCommand("/tokens", "", "Show token usage and context window status", "トークン使用量を表示", CommandCategoryContext, 60)
}

func commandCopy() CommandInfo {
	return legacyAgentTUILocalCommand(
		"/copy",
		"[code] [-n N]",
		"Copy last AI output to clipboard (code=code blocks only, -n=N-th last output)",
		"AI出力をクリップボードにコピー",
		TUILocalActionCopyMouseSelection,
		TUILocalArgBareOnly,
		TUILocalWhenHasMouseSelection,
		CommandCategorySession,
		100,
		nil,
		nil,
	)
}

func commandAttach() CommandInfo {
	return attachmentTUILocalCommand(
		"/attach",
		"<path>",
		"Attach a file or image to the current composer draft (combined limit: up to 12 attachments per draft)",
		"現在の入力ドラフトにファイル/画像を添付",
		101,
	)
}

func commandDetach() CommandInfo {
	return attachmentTUILocalCommand(
		"/detach",
		"<index>",
		"Detach one attachment by index",
		"指定番号の添付を1件外す",
		102,
	)
}

func commandDetachAll() CommandInfo {
	return attachmentTUILocalCommand(
		"/detach-all",
		"",
		"Detach all attachments from the current draft",
		"現在の入力ドラフトの添付をすべて外す",
		103,
	)
}

func commandReview() CommandInfo {
	return tuiRouterBareLocalCommand(
		"/review",
		"Review current changes and find issues",
		"現在の変更をレビュー",
		TUILocalActionOpenReview,
		CommandCategoryReview,
		70,
		CommandLifecyclePreview,
	)
}

func commandCompress() CommandInfo {
	return legacyDiscoverableCommand("/compress", "[N] [-c]", "Compress history (keep recent N, -c: use OpenAI Compact API)", "履歴を圧縮", CommandCategoryContext, 110)
}

func commandUse() CommandInfo {
	return legacyDiscoverableCommand("/use", "<provider> [model]", "Switch provider and optionally model (e.g., /use gemini gemini-2.0-flash-exp)", "プロバイダーを切り替え", CommandCategoryModel, 20)
}

func commandProviders() CommandInfo {
	return legacyDiscoverableCommand("/providers", "", "List available providers and their API key status", "利用可能なプロバイダー一覧", CommandCategoryModel, 30)
}

func commandConfig() CommandInfo {
	return legacyAgentTUILocalCommand(
		"/config",
		"",
		"Edit global config.yaml settings",
		"対話式設定変更",
		TUILocalActionOpenConfig,
		TUILocalArgBareOnly,
		TUILocalWhenNone,
		CommandCategoryConfig,
		90,
		nil,
		configSubCommands(),
	)
}

func configSubCommands() []SubCommand {
	return []SubCommand{
		{Name: "/config show", Description: "Show all settings with diff from defaults"},
		{Name: "/config model <name>", Description: "Change default model"},
	}
}

func commandModel() CommandInfo {
	return legacyDiscoverableCommand("/model", "[name]", "Show current model or switch model without restart", "モデルを表示/切り替え", CommandCategoryModel, 10)
}

func commandInit() CommandInfo {
	return legacyDiscoverableCommand("/init", "", "Create xelyon.yaml project template", "xelyon.yamlテンプレートを作成", CommandCategoryConfig, 180)
}

func commandProject() CommandInfo {
	return tuiRouterBareLocalCommand(
		"/project",
		"Edit project xelyon.yaml interactively",
		"xelyon.yamlを対話式で編集",
		TUILocalActionOpenProject,
		CommandCategoryConfig,
		80,
		"",
	)
}

func commandPlan() CommandInfo {
	return legacyDiscoverableCommand("/plan", "[on|off]", "Toggle Plan Mode (investigation -> plan -> approval; implementation happens on next normal turn)", "Plan Modeを切り替え", CommandCategoryDev, 120)
}

func commandThink() CommandInfo {
	return legacyDiscoverableCommand("/think", "[on|off|level]", "Toggle Extended Thinking mode (level: low/medium/high/xhigh)", "Extended Thinkingを切り替え", CommandCategoryModel, 40)
}

func commandLSP() CommandInfo {
	return classicOnlyHiddenCommand("/lsp", "[status]", "Show LSP server status (running/not started/disabled)", "LSPサーバー状態を表示", CommandCategoryDev, 190)
}

func commandVersion() CommandInfo {
	return legacyHiddenCommand("/version", "", "Show version information", "バージョンを表示", CommandCategorySystem, 920)
}

func commandHelp() CommandInfo {
	return legacyHiddenCommand("/help", "", "Show this help", "ヘルプを表示", CommandCategorySystem, 910)
}

// Tips はTips一覧
var Tips = []string{
	"Just describe what you want in natural language",
	"AI will ask confirmation for dangerous operations",
	"Use Ctrl+C to cancel current operation",
	"Use git to revert file changes when needed",
}
