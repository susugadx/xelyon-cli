package claudestream

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type runnerOutputState struct {
	ctx                   context.Context
	spinner               *ui.Spinner
	out                   io.Writer
	errOut                io.Writer
	streamAssistantText   bool
	firstChunk            bool
	contentNewlineEmitted bool

	fullResponse strings.Builder
}

func newRunnerOutputState(ctx context.Context, spinner *ui.Spinner) *runnerOutputState {
	return &runnerOutputState{
		ctx:                 ctx,
		spinner:             spinner,
		out:                 api.OutputWriterFromContext(ctx),
		errOut:              api.ErrorWriterFromContext(ctx),
		streamAssistantText: api.ShouldStreamAssistantText(ctx),
		firstChunk:          true,
	}
}

func (s *runnerOutputState) response() string {
	return s.fullResponse.String()
}

func (s *runnerOutputState) stopSpinner() {
	if s.spinner == nil {
		return
	}
	s.spinner.Stop()
}

func (s *runnerOutputState) syncNewlineForActiveSpinner() {
	if !s.streamAssistantText || s.firstChunk || s.contentNewlineEmitted || s.spinner == nil || !s.spinner.IsActive() {
		return
	}
	_, _ = fmt.Fprintln(s.out)
	s.contentNewlineEmitted = true
}

func (s *runnerOutputState) printTrailingNewlineIfNeeded() {
	if !s.streamAssistantText || s.firstChunk || s.contentNewlineEmitted {
		return
	}
	_, _ = fmt.Fprintln(s.out)
}

func (s *runnerOutputState) appendTextDelta(textDelta string) {
	if textDelta == "" {
		return
	}

	if s.firstChunk {
		s.stopSpinner()
		s.firstChunk = false
		api.PrintAIHeaderWithContext(s.ctx)
	}
	if s.streamAssistantText {
		_, _ = fmt.Fprint(s.out, textDelta)
		s.contentNewlineEmitted = false
	}
	s.fullResponse.WriteString(textDelta)
}

func (s *runnerOutputState) finalizeWith(err error, printTrailingNewline bool) (string, error) {
	return s.finalizeWithResponse(s.currentResponse(), err, printTrailingNewline)
}

func (s *runnerOutputState) currentResponse() string {
	return s.response()
}

func (s *runnerOutputState) finalizeWithResponse(response string, err error, printTrailingNewline bool) (string, error) {
	s.stopSpinner()
	if printTrailingNewline {
		s.printTrailingNewlineIfNeeded()
	}
	return response, err
}

func (s *runnerOutputState) finalizeContextDone(opts RunnerOptions, eventErr error) (string, error) {
	policy := newRunnerContextDonePolicy(opts)
	response := s.currentResponse()
	resolution := policy.resolve(response, eventErr, s.ctx.Err())
	if resolution.warnPartial {
		cancelWarnColor.Fprintln(s.errOut, "\n⚠️  Response interrupted. Partial result returned.")
	}
	return s.finalizeWithResponse(response, resolution.err, resolution.printTrailingNewline)
}

func (s *runnerOutputState) finalizeIdleTimeout(idleTimeout time.Duration) (string, error) {
	return s.finalizeWith(fmt.Errorf("idle timeout: no data received for %v", idleTimeout), false)
}

func (s *runnerOutputState) finalizeScannerDone(scanErr error) (string, error) {
	if scanErr != nil {
		return s.finalizeWith(fmt.Errorf("stream reading error: %w", scanErr), false)
	}
	return s.finalizeWith(nil, true)
}

func (s *runnerOutputState) finalizeDecodeError(err error) (string, error) {
	return s.finalizeWith(err, true)
}

func (s *runnerOutputState) finalizeHandlerError(err error) (string, error) {
	return s.finalizeWith(err, true)
}

func (s *runnerOutputState) finalizeDone() (string, error) {
	return s.finalizeWith(nil, true)
}
