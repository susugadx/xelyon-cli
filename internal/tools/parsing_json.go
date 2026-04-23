package tools

type jsonToolCallCandidate struct {
	json string
}

type jsonToolCallScanner struct {
	response        string
	codeBlockRanges [][2]int
	startFinder     jsonToolCallStartFinder
	errorPolicy     jsonToolCallScanErrorPolicy
	logger          *parseDebugLogger
	state           *jsonToolCallScanState
}

func parseJSONToolCalls(response string, codeBlockRanges [][2]int, startFinder jsonToolCallStartFinder, logger *parseDebugLogger) []*ToolCall {
	scanner := newJSONToolCallScanner(response, codeBlockRanges, startFinder, logger)
	decoder := newJSONToolCallDecoder(logger)

	var results []*ToolCall
	for {
		candidate, ok := scanner.Next()
		if !ok {
			break
		}
		if toolCall, ok := decoder.Decode(candidate); ok {
			results = append(results, toolCall)
		}
	}

	return results
}

func newJSONToolCallScanner(response string, codeBlockRanges [][2]int, startFinder jsonToolCallStartFinder, logger *parseDebugLogger) *jsonToolCallScanner {
	if startFinder == nil {
		startFinder = newDefaultJSONToolCallStartFinder()
	}

	return &jsonToolCallScanner{
		response:        response,
		codeBlockRanges: codeBlockRanges,
		startFinder:     startFinder,
		errorPolicy:     defaultJSONToolCallScanErrorPolicy(),
		logger:          logger,
		state:           newJSONToolCallScanState(),
	}
}

func (s *jsonToolCallScanner) Next() (jsonToolCallCandidate, bool) {
	if s.state.IsDone() {
		return jsonToolCallCandidate{}, false
	}

	for s.state.SearchFrom() < len(s.response) {
		start, ok := s.findNextStart()
		if !ok {
			return jsonToolCallCandidate{}, false
		}

		candidate, emitted := s.scanFromStart(start)
		if !emitted {
			if s.state.IsDone() {
				return jsonToolCallCandidate{}, false
			}
			continue
		}
		return candidate, true
	}

	s.state.MarkDone()
	return jsonToolCallCandidate{}, false
}

func (s *jsonToolCallScanner) findNextStart() (int, bool) {
	start := s.startFinder.Find(s.response, s.state.SearchFrom())
	if start == -1 {
		s.state.MarkDone()
		return 0, false
	}
	return start, true
}

func (s *jsonToolCallScanner) scanFromStart(start int) (jsonToolCallCandidate, bool) {
	if s.skipCodeBlockCandidate(start) {
		return jsonToolCallCandidate{}, false
	}

	candidate, end, scanErr := extractJSONObjectCandidate(s.response, start, s.logger)
	if scanErr != nil {
		s.handleScanError(start, *scanErr)
		return jsonToolCallCandidate{}, false
	}

	s.state.AdvanceTo(end)
	return candidate, true
}

func (s *jsonToolCallScanner) skipCodeBlockCandidate(start int) bool {
	if !shouldSkipCodeBlockJSON(start, s.codeBlockRanges, s.logger) {
		return false
	}
	s.state.AdvancePast(start)
	return true
}

func (s *jsonToolCallScanner) handleScanError(start int, scanErr jsonToolCallScanError) {
	if s.errorPolicy.Decide(scanErr) == jsonToolCallScanDecisionStop {
		s.state.MarkDone()
		return
	}
	s.state.AdvancePast(start)
}

func newJSONToolCallDecoder(logger *parseDebugLogger) *jsonToolCallDecoder {
	return &jsonToolCallDecoder{logger: logger}
}

func shouldSkipCodeBlockJSON(start int, codeBlockRanges [][2]int, logger *parseDebugLogger) bool {
	if !isInCodeBlock(start, codeBlockRanges) {
		return false
	}
	logger.LogEvent(newParseDebugSkipJSONInCodeBlockEvent(start))
	return true
}

func extractJSONObjectCandidate(response string, start int, logger *parseDebugLogger) (jsonToolCallCandidate, int, *jsonToolCallScanError) {
	end := findBalancedJSONObjectEnd(response, start)
	if end == -1 {
		logger.LogEvent(newParseDebugIncompleteJSONObjectEvent(start, response))
		return jsonToolCallCandidate{}, 0, &jsonToolCallScanError{
			kind:  jsonToolCallScanErrorIncompleteJSONObject,
			start: start,
		}
	}

	jsonStr := response[start:end]
	logger.LogEvent(newParseDebugExtractedJSONCandidateEvent(jsonStr))
	return jsonToolCallCandidate{json: jsonStr}, end, nil
}
