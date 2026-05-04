package commandcatalog

import "sort"

type indexedCommand struct {
	catalogIndex int
	cmd          CommandInfo
}

type catalogIndex struct {
	lookup                map[string]CommandInfo
	commandsBySurface     map[CommandSurface][]CommandInfo
	discoverableBySurface map[CommandSurface][]CommandInfo
}

var globalCatalogIndex = buildCatalogIndex(Commands)

func buildCatalogIndex(commands []CommandInfo) catalogIndex {
	if err := ValidateCommands(commands); err != nil {
		panic(err)
	}
	lookup := buildCommandLookup(commands)
	commandsBySurface := buildCommandsBySurface(commands)
	discoverableBySurface := buildDiscoverableBySurface(commands)
	return catalogIndex{
		lookup:                lookup,
		commandsBySurface:     commandsBySurface,
		discoverableBySurface: discoverableBySurface,
	}
}

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

func buildCommandsBySurface(commands []CommandInfo) map[CommandSurface][]CommandInfo {
	bySurface := make(map[CommandSurface][]CommandInfo)
	for _, cmd := range commands {
		for _, surface := range cmd.effectiveSurfaces() {
			bySurface[surface] = append(bySurface[surface], cmd)
		}
	}
	return bySurface
}

func buildDiscoverableBySurface(commands []CommandInfo) map[CommandSurface][]CommandInfo {
	indexedBySurface := make(map[CommandSurface][]indexedCommand)
	for i, cmd := range commands {
		if !cmd.Discoverable {
			continue
		}
		for _, surface := range cmd.effectiveSurfaces() {
			indexedBySurface[surface] = append(indexedBySurface[surface], indexedCommand{catalogIndex: i, cmd: cmd})
		}
	}

	discoverable := make(map[CommandSurface][]CommandInfo, len(indexedBySurface))
	for surface, indexed := range indexedBySurface {
		sort.SliceStable(indexed, func(i, j int) bool {
			left := indexed[i].cmd.EffectiveSortWeight()
			right := indexed[j].cmd.EffectiveSortWeight()
			if left == right {
				return indexed[i].catalogIndex < indexed[j].catalogIndex
			}
			return left < right
		})

		commands := make([]CommandInfo, 0, len(indexed))
		for _, item := range indexed {
			commands = append(commands, item.cmd)
		}
		discoverable[surface] = commands
	}

	return discoverable
}

func (idx catalogIndex) find(name string) (CommandInfo, bool) {
	cmd, ok := idx.lookup[name]
	return cmd, ok
}

func (idx catalogIndex) commandsForSurface(surface CommandSurface) []CommandInfo {
	return copyCommands(idx.commandsBySurface[surface])
}

func (idx catalogIndex) discoverableForSurface(surface CommandSurface) []CommandInfo {
	return copyCommands(idx.discoverableBySurface[surface])
}

func copyCommands(commands []CommandInfo) []CommandInfo {
	if len(commands) == 0 {
		return nil
	}
	out := make([]CommandInfo, len(commands))
	copy(out, commands)
	return out
}
