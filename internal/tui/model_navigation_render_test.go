package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestNavMode_ViewShowsColumnCursorInNormalMode(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.navigationMode = true
	m.rawLines = []string{"hello"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorLine = 0
	m.cursorCol = 2
	m.rebuildChrome()

	view := m.View()
	if !strings.Contains(view, "\033[48;5;255;38;5;16ml") {
		t.Fatalf("view should contain highlighted cursor character, got %q", view)
	}
	if !strings.Contains(view, "\033[48;5;236m                                                                           \033[0m") {
		t.Fatalf("view should extend line highlight into padding, got %q", view)
	}
}

func TestNavMode_CharVisualSelectionMapsAcrossWrappedRows(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.width = 5
	m.height = 8
	m.vp = lightViewport{width: 5, height: 4}
	m.rawLines = []string{"abcdefghij"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorLine = 0
	m.cursorCol = 2

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	m.cursorCol = 6

	lines := strings.Split(m.viewportView(), "\n")
	if len(lines) < 2 {
		t.Fatalf("viewport lines = %d, want at least 2", len(lines))
	}
	if !strings.Contains(lines[0], "\033[48;5;240m") {
		t.Fatalf("first wrapped row should have visual selection bg, got %q", lines[0])
	}
	plain0 := strings.TrimRight(stripANSI(lines[0]), " ")
	if plain0 != "abcde" {
		t.Fatalf("first wrapped row plain content = %q, want abcde", plain0)
	}
	if !strings.Contains(lines[1], "\033[48;5;240mf\033[0m") {
		t.Fatalf("second wrapped row should highlight f, got %q", lines[1])
	}
	if !strings.Contains(lines[1], "\033[48;5;255;38;5;16mg\033[0m") {
		t.Fatalf("second wrapped row should place visual cursor on g, got %q", lines[1])
	}
	if strings.Contains(lines[1], "\033[48;5;240mh\033[0m") || strings.Contains(lines[1], "\033[48;5;255;38;5;16mh\033[0m") {
		t.Fatalf("second wrapped row should not highlight h.., got %q", lines[1])
	}
}

func TestNavMode_CharVisualSelectionStopsAtWrappedBoundary(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.width = 5
	m.height = 8
	m.vp = lightViewport{width: 5, height: 4}
	m.rawLines = []string{"abcdefghij"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorLine = 0
	m.cursorCol = 2

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	m.cursorCol = 4

	lines := strings.Split(m.viewportView(), "\n")
	if len(lines) < 2 {
		t.Fatalf("viewport lines = %d, want at least 2", len(lines))
	}
	if !strings.Contains(lines[0], "\033[48;5;255;38;5;16me\033[0m") {
		t.Fatalf("boundary selection should end with cursor on e, got %q", lines[0])
	}
	if strings.Contains(lines[1], "\033[48;5;240m") || strings.Contains(lines[1], "\033[48;5;255;38;5;16m") {
		t.Fatalf("next wrapped row should not be highlighted when selection ends at boundary, got %q", lines[1])
	}
}

func TestNavMode_CharVisualSelectionOnLongANSILineKeepsCursorVisible(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.width = 6
	m.height = 8
	m.vp = lightViewport{width: 6, height: 4}
	m.rawLines = []string{"\033[31mabcdef ghi\033[0m"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorLine = 0
	m.cursorCol = 1

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	m.cursorCol = 7
	m.rebuildChrome()

	view := m.View()
	firstLine := strings.Split(view, "\n")[0]
	if !strings.Contains(view, "\033[48;5;255;38;5;16mg") {
		t.Fatalf("view should keep visual cursor visible on truncated ANSI line, got %q", view)
	}
	if lipgloss.Width(stripANSI(firstLine)) != 6 {
		t.Fatalf("first line width = %d, want 6; line=%q", lipgloss.Width(stripANSI(firstLine)), firstLine)
	}
	if strings.Contains(stripANSI(firstLine), "g") {
		t.Fatalf("first line should stay truncated to visible width, got %q", firstLine)
	}
}

func TestNavMode_ViewShowsCursorOnEmptyLine(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.navigationMode = true
	m.rawLines = []string{""}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorLine = 0
	m.cursorCol = 0
	m.rebuildChrome()

	view := m.View()
	if !strings.Contains(view, "\033[48;5;255;38;5;16m \033[0m") {
		t.Fatalf("view should contain cursor placeholder for empty line, got %q", view)
	}
}

func TestNavMode_ViewportViewPadsTrailingBlankRowsToWidth(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.width = 8
	m.height = 8
	m.vp = lightViewport{width: 8, height: 4}
	m.rawLines = []string{"line1", "line2"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoBottom()

	lines := strings.Split(m.viewportView(), "\n")
	if len(lines) != m.vp.height {
		t.Fatalf("viewport lines = %d, want %d", len(lines), m.vp.height)
	}

	for i, line := range lines {
		if got := lipgloss.Width(line); got != m.vp.width {
			t.Fatalf("line %d width = %d, want %d; line=%q", i, got, m.vp.width, line)
		}
	}
}

func TestNavMode_LineVisualSelectionPadsTrailingBlankRowsToWidth(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.width = 10
	m.height = 8
	m.vp = lightViewport{width: 10, height: 4}
	m.rawLines = []string{"\033[31m日本語\033[0m", "tail"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	m = updated.(Model)
	lines := strings.Split(m.viewportView(), "\n")
	if len(lines) != m.vp.height {
		t.Fatalf("viewport lines = %d, want %d", len(lines), m.vp.height)
	}

	for i, line := range lines {
		if got := lipgloss.Width(line); got != m.vp.width {
			t.Fatalf("line %d width = %d, want %d; line=%q", i, got, m.vp.width, line)
		}
	}
}
