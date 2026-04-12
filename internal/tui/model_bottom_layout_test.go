package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestBottom_ShortContent_FooterStaysAtBottom(t *testing.T) {
	m := setupBottomTestModel(3)

	verifyViewLines(t, m, "short normal")
	verifyFooterPosition(t, m, "short normal")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	verifyViewLines(t, m, "short G")
	verifyFooterPosition(t, m, "short G")
}

func TestInvariant_VpHeightPlusFooter_AllModes(t *testing.T) {
	m := setupBottomTestModel(50)

	check := func(label string) {
		t.Helper()
		if m.vp.height+m.footerHeight() != m.height {
			t.Errorf("[%s] vp.height(%d) + footer(%d) = %d, want %d",
				label, m.vp.height, m.footerHeight(), m.vp.height+m.footerHeight(), m.height)
		}
	}

	check("normal")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	check("NAV")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	check("visual")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	check("G")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	check("back to normal")

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 15})
	m = updated.(Model)
	check("after resize")
}

func TestBottom_TrailingEmptyLines_FooterStaysFixed(t *testing.T) {
	m := setupBottomTestModel(10)
	m.appendContentLines("", "", "", "")
	m.rebuildChrome()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	verifyViewLines(t, m, "trailing empties G")
	verifyFooterPosition(t, m, "trailing empties G")

	for range 5 {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = updated.(Model)
	}
	verifyViewLines(t, m, "trailing empties j past")
	verifyFooterPosition(t, m, "trailing empties j past")
}

func TestBottom_NoEmbeddedNewlines(t *testing.T) {
	m := setupBottomTestModel(50)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)

	view := m.viewportView()
	viewLines := strings.Split(view, "\n")
	if len(viewLines) != m.vp.height {
		t.Fatalf("viewport line count = %d, want %d", len(viewLines), m.vp.height)
	}

	chrome := m.chromeCache
	chromeLines := strings.Split(chrome, "\n")
	if len(chromeLines) != m.footerHeight() {
		t.Fatalf("chrome line count = %d, want %d", len(chromeLines), m.footerHeight())
	}
}
