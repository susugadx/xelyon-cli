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
	errOut                io.Writer
	streamAssistantText   bool
	headerPrinted         bool
	contentNewlineEmitted bool
}

func newSSEDisplayState(spinner *ui.Spinner, out io.Writer, errOut io.Writer, streamAssistantText bool) *sseDisplayState {
	return &sseDisplayState{
		spinner:             spinner,
		out:                 out,
		errOut:              errOut,
		streamAssistantText: streamAssistantText,
	}
}

func (s *sseDisplayState) stopSpinner() {
	if s == nil || s.spinner == nil {
		return
	}
	s.spinner.Stop()
}

func (s *sseDisplayState) restartSpinner(message string) {
	if s == nil || s.spinner == nil || message == "" {
		return
	}
	s.spinner.Stop()
	s.spinner.Start(message)
}

func (s *sseDisplayState) ensureHeader(ctx context.Context) {
	if s == nil {
		return
	}
	if s.headerPrinted {
		return
	}
	s.stopSpinner()
	api.PrintAIHeaderWithContext(ctx)
	s.headerPrinted = true
}

func (s *sseDisplayState) printText(text string) {
	if s == nil {
		return
	}
	if !s.streamAssistantText || text == "" {
		return
	}
	_, _ = fmt.Fprint(s.out, text)
	s.contentNewlineEmitted = false
}

func (s *sseDisplayState) showToolSpinner(toolName string) {
	if s == nil || s.spinner == nil {
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
	if s == nil {
		return
	}
	if !s.streamAssistantText || s.contentNewlineEmitted {
		return
	}
	_, _ = fmt.Fprintln(s.out)
}

func (s *sseDisplayState) warnFunctionCallRescue(count int) {
	if s == nil || s.errOut == nil {
		return
	}
	_, _ = fmt.Fprintf(s.errOut, "⚠️  FC rescue: %d tool call(s) extracted from text response\n", count)
}
