package ui

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func normalizeLSPServerLanguage(raw string) string {
	return strings.TrimSpace(raw)
}

func withAddedLSPServer(existing map[string]config.LSPServerConfig, lang, command string) map[string]config.LSPServerConfig {
	updated := cloneLSPServerConfigs(existing)
	updated[lang] = config.LSPServerConfig{Command: strings.TrimSpace(command)}
	return updated
}

func cloneLSPServerConfigs(existing map[string]config.LSPServerConfig) map[string]config.LSPServerConfig {
	if len(existing) == 0 {
		return make(map[string]config.LSPServerConfig)
	}

	copied := make(map[string]config.LSPServerConfig, len(existing))
	for key, value := range existing {
		copied[key] = value
	}
	return copied
}

func selectLSPServerByInput(input string, servers []string) (string, bool) {
	idx, ok := parseConfigEditorIndex(input, len(servers))
	if !ok {
		return "", false
	}
	return servers[idx], true
}

func updateLSPServerCommandValue(current config.LSPServerConfig, rawCommand string) (config.LSPServerConfig, bool) {
	command := strings.TrimSpace(rawCommand)
	if command == "" {
		return current, false
	}
	current.Command = command
	return current, true
}

func toggleLSPServerDisabledValue(current config.LSPServerConfig) config.LSPServerConfig {
	current.Disabled = !current.Disabled
	return current
}
