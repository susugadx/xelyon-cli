package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type assistantResponse struct {
	raw                 string
	display             string
	hasCompactionNotice bool
}

func prepareAssistantResponse(response string) assistantResponse {
	prepared := assistantResponse{
		raw:     response,
		display: response,
	}

	if !strings.Contains(response, "[COMPACTION]") {
		return prepared
	}

	startIdx := strings.Index(response, "[COMPACTION]")
	endIdx := strings.Index(response, "[/COMPACTION]")
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return prepared
	}

	prepared.hasCompactionNotice = true
	prepared.display = strings.TrimSpace(response[:startIdx] + response[endIdx+len("[/COMPACTION]"):])
	return prepared
}

func (a *Agent) emitAssistantResponseNotices(prepared assistantResponse) {
	if !prepared.hasCompactionNotice {
		return
	}
	cyan.Fprintln(a.output(), "📦 Context compacted by Claude")
}

func (a *Agent) appendAssistantResponseHistory(prepared assistantResponse) {
	a.History = append(a.History, api.Message{
		Role:             "assistant",
		Content:          prepared.raw,
		ReasoningContent: a.getLastReasoningContent(),
	})

	if a.Stats != nil {
		a.Stats.AssistantMessages++
	}

	a.recordAssistantDisplayOutput(prepared.display)

	if a.session != nil {
		a.appendSessionMessage("assistant", prepared.display, a.CurrentModel)
	}
}

func (a *Agent) recordAssistantDisplayOutput(display string) {
	a.historyMu.Lock()
	a.lastOutputs = append(a.lastOutputs, display)
	if len(a.lastOutputs) > config.MaxLastOutputs {
		a.lastOutputs = a.lastOutputs[1:]
	}
	a.historyMu.Unlock()
}

func (a *Agent) handleNormalResponse(response string) {
	prepared := prepareAssistantResponse(response)
	a.emitAssistantResponseNotices(prepared)
	a.appendAssistantResponseHistory(prepared)
	a.printFinalAssistantResponse(prepared.display)
}
