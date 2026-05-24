package gemini

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type sseInterpretState struct {
	display     *sseDisplayState
	errOut      io.Writer
	thinkingMsg string
	debug       bool

	fullResponse        strings.Builder
	functionCalls       []*api.GeminiFunctionCall
	thoughtParts        []map[string]any
	rescuedToolJSONs    []string
	usage               *GeminiUsageMetadata
	billingServiceTier  string
	suppressingToolJSON bool
	toolJSONDepth       int
	toolJSONInStr       bool
	streamStarted       bool
	hadActionableOutput bool
	thinkingRetries     int
}

func newSSEInterpretState(ctx context.Context, spinner *ui.Spinner, thinkingMsg string, debug bool) *sseInterpretState {
	out := api.OutputWriterFromContext(ctx)
	errOut := api.ErrorWriterFromContext(ctx)
	streamAssistantText := api.ShouldStreamAssistantText(ctx)
	return &sseInterpretState{
		display:     newSSEDisplayState(spinner, out, errOut, streamAssistantText),
		errOut:      errOut,
		thinkingMsg: thinkingMsg,
		debug:       debug,
	}
}

func (s *sseInterpretState) response() string {
	return s.fullResponse.String()
}

func (s *sseInterpretState) stopSpinner() {
	if s.display == nil {
		return
	}
	s.display.stopSpinner()
}

func (s *sseInterpretState) debugf(format string, args ...interface{}) {
	if !s.debug {
		return
	}
	fmt.Fprintf(s.errOut, format, args...)
}

func (s *sseInterpretState) handleThinkingTimeout(thinkingTimeout time.Duration) error {
	if !s.hadActionableOutput {
		s.stopSpinner()
		return &ErrThinkingTimeout{Message: fmt.Sprintf("thinking timeout: no Gemini progress or actionable output received for %v", thinkingTimeout)}
	}

	s.thinkingRetries++
	if s.thinkingRetries >= 2 {
		s.stopSpinner()
		return &ErrThinkingTimeout{Message: fmt.Sprintf("thinking timeout: extended thinking exceeded %v with no new output", thinkingTimeout*time.Duration(s.thinkingRetries+1))}
	}
	return nil
}

func (s *sseInterpretState) resetThinkingTimer(timer *time.Timer, timeout time.Duration) {
	resetTimer(timer, timeout)
}

func (s *sseInterpretState) resetThinkingProgress(timer *time.Timer, timeout time.Duration) {
	s.hadActionableOutput = true
	s.thinkingRetries = 0
	s.resetThinkingTimer(timer, timeout)
}

func (s *sseInterpretState) processChunk(ctx context.Context, chunk GeminiFunctionResponse, thinkingTimer *time.Timer, thinkingTimeout time.Duration) {
	if !s.streamStarted {
		s.streamStarted = true
		if s.display != nil {
			s.display.restartSpinner(s.thinkingMsg)
		}
	}

	if chunk.UsageMetadata != nil {
		s.usage = chunk.UsageMetadata
	}
	if len(chunk.Candidates) == 0 {
		return
	}

	for _, part := range chunk.Candidates[0].Content.Parts {
		s.processPart(ctx, part, thinkingTimer, thinkingTimeout)
	}
}

func (s *sseInterpretState) processLine(ctx context.Context, line string, thinkingTimer *time.Timer, thinkingTimeout time.Duration) bool {
	data, handled := parseGeminiSSEDataLine(line)
	if !handled {
		return false
	}

	chunk, err := decodeGeminiSSEChunk(data)
	if err != nil {
		s.debugf("[DEBUG Gemini SSE] Failed to unmarshal chunk: %v\n", err)
		return false
	}

	s.processChunk(ctx, chunk, thinkingTimer, thinkingTimeout)
	return true
}

func (s *sseInterpretState) processPart(ctx context.Context, part GeminiFunctionPart, thinkingTimer *time.Timer, thinkingTimeout time.Duration) {
	action := buildPartAction(part)
	if action.collectThought {
		s.collectThoughtPart(part)
		s.resetThinkingTimer(thinkingTimer, thinkingTimeout)
		return
	}
	if action.collectSignature {
		s.collectSignaturePart(part)
		if action.functionCall != nil {
			s.handleFunctionCall(action.functionCall, action.thoughtSignature, thinkingTimer, thinkingTimeout)
		} else if action.text != "" {
			s.handleTextPart(ctx, action.text, thinkingTimer, thinkingTimeout)
		} else {
			s.resetThinkingTimer(thinkingTimer, thinkingTimeout)
		}
		return
	}

	if action.text != "" {
		s.handleTextPart(ctx, action.text, thinkingTimer, thinkingTimeout)
	}
	if action.functionCall != nil {
		s.handleFunctionCall(action.functionCall, action.thoughtSignature, thinkingTimer, thinkingTimeout)
	}
}

func (s *sseInterpretState) collectThoughtPart(part GeminiFunctionPart) {
	tp := map[string]any{"thought": true}
	if part.Text != "" {
		tp["text"] = part.Text
	}
	if part.ThoughtSignature != "" {
		tp["thought_signature"] = part.ThoughtSignature
	}
	s.thoughtParts = append(s.thoughtParts, tp)

	sig := part.ThoughtSignature
	if len(sig) > 20 {
		sig = sig[:20] + "..."
	}
	s.debugf("[DEBUG Gemini SSE] Collected thought part (text=%d chars, sig=%q)\n", len(part.Text), sig)
}

func (s *sseInterpretState) collectSignaturePart(part GeminiFunctionPart) {
	tp := map[string]any{"thought_signature": part.ThoughtSignature}
	if part.Text != "" {
		tp["text"] = part.Text
	}
	s.thoughtParts = append(s.thoughtParts, tp)

	sig := part.ThoughtSignature
	if len(sig) > 20 {
		sig = sig[:20] + "..."
	}
	s.debugf("[DEBUG Gemini SSE] Collected signature part (text=%d chars, sig=%q, hasFC=%v)\n", len(part.Text), sig, part.FunctionCall != nil)
}

func (s *sseInterpretState) handleTextPart(ctx context.Context, text string, thinkingTimer *time.Timer, thinkingTimeout time.Duration) {
	s.resetThinkingProgress(thinkingTimer, thinkingTimeout)
	textAction := s.interpretTextPart(text)
	s.applyTextAction(ctx, textAction)
}

func (s *sseInterpretState) ensureHeaderPrinted(ctx context.Context) {
	if s.display == nil {
		return
	}
	s.display.ensureHeader(ctx)
}

func (s *sseInterpretState) handleFunctionCall(functionCall *api.GeminiFunctionCall, thoughtSignature string, thinkingTimer *time.Timer, thinkingTimeout time.Duration) {
	s.resetThinkingProgress(thinkingTimer, thinkingTimeout)
	if s.display != nil {
		s.display.showToolSpinner(functionCall.Name)
	}
	functionCall.ThoughtSignature = thoughtSignature
	s.functionCalls = append(s.functionCalls, functionCall)
}

func (s *sseInterpretState) finalize(p *Provider) (string, error) {
	s.stopSpinner()
	return newSSEFinalizeState(s).finalize(p)
}
