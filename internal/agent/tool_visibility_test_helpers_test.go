package agent

import "github.com/susugadx/xelyon-cli/internal/tools"

func toolNameInList(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}

func toolDefinitionNamed(defs []tools.ToolDefinition, name string) bool {
	for _, def := range defs {
		if def.Name == name {
			return true
		}
	}
	return false
}
