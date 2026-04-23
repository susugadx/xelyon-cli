package tools

import (
	"errors"
	"io"
)

type parseDebugEvent interface {
	log(*parseDebugLogger)
}

type parseDebugSkipJSONInCodeBlockEvent struct {
	start int
}

type parseDebugIncompleteJSONObjectEvent struct {
	start    int
	response string
}

type parseDebugExtractedJSONCandidateEvent struct {
	jsonStr string
}

type parseDebugSkipEmptyToolFieldEvent struct{}

type parseDebugJSONRepairedEvent struct{}

type parseDebugJSONParseErrorEvent struct {
	err error
}

type parseDebugXMLRescueSkipInCodeBlockEvent struct {
	tool string
}

type parseDebugXMLRescueSkipUnknownToolEvent struct {
	tool string
}

type parseDebugXMLRescueToolCallEvent struct {
	tool string
	args map[string]string
}

func newParseDebugSkipJSONInCodeBlockEvent(start int) parseDebugEvent {
	return parseDebugSkipJSONInCodeBlockEvent{start: start}
}

func newParseDebugIncompleteJSONObjectEvent(start int, response string) parseDebugEvent {
	return parseDebugIncompleteJSONObjectEvent{
		start:    start,
		response: response,
	}
}

func newParseDebugExtractedJSONCandidateEvent(jsonStr string) parseDebugEvent {
	return parseDebugExtractedJSONCandidateEvent{jsonStr: jsonStr}
}

func newParseDebugSkipEmptyToolFieldEvent() parseDebugEvent {
	return parseDebugSkipEmptyToolFieldEvent{}
}

func newParseDebugJSONRepairedEvent() parseDebugEvent {
	return parseDebugJSONRepairedEvent{}
}

func newParseDebugJSONParseErrorEvent(err error) parseDebugEvent {
	return parseDebugJSONParseErrorEvent{err: err}
}

func newParseDebugXMLRescueSkipInCodeBlockEvent(tool string) parseDebugEvent {
	return parseDebugXMLRescueSkipInCodeBlockEvent{tool: tool}
}

func newParseDebugXMLRescueSkipUnknownToolEvent(tool string) parseDebugEvent {
	return parseDebugXMLRescueSkipUnknownToolEvent{tool: tool}
}

func newParseDebugXMLRescueToolCallEvent(tool string, args map[string]string) parseDebugEvent {
	return parseDebugXMLRescueToolCallEvent{
		tool: tool,
		args: args,
	}
}

func (e parseDebugSkipJSONInCodeBlockEvent) log(l *parseDebugLogger) {
	l.Logf("[DEBUG ParseToolCalls] skipping: in code block at %d\n", e.start)
}

func (e parseDebugIncompleteJSONObjectEvent) log(l *parseDebugLogger) {
	l.Logf("[DEBUG ParseToolCalls] incomplete JSON: no closing brace found from index %d\n", e.start)
	showStart := e.start
	if len(e.response)-showStart > 200 {
		showStart = len(e.response) - 200
	}
	l.Logf("[DEBUG ParseToolCalls] tail: ...%s\n", e.response[showStart:])
}

func (e parseDebugExtractedJSONCandidateEvent) log(l *parseDebugLogger) {
	l.Logf("[DEBUG ParseToolCalls] extracted JSON (%d bytes): %s\n", len(e.jsonStr), truncateDebug(e.jsonStr, 200))
}

func (parseDebugSkipEmptyToolFieldEvent) log(l *parseDebugLogger) {
	l.Logf("[DEBUG ParseToolCalls] skipping: empty tool field\n")
}

func (parseDebugJSONRepairedEvent) log(l *parseDebugLogger) {
	l.Logf("[DEBUG ParseToolCalls] JSON repaired: fixed raw control characters in string values\n")
}

func (e parseDebugJSONParseErrorEvent) log(l *parseDebugLogger) {
	if errors.Is(e.err, io.EOF) {
		l.Logf("[DEBUG ParseToolCalls] JSON parse error: EOF\n")
		return
	}
	l.Logf("[DEBUG ParseToolCalls] JSON parse error: %v\n", e.err)
}

func (e parseDebugXMLRescueSkipInCodeBlockEvent) log(l *parseDebugLogger) {
	l.Logf("[DEBUG ParseToolCalls] XML rescue: skipping %q in code block\n", e.tool)
}

func (e parseDebugXMLRescueSkipUnknownToolEvent) log(l *parseDebugLogger) {
	l.Logf("[DEBUG ParseToolCalls] XML rescue: skipping unknown tool %q\n", e.tool)
}

func (e parseDebugXMLRescueToolCallEvent) log(l *parseDebugLogger) {
	l.Logf("[DEBUG ParseToolCalls] XML rescue: tool=%s, args=%v\n", e.tool, e.args)
}
