package gemini

import (
	"context"
	"fmt"
	"io"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type sseDisplayState struct {
	spinner               *ui.Spinner
	out                   io.Writer
	streamAssistantText   bool
	headerPrinted         bool
	contentNewlineEmitted bool
}

func newSSEDisplayState(spinner *ui.Spinner, out io.Writer, streamAssistantText bool) *sseDisplayState {
	return &sseDisplayState{
		spinner:             spinner,
		out:                 out,
		streamAssistantText: streamAssistantText,
	}
}

func (s *sseDisplayState) stopSpinner() {
	if s.spinner == nil {
		return
	}
	s.spinner.Stop()
}

func (s *sseDisplayState) restartSpinner(message string) {
	if s.spinner == nil || message == "" {
		return
	}
	s.spinner.Stop()
	s.spinner.Start(message)
}

func (s *sseDisplayState) ensureHeader(ctx context.Context) {
	if s.headerPrinted {
		return
	}
	s.stopSpinner()
	api.PrintAIHeaderWithContext(ctx)
	s.headerPrinted = true
}

func (s *sseDisplayState) printText(text string) {
	if !s.streamAssistantText || text == "" {
		return
	}
	_, _ = fmt.Fprint(s.out, text)
	s.contentNewlineEmitted = false
}

func (s *sseDisplayState) showToolSpinner(toolName string) {
	if s.spinner == nil {
		return
	}
	s.spinner.Stop()
	if s.streamAssistantText && s.headerPrinted && !s.contentNewlineEmitted {
		_, _ = fmt.Fprintln(s.out)
		s.contentNewlineEmitted = true
	}
	s.spinner.Start(ui.SpinnerMessageForTool(toolName))
}

func (s *sseDisplayState) printTrailingNewlineIfNeeded() {
	if !s.streamAssistantText || s.contentNewlineEmitted {
		return
	}
	_, _ = fmt.Fprintln(s.out)
}
