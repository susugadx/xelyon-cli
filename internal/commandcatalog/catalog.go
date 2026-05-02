package commandcatalog

import (
	"fmt"
	"sort"
	"strings"
)

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

// Commands はコマンド一覧
var Commands = []CommandInfo{
	{
		Name:          "/exit",
		Aliases:       []string{"/quit", "/q"},
		Description:   "Exit the CLI",
		DescriptionJP: "CLIを終了",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategorySystem,
		Discoverable:  true,
		SortWeight:    900,
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
		Name:          "/copy",
		Args:          "[code] [-n N]",
		Description:   "Copy last AI output to clipboard (code=code blocks only, -n=N-th last output)",
		DescriptionJP: "AI出力をクリップボードにコピー",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategorySession,
		Discoverable:  true,
		SortWeight:    100,
	},
	{
		Name:          "/attach",
		Args:          "<path>",
		Description:   "Attach a file or image to the current composer draft",
		DescriptionJP: "現在の入力ドラフトにファイル/画像を添付",
		Surfaces:      []CommandSurface{CommandSurfaceTUI},
		Owner:         CommandOwnerTUIRouter,
		Category:      CommandCategorySession,
		Discoverable:  true,
		SortWeight:    101,
	},
	{
		Name:          "/detach",
		Args:          "<index>",
		Description:   "Detach one attachment by index",
		DescriptionJP: "指定番号の添付を1件外す",
		Surfaces:      []CommandSurface{CommandSurfaceTUI},
		Owner:         CommandOwnerTUIRouter,
		Category:      CommandCategorySession,
		Discoverable:  true,
		SortWeight:    102,
	},
	{
		Name:          "/detach-all",
		Description:   "Detach all attachments from the current draft",
		DescriptionJP: "現在の入力ドラフトの添付をすべて外す",
		Surfaces:      []CommandSurface{CommandSurfaceTUI},
		Owner:         CommandOwnerTUIRouter,
		Category:      CommandCategorySession,
		Discoverable:  true,
		SortWeight:    103,
	},
	{
		Name:          "/review",
		Description:   "Review current changes and find issues",
		DescriptionJP: "現在の変更をレビュー",
		Surfaces:      []CommandSurface{CommandSurfaceTUI},
		Owner:         CommandOwnerTUIRouter,
		Lifecycle:     CommandLifecyclePreview,
		Category:      CommandCategoryReview,
		Discoverable:  true,
		SortWeight:    70,
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
		Name:          "/config",
		Description:   "Edit global config.yaml settings",
		DescriptionJP: "対話式設定変更",
		Surfaces:      legacyFallbackSurfaces(),
		Category:      CommandCategoryConfig,
		Discoverable:  true,
		SortWeight:    90,
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
		Name:          "/project",
		Description:   "Edit project xelyon.yaml interactively",
		DescriptionJP: "xelyon.yamlを対話式で編集",
		Surfaces:      []CommandSurface{CommandSurfaceTUI},
		Owner:         CommandOwnerTUIRouter,
		Category:      CommandCategoryConfig,
		Discoverable:  true,
		SortWeight:    80,
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

// RenderCommandsText は全 surface 向けの help 表示用コマンド一覧を返す。
func RenderCommandsText() string {
	return renderCommandsText(Commands)
}

// RenderCommandsTextForSurface は指定 surface の help 表示用コマンド一覧を返す。
func RenderCommandsTextForSurface(surface CommandSurface) string {
	return renderCommandsText(CommandsForSurface(surface))
}

func renderCommandsText(commands []CommandInfo) string {
	var b strings.Builder
	b.WriteString("Commands:\n")
	for _, cmd := range commands {
		name := cmd.Name
		if len(cmd.Aliases) > 0 {
			name += ", " + strings.Join(cmd.Aliases, ", ")
		}
		if cmd.Args != "" {
			name += " " + cmd.Args
		}
		fmt.Fprintf(&b, "  %-25s - %s\n", name, cmd.Description)
		for _, sub := range cmd.SubCommands {
			fmt.Fprintf(&b, "                            %s - %s\n", sub.Name, sub.Description)
		}
	}
	return b.String()
}

// RenderTipsText は help 表示用の Tips 一覧を返す。
func RenderTipsText() string {
	var b strings.Builder
	b.WriteString("Tips:\n")
	for _, tip := range Tips {
		fmt.Fprintf(&b, "  - %s\n", tip)
	}
	return b.String()
}

// MatchPrefix は prefix に一致するコマンドを catalog 順で返す。
func MatchPrefix(prefix string) []CommandInfo {
	return matchPrefixInCommands(prefix, Commands)
}

// MatchPrefixForSurface は prefix と surface に一致するコマンドを catalog 順で返す。
func MatchPrefixForSurface(prefix string, surface CommandSurface) []CommandInfo {
	return matchPrefixInCommands(prefix, CommandsForSurface(surface))
}

// MatchDiscoverablePrefixForSurface は prefix と surface に一致する discoverable command を候補順で返す。
func MatchDiscoverablePrefixForSurface(prefix string, surface CommandSurface) []CommandInfo {
	return matchPrefixInCommands(prefix, DiscoverableCommandsForSurface(surface))
}

func matchPrefixInCommands(prefix string, commands []CommandInfo) []CommandInfo {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return append([]CommandInfo(nil), commands...)
	}
	matches := make([]CommandInfo, 0, len(commands))
	for _, cmd := range commands {
		if commandMatchesPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}
	return matches
}

// CommandsForSurface は指定 surface で利用可能な command を catalog 順で返す。
func CommandsForSurface(surface CommandSurface) []CommandInfo {
	filtered := make([]CommandInfo, 0, len(Commands))
	for _, cmd := range Commands {
		if cmd.SupportsSurface(surface) {
			filtered = append(filtered, cmd)
		}
	}
	return filtered
}

// Find は command 名または alias に一致する command を返す。
func Find(name string) (CommandInfo, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CommandInfo{}, false
	}
	for _, cmd := range Commands {
		if cmd.Name == name {
			return cmd, true
		}
		for _, alias := range cmd.Aliases {
			if alias == name {
				return cmd, true
			}
		}
	}
	return CommandInfo{}, false
}

// DiscoverableCommandsForSurface は指定 surface の候補表示対象 command を候補順で返す。
func DiscoverableCommandsForSurface(surface CommandSurface) []CommandInfo {
	type indexedCommand struct {
		index int
		cmd   CommandInfo
	}
	indexed := make([]indexedCommand, 0, len(Commands))
	for i, cmd := range Commands {
		if cmd.Discoverable && cmd.SupportsSurface(surface) {
			indexed = append(indexed, indexedCommand{index: i, cmd: cmd})
		}
	}
	sort.SliceStable(indexed, func(i, j int) bool {
		left := indexed[i].cmd.EffectiveSortWeight()
		right := indexed[j].cmd.EffectiveSortWeight()
		if left == right {
			return indexed[i].index < indexed[j].index
		}
		return left < right
	})
	result := make([]CommandInfo, 0, len(indexed))
	for _, item := range indexed {
		result = append(result, item.cmd)
	}
	return result
}

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

func commandMatchesPrefix(cmd CommandInfo, prefix string) bool {
	if strings.HasPrefix(cmd.Name, prefix) {
		return true
	}
	for _, alias := range cmd.Aliases {
		if strings.HasPrefix(alias, prefix) {
			return true
		}
	}
	for _, sub := range cmd.SubCommands {
		if strings.HasPrefix(sub.Name, prefix) {
			return true
		}
	}
	return false
}
