package openairesponses

type responsesChunkActionResult struct {
	textDelta string
	done      bool
	err       error
}

type responsesChunkAction func(*responsesStreamState, StreamChunk) responsesChunkActionResult

func responsesErrorAction(s *responsesStreamState, chunk StreamChunk) responsesChunkActionResult {
	done, err := s.handleErrorEvent(chunk)
	return responsesChunkActionResult{done: done, err: err}
}

func responsesCompletionAction(s *responsesStreamState, chunk StreamChunk) responsesChunkActionResult {
	s.handleCompletionEvent(chunk)
	return responsesChunkActionResult{done: true}
}

func responsesFunctionCallAddedAction(s *responsesStreamState, chunk StreamChunk) responsesChunkActionResult {
	s.showFunctionCallSpinner(chunk.Item)
	s.handleOutputItemAdded(chunk)
	return responsesChunkActionResult{}
}

func responsesOutputItemDoneAction(s *responsesStreamState, chunk StreamChunk) responsesChunkActionResult {
	s.handleOutputItemDone(chunk)
	return responsesChunkActionResult{}
}

func responsesFunctionCallArgumentsDeltaAction(s *responsesStreamState, chunk StreamChunk) responsesChunkActionResult {
	s.handleFunctionCallArgumentsDelta(chunk)
	return responsesChunkActionResult{}
}

func responsesFunctionCallArgumentsDoneAction(s *responsesStreamState, chunk StreamChunk) responsesChunkActionResult {
	s.handleFunctionCallArgumentsDone(chunk)
	return responsesChunkActionResult{}
}

func responsesCreatedAction(s *responsesStreamState, chunk StreamChunk) responsesChunkActionResult {
	s.captureResponseID(chunk)
	return responsesChunkActionResult{}
}

func responsesTextDeltaAction(s *responsesStreamState, chunk StreamChunk) responsesChunkActionResult {
	s.handleTextDelta(chunk)
	return responsesChunkActionResult{textDelta: chunk.Delta}
}

var responsesChunkActionTable = map[string]responsesChunkAction{
	"response.created":                       responsesCreatedAction,
	"error":                                  responsesErrorAction,
	"response.failed":                        responsesErrorAction,
	"response.output_item.added":             responsesFunctionCallAddedAction,
	"response.output_item.done":              responsesOutputItemDoneAction,
	"response.function_call_arguments.delta": responsesFunctionCallArgumentsDeltaAction,
	"response.function_call_arguments.done":  responsesFunctionCallArgumentsDoneAction,
	"response.output_text.delta":             responsesTextDeltaAction,
	"response.completed":                     responsesCompletionAction,
	"response.done":                          responsesCompletionAction,
}
