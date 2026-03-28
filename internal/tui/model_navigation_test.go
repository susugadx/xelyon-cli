package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestNavMode_EscEntersNavWhenInputEmpty(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.vp = lightViewport{width: 10, height: 5, yOffset: 3}
	setModelRawLines(&m, 20)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(Model)
	if !got.navigationMode {
		t.Fatal("Esc with empty input should enter navigation mode")
	}
	if got.cursorLine != 3 {
		t.Fatalf("cursorLine = %d, want 3", got.cursorLine)
	}
}

func TestNavMode_EscDoesNotEnterNavWhenInputHasText(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.textInput.SetValue("hello")

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(Model)
	if got.navigationMode {
		t.Fatal("Esc with text in input should NOT enter navigation mode")
	}
}

func TestNavMode_QExitsNav(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	got := updated.(Model)
	if got.navigationMode {
		t.Fatal("q should exit navigation mode")
	}
}

func TestNavMode_JKScrolls(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 5}
	setModelRawLines(&m, 20)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got := updated.(Model)
	if got.cursorLine != 1 {
		t.Fatalf("j: cursorLine = %d, want 1", got.cursorLine)
	}

	updated, _ = got.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	got = updated.(Model)
	if got.cursorLine != 0 {
		t.Fatalf("k: cursorLine = %d, want 0", got.cursorLine)
	}
}

func TestNavMode_DUHalfPage(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 10}
	setModelRawLines(&m, 40)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	got := updated.(Model)
	if got.cursorLine != 5 {
		t.Fatalf("d: cursorLine = %d, want 5", got.cursorLine)
	}

	updated, _ = got.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	got = updated.(Model)
	if got.cursorLine != 0 {
		t.Fatalf("u: cursorLine = %d, want 0", got.cursorLine)
	}
}

func TestNavMode_CountPrefixAppliesToMoves(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 10}
	setModelRawLines(&m, 40)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = updated.(Model)
	if m.pendingCount != 3 {
		t.Fatalf("pendingCount = %d, want 3", m.pendingCount)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.cursorLine != 3 {
		t.Fatalf("cursorLine = %d, want 3", m.cursorLine)
	}
	if m.pendingCount != 0 {
		t.Fatalf("pendingCount = %d, want 0", m.pendingCount)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)
	if m.cursorLine != 13 {
		t.Fatalf("cursorLine after 2d = %d, want 13", m.cursorLine)
	}
}

func TestNavMode_GGAndG(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 5}
	setModelRawLines(&m, 20)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	got := updated.(Model)
	if got.cursorLine != 19 {
		t.Fatalf("G: cursorLine = %d, want 19", got.cursorLine)
	}

	updated, _ = got.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	got = updated.(Model)
	if !got.gPressed {
		t.Fatal("first g should set gPressed")
	}
	updated, _ = got.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	got = updated.(Model)
	if got.cursorLine != 0 {
		t.Fatalf("gg: cursorLine = %d, want 0", got.cursorLine)
	}
	if got.gPressed {
		t.Fatal("gPressed should be reset after gg")
	}
}

func TestNavMode_CountPrefixAppliesToGGAndG(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 5}
	setModelRawLines(&m, 20)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)
	if m.cursorLine != 11 {
		t.Fatalf("cursorLine after 12G = %d, want 11", m.cursorLine)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(Model)
	if m.cursorLine != 2 {
		t.Fatalf("cursorLine after 3gg = %d, want 2", m.cursorLine)
	}
}

func TestNavMode_GFollowedByOtherKeyResetsG(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.gPressed = true

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got := updated.(Model)
	if got.gPressed {
		t.Fatal("gPressed should be reset after non-g key")
	}
}

func TestNavMode_CtrlCWorksInNav(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true

	updated, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyCtrlC})
	got := updated.(Model)
	if cmd != nil {
		t.Fatal("first ctrl+c should not quit")
	}
	if !got.lastInterrupt.After(time.Now().Add(-time.Second)) {
		t.Fatal("lastInterrupt should be set")
	}
}

func TestNavMode_YCallsCopy(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 5}
	setModelRawLines(&m, 5)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	got := updated.(Model)
	if agent.copyCalls != 0 {
		t.Fatalf("first y should wait for yy, copyCalls = %d", agent.copyCalls)
	}
	if !got.yPressed {
		t.Fatal("first y should set yPressed")
	}

	updated, _ = got.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	got = updated.(Model)
	if agent.copyCalls != 1 {
		t.Fatalf("copyCalls = %d, want 1", agent.copyCalls)
	}
	if len(agent.copyTexts) != 1 || agent.copyTexts[0] != "line0" {
		t.Fatalf("copied text = %#v, want [line0]", agent.copyTexts)
	}
	if got.transientStatus == "" {
		t.Fatal("transientStatus should be set after copy")
	}
}

func TestNavMode_VisualSelectionCopiesPlainText(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 20, height: 5}
	m.rawLines = []string{"one", "\033[31m二行目\033[0m", "three"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	m = updated.(Model)
	if m.visualMode != visualModeLine {
		t.Fatalf("visualMode = %d, want %d", m.visualMode, visualModeLine)
	}
	if m.visualStart.line != 0 {
		t.Fatalf("visualStart.line = %d, want 0", m.visualStart.line)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.cursorLine != 1 {
		t.Fatalf("cursorLine = %d, want 1", m.cursorLine)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)
	if m.visualMode != visualModeOff {
		t.Fatalf("visualMode after copy = %d, want %d", m.visualMode, visualModeOff)
	}
	if len(agent.copyTexts) != 1 {
		t.Fatalf("copyTexts len = %d, want 1", len(agent.copyTexts))
	}
	if agent.copyTexts[0] != "one\n二行目" {
		t.Fatalf("copied text = %q, want %q", agent.copyTexts[0], "one\n二行目")
	}
}

func TestNavMode_EscCancelsVisualSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.visualMode = visualModeLine
	m.visualStart = visualPosition{line: 1, col: 0}
	m.cursorLine = 2

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(Model)
	if got.visualMode != visualModeOff {
		t.Fatalf("visualMode = %d, want %d", got.visualMode, visualModeOff)
	}
	if !got.navigationMode {
		t.Fatal("Esc in visual mode should stay in navigation mode")
	}
}

func TestNavMode_CharVisualSelectionCopiesRange(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 20, height: 5}
	m.rawLines = []string{"hello", "world"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorLine = 0
	m.cursorCol = 1

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	if m.visualMode != visualModeChar {
		t.Fatalf("visualMode = %d, want %d", m.visualMode, visualModeChar)
	}
	if m.visualStart.col != 1 {
		t.Fatalf("visualStart.col = %d, want 1", m.visualStart.col)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)

	if len(agent.copyTexts) != 1 {
		t.Fatalf("copyTexts len = %d, want 1", len(agent.copyTexts))
	}
	if agent.copyTexts[0] != "ell" {
		t.Fatalf("copied text = %q, want %q", agent.copyTexts[0], "ell")
	}
	if m.visualMode != visualModeOff {
		t.Fatalf("visualMode after copy = %d, want %d", m.visualMode, visualModeOff)
	}
}

func TestNavMode_CharVisualSelectionSupportsLineMoveAndColumnClamp(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 20, height: 5}
	m.rawLines = []string{"abcdef", "xy"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorCol = 4

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	if m.cursorLine != 1 {
		t.Fatalf("cursorLine = %d, want 1", m.cursorLine)
	}
	if m.cursorCol != 1 {
		t.Fatalf("cursorCol = %d, want 1", m.cursorCol)
	}
}

func TestNavMode_HLMoveCursorColWithoutVisualMode(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 20, height: 5}
	m.rawLines = []string{"hello"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = updated.(Model)

	if m.cursorCol != 1 {
		t.Fatalf("cursorCol = %d, want 1", m.cursorCol)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	if m.visualStart.col != 1 {
		t.Fatalf("visualStart.col = %d, want 1", m.visualStart.col)
	}
}

func TestNavMode_ZeroMovesToLineStartWithoutCount(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 20, height: 5}
	m.rawLines = []string{"hello world"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorCol = 5

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	m = updated.(Model)
	if m.cursorCol != 0 {
		t.Fatalf("cursorCol = %d, want 0", m.cursorCol)
	}
}

func TestNavMode_CountCanIncludeZero(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 5}
	setModelRawLines(&m, 20)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'0'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.cursorLine != 10 {
		t.Fatalf("cursorLine = %d, want 10", m.cursorLine)
	}
}

func TestNavMode_WBEWordMotions(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 30, height: 5}
	m.rawLines = []string{"alpha beta gamma"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = updated.(Model)
	if m.cursorCol != 6 {
		t.Fatalf("cursorCol after w = %d, want 6", m.cursorCol)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = updated.(Model)
	if m.cursorCol != 9 {
		t.Fatalf("cursorCol after e = %d, want 9", m.cursorCol)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = updated.(Model)
	if m.cursorCol != 6 {
		t.Fatalf("cursorCol after b = %d, want 6", m.cursorCol)
	}
}

func TestNavMode_WordMotionsWorkInVisualMode(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 30, height: 5}
	m.rawLines = []string{"alpha beta gamma"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)

	if len(agent.copyTexts) != 1 {
		t.Fatalf("copyTexts len = %d, want 1", len(agent.copyTexts))
	}
	if agent.copyTexts[0] != "alpha beta" {
		t.Fatalf("copied text = %q, want %q", agent.copyTexts[0], "alpha beta")
	}
}

func TestNavMode_LineStartAndEndMotions(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 30, height: 5}
	m.rawLines = []string{"  alpha", " beta", "gamma"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'^'}})
	m = updated.(Model)
	if m.cursorCol != 2 {
		t.Fatalf("cursorCol after ^ = %d, want 2", m.cursorCol)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'$'}})
	m = updated.(Model)
	if m.cursorCol != 6 {
		t.Fatalf("cursorCol after $ = %d, want 6", m.cursorCol)
	}

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'$'}})
	m = updated.(Model)
	if m.cursorLine != 1 {
		t.Fatalf("cursorLine after 2$ = %d, want 1", m.cursorLine)
	}
	if m.cursorCol != 4 {
		t.Fatalf("cursorCol after 2$ = %d, want 4", m.cursorCol)
	}
}

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

func TestNavMode_VerticalMoveClampsCursorToVisibleWidth(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.width = 5
	m.height = 8
	m.vp = lightViewport{width: 5, height: 4}
	m.rawLines = []string{"abcdefghij", "klmnopqrst"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorLine = 0
	m.cursorCol = 4

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	if m.cursorLine != 1 {
		t.Fatalf("cursorLine = %d, want 1", m.cursorLine)
	}
	if m.cursorCol != 4 {
		t.Fatalf("cursorCol = %d, want 4", m.cursorCol)
	}

	m.cursorCol = 8
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = updated.(Model)

	if m.cursorLine != 0 {
		t.Fatalf("cursorLine after k = %d, want 0", m.cursorLine)
	}
	if m.cursorCol != 8 {
		t.Fatalf("cursorCol after k = %d, want 8", m.cursorCol)
	}

	m.rebuildChrome()
	view := m.View()
	if !strings.Contains(view, "\033[48;5;255;38;5;16mi") {
		t.Fatalf("view should contain visible cursor after vertical move, got %q", view)
	}
}

func TestNavMode_WordMotionKeepsLogicalColumnOnLongLine(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.width = 5
	m.height = 8
	m.vp = lightViewport{width: 5, height: 4}
	m.rawLines = []string{"abcde fg", "tail"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorLine = 0
	m.cursorCol = 4
	m.rebuildChrome()

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	m = updated.(Model)

	if m.cursorLine != 0 {
		t.Fatalf("cursorLine after w = %d, want 0", m.cursorLine)
	}
	if m.cursorCol != 6 {
		t.Fatalf("cursorCol after w = %d, want 6", m.cursorCol)
	}

	view := m.View()
	if !strings.Contains(view, "\033[48;5;255;38;5;16mf") {
		t.Fatalf("view should keep cursor visible at edge for off-screen column, got %q", view)
	}
}

func TestModel_MouseWheelDoesNotChangeVisualSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.vp = lightViewport{width: 20, height: 4}
	setModelRawLines(&m, 20)
	m.cursorLine = 2
	m.cursorCol = 0

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	startBefore := m.visualStart
	cursorLineBefore := m.cursorLine
	cursorColBefore := m.cursorCol
	yOffsetBefore := m.vp.yOffset

	updated, _ = m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	m = updated.(Model)

	if m.vp.yOffset != yOffsetBefore+3 {
		t.Fatalf("yOffset after wheel = %d, want %d", m.vp.yOffset, yOffsetBefore+3)
	}
	if m.visualStart != startBefore {
		t.Fatalf("visualStart changed from %+v to %+v", startBefore, m.visualStart)
	}
	if m.cursorLine != cursorLineBefore {
		t.Fatalf("cursorLine changed from %d to %d during visual wheel scroll", cursorLineBefore, m.cursorLine)
	}
	if m.cursorCol != cursorColBefore {
		t.Fatalf("cursorCol changed from %d to %d during visual wheel scroll", cursorColBefore, m.cursorCol)
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

func TestNavMode_PendingYFallsBackToCopyLastOutput(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 5}
	setModelRawLines(&m, 5)

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	if agent.copyCalls != 1 {
		t.Fatalf("copyCalls = %d, want 1", agent.copyCalls)
	}
	if m.cursorLine != 1 {
		t.Fatalf("cursorLine = %d, want 1", m.cursorLine)
	}
}
