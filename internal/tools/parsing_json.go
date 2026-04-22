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
	searchFrom      int
	done            bool
}

type jsonToolCallDecoder struct {
	logger *parseDebugLogger
}

type jsonToolCallScanDecision int

const (
	jsonToolCallScanDecisionContinue jsonToolCallScanDecision = iota
	jsonToolCallScanDecisionStop
)

type jsonToolCallScanErrorKind int

const (
	jsonToolCallScanErrorIncompleteJSONObject jsonToolCallScanErrorKind = iota + 1
)

type jsonToolCallScanError struct {
	kind  jsonToolCallScanErrorKind
	start int
}

type jsonToolCallScanErrorPolicy struct {
	onIncompleteJSONObject jsonToolCallScanDecision
}

func defaultJSONToolCallScanErrorPolicy() jsonToolCallScanErrorPolicy {
	// 既存挙動互換: 途中で不完全 JSON を見つけたら JSON 走査を終了する。
	return jsonToolCallScanErrorPolicy{
		onIncompleteJSONObject: jsonToolCallScanDecisionStop,
	}
}

func (p jsonToolCallScanErrorPolicy) Decide(err jsonToolCallScanError) jsonToolCallScanDecision {
	switch err.kind {
	case jsonToolCallScanErrorIncompleteJSONObject:
		return p.onIncompleteJSONObject
	default:
		return jsonToolCallScanDecisionStop
	}
}

func parseJSONToolCalls(response string, codeBlockRanges [][2]int, logger *parseDebugLogger) []*ToolCall {
	scanner := newJSONToolCallScanner(response, codeBlockRanges, logger)
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

func newJSONToolCallScanner(response string, codeBlockRanges [][2]int, logger *parseDebugLogger) *jsonToolCallScanner {
	return &jsonToolCallScanner{
		response:        response,
		codeBlockRanges: codeBlockRanges,
		startFinder:     newDefaultJSONToolCallStartFinder(),
		errorPolicy:     defaultJSONToolCallScanErrorPolicy(),
		logger:          logger,
	}
}

func (s *jsonToolCallScanner) Next() (jsonToolCallCandidate, bool) {
	if s.done {
		return jsonToolCallCandidate{}, false
	}

	for s.searchFrom < len(s.response) {
		start := s.startFinder.Find(s.response, s.searchFrom)
		if start == -1 {
			s.done = true
			return jsonToolCallCandidate{}, false
		}

		if shouldSkipCodeBlockJSON(start, s.codeBlockRanges, s.logger) {
			s.searchFrom = start + 1
			continue
		}

		candidate, end, scanErr := extractJSONObjectCandidate(s.response, start, s.logger)
		if scanErr != nil {
			if s.errorPolicy.Decide(*scanErr) == jsonToolCallScanDecisionStop {
				s.done = true
				return jsonToolCallCandidate{}, false
			}
			s.searchFrom = start + 1
			continue
		}

		s.searchFrom = end
		return candidate, true
	}

	s.done = true
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
	end := findJSONObjectEnd(response, start)
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

func findJSONObjectEnd(response string, start int) int {
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(response); i++ {
		ch := response[i]

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			escaped = true
			continue
		}

		if ch == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}

	return -1
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
