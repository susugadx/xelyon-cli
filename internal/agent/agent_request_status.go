package agent

import (
	"fmt"
	"os"
	"strings"
)

func summarizeStatusError(err error) string {
	if err == nil {
		return "Request failed"
	}

	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "Request failed"
	}

	if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
		msg = strings.TrimSpace(msg[:idx])
	}

	const maxReasonLen = 120
	if len(msg) > maxReasonLen {
		msg = msg[:maxReasonLen-3] + "..."
	}

	return msg
}

func cancelDebugEnabled() bool {
	return os.Getenv("XELYON_DEBUG_CANCEL") == "1"
}

func (a *Agent) debugCancelf(format string, args ...any) {
	if a == nil || !cancelDebugEnabled() {
		return
	}
	_, _ = fmt.Fprintf(a.errorOutput(), "[DEBUG Cancel] "+format+"\n", args...)
}

func (a *Agent) cancelActiveRequest(reason string) {
	if a == nil {
		return
	}

	if reason != "" {
		a.lastCancelReason = reason
	}

	if a.cancelFunc == nil {
		a.debugCancelf("cancel requested without active request (reason=%q)", reason)
		return
	}

	a.debugCancelf("canceling active request (reason=%q)", reason)
	a.cancelFunc()
}

func (a *Agent) statusReasonForError(err error) string {
	reason := summarizeStatusError(err)
	if a == nil {
		return reason
	}

	if reason != "Request failed" && strings.TrimSpace(a.lastCancelReason) != "" && strings.Contains(reason, "context canceled") {
		return reason + " [" + a.lastCancelReason + "]"
	}

	return reason
}
