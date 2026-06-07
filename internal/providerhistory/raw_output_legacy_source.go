package providerhistory

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/rawoutputs"
)

func providerHistoryLegacyRawOutputExactSourceID(sessionID string, surface rawoutputs.Surface, source rawoutputs.SourceMetadata, content string) (string, bool) {
	sessionID = strings.TrimSpace(sessionID)
	surfaceName := strings.TrimSpace(string(surface))
	toolName := strings.TrimSpace(source.ToolName)
	toolCallID := strings.TrimSpace(source.ToolCallID)
	contentHash := commandHash(content)
	ambiguous := sessionID == "" ||
		surfaceName == "" ||
		toolName == "" ||
		toolCallID == "" ||
		source.HistoryIndex < 0 ||
		strings.TrimSpace(contentHash) == ""
	if ambiguous {
		return "", true
	}
	parts := []string{
		"legacy_raw_output",
		sessionID,
		surfaceName,
		fmt.Sprintf("history:%d", source.HistoryIndex),
		toolName,
		toolCallID,
		strings.TrimSpace(source.EventID),
		strings.TrimSpace(source.CommandHash),
		contentHash,
	}
	return strings.Join(parts, "\x00"), false
}
