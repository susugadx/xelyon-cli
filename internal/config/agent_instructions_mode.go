package config

import "strings"

const (
	AgentInstructionProjectModeOff      = "off"
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
		return AgentInstructionProjectModeFallback
	}
	return normalized
}
