package toolblock

import "strings"

type renderState struct {
	summary   string
	detail    string
	collapsed bool
	focused   bool
}

func newRenderState(summary string, detail string, collapsed bool, focused bool) renderState {
	return renderState{
		summary:   summary,
		detail:    detail,
		collapsed: collapsed,
		focused:   focused,
	}
}

// SummaryLine は tool result block の summary 行を生成する。
func SummaryLine(summary string, collapsed bool, focused bool) string {
	return newRenderState(summary, "", collapsed, focused).summaryLine()
}

func (s renderState) summaryLine() string {
	indicator := "  "
	if s.focused {
		indicator = "▶ "
	}

	return indicator + s.summary
}

// Lines は tool result block の表示行を生成する。
func Lines(summary string, detail string, collapsed bool, focused bool) []string {
	return newRenderState(summary, detail, collapsed, focused).lines()
}

func (s renderState) lines() []string {
	summaryLine := s.summaryLine()
	if s.collapsed {
		return []string{summaryLine}
	}

	lines := []string{summaryLine}
	for _, line := range strings.Split(s.detail, "\n") {
		lines = append(lines, "  "+line)
	}
	return lines
}
