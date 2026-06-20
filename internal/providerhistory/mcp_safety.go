package providerhistory

import (
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

const mcpBareSecretKeyPattern = `password|passwd|secret|api[_-]?key|api\s+key|authorization|auth[_-]?token|access[_-]?token|refresh[_-]?token|id[_-]?token|session[_-]?token|client[_-]?secret|private[_-]?key|jwt|token`

var mcpBareSecretPattern = regexp.MustCompile(`(?i)["']?\b(` + mcpBareSecretKeyPattern + `)\b["']?\s*[:=]\s*["']?(bearer\s+)?[^\s"'\]\}),;]+`)

const (
	// MCPSensitiveOrPrivateResultKeepReason は private-looking MCP result を artifact 化しない理由。
	MCPSensitiveOrPrivateResultKeepReason = "mcp_sensitive_or_private_result_keep"
)

// MCPRawOutputArtifactOmitReason は MCP tool result を normal raw output store に保存しない理由を返す。
func MCPRawOutputArtifactOmitReason(content string) string {
	if rawoutputs.LooksSensitiveContent(content) || providerHistoryMCPLooksBareSecret(content) {
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

func providerHistoryMCPLooksBareSecret(content string) bool {
	return mcpBareSecretPattern.MatchString(content)
}
