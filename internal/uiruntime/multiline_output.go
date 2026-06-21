package uiruntime

import (
	"fmt"
	"io"
)

func (m *MultilineReader) outputWriter() io.Writer {
	if m.out == nil {
		return DefaultRuntime().Output()
	}
	return m.out
}

func (m *MultilineReader) errorWriter() io.Writer {
	if m.err == nil {
		return DefaultRuntime().ErrorOutput()
	}
	return m.err
}

func (m *MultilineReader) print(args ...interface{}) {
	_, _ = fmt.Fprint(m.outputWriter(), args...)
}

func (m *MultilineReader) printf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(m.outputWriter(), format, args...)
}

func (m *MultilineReader) println(args ...interface{}) {
	_, _ = fmt.Fprintln(m.outputWriter(), args...)
}

func (m *MultilineReader) writeString(s string) {
	_, _ = io.WriteString(m.outputWriter(), s)
}
