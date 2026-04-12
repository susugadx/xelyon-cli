package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBottom_G_FooterStaysFixed(t *testing.T) {
	m := setupBottomTestModel(50)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewLines(t, m, "enter NAV")
	verifyFooterPosition(t, m, "enter NAV")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	verifyViewLines(t, m, "G")
	verifyFooterPosition(t, m, "G")
}

func TestBottom_RepeatedJ_FooterStaysFixed(t *testing.T) {
	m := setupBottomTestModel(50)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	for i := range 60 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = updated.(Model)
		verifyViewLines(t, m, fmt.Sprintf("j_%d", i))
	}
	verifyFooterPosition(t, m, "after j spam bottom")
}

func TestBottom_RepeatedDown_PastContent(t *testing.T) {
	m := setupBottomTestModel(50)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)

	for i := range 20 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = updated.(Model)
		verifyViewLines(t, m, fmt.Sprintf("j_past_%d", i))
	}
	verifyFooterPosition(t, m, "j past bottom")
}

func TestBottom_GG_Then_G(t *testing.T) {
	m := setupBottomTestModel(50)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(Model)
	verifyViewLines(t, m, "gg")
	verifyFooterPosition(t, m, "gg")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	verifyViewLines(t, m, "G after gg")
	verifyFooterPosition(t, m, "G after gg")
}

func TestBottom_DU_HalfPage(t *testing.T) {
	m := setupBottomTestModel(50)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	for i := range 5 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
		m = updated.(Model)
		verifyViewLines(t, m, fmt.Sprintf("d_%d", i))
	}
	verifyFooterPosition(t, m, "after d spam")

	for i := range 5 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
		m = updated.(Model)
		verifyViewLines(t, m, fmt.Sprintf("u_%d", i))
	}
	verifyFooterPosition(t, m, "after u spam")
}

func TestBottom_BlockTab_NearBottom(t *testing.T) {
	m := setupBottomTestModel(30)
	m.appendToolResult(ToolResult{Name: "t1", Summary: "Tool near end", Detail: "detail"})
	m.appendToolResult(ToolResult{Name: "t2", Summary: "Last tool", Detail: "d1\nd2\nd3"})
	m.rebuildChrome()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	verifyViewLines(t, m, "Tab first block")
	verifyFooterPosition(t, m, "Tab first block")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(Model)
	verifyViewLines(t, m, "ShiftTab")
	verifyFooterPosition(t, m, "ShiftTab")
}

func TestBottom_NavThenEsc(t *testing.T) {
	m := setupBottomTestModel(50)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewLines(t, m, "bottom then ESC")
	verifyFooterPosition(t, m, "bottom then ESC")
}
