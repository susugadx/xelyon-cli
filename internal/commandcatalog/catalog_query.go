package commandcatalog

import (
	"sort"
	"strings"
)

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
