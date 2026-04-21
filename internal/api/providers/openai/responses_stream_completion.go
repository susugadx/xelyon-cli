package openai

import "github.com/susugadx/xelyon-cli/internal/api"

func (s *responsesStreamState) handleCompletionEvent(chunk ResponsesStreamChunk) {
	s.captureUsage(chunk)
	s.appendFunctionCallsToOutput()
}

func isResponsesCompletionEvent(eventType string) bool {
	return eventType == "response.completed" || eventType == "response.done"
}

func (s *responsesStreamState) captureUsage(chunk ResponsesStreamChunk) {
	var usage *ResponsesUsage
	if chunk.Response != nil && chunk.Response.Usage != nil {
		usage = chunk.Response.Usage
	} else if chunk.Usage != nil {
		usage = chunk.Usage
	}

	if usage == nil {
		s.debugf("[DEBUG OpenAI Responses] %s event but usage is nil\n", chunk.Type)
		return
	}

	cachedTokens := 0
	if usage.InputTokensDetails != nil {
		cachedTokens = usage.InputTokensDetails.CachedTokens
	}
	reasoningTokens := 0
	if usage.OutputTokensDetails != nil {
		reasoningTokens = usage.OutputTokensDetails.ReasoningTokens
	}
	s.lastUsage = &api.Usage{
		InputTokens:       usage.InputTokens,
		OutputTokens:      usage.OutputTokens,
		ThinkingTokens:    reasoningTokens,
		CachedInputTokens: cachedTokens,
	}
	s.debugf("[DEBUG OpenAI Responses] usage received: input=%d, output=%d, cached=%d\n",
		usage.InputTokens, usage.OutputTokens, cachedTokens)
}
