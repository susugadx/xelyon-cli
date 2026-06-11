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

type assistantSessionMode int

const (
	assistantSessionNone assistantSessionMode = iota
	assistantSessionRawAPI
	assistantSessionRawText
	assistantSessionDisplayText
)

type assistantAppendOptions struct {
	incrementStats bool
	recordOutput   bool
	sessionMode    assistantSessionMode
}

func rawAssistantResponse(response string) assistantResponse {
	return assistantResponse{
		raw:     response,
		display: response,
	}
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

func (a *Agent) appendAssistantResponse(prepared assistantResponse, opts assistantAppendOptions) api.Message {
	msg := api.Message{
		Role:             "assistant",
		Content:          prepared.raw,
		ReasoningContent: a.getLastReasoningContent(),
	}
	msg.SetOpenAIResponsesInputItems(a.getLastOpenAIResponsesInputItems())
	a.History = append(a.History, msg)

	if opts.incrementStats && a.Stats != nil {
		a.Stats.AssistantMessages++
	}

	if opts.recordOutput {
		a.recordAssistantDisplayOutput(prepared.display)
	}

	switch opts.sessionMode {
	case assistantSessionRawAPI:
		a.appendSessionMessageFromAPI(msg, a.CurrentModel)
	case assistantSessionRawText:
		a.appendSessionMessage("assistant", prepared.raw, a.CurrentModel)
	case assistantSessionDisplayText:
		a.appendSessionMessageFromAPIWithStoredContent(msg, prepared.display, a.CurrentModel)
	}

	return msg
}

func (a *Agent) appendAssistantResponseHistory(prepared assistantResponse) {
	a.appendAssistantResponse(prepared, assistantAppendOptions{
		incrementStats: true,
		recordOutput:   true,
		sessionMode:    assistantSessionDisplayText,
	})
}

func (a *Agent) recordAssistantRawResponse(response string) {
	a.appendAssistantResponse(rawAssistantResponse(response), assistantAppendOptions{})
}

func (a *Agent) recordAssistantAPITurn(response string) {
	a.appendAssistantResponse(rawAssistantResponse(response), assistantAppendOptions{
		incrementStats: true,
		sessionMode:    assistantSessionRawAPI,
	})
}

func (a *Agent) recordAssistantDisplayOutput(display string) {
	a.historyMu.Lock()
	a.lastOutputs = append(a.lastOutputs, display)
	if len(a.lastOutputs) > config.MaxLastOutputs {
		a.lastOutputs = a.lastOutputs[1:]
	}
	a.historyMu.Unlock()
}

func (a *Agent) displayAssistantResponse(prepared assistantResponse) {
	a.emitAssistantResponseNotices(prepared)
	a.printFinalAssistantResponse(prepared.display)
}

func (a *Agent) showAssistantResponse(response string) {
	a.displayAssistantResponse(prepareAssistantResponse(response))
}

func (a *Agent) recordAndShowAssistantResponse(response string) {
	prepared := prepareAssistantResponse(response)
	a.appendAssistantResponseHistory(prepared)
	a.displayAssistantResponse(prepared)
}

func (a *Agent) handleNormalResponse(response string) {
	a.recordAndShowAssistantResponse(response)
}
