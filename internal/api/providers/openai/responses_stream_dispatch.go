package openai

type responsesChunkActionResult struct {
	textDelta string
	done      bool
	err       error
}

type responsesChunkAction func(*responsesStreamState, ResponsesStreamChunk) responsesChunkActionResult

func responsesErrorAction(s *responsesStreamState, chunk ResponsesStreamChunk) responsesChunkActionResult {
	done, err := s.handleErrorEvent(chunk)
	return responsesChunkActionResult{done: done, err: err}
}

func responsesCompletionAction(s *responsesStreamState, chunk ResponsesStreamChunk) responsesChunkActionResult {
	s.handleCompletionEvent(chunk)
	return responsesChunkActionResult{done: true}
}

func responsesFunctionCallAddedAction(s *responsesStreamState, chunk ResponsesStreamChunk) responsesChunkActionResult {
	s.handleFunctionCallAdded(chunk.Item)
	return responsesChunkActionResult{}
}

func responsesFunctionCallArgumentsDeltaAction(s *responsesStreamState, chunk ResponsesStreamChunk) responsesChunkActionResult {
	s.handleFunctionCallArgumentsDelta(chunk)
	return responsesChunkActionResult{}
}

func responsesFunctionCallArgumentsDoneAction(s *responsesStreamState, chunk ResponsesStreamChunk) responsesChunkActionResult {
	s.handleFunctionCallArgumentsDone(chunk)
	return responsesChunkActionResult{}
}

var responsesChunkActionTable = map[string]responsesChunkAction{
	"response.created": func(s *responsesStreamState, chunk ResponsesStreamChunk) responsesChunkActionResult {
		s.captureResponseID(chunk)
		return responsesChunkActionResult{}
	},
	"error":                                  responsesErrorAction,
	"response.failed":                        responsesErrorAction,
	"response.output_item.added":             responsesFunctionCallAddedAction,
	"response.function_call_arguments.delta": responsesFunctionCallArgumentsDeltaAction,
	"response.function_call_arguments.done":  responsesFunctionCallArgumentsDoneAction,
	"response.output_text.delta": func(_ *responsesStreamState, chunk ResponsesStreamChunk) responsesChunkActionResult {
		return responsesChunkActionResult{textDelta: chunk.Delta}
	},
	"response.completed": responsesCompletionAction,
	"response.done":      responsesCompletionAction,
}
