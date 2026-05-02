package commandcatalog

import "strings"

// CommandsForSurface は指定 surface で利用可能な command を catalog 順で返す。
func CommandsForSurface(surface CommandSurface) []CommandInfo {
	return globalCatalogIndex.commandsForSurface(surface)
}

// Find は command 名または alias に一致する command を返す。
func Find(name string) (CommandInfo, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return CommandInfo{}, false
	}
	return globalCatalogIndex.find(name)
}

// DiscoverableCommandsForSurface は指定 surface の候補表示対象 command を候補順で返す。
func DiscoverableCommandsForSurface(surface CommandSurface) []CommandInfo {
	return globalCatalogIndex.discoverableForSurface(surface)
}
