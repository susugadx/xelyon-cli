package tui

import "strings"

type sessionPickerState struct {
	candidates []SessionCandidate
	filter     string
	filtering  bool
	selected   int
	all        bool
}

func newSessionPickerState(candidates []SessionCandidate, all bool) *sessionPickerState {
	return &sessionPickerState{
		candidates: append([]SessionCandidate(nil), candidates...),
		all:        all,
	}
}

func (p *sessionPickerState) rows() []SessionCandidate {
	if p == nil {
		return nil
	}
	filter := strings.ToLower(strings.TrimSpace(p.filter))
	if filter == "" {
		return p.candidates
	}
	rows := make([]SessionCandidate, 0, len(p.candidates))
	for _, row := range p.candidates {
		haystack := strings.ToLower(strings.Join([]string{
			row.ID,
			row.Preview,
			row.ProviderName,
			row.Model,
			row.WorkingDir,
		}, " "))
		if strings.Contains(haystack, filter) {
			rows = append(rows, row)
		}
	}
	return rows
}

func (p *sessionPickerState) moveSelection(delta int) {
	if p == nil {
		return
	}
	p.selected += delta
	p.clampSelection()
}

func (p *sessionPickerState) clampSelection() {
	if p == nil {
		return
	}
	count := len(p.rows())
	if count == 0 {
		p.selected = 0
		return
	}
	if p.selected < 0 {
		p.selected = 0
	}
	if p.selected >= count {
		p.selected = count - 1
	}
}

func (p *sessionPickerState) selectedSession() (SessionCandidate, bool) {
	if p == nil {
		return SessionCandidate{}, false
	}
	rows := p.rows()
	if p.selected < 0 || p.selected >= len(rows) {
		return SessionCandidate{}, false
	}
	return rows[p.selected], true
}
