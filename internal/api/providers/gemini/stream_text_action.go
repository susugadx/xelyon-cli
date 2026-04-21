package gemini

import "strings"

type sseTextAction struct {
	responseText     string
	displayText      string
	rescuedToolJSONs []string
	shouldDisplay    bool
}

func (s *sseInterpretState) interpretTextPart(text string) sseTextAction {
	if s.suppressingToolJSON {
		s.updateToolJSONSuppression(text)
		return sseTextAction{responseText: text}
	}

	trimmed := strings.TrimSpace(text)
	if isToolJSONPrefix(trimmed) {
		s.suppressingToolJSON = true
		s.toolJSONDepth = 0
		s.toolJSONInStr = false
		s.updateToolJSONSuppression(text)
		return sseTextAction{responseText: text}
	}

	extracted, remaining := extractCodeBlockToolJSON(text)
	if len(extracted) > 0 {
		action := sseTextAction{
			responseText:     remaining,
			rescuedToolJSONs: extracted,
		}
		if strings.TrimSpace(remaining) != "" {
			action.displayText = remaining
			action.shouldDisplay = true
		}
		return action
	}

	return sseTextAction{
		responseText:  text,
		displayText:   text,
		shouldDisplay: true,
	}
}

func (s *sseInterpretState) updateToolJSONSuppression(text string) {
	updateToolJSONDepth(text, &s.toolJSONDepth, &s.toolJSONInStr)
	if s.toolJSONDepth <= 0 {
		s.suppressingToolJSON = false
	}
}
