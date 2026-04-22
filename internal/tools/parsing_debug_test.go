package tools

import (
	"bytes"
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
