package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

var toolCallJSONStartPatterns = []string{
	"{\"id\"",     // {"id" (Function Calling)
	"{ \"id\"",    // { "id" (Function Calling)
	"{\"tool\"",   // {"tool"
	"{ \"tool\"",  // { "tool"
	"{\"tool\":",  // {"tool":
	"{ \"tool\":", // { "tool":
}

type jsonToolCallCandidate struct {
	json string
}

type jsonToolCallScanner struct {
	response        string
	codeBlockRanges [][2]int
	debug           bool
	debugOut        io.Writer
	searchFrom      int
	done            bool
}

type jsonToolCallDecoder struct {
	debug    bool
	debugOut io.Writer
}

func logParseResponseDebug(response string, debugOut io.Writer) {
	fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] response length: %d\n", len(response))
	for _, p := range []string{`{"tool"`, `{ "tool"`} {
		if idx := strings.Index(response, p); idx != -1 {
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] found pattern %q at index %d\n", p, idx)
			start := idx
			if start > 50 {
				start = idx - 50
			}
			end := idx + 100
			if end > len(response) {
				end = len(response)
			}
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] context: ...%s...\n", response[start:end])
		}
	}
}

func parseJSONToolCalls(response string, codeBlockRanges [][2]int, debug bool, debugOut io.Writer) []*ToolCall {
	scanner := newJSONToolCallScanner(response, codeBlockRanges, debug, debugOut)
	decoder := newJSONToolCallDecoder(debug, debugOut)

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

func newJSONToolCallScanner(response string, codeBlockRanges [][2]int, debug bool, debugOut io.Writer) *jsonToolCallScanner {
	return &jsonToolCallScanner{
		response:        response,
		codeBlockRanges: codeBlockRanges,
		debug:           debug,
		debugOut:        debugOut,
	}
}

func (s *jsonToolCallScanner) Next() (jsonToolCallCandidate, bool) {
	if s.done {
		return jsonToolCallCandidate{}, false
	}

	for s.searchFrom < len(s.response) {
		start := findNextToolCallJSONStart(s.response, s.searchFrom)
		if start == -1 {
			s.done = true
			return jsonToolCallCandidate{}, false
		}

		if shouldSkipCodeBlockJSON(start, s.codeBlockRanges, s.debug, s.debugOut) {
			s.searchFrom = start + 1
			continue
		}

		jsonStr, end, ok := extractJSONObject(s.response, start, s.debug, s.debugOut)
		if !ok {
			s.done = true
			return jsonToolCallCandidate{}, false
		}

		s.searchFrom = end
		return jsonToolCallCandidate{json: jsonStr}, true
	}

	s.done = true
	return jsonToolCallCandidate{}, false
}

func newJSONToolCallDecoder(debug bool, debugOut io.Writer) *jsonToolCallDecoder {
	return &jsonToolCallDecoder{debug: debug, debugOut: debugOut}
}

func (d *jsonToolCallDecoder) Decode(candidate jsonToolCallCandidate) (*ToolCall, bool) {
	return decodeToolCallJSON(candidate.json, d.debug, d.debugOut)
}

func findNextToolCallJSONStart(response string, searchFrom int) int {
	start := -1
	for _, pattern := range toolCallJSONStartPatterns {
		idx := strings.Index(response[searchFrom:], pattern)
		if idx == -1 {
			continue
		}
		absIdx := searchFrom + idx
		if start == -1 || absIdx < start {
			start = absIdx
		}
	}
	return start
}

func shouldSkipCodeBlockJSON(start int, codeBlockRanges [][2]int, debug bool, debugOut io.Writer) bool {
	if !isInCodeBlock(start, codeBlockRanges) {
		return false
	}
	if debug {
		fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] skipping: in code block at %d\n", start)
	}
	return true
}

func extractJSONObject(response string, start int, debug bool, debugOut io.Writer) (string, int, bool) {
	end := findJSONObjectEnd(response, start)
	if end == -1 {
		if debug {
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] incomplete JSON: no closing brace found from index %d\n", start)
			showStart := start
			if len(response)-showStart > 200 {
				showStart = len(response) - 200
			}
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] tail: ...%s\n", response[showStart:])
		}
		return "", 0, false
	}

	jsonStr := response[start:end]
	if debug {
		fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] extracted JSON (%d bytes): %s\n", len(jsonStr), truncateDebug(jsonStr, 200))
	}
	return jsonStr, end, true
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

func decodeToolCallJSON(jsonStr string, debug bool, debugOut io.Writer) (*ToolCall, bool) {
	var toolCall ToolCall
	if !unmarshalToolCallJSONWithRepair(jsonStr, &toolCall, debug, debugOut) {
		return nil, false
	}

	if toolCall.Tool == "" {
		if debug {
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] skipping: empty tool field\n")
		}
		return nil, false
	}

	toolCall.NormalizeArgs()
	return &toolCall, true
}

func unmarshalToolCallJSONWithRepair(jsonStr string, toolCall *ToolCall, debug bool, debugOut io.Writer) bool {
	if err := json.Unmarshal([]byte(jsonStr), toolCall); err == nil {
		return true
	} else {
		repaired := repairJSONStringValues(jsonStr)
		if repaired != jsonStr {
			if err2 := json.Unmarshal([]byte(repaired), toolCall); err2 == nil {
				if debug {
					fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] JSON repaired: fixed raw control characters in string values\n")
				}
				return true
			}
		}

		if debug {
			fmt.Fprintf(debugOut, "[DEBUG ParseToolCalls] JSON parse error: %v\n", err)
		}
		return false
	}
}
