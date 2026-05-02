package commandcatalog

import (
	"fmt"
	"sort"
	"strings"
)

var commandLookup = buildCommandLookup(Commands)

func buildCommandLookup(commands []CommandInfo) map[string]CommandInfo {
	lookup := make(map[string]CommandInfo, len(commands)*2)
	for _, cmd := range commands {
		if _, exists := lookup[cmd.Name]; !exists {
			lookup[cmd.Name] = cmd
		}
		for _, alias := range cmd.Aliases {
			if _, exists := lookup[alias]; !exists {
				lookup[alias] = cmd
			}
		}
	}
	return lookup
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
	cmd, ok := commandLookup[name]
	return cmd, ok
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
