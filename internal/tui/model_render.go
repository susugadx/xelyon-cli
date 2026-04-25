package tui

import termtext "github.com/susugadx/xelyon-cli/internal/tui/termtext"

func (m *Model) rebuildLayout() {
	m.layout = termtext.BuildLayout(m.rawLines, m.width)
}

func (m *Model) getVisualRowContents() []string {
	if m.layout == nil {
		return nil
	}
	res := make([]string, len(m.layout.Rows))
	for i, r := range m.layout.Rows {
		res[i] = r.Content
	}
	return res
}
