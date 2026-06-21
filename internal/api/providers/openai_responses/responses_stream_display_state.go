package openairesponses

import "github.com/susugadx/xelyon-cli/internal/uitoolview"

func (s *responsesStreamState) showFunctionCallSpinner(item *Item) {
	if s.spinner == nil || item == nil || item.Type != "function_call" {
		return
	}

	s.spinner.Stop()
	s.spinner.Start(uitoolview.SpinnerMessageForTool(item.Name))
}
