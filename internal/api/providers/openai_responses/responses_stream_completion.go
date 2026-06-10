package openairesponses

func (s *responsesStreamState) handleCompletionEvent(chunk StreamChunk) {
	s.captureUsage(chunk)
	s.appendFunctionCallsToOutput()
}

func (s *responsesStreamState) captureUsage(chunk StreamChunk) {
	var usage *Usage
	if chunk.Response != nil && chunk.Response.Usage != nil {
		usage = chunk.Response.Usage
	} else if chunk.Usage != nil {
		usage = chunk.Usage
	}

	if usage == nil {
		s.debugf("[DEBUG %s Responses] %s event but usage is nil\n", s.debugName, chunk.Type)
		return
	}

	s.lastUsage = responsesUsageToAPIUsage(usage)
	s.debugf("[DEBUG %s Responses] usage received: input=%d, output=%d, cached=%d\n",
		s.debugName,
		usage.InputTokens, usage.OutputTokens, s.lastUsage.CachedInputTokens)
}
