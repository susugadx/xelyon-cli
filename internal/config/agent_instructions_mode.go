package config

import "strings"

const (
	AgentInstructionProjectModeOff = "off"
	// AgentInstructionProjectModeFallback は互換用に受け付ける deprecated alias。
	// AGENTS-first 方針では always と同じく project guidance を読み込む。
	AgentInstructionProjectModeFallback = "fallback"
	AgentInstructionProjectModeAlways   = "always"
)

func isValidAgentInstructionProjectMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case AgentInstructionProjectModeOff, AgentInstructionProjectModeFallback, AgentInstructionProjectModeAlways:
		return true
	default:
		return false
	}
}

func normalizeAgentInstructionProjectMode(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	if !isValidAgentInstructionProjectMode(normalized) {
		return AgentInstructionProjectModeAlways
	}
	return normalized
}
