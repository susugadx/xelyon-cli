package gemini

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
)

type sseFinalizeState struct {
	interpret *sseInterpretState
	output    sseFinalizeOutput
}

func newSSEFinalizeState(interpret *sseInterpretState) *sseFinalizeState {
	return &sseFinalizeState{
		interpret: interpret,
		output:    newSSEFinalizeBuilderOutput(&interpret.fullResponse),
	}
}

func (s *sseFinalizeState) finalize(p *Provider) (string, error) {
	s.attachThoughtPartsToFunctionCalls()
	s.appendRescuedToolJSONIfNeeded()
	s.appendUniqueFunctionCalls()
	s.emitUsage(p)
	return s.finalizeOutput()
}

func (s *sseFinalizeState) attachThoughtPartsToFunctionCalls() {
	if len(s.interpret.thoughtParts) == 0 {
		return
	}
	for _, fc := range s.interpret.functionCalls {
		fc.ThoughtParts = s.interpret.thoughtParts
	}
}

func (s *sseFinalizeState) appendRescuedToolJSONIfNeeded() {
	if len(s.interpret.functionCalls) != 0 || len(s.interpret.rescuedToolJSONs) == 0 {
		return
	}

	s.interpret.debugf("[DEBUG Gemini SSE] Rescuing %d tool call(s) from text\n", len(s.interpret.rescuedToolJSONs))
	s.interpret.display.warnFunctionCallRescue(len(s.interpret.rescuedToolJSONs))
	for _, toolJSON := range s.interpret.rescuedToolJSONs {
		s.output.Append(toolJSON)
	}
}

func (s *sseFinalizeState) appendUniqueFunctionCalls() {
	seenTools := make(map[string]bool)
	for _, fc := range s.interpret.functionCalls {
		displayKey := convertFunctionCallToDisplayJSON(fc)
		if seenTools[displayKey] {
			continue
		}
		seenTools[displayKey] = true
		s.output.Append(convertFunctionCallToToolJSON(fc))
	}
}

func (s *sseFinalizeState) emitUsage(p *Provider) {
	if s.interpret.usage == nil || p.usageCallback == nil {
		return
	}

	p.usageCallback(api.Usage{
		InputTokens:       s.interpret.usage.PromptTokenCount,
		OutputTokens:      s.interpret.usage.CandidatesTokenCount,
		ThinkingTokens:    s.interpret.usage.ThoughtsTokenCount,
		CachedInputTokens: s.interpret.usage.CachedContentTokenCount,
	})
}

func (s *sseFinalizeState) finalizeOutput() (string, error) {
	if s.output.Len() == 0 {
		return "", fmt.Errorf("no content in Gemini SSE response (stream ended without generating any text or function calls)")
	}

	s.interpret.display.printTrailingNewlineIfNeeded()
	return s.output.Response(), nil
}
