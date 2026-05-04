package commandcatalog

import "strings"

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
		return copyCommands(commands)
	}
	matches := make([]CommandInfo, 0, len(commands))
	for _, cmd := range commands {
		if commandMatchesPrefix(cmd, prefix) {
			matches = append(matches, cmd)
		}
	}
	return matches
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
