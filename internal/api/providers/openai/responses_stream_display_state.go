package openai

import "github.com/susugadx/xelyon-cli/internal/ui"

func (s *responsesStreamState) showFunctionCallSpinner(item *ResponsesItem) {
	if s.spinner == nil || item == nil || item.Type != "function_call" {
		return
	}

	s.spinner.Stop()
	s.spinner.Start(ui.SpinnerMessageForTool(item.Name))
}
