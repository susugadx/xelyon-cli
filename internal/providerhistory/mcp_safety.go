package providerhistory

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

const (
	// MCPSensitiveOrPrivateResultKeepReason は private-looking MCP result を artifact 化しない理由。
	MCPSensitiveOrPrivateResultKeepReason = "mcp_sensitive_or_private_result_keep"
)

// MCPRawOutputArtifactOmitReason は MCP tool result を normal raw output store に保存しない理由を返す。
func MCPRawOutputArtifactOmitReason(content string) string {
	if rawoutputs.LooksSensitiveContent(content) || providerHistoryLooksBareSecret(content) {
		return string(rawoutputs.ReasonSensitiveArtifactForbidden)
	}
	if providerHistoryMCPLooksPrivate(content) {
		return MCPSensitiveOrPrivateResultKeepReason
	}
	return ""
}

// MCPRawOutputArtifactOmitReasonAllowsRuntimeExcerpt は artifact 化しない MCP result でも runtime placeholder に bounded excerpt を残せるかを返す。
func MCPRawOutputArtifactOmitReasonAllowsRuntimeExcerpt(reason string) bool {
	switch strings.TrimSpace(reason) {
	case string(rawoutputs.ReasonSensitiveArtifactForbidden):
		return false
	default:
		return true
	}
}

func providerHistoryMCPLooksPrivate(content string) bool {
	lower := strings.ToLower(content)
	for _, marker := range []string{
		"customer",
		"email",
		"private",
		"issue body",
		"message body",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
