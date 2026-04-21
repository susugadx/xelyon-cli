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

func (s *runnerOutputState) finalizeContextDone(opts RunnerOptions, eventErr error) (string, error) {
	s.stopSpinner()
	partial := s.response()
	if partial != "" {
		s.printTrailingNewlineIfNeeded()
		if opts.WarnOnPartial {
			cancelWarnColor.Fprintln(s.errOut, "\n⚠️  Response interrupted. Partial result returned.")
		}
		if opts.CancelMode == CancelModePartialAsError {
			return partial, s.ctx.Err()
		}
		return partial, nil
	}

	if eventErr != nil {
		return "", eventErr
	}
	return "", s.ctx.Err()
}

func (s *runnerOutputState) finalizeIdleTimeout(idleTimeout time.Duration) (string, error) {
	s.stopSpinner()
	return s.response(), fmt.Errorf("idle timeout: no data received for %v", idleTimeout)
}

func (s *runnerOutputState) finalizeScannerDone(scanErr error) (string, error) {
	s.stopSpinner()
	if scanErr != nil {
		return s.response(), fmt.Errorf("stream reading error: %w", scanErr)
	}
	s.printTrailingNewlineIfNeeded()
	return s.response(), nil
}

func (s *runnerOutputState) finalizeDecodeError(err error) (string, error) {
	s.stopSpinner()
	s.printTrailingNewlineIfNeeded()
	return s.response(), err
}

func (s *runnerOutputState) finalizeHandlerError(err error) (string, error) {
	s.stopSpinner()
	s.printTrailingNewlineIfNeeded()
	return s.response(), err
}

func (s *runnerOutputState) finalizeDone() (string, error) {
	s.stopSpinner()
	s.printTrailingNewlineIfNeeded()
	return s.response(), nil
}
