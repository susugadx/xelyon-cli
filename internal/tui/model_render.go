package tui

func (m *Model) rebuildLayout() {
	m.layout = BuildLayout(m.rawLines, m.width)
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
