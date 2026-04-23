package tools

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestParseDebugLogger_LogParseResponse_UsesFinderPatterns(t *testing.T) {
	var buf bytes.Buffer
	logger := newParseDebugLogger(true, &buf)
	finder := newPatternJSONToolCallStartFinder([]string{"<<tool>>"})

	logger.LogParseResponse("prefix <<tool>> suffix", finder)
	out := buf.String()
	if !strings.Contains(out, "found pattern \"<<tool>>\"") {
		t.Fatalf("debug log should include finder pattern, got: %s", out)
	}
}

func TestParseDebugLogger_LogParseResponse_WithoutFinder(t *testing.T) {
	var buf bytes.Buffer
	logger := newParseDebugLogger(true, &buf)

	logger.LogParseResponse("plain response", nil)
	out := buf.String()
	if !strings.Contains(out, "response length") {
		t.Fatalf("debug log should include response length, got: %s", out)
	}
}

func TestParseDebugLogger_EventMethods(t *testing.T) {
	var buf bytes.Buffer
	logger := newParseDebugLogger(true, &buf)

	logger.LogEvent(newParseDebugSkipJSONInCodeBlockEvent(12))
	logger.LogEvent(newParseDebugIncompleteJSONObjectEvent(5, "prefix {\"tool\": \"read_file\""))
	logger.LogEvent(newParseDebugExtractedJSONCandidateEvent(`{"tool":"bash"}`))
	logger.LogEvent(newParseDebugSkipEmptyToolFieldEvent())
	logger.LogEvent(newParseDebugJSONRepairedEvent())
	logger.LogEvent(newParseDebugJSONParseErrorEvent(io.EOF))
	logger.LogEvent(newParseDebugXMLRescueSkipInCodeBlockEvent("read_file"))
	logger.LogEvent(newParseDebugXMLRescueSkipUnknownToolEvent("unknown"))
	logger.LogEvent(newParseDebugXMLRescueToolCallEvent("bash", map[string]string{"command": "ls"}))

	out := buf.String()
	for _, want := range []string{
		"skipping: in code block at 12",
		"incomplete JSON",
		"extracted JSON",
		"skipping: empty tool field",
		"JSON repaired",
		"JSON parse error",
		`XML rescue: skipping "read_file" in code block`,
		`XML rescue: skipping unknown tool "unknown"`,
		"XML rescue: tool=bash",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("debug log should include %q, got: %s", want, out)
		}
	}
}
