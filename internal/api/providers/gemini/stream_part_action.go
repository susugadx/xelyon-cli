package gemini

import "github.com/susugadx/xelyon-cli/internal/api"

type ssePartAction struct {
	collectThought   bool
	collectSignature bool
	text             string
	functionCall     *api.GeminiFunctionCall
	thoughtSignature string
}

func buildPartAction(part GeminiFunctionPart) ssePartAction {
	action := ssePartAction{
		thoughtSignature: part.ThoughtSignature,
		functionCall:     part.FunctionCall,
	}

	if part.Thought {
		action.collectThought = true
		return action
	}
	if part.ThoughtSignature != "" {
		action.collectSignature = true
		if part.FunctionCall == nil {
			action.text = part.Text
		}
		return action
	}
	if part.Text != "" {
		action.text = part.Text
	}
	return action
}
