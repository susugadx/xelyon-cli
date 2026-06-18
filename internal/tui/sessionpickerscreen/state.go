package sessionpickerscreen

import (
	"strings"
	"time"
)

// Candidate は session picker が表示・選択に使う session 候補を表す。
type Candidate struct {
	ID           string
	Preview      string
	Model        string
	ProviderName string
	WorkingDir   string
	LastModified time.Time
}

// Command は session picker が root Model に要求する操作を表す。
type Command int

const (
	// CommandNone は root 側の操作が不要な入力処理を表す。
	CommandNone Command = iota
	// CommandClose は picker を閉じる要求を表す。
	CommandClose
	// CommandResume は選択 session の resume 要求を表す。
	CommandResume
)

// KeyResult はキー入力処理の結果を表す。
type KeyResult struct {
	Command   Command
	Candidate Candidate
}

// Snapshot は root tests が picker の公開状態を確認するための読み取り専用状態。
type Snapshot struct {
	All       bool
	Startup   bool
	Filtering bool
	Filter    string
	Selected  int
}

// Screen は resume session picker の UI state/input/render を保持する。
type Screen struct {
	candidates []Candidate
	filter     string
	filtering  bool
	selected   int
	all        bool
	startup    bool
}

// New は session picker を構築する。
func New(candidates []Candidate, all bool, startup bool) *Screen {
	return &Screen{
		candidates: append([]Candidate(nil), candidates...),
		all:        all,
		startup:    startup,
	}
}

// Snapshot は picker の公開状態を返す。
func (p *Screen) Snapshot() Snapshot {
	if p == nil {
		return Snapshot{}
	}
	return Snapshot{
		All:       p.all,
		Startup:   p.startup,
		Filtering: p.filtering,
		Filter:    p.filter,
		Selected:  p.selected,
	}
}

// All は全 working directory 対象の resume picker かどうかを返す。
func (p *Screen) All() bool {
	return p != nil && p.all
}

// Startup は startup resume picker かどうかを返す。
func (p *Screen) Startup() bool {
	return p != nil && p.startup
}

func (p *Screen) rows() []Candidate {
	if p == nil {
		return nil
	}
	filter := strings.ToLower(strings.TrimSpace(p.filter))
	if filter == "" {
		return p.candidates
	}
	rows := make([]Candidate, 0, len(p.candidates))
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

func (p *Screen) moveSelection(delta int) {
	if p == nil {
		return
	}
	p.selected += delta
	p.clampSelection()
}

func (p *Screen) clampSelection() {
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

func (p *Screen) selectedSession() (Candidate, bool) {
	if p == nil {
		return Candidate{}, false
	}
	rows := p.rows()
	if p.selected < 0 || p.selected >= len(rows) {
		return Candidate{}, false
	}
	return rows[p.selected], true
}
