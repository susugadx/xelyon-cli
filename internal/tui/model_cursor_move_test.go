package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// l キーで幅2文字を1回で通過するかテスト
func TestNavL_SkipsOverWide2Char(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"ab日cd"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.rebuildChrome()
	m.navigationMode = true
	m.cursorLine = 0

	// "ab日cd" → col: a(0) b(1) 日(2-3) c(4) d(5)
	// カーソルを b(col=1) に置く
	m.cursorCol = 1

	// l 1回: b→日 (col 1→2, 次の grapheme cluster 先頭)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	if m.cursorCol != 2 {
		t.Errorf("l from b(1): cursorCol = %d, want 2 (land on 日)", m.cursorCol)
	}

	// l 1回: 日→c (col 2→4, 幅2文字を1回で通過)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	if m.cursorCol != 4 {
		t.Errorf("l from 日(2): cursorCol = %d, want 4 (skip over width-2 char)", m.cursorCol)
	}
}

func TestNavH_SkipsOverWide2Char(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"ab日cd"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.rebuildChrome()
	m.navigationMode = true
	m.cursorLine = 0

	// カーソルを c(col=4) に置く
	m.cursorCol = 4

	// h 1回で日の先頭(col=2) に行くべき（日を飛び越えない、日の上に着地）
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated.(Model)

	if m.cursorCol != 2 {
		t.Errorf("after h from col 4: cursorCol = %d, want 2 (should land on 日 start)", m.cursorCol)
	}
}

func TestNavL_SkipsOverEmoji(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"a✅b"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.rebuildChrome()
	m.navigationMode = true
	m.cursorLine = 0

	// "a✅b" → col: a(0) ✅(1-2) b(3)
	m.cursorCol = 0

	// l 1回: a→✅ (col 0→1)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	if m.cursorCol != 1 {
		t.Errorf("l from col 0: cursorCol = %d, want 1", m.cursorCol)
	}

	// l 1回: ✅→b (col 1→3, 幅2を飛び越える)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	if m.cursorCol != 3 {
		t.Errorf("l from col 1 (on ✅): cursorCol = %d, want 3 (should skip width-2)", m.cursorCol)
	}
}

func TestNavH_FromAfterEmoji(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"a✅b"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.rebuildChrome()
	m.navigationMode = true
	m.cursorLine = 0

	// "a✅b" → col: a(0) ✅(1-2) b(3)
	m.cursorCol = 3

	// h 1回: b→✅ (col 3→1)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated.(Model)
	if m.cursorCol != 1 {
		t.Errorf("h from col 3: cursorCol = %d, want 1 (should land on ✅ start)", m.cursorCol)
	}
}

func TestNavL_CJKSequence(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"日本語"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.rebuildChrome()
	m.navigationMode = true
	m.cursorLine = 0

	// "日本語" → col: 日(0-1) 本(2-3) 語(4-5)
	m.cursorCol = 0

	// l: 日→本 (col 0→2)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	if m.cursorCol != 2 {
		t.Errorf("l from 日(0): cursorCol = %d, want 2", m.cursorCol)
	}

	// l: 本→語 (col 2→4)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	if m.cursorCol != 4 {
		t.Errorf("l from 本(2): cursorCol = %d, want 4", m.cursorCol)
	}
}

func TestNavH_CJKSequence(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"日本語"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.rebuildChrome()
	m.navigationMode = true
	m.cursorLine = 0

	m.cursorCol = 4

	// h: 語→本 (col 4→2)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated.(Model)
	if m.cursorCol != 2 {
		t.Errorf("h from 語(4): cursorCol = %d, want 2", m.cursorCol)
	}

	// h: 本→日 (col 2→0)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated.(Model)
	if m.cursorCol != 0 {
		t.Errorf("h from 本(2): cursorCol = %d, want 0", m.cursorCol)
	}
}
