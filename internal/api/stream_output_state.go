package api

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

type streamOutputState struct {
	ctx                 context.Context
	spinner             *ui.Spinner
	out                 io.Writer
	streamAssistantText bool

	fullResponse strings.Builder
	firstChunk   bool

	inToolJSON bool
	jsonDepth  int
	inString   bool
	escaped    bool

	pendingChunk          string
	contentNewlineEmitted bool
}

func newStreamOutputState(ctx context.Context, spinner *ui.Spinner) *streamOutputState {
	return &streamOutputState{
		ctx:                 ctx,
		spinner:             spinner,
		out:                 outputWriterFromContext(ctx),
		streamAssistantText: ShouldStreamAssistantText(ctx),
		firstChunk:          true,
	}
}

func (s *streamOutputState) response() string {
	return s.fullResponse.String()
}

func (s *streamOutputState) stopSpinner() {
	if s.spinner == nil {
		return
	}
	s.spinner.Stop()
}

func (s *streamOutputState) processLine(line string, parser StreamParser) (bool, error) {
	if line == "" {
		return false, nil
	}

	content, done, err := parser(line)
	s.syncNewlineForParserSpinner()
	if err != nil {
		return false, err
	}
	if done {
		return true, nil
	}
	if content == "" {
		return false, nil
	}

	s.appendContent(content)
	return false, nil
}

func (s *streamOutputState) syncNewlineForParserSpinner() {
	if !s.streamAssistantText || s.firstChunk || s.contentNewlineEmitted || s.spinner == nil || !s.spinner.IsActive() {
		return
	}
	_, _ = fmt.Fprintln(s.out)
	s.contentNewlineEmitted = true
}

func (s *streamOutputState) appendContent(content string) {
	// fullResponse には表示フィルタ前の生コンテンツを保持する。
	s.fullResponse.WriteString(content)

	if s.pendingChunk != "" {
		content = s.pendingChunk + content
		s.pendingChunk = ""
	}

	if !s.inToolJSON {
		if prefixLen := matchesPatternPrefix(content); prefixLen > 0 {
			s.pendingChunk = content[len(content)-prefixLen:]
			content = content[:len(content)-prefixLen]
		}
	}

	displayContent := filterToolJSON(content, &s.inToolJSON, &s.jsonDepth, &s.inString, &s.escaped)
	s.startToolSpinnerIfNeeded()

	if s.firstChunk {
		s.stopSpinner()
		s.firstChunk = false
		PrintAIHeaderWithContext(s.ctx)
	}
	if s.streamAssistantText && displayContent != "" {
		_, _ = fmt.Fprint(s.out, displayContent)
	}
}

func (s *streamOutputState) startToolSpinnerIfNeeded() {
	if !s.inToolJSON || s.spinner == nil || s.spinner.IsActive() {
		return
	}
	if s.streamAssistantText && !s.firstChunk && !s.contentNewlineEmitted {
		_, _ = fmt.Fprintln(s.out)
		s.contentNewlineEmitted = true
	}

	msg := "Preparing..."
	responseStr := s.response()
	if strings.Contains(responseStr, "write_file") {
		msg = ui.SpinnerMessageForTool("write_file")
	} else if strings.Contains(responseStr, "str_replace") {
		msg = ui.SpinnerMessageForTool("str_replace")
	}
	s.spinner.Start(msg)
}

func (s *streamOutputState) finalizeDone() (string, error) {
	s.stopSpinner()
	if s.streamAssistantText && s.pendingChunk != "" {
		_, _ = fmt.Fprint(s.out, s.pendingChunk)
		s.contentNewlineEmitted = false
	}
	s.printTrailingNewlineIfNeeded()
	return s.response(), nil
}

func (s *streamOutputState) finalizeParserError(err error) (string, error) {
	s.stopSpinner()
	s.printTrailingNewlineIfNeeded()
	return s.response(), err
}

func (s *streamOutputState) finalizeScannerDone(scanErr error) (string, error) {
	s.stopSpinner()
	if scanErr != nil {
		return s.response(), fmt.Errorf("scanner error: %w", scanErr)
	}
	s.printTrailingNewlineIfNeeded()
	return s.response(), nil
}

func (s *streamOutputState) finalizeContextDone(errOut io.Writer, ctxErr error) (string, error) {
	s.stopSpinner()
	partialResponse := s.response()
	if partialResponse != "" {
		s.printTrailingNewlineIfNeeded()
		yellow.Fprintln(errOut, "\n⚠️  Response interrupted. Partial result returned.")
		return partialResponse, nil
	}
	if ctxErr != nil {
		return "", ctxErr
	}
	return "", context.Canceled
}

func (s *streamOutputState) printTrailingNewlineIfNeeded() {
	if !s.streamAssistantText || s.firstChunk || s.contentNewlineEmitted {
		return
	}
	_, _ = fmt.Fprintln(s.out)
}
