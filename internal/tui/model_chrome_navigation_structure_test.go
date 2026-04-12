package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestView_NavJK_StructureConsistent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewStructure(t, m, "enter NAV")

	for i := range 15 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = updated.(Model)
		verifyViewStructure(t, m, "j press "+string(rune('0'+i%10)))
	}

	for i := range 10 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
		m = updated.(Model)
		verifyViewStructure(t, m, "k press "+string(rune('0'+i%10)))
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewStructure(t, m, "exit NAV")
}

func TestView_NavWordMovement_StructureConsistent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	for i := range 5 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
		m = updated.(Model)
		verifyViewStructure(t, m, "w press "+string(rune('0'+i)))
	}
	for i := range 3 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
		m = updated.(Model)
		verifyViewStructure(t, m, "b press "+string(rune('0'+i)))
	}
	for i := range 2 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		m = updated.(Model)
		verifyViewStructure(t, m, "e press "+string(rune('0'+i)))
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewStructure(t, m, "exit NAV after word movement")
}

func TestView_NavLineJumps_StructureConsistent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "0")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'$'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "$")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'^'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "^")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "G")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "gg")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "d")
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "u")
}

func TestView_NavBlockFocus_StructureConsistent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	m.appendToolResult(ToolResult{Name: "test1", Summary: "Tool 1", Detail: "detail1"})
	m.appendToolResult(ToolResult{Name: "test2", Summary: "Tool 2", Detail: "detail2\nline2"})
	m.rebuildChrome()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	verifyViewStructure(t, m, "Tab focus")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "j in block focus")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "k in block focus")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	verifyViewStructure(t, m, "arrow down block focus")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	verifyViewStructure(t, m, "toggle block")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewStructure(t, m, "unfocus block")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewStructure(t, m, "exit NAV after block ops")
}

func TestView_NavVisualWordMovement_StructureConsistent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := setupModelForChromeTest(agent)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "visual start")

	for i := range 3 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
		m = updated.(Model)
		verifyViewStructure(t, m, "visual w "+string(rune('0'+i)))
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'$'}})
	m = updated.(Model)
	verifyViewStructure(t, m, "visual $")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewStructure(t, m, "cancel visual")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	verifyViewStructure(t, m, "exit NAV after visual")
}
