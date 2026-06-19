package agent

import (
	"reflect"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	promptnormal "github.com/susugadx/xelyon-cli/internal/prompt/normal"
)

func (a *Agent) persistableHistoryForCompression() []api.Message {
	if a == nil {
		return nil
	}

	runtimeHistory := stripNormalModePromptFromHistory(a.History)
	if a.session == nil {
		return runtimeHistory
	}

	sessionHistory := a.session.ToAPIMessages()
	if historiesMatchForCompressionPersistence(runtimeHistory, sessionHistory) {
		return sessionHistory
	}
	return runtimeHistory
}

func stripNormalModePromptFromHistory(history []api.Message) []api.Message {
	out := make([]api.Message, len(history))
	copy(out, history)
	for i := range out {
		if out[i].Role == "user" {
			// 過去 session / 旧 runtime が user message へ付けた suffix の互換処理。
			// 新規 Normal mode request は mode directive を system prompt 側に置く。
			out[i].Content = strings.TrimSuffix(out[i].Content, promptnormal.NormalModePrompt)
		}
	}
	return out
}

func historiesMatchForCompressionPersistence(runtimeHistory, sessionHistory []api.Message) bool {
	if len(runtimeHistory) != len(sessionHistory) {
		return false
	}
	for i := range runtimeHistory {
		if !messagesMatchForCompressionPersistence(runtimeHistory[i], sessionHistory[i]) {
			return false
		}
	}
	return true
}

func messagesMatchForCompressionPersistence(runtimeMsg, sessionMsg api.Message) bool {
	return runtimeMsg.Role == sessionMsg.Role &&
		runtimeMsg.Content == sessionMsg.Content &&
		runtimeMsg.ToolCallID == sessionMsg.ToolCallID &&
		runtimeMsg.ToolName == sessionMsg.ToolName &&
		reflect.DeepEqual(runtimeMsg.ToolCalls, sessionMsg.ToolCalls)
}
