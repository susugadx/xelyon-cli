package configedit

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func normalizeLSPServerLanguage(raw string) string {
	return strings.TrimSpace(raw)
}

// NormalizeLSPServerLanguage は LSP server language key の入力を正規化する。
func NormalizeLSPServerLanguage(raw string) string {
	return normalizeLSPServerLanguage(raw)
}

func withAddedLSPServer(existing map[string]config.LSPServerConfig, lang, command string) map[string]config.LSPServerConfig {
	updated := cloneLSPServerConfigs(existing)
	updated[lang] = config.LSPServerConfig{Command: strings.TrimSpace(command)}
	return updated
}

// WithAddedLSPServer は既存 map を破壊せず LSP server 設定を追加した map を返す。
func WithAddedLSPServer(existing map[string]config.LSPServerConfig, lang, command string) map[string]config.LSPServerConfig {
	return withAddedLSPServer(existing, lang, command)
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

// SelectLSPServerByInput は番号入力から LSP server key を選択する。
func SelectLSPServerByInput(input string, servers []string) (string, bool) {
	return selectLSPServerByInput(input, servers)
}

func updateLSPServerCommandValue(current config.LSPServerConfig, rawCommand string) (config.LSPServerConfig, bool) {
	command := strings.TrimSpace(rawCommand)
	if command == "" {
		return current, false
	}
	current.Command = command
	return current, true
}

// UpdateLSPServerCommandValue は空入力を無視し、LSP server command を更新する。
func UpdateLSPServerCommandValue(current config.LSPServerConfig, rawCommand string) (config.LSPServerConfig, bool) {
	return updateLSPServerCommandValue(current, rawCommand)
}

func toggleLSPServerDisabledValue(current config.LSPServerConfig) config.LSPServerConfig {
	current.Disabled = !current.Disabled
	return current
}

// ToggleLSPServerDisabledValue は LSP server の disabled flag を反転する。
func ToggleLSPServerDisabledValue(current config.LSPServerConfig) config.LSPServerConfig {
	return toggleLSPServerDisabledValue(current)
}
