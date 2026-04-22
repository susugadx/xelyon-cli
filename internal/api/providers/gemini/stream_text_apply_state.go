package gemini

import "context"

func (s *sseInterpretState) applyTextAction(ctx context.Context, action sseTextAction) {
	s.collectRescuedToolJSONs(action.rescuedToolJSONs)
	s.displayTextAction(ctx, action)
	s.appendResponseText(action.responseText)
}

func (s *sseInterpretState) collectRescuedToolJSONs(toolJSONs []string) {
	if len(toolJSONs) == 0 {
		return
	}
	s.rescuedToolJSONs = append(s.rescuedToolJSONs, toolJSONs...)
}

func (s *sseInterpretState) displayTextAction(ctx context.Context, action sseTextAction) {
	if !action.shouldDisplay || s.display == nil {
		return
	}
	s.ensureHeaderPrinted(ctx)
	s.display.printText(action.displayText)
}

func (s *sseInterpretState) appendResponseText(text string) {
	s.fullResponse.WriteString(text)
}
