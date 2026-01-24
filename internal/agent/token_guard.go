package agent

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
)

const (
	defaultToolOutputTailLines = 300
	minToolOutputTailLines     = 50
)

// trimToolOutputsForSend returns a copy of history where tool outputs are truncated.
//
// XELYON stores tool results into history as user messages in the form:
//
//	[Tool Result for <tool>]\n<output>
//
// Those outputs can be extremely large (e.g. go test logs) and easily blow up
// the provider context limit. This function keeps only the last N lines.
func trimToolOutputsForSend(history []api.Message, tailLines int) []api.Message {
	if tailLines <= 0 {
		tailLines = defaultToolOutputTailLines
	}
	if tailLines < minToolOutputTailLines {
		tailLines = minToolOutputTailLines
	}

	out := make([]api.Message, 0, len(history))
	for _, msg := range history {
		if msg.Role != "user" {
			out = append(out, msg)
			continue
		}
		if !strings.HasPrefix(msg.Content, "[Tool Result for ") {
			out = append(out, msg)
			continue
		}

		trimmed, changed := token.TruncateToLastLines(msg.Content, tailLines)
		if changed {
			trimmed = fmt.Sprintf("%s\n\n[SYSTEM NOTE] Tool output was truncated to the last %d lines to fit context limits.", trimmed, tailLines)
		}

		out = append(out, api.Message{Role: msg.Role, Content: trimmed})
	}
	return out
}
