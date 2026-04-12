package tui

import (
	"strconv"
)

func lineLabel(n int) string {
	if n == 1 {
		return "1 line"
	}
	return strconv.Itoa(n) + " lines"
}

func (m *Model) setCopyError(err error) {
	m.setTransientStatus("Copy failed: " + err.Error())
}

func (m *Model) setCopySuccess(msg string) {
	m.setTransientStatus("✅ " + msg)
}
