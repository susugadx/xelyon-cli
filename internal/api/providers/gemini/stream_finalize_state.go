package gemini

import "fmt"

type sseFinalizeState struct {
	interpret *sseInterpretState
	output    sseFinalizeOutput
}

type sseFinalizeBuildResult struct {
	response             string
	rescuedToolCallCount int
}

func newSSEFinalizeState(interpret *sseInterpretState) *sseFinalizeState {
	return &sseFinalizeState{
		interpret: interpret,
		output:    newSSEFinalizeBuilderOutput(&interpret.fullResponse),
	}
}

func (s *sseFinalizeState) finalize(p *Provider) (string, error) {
	buildResult, err := s.buildOutput()
	if err != nil {
		return "", err
	}

	s.emitFinalizeEffects(p, buildResult.rescuedToolCallCount)
	return buildResult.response, nil
}

func (s *sseFinalizeState) buildOutput() (sseFinalizeBuildResult, error) {
	s.attachThoughtPartsToFunctionCalls()
	rescuedToolCallCount := s.appendRescuedToolJSONIfNeeded()
	s.appendUniqueFunctionCalls()

	if s.output.Len() == 0 {
		return sseFinalizeBuildResult{}, fmt.Errorf("no content in Gemini SSE response (stream ended without generating any text or function calls)")
	}

	return sseFinalizeBuildResult{
		response:             s.output.Response(),
		rescuedToolCallCount: rescuedToolCallCount,
	}, nil
}

func (s *sseFinalizeState) attachThoughtPartsToFunctionCalls() {
	if len(s.interpret.thoughtParts) == 0 {
		return
	}
	for _, fc := range s.interpret.functionCalls {
		fc.ThoughtParts = s.interpret.thoughtParts
	}
}

func (s *sseFinalizeState) appendRescuedToolJSONIfNeeded() int {
	if len(s.interpret.functionCalls) != 0 || len(s.interpret.rescuedToolJSONs) == 0 {
		return 0
	}

	s.interpret.debugf("[DEBUG Gemini SSE] Rescuing %d tool call(s) from text\n", len(s.interpret.rescuedToolJSONs))
	for _, toolJSON := range s.interpret.rescuedToolJSONs {
		s.output.Append(toolJSON)
	}
	return len(s.interpret.rescuedToolJSONs)
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
	p.emitUsageMetadata(s.interpret.usage)
}

func (s *sseFinalizeState) emitFinalizeEffects(p *Provider, rescuedToolCallCount int) {
	s.emitFunctionCallRescueWarning(rescuedToolCallCount)
	s.emitUsage(p)
	if s.interpret.display != nil {
		s.interpret.display.printTrailingNewlineIfNeeded()
	}
}

func (s *sseFinalizeState) emitFunctionCallRescueWarning(rescuedToolCallCount int) {
	if rescuedToolCallCount == 0 || s.interpret.display == nil {
		return
	}
	s.interpret.display.warnFunctionCallRescue(rescuedToolCallCount)
}
