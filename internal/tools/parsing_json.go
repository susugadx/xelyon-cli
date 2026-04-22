package tools

import "encoding/json"

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

type jsonToolCallDecoder struct {
	logger *parseDebugLogger
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
		start := s.startFinder.Find(s.response, s.state.SearchFrom())
		if start == -1 {
			s.state.MarkDone()
			return jsonToolCallCandidate{}, false
		}

		if shouldSkipCodeBlockJSON(start, s.codeBlockRanges, s.logger) {
			s.state.AdvancePast(start)
			continue
		}

		candidate, end, scanErr := extractJSONObjectCandidate(s.response, start, s.logger)
		if scanErr != nil {
			if s.errorPolicy.Decide(*scanErr) == jsonToolCallScanDecisionStop {
				s.state.MarkDone()
				return jsonToolCallCandidate{}, false
			}
			s.state.AdvancePast(start)
			continue
		}

		s.state.AdvanceTo(end)
		return candidate, true
	}

	s.state.MarkDone()
	return jsonToolCallCandidate{}, false
}

func newJSONToolCallDecoder(logger *parseDebugLogger) *jsonToolCallDecoder {
	return &jsonToolCallDecoder{logger: logger}
}

func (d *jsonToolCallDecoder) Decode(candidate jsonToolCallCandidate) (*ToolCall, bool) {
	return decodeToolCallJSON(candidate.json, d.logger)
}

func shouldSkipCodeBlockJSON(start int, codeBlockRanges [][2]int, logger *parseDebugLogger) bool {
	if !isInCodeBlock(start, codeBlockRanges) {
		return false
	}
	logger.Logf("[DEBUG ParseToolCalls] skipping: in code block at %d\n", start)
	return true
}

func extractJSONObjectCandidate(response string, start int, logger *parseDebugLogger) (jsonToolCallCandidate, int, *jsonToolCallScanError) {
	end := findBalancedJSONObjectEnd(response, start)
	if end == -1 {
		logger.Logf("[DEBUG ParseToolCalls] incomplete JSON: no closing brace found from index %d\n", start)
		showStart := start
		if len(response)-showStart > 200 {
			showStart = len(response) - 200
		}
		logger.Logf("[DEBUG ParseToolCalls] tail: ...%s\n", response[showStart:])
		return jsonToolCallCandidate{}, 0, &jsonToolCallScanError{
			kind:  jsonToolCallScanErrorIncompleteJSONObject,
			start: start,
		}
	}

	jsonStr := response[start:end]
	logger.Logf("[DEBUG ParseToolCalls] extracted JSON (%d bytes): %s\n", len(jsonStr), truncateDebug(jsonStr, 200))
	return jsonToolCallCandidate{json: jsonStr}, end, nil
}

func decodeToolCallJSON(jsonStr string, logger *parseDebugLogger) (*ToolCall, bool) {
	var toolCall ToolCall
	if !unmarshalToolCallJSONWithRepair(jsonStr, &toolCall, logger) {
		return nil, false
	}

	if toolCall.Tool == "" {
		logger.Logf("[DEBUG ParseToolCalls] skipping: empty tool field\n")
		return nil, false
	}

	toolCall.NormalizeArgs()
	return &toolCall, true
}

func unmarshalToolCallJSONWithRepair(jsonStr string, toolCall *ToolCall, logger *parseDebugLogger) bool {
	err := json.Unmarshal([]byte(jsonStr), toolCall)
	if err == nil {
		return true
	}

	repaired := repairJSONStringValues(jsonStr)
	if repaired != jsonStr {
		if err2 := json.Unmarshal([]byte(repaired), toolCall); err2 == nil {
			logger.Logf("[DEBUG ParseToolCalls] JSON repaired: fixed raw control characters in string values\n")
			return true
		}
	}

	logger.Logf("[DEBUG ParseToolCalls] JSON parse error: %v\n", err)
	return false
}
