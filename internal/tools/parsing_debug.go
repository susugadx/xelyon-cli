package tools

import (
	"fmt"
	"io"
	"strings"
)

type parseDebugLogger struct {
	enabled bool
	out     io.Writer
}

func newParseDebugLogger(enabled bool, out io.Writer) *parseDebugLogger {
	if out == nil {
		out = io.Discard
	}
	return &parseDebugLogger{
		enabled: enabled,
		out:     out,
	}
}

func (l *parseDebugLogger) Logf(format string, args ...interface{}) {
	if l == nil || !l.enabled {
		return
	}
	fmt.Fprintf(l.out, format, args...)
}

func (l *parseDebugLogger) LogParseResponse(response string, finder jsonToolCallStartFinder) {
	if l == nil || !l.enabled {
		return
	}

	l.Logf("[DEBUG ParseToolCalls] response length: %d\n", len(response))
	if finder == nil {
		return
	}
	for _, p := range finder.DebugPatterns() {
		if idx := strings.Index(response, p); idx != -1 {
			l.Logf("[DEBUG ParseToolCalls] found pattern %q at index %d\n", p, idx)
			start := idx
			if start > 50 {
				start = idx - 50
			}
			end := idx + 100
			if end > len(response) {
				end = len(response)
			}
			l.Logf("[DEBUG ParseToolCalls] context: ...%s...\n", response[start:end])
		}
	}
}

func (l *parseDebugLogger) LogEvent(event parseDebugEvent) {
	if l == nil || !l.enabled || event == nil {
		return
	}
	event.log(l)
}
