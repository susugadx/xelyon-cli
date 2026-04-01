package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- screenToRawPosition ---

func TestScreenToRawPosition_Basic(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 20)
	m.vp.gotoTop()

	rawLine, rawCol, ok := m.screenToRawPosition(3, 0)
	if !ok {
		t.Fatal("expected ok")
	}
	if rawLine != 0 {
		t.Fatalf("rawLine = %d, want 0", rawLine)
	}
	if rawCol != 3 {
		t.Fatalf("rawCol = %d, want 3", rawCol)
	}
}

func TestScreenToRawPosition_WithOffset(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 50)
	m.vp.yOffset = 10

	rawLine, rawCol, ok := m.screenToRawPosition(3, 2)
	if !ok {
		t.Fatal("expected ok")
	}
	// Visual row 12 should map to rawLine 12 (no wrapping for short lines)
	if rawLine != 12 {
		t.Fatalf("rawLine = %d, want 12", rawLine)
	}
	if rawCol != 3 {
		t.Fatalf("rawCol = %d, want 3", rawCol)
	}
}

func TestScreenToRawPosition_ClampsToContent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	// "line0" is 5 chars, maxCol = 4
	setModelRawLines(&m, 5)

	rawLine, rawCol, ok := m.screenToRawPosition(99, 0)
	if !ok {
		t.Fatal("expected ok")
	}
	if rawLine != 0 {
		t.Fatalf("rawLine = %d, want 0", rawLine)
	}
	// Should be clamped to maxCol (4 for "line0")
	if rawCol > 4 {
		t.Fatalf("rawCol = %d, want <= 4", rawCol)
	}
}

func TestScreenToRawPosition_EmptyLayout(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	_, _, ok := m.screenToRawPosition(0, 0)
	if ok {
		t.Fatal("expected !ok for empty layout")
	}
}

func TestScreenToRawPosition_NegativeY(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.vp.yOffset = 5

	rawLine, _, ok := m.screenToRawPosition(0, -3)
	if !ok {
		t.Fatal("expected ok")
	}
	// Y=-3 + offset=5 → visRow=2, which should map to rawLine 2
	if rawLine != 2 {
		t.Fatalf("rawLine = %d, want 2", rawLine)
	}
}

// --- mouseSelectionColumnsForLine ---

func TestMouseSelectionColumnsForLine_SingleLine(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.mouseSelAnchor = visualPosition{line: 3, col: 2}
	m.mouseSelEnd = visualPosition{line: 3, col: 4}

	start, end, ok := m.mouseSelectionColumnsForLine(3)
	if !ok {
		t.Fatal("expected ok")
	}
	if start != 2 {
		t.Fatalf("startCol = %d, want 2", start)
	}
	if end != 5 { // endCol = col+1 (exclusive)
		t.Fatalf("endCol = %d, want 5", end)
	}
}

func TestMouseSelectionColumnsForLine_MultiLine(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.mouseSelAnchor = visualPosition{line: 2, col: 1}
	m.mouseSelEnd = visualPosition{line: 5, col: 3}

	// First line
	start, _, ok := m.mouseSelectionColumnsForLine(2)
	if !ok {
		t.Fatal("expected ok for first line")
	}
	if start != 1 {
		t.Fatalf("first line startCol = %d, want 1", start)
	}

	// Intermediate line
	start, end, ok := m.mouseSelectionColumnsForLine(3)
	if !ok {
		t.Fatal("expected ok for intermediate line")
	}
	if start != 0 {
		t.Fatalf("intermediate startCol = %d, want 0", start)
	}
	if end != 9999 {
		t.Fatalf("intermediate endCol = %d, want 9999", end)
	}

	// Last line
	_, end, ok = m.mouseSelectionColumnsForLine(5)
	if !ok {
		t.Fatal("expected ok for last line")
	}
	if end != 4 { // endCol = col+1
		t.Fatalf("last line endCol = %d, want 4", end)
	}

	// Outside selection
	_, _, ok = m.mouseSelectionColumnsForLine(1)
	if ok {
		t.Fatal("expected !ok for line before selection")
	}
	_, _, ok = m.mouseSelectionColumnsForLine(6)
	if ok {
		t.Fatal("expected !ok for line after selection")
	}
}

func TestMouseSelectionColumnsForLine_ReversedSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	// Anchor is after end (drag upward)
	m.mouseSelAnchor = visualPosition{line: 5, col: 3}
	m.mouseSelEnd = visualPosition{line: 2, col: 1}

	// Should still work due to normalization
	start, _, ok := m.mouseSelectionColumnsForLine(2)
	if !ok {
		t.Fatal("expected ok")
	}
	if start != 1 {
		t.Fatalf("startCol = %d, want 1", start)
	}
}

// --- mouseSelectionText ---

func TestMouseSelectionText_SingleLine(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"Hello, World!"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 4}

	text, lines := m.mouseSelectionText()
	if text != "Hello" {
		t.Fatalf("text = %q, want %q", text, "Hello")
	}
	if lines != 1 {
		t.Fatalf("lines = %d, want 1", lines)
	}
}

func TestMouseSelectionText_MultiLine(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"first line", "second line", "third line"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.mouseSelAnchor = visualPosition{line: 0, col: 6}
	m.mouseSelEnd = visualPosition{line: 2, col: 4}

	text, lines := m.mouseSelectionText()
	expected := "line\nsecond line\nthird"
	if text != expected {
		t.Fatalf("text = %q, want %q", text, expected)
	}
	if lines != 3 {
		t.Fatalf("lines = %d, want 3", lines)
	}
}

func TestMouseSelectionText_WithANSI(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"\033[31mred text\033[0m"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 2}

	text, _ := m.mouseSelectionText()
	if text != "red" {
		t.Fatalf("text = %q, want %q (ANSI should be stripped)", text, "red")
	}
}

// --- normalizedMouseSelection ---

func TestNormalizedMouseSelection_NoSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	_, _, ok := m.normalizedMouseSelection()
	if ok {
		t.Fatal("expected !ok when no selection")
	}
}

func TestNormalizedMouseSelection_Reversed(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.mouseSelAnchor = visualPosition{line: 5, col: 3}
	m.mouseSelEnd = visualPosition{line: 2, col: 1}

	start, end, ok := m.normalizedMouseSelection()
	if !ok {
		t.Fatal("expected ok")
	}
	if start.line != 2 || start.col != 1 {
		t.Fatalf("start = %v, want {2, 1}", start)
	}
	if end.line != 5 || end.col != 3 {
		t.Fatalf("end = %v, want {5, 3}", end)
	}
}

// --- hasActiveMouseSelection ---

func TestHasActiveMouseSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	if m.hasActiveMouseSelection() {
		t.Fatal("expected false for initial state")
	}

	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 0}
	if m.hasActiveMouseSelection() {
		t.Fatal("expected false when anchor == end")
	}

	m.mouseSelEnd = visualPosition{line: 0, col: 5}
	if !m.hasActiveMouseSelection() {
		t.Fatal("expected true when anchor != end")
	}
}

// --- Mouse drag flow ---

func TestMouseDrag_BasicFlow(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 20)
	m.vp.gotoTop()
	m.rebuildChrome()

	// Press
	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
		X:      5, Y: 2,
	})
	m = updated.(Model)
	if !m.mouseDragging {
		t.Fatal("expected mouseDragging after press")
	}
	if m.mouseSelAnchor.line < 0 {
		t.Fatal("expected anchor to be set")
	}

	// Motion
	updated, _ = m.Update(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionMotion,
		X:      10, Y: 5,
	})
	m = updated.(Model)
	if !m.mouseDragging {
		t.Fatal("expected still dragging")
	}
	if m.mouseSelAnchor == m.mouseSelEnd {
		t.Fatal("expected selection to differ from anchor")
	}

	// Release
	updated, _ = m.Update(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
		X:      10, Y: 5,
	})
	m = updated.(Model)
	if m.mouseDragging {
		t.Fatal("expected dragging to stop after release")
	}
	if !m.hasActiveMouseSelection() {
		t.Fatal("expected active selection after drag")
	}
}

func TestMouseDrag_ClickClearsSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 20)
	m.vp.gotoTop()
	m.rebuildChrome()

	// Set an existing selection
	m.mouseSelAnchor = visualPosition{line: 1, col: 0}
	m.mouseSelEnd = visualPosition{line: 3, col: 5}

	// Click (press+release at same position)
	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
		X:      2, Y: 4,
	})
	m = updated.(Model)
	updated, _ = m.Update(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionRelease,
		X:      2, Y: 4,
	})
	m = updated.(Model)

	if m.hasActiveMouseSelection() {
		t.Fatal("expected selection cleared after click without drag")
	}
}

func TestMouseDrag_InputDockIgnored(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 20)
	m.vp.gotoTop()
	m.rebuildChrome()

	// Press in input dock area (Y >= viewport height)
	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
		X:      5, Y: m.vp.height + 1,
	})
	m = updated.(Model)
	if m.mouseDragging {
		t.Fatal("should not start drag in input dock area")
	}
}

func TestMouseDrag_ClearsKeyboardVisualSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 20)
	m.vp.gotoTop()
	m.rebuildChrome()
	m.navigationMode = true
	m.visualMode = visualModeChar
	m.visualStart = visualPosition{line: 1, col: 0}

	// Start mouse drag
	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
		X:      2, Y: 3,
	})
	m = updated.(Model)

	if m.visualMode != visualModeOff {
		t.Fatal("expected keyboard visual selection to be cleared on drag start")
	}
}

// --- Auto-scroll ---

func TestAutoScrollAmount(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.vp.height = 20

	tests := []struct {
		name     string
		y        int
		wantSign int // -1=up, 0=none, 1=down
	}{
		{"top edge y=0", 0, -1},
		{"top edge y=1", 1, -1},
		{"top edge y=2", 2, -1},
		{"middle y=5", 5, 0},
		{"middle y=10", 10, 0},
		{"bottom edge y=17", 17, 1},
		{"bottom edge y=18", 18, 1},
		{"bottom edge y=19", 19, 1},
		{"below viewport y=20", 20, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.mouseLastScreenY = tt.y
			amount := m.autoScrollAmount()
			switch {
			case tt.wantSign < 0 && amount >= 0:
				t.Fatalf("amount = %d, want negative", amount)
			case tt.wantSign == 0 && amount != 0:
				t.Fatalf("amount = %d, want 0", amount)
			case tt.wantSign > 0 && amount <= 0:
				t.Fatalf("amount = %d, want positive", amount)
			}
		})
	}
}

func TestAutoScroll_SpeedIncreases(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.vp.height = 20

	// Closer to edge = faster
	m.mouseLastScreenY = 2
	slow := m.autoScrollAmount()
	m.mouseLastScreenY = 0
	fast := m.autoScrollAmount()

	if fast >= slow { // both negative, fast should be more negative
		t.Fatalf("speed at y=0 (%d) should be faster than y=2 (%d)", fast, slow)
	}
}

func TestHandleAutoScroll_StopsWhenNotDragging(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 50)
	m.mouseDragging = false
	m.mouseAutoScrolling = true

	cmd := m.handleAutoScroll()
	if cmd != nil {
		t.Fatal("expected nil cmd when not dragging")
	}
	if m.mouseAutoScrolling {
		t.Fatal("expected mouseAutoScrolling to be cleared")
	}
}

func TestHandleAutoScroll_ScrollsAndUpdates(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 50)
	m.vp.gotoTop()
	m.mouseDragging = true
	m.mouseAutoScrolling = true
	m.mouseLastScreenY = m.vp.height // Below viewport
	m.mouseLastScreenX = 5
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 5}

	oldOffset := m.vp.yOffset
	cmd := m.handleAutoScroll()

	if cmd == nil {
		t.Fatal("expected non-nil cmd for continued scrolling")
	}
	if m.vp.yOffset <= oldOffset {
		t.Fatalf("expected viewport to scroll down: offset %d -> %d", oldOffset, m.vp.yOffset)
	}
	if m.mouseSelEnd.line == 0 && m.mouseSelEnd.col == 5 {
		t.Fatal("expected selection end to update after scroll")
	}
}

// --- Copy operations ---

func TestCtrlC_CopiesMouseSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"Hello, World!"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.rebuildChrome()
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 4}

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if len(agent.copyTexts) != 1 || agent.copyTexts[0] != "Hello" {
		t.Fatalf("copyTexts = %v, want [\"Hello\"]", agent.copyTexts)
	}
	if m.hasActiveMouseSelection() {
		t.Fatal("expected selection cleared after copy")
	}
}

func TestCtrlC_CopyTakesPriorityOverInterrupt(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	agent.setProcessing(true)
	m := newModelWithViewport(agent)
	m.rawLines = []string{"Hello"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.rebuildChrome()
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 4}

	m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyCtrlC})

	// マウス選択中はコピーが優先（AI回答中でもキャンセルしない）
	if agent.copyCalls != 1 {
		t.Fatalf("copyCalls = %d, want 1 (copy should take priority over interrupt)", agent.copyCalls)
	}
	if agent.cancelCalls != 0 {
		t.Fatalf("cancelCalls = %d, want 0", agent.cancelCalls)
	}
}

func TestNavY_CopiesMouseSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"first", "second", "third"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.rebuildChrome()
	m.navigationMode = true
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 1, col: 5}

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	m = updated.(Model)

	if len(agent.copyTexts) != 1 {
		t.Fatalf("copyTexts = %v, want 1 entry", agent.copyTexts)
	}
	expected := "first\nsecond"
	if agent.copyTexts[0] != expected {
		t.Fatalf("copyTexts[0] = %q, want %q", agent.copyTexts[0], expected)
	}
}

func TestCopyCommand_CopiesMouseSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"copy me"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.rebuildChrome()
	m.textInput.SetValue("/copy")
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 6}

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if len(agent.copyTexts) != 1 || agent.copyTexts[0] != "copy me" {
		t.Fatalf("copyTexts = %v, want [\"copy me\"]", agent.copyTexts)
	}
}

// --- Esc clears selection ---

func TestEsc_ClearsMouseSelection_InputMode(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.rebuildChrome()
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 2, col: 5}

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.hasActiveMouseSelection() {
		t.Fatal("expected selection cleared by Esc")
	}
	if m.navigationMode {
		t.Fatal("Esc should clear selection first, not enter nav mode")
	}
}

func TestEsc_ClearsMouseSelection_NavMode(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.rebuildChrome()
	m.navigationMode = true
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 2, col: 5}

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.hasActiveMouseSelection() {
		t.Fatal("expected selection cleared by Esc in nav mode")
	}
	if !m.navigationMode {
		t.Fatal("should stay in nav mode after clearing selection")
	}
}

// --- Conflict resolution ---

func TestVisualV_ClearsMouseSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.rebuildChrome()
	m.navigationMode = true
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 2, col: 5}

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)

	if m.hasActiveMouseSelection() {
		t.Fatal("expected mouse selection cleared when entering visual mode")
	}
}

func TestVisualV_Line_ClearsMouseSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.rebuildChrome()
	m.navigationMode = true
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 2, col: 5}

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	m = updated.(Model)

	if m.hasActiveMouseSelection() {
		t.Fatal("expected mouse selection cleared when entering visual line mode")
	}
}

// --- Selection rendering ---

func TestViewportView_ShowsMouseSelectionOverlay(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"line zero", "line one", "line two"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 1, col: 3}

	view := m.viewportView()
	// Should contain selection background escape code
	if !strings.Contains(view, "\033[48;5;240m") {
		t.Fatal("expected mouse selection background in viewport view")
	}
}

func TestViewportView_NoOverlayWithoutSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"line zero", "line one"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()

	view := m.viewportView()
	// Should NOT contain selection background (no nav mode, no mouse sel)
	if strings.Contains(view, "\033[48;5;240m") {
		t.Fatal("expected no selection background without active selection")
	}
}

// --- Status bar hints ---

func TestStatusBar_ShowsMouseSelHints(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 2, col: 5}

	bar := m.renderStatusBar()
	if !strings.Contains(bar, "SELECT") {
		t.Fatal("expected SELECT in status bar when mouse selection active")
	}
}

func TestStatusBar_ShowsDragHint(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 10)
	m.mouseDragging = true

	bar := m.renderStatusBar()
	if !strings.Contains(bar, "SELECT") {
		t.Fatal("expected SELECT in status bar during drag")
	}
}

// --- Resize and stream stability ---

func TestMouseSelection_SurvivesResize(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 20)
	m.rebuildChrome()
	m.mouseSelAnchor = visualPosition{line: 3, col: 2}
	m.mouseSelEnd = visualPosition{line: 7, col: 5}

	// Simulate resize
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 25})
	m = updated.(Model)

	// Completed selection should survive resize
	if !m.hasActiveMouseSelection() {
		t.Fatal("expected selection to survive resize")
	}
	if m.mouseSelAnchor.line != 3 || m.mouseSelEnd.line != 7 {
		t.Fatal("expected selection positions to be preserved")
	}
}

func TestMouseDrag_StopsOnResize(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 20)
	m.rebuildChrome()
	m.mouseDragging = true
	m.mouseAutoScrolling = true

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 25})
	m = updated.(Model)

	if m.mouseDragging {
		t.Fatal("expected drag to stop on resize")
	}
	if m.mouseAutoScrolling {
		t.Fatal("expected auto-scroll to stop on resize")
	}
}

func TestMouseSelection_SurvivesStreamUpdate(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 20)
	m.rebuildChrome()
	m.mouseSelAnchor = visualPosition{line: 3, col: 2}
	m.mouseSelEnd = visualPosition{line: 7, col: 5}

	// Append new content
	m.appendContentLines("new line appended")

	if !m.hasActiveMouseSelection() {
		t.Fatal("expected selection to survive stream update")
	}
	if m.mouseSelAnchor.line != 3 || m.mouseSelEnd.line != 7 {
		t.Fatal("expected selection positions to be preserved after stream update")
	}
}

// --- mouseSelectionColumnsForVisualRow ---

func TestMouseSelectionColumnsForVisualRow_Basic(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"short"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.mouseSelAnchor = visualPosition{line: 0, col: 1}
	m.mouseSelEnd = visualPosition{line: 0, col: 3}

	startCol, endCol, ok := m.mouseSelectionColumnsForLine(0)
	if !ok {
		t.Fatal("expected ok")
	}

	localStart, localEnd := m.mouseSelectionColumnsForVisualRow(0, 0, startCol, endCol)
	if localStart != 1 {
		t.Fatalf("localStart = %d, want 1", localStart)
	}
	if localEnd != 4 {
		t.Fatalf("localEnd = %d, want 4", localEnd)
	}
}

// --- Wheel scroll preserved ---

func TestWheelScroll_PreservedWithSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	setModelRawLines(&m, 50)
	m.vp.gotoTop()
	m.rebuildChrome()
	m.mouseSelAnchor = visualPosition{line: 5, col: 0}
	m.mouseSelEnd = visualPosition{line: 10, col: 5}

	oldOffset := m.vp.yOffset
	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	m = updated.(Model)

	if m.vp.yOffset <= oldOffset {
		t.Fatal("expected wheel scroll to work with active selection")
	}
	if !m.hasActiveMouseSelection() {
		t.Fatal("expected selection to survive wheel scroll")
	}
}

// --- Rendering precision tests ---

// helperSplitViewLines splits viewport output into per-line strings.
func helperSplitViewLines(view string, height int) []string {
	lines := strings.SplitN(view, "\n", height+1)
	if len(lines) > height {
		lines = lines[:height]
	}
	return lines
}

func TestRender_EachLineExactWidth_ASCII(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"Hello World", "Second line here", "Third"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 2}
	m.mouseSelEnd = visualPosition{line: 2, col: 3}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != m.vp.width {
			t.Errorf("line %d: lipgloss.Width = %d, want %d", i, w, m.vp.width)
		}
	}
}

func TestRender_EachLineExactWidth_CJK(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"日本語テスト", "abc日本語def", "テスト完了"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 2, col: 7}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != m.vp.width {
			t.Errorf("line %d: lipgloss.Width = %d, want %d (line=%q)", i, w, m.vp.width, line)
		}
	}
}

func TestRender_EachLineExactWidth_ANSI(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{
		"\033[31mred text\033[0m normal",
		"\033[1;32mbold green\033[0m end",
		"plain text",
	}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 3}
	m.mouseSelEnd = visualPosition{line: 2, col: 5}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != m.vp.width {
			t.Errorf("line %d: lipgloss.Width = %d, want %d", i, w, m.vp.width)
		}
	}
}

func TestRender_EachLineExactWidth_MixedANSICJK(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{
		"\033[31m日本語\033[0mテスト",
		"\033[34mABC\033[0m全角ＤＥＦ",
	}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 2}
	m.mouseSelEnd = visualPosition{line: 1, col: 7}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w != m.vp.width {
			t.Errorf("line %d: lipgloss.Width = %d, want %d (line=%q)", i, w, m.vp.width, line)
		}
	}
}

func TestRender_SelectionOnlyOnSelectedLines(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"line0", "line1", "line2", "line3", "line4"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 1, col: 0}
	m.mouseSelEnd = visualPosition{line: 3, col: 4}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)

	// Line 0 (not selected) should NOT contain selection bg
	if strings.Contains(lines[0], mouseSelBg) {
		t.Error("line 0 should not have selection background")
	}
	// Lines 1-3 should contain selection bg
	for i := 1; i <= 3; i++ {
		if !strings.Contains(lines[i], mouseSelBg) {
			t.Errorf("line %d should have selection background", i)
		}
	}
	// Line 4 should NOT contain selection bg
	if strings.Contains(lines[4], mouseSelBg) {
		t.Error("line 4 should not have selection background")
	}
}

func TestRender_CJKBoundary_SelectionHighlightsWholeChar(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"ab日cd"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	// col 2-3 = 日 (width 2), so selecting col 2 to col 3 should highlight 日
	m.mouseSelAnchor = visualPosition{line: 0, col: 2}
	m.mouseSelEnd = visualPosition{line: 0, col: 3}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)
	line0 := lines[0]

	// The selection bg should appear exactly once for the 日 character
	if !strings.Contains(line0, mouseSelBg) {
		t.Fatal("expected selection background for CJK char")
	}
	// The 'a' and 'b' before should NOT be highlighted
	plainBefore := stripANSI(line0)
	if !strings.HasPrefix(plainBefore, "ab") {
		t.Fatalf("expected plain content to start with 'ab', got %q", plainBefore)
	}
}

func TestRender_ANSIResetBalanced(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"Hello World"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 2}
	m.mouseSelEnd = visualPosition{line: 0, col: 7}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)
	line0 := lines[0]

	// Every mouseSelBg should be followed by a reset
	bgCount := strings.Count(line0, mouseSelBg)
	resetCount := strings.Count(line0, "\033[0m")
	if bgCount > 0 && resetCount < bgCount {
		t.Errorf("ANSI unbalanced: %d bg opens, %d resets", bgCount, resetCount)
	}
}

func TestRender_EmptyLineInSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"hello", "", "world"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 2, col: 4}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)

	// All 3 lines should have correct width (empty line too)
	for i := 0; i < 3; i++ {
		w := lipgloss.Width(lines[i])
		if w != m.vp.width {
			t.Errorf("line %d: width = %d, want %d", i, w, m.vp.width)
		}
	}
}

func TestRender_NonSelectedLineSameAsNormal(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"\033[31mcolored text\033[0m", "plain text", "another"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()

	// Capture normal rendering
	normalView := m.vp.view()
	normalLines := helperSplitViewLines(normalView, m.vp.height)

	// Set selection on line 1 only
	m.mouseSelAnchor = visualPosition{line: 1, col: 0}
	m.mouseSelEnd = visualPosition{line: 1, col: 5}
	selView := m.renderViewportWithMouseSelection()
	selLines := helperSplitViewLines(selView, m.vp.height)

	// Line 0 (not in selection) should be identical to normal rendering
	if selLines[0] != normalLines[0] {
		t.Errorf("non-selected line differs from normal:\n  normal: %q\n  select: %q", normalLines[0], selLines[0])
	}
	// Line 2 (not in selection) should also be identical
	if selLines[2] != normalLines[2] {
		t.Errorf("non-selected line 2 differs from normal:\n  normal: %q\n  select: %q", normalLines[2], selLines[2])
	}
}

func TestRender_SelectedLineContentPreserved(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"Hello, World!"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 12}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)
	plain := stripANSI(lines[0])
	trimmed := strings.TrimRight(plain, " ")
	if trimmed != "Hello, World!" {
		t.Errorf("plain content = %q, want %q", trimmed, "Hello, World!")
	}
}

func TestRender_WrappedLine_SelectionAcrossBoundary(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 20
	m.height = 15
	m.vp = lightViewport{width: 20, height: m.height - m.footerHeight()}
	m.padLineCache = strings.Repeat(" ", 20)
	// 30 chars wraps to 2 visual rows at width 20
	m.rawLines = []string{"ABCDEFGHIJKLMNOPQRSTUVWXYZ1234"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()

	if len(m.layout.Rows) < 2 {
		t.Fatalf("expected >=2 visual rows, got %d", len(m.layout.Rows))
	}

	// Selection spans the wrap boundary (rawCol 15 to 25)
	m.mouseSelAnchor = visualPosition{line: 0, col: 15}
	m.mouseSelEnd = visualPosition{line: 0, col: 25}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)

	// Both visual rows should have exact width
	for i := 0; i < 2; i++ {
		w := lipgloss.Width(lines[i])
		if w != 20 {
			t.Errorf("visual row %d: width = %d, want 20 (line=%q)", i, w, lines[i])
		}
	}
	// Row 0 should have selection (covers cols 15-19)
	if !strings.Contains(lines[0], mouseSelBg) {
		t.Error("visual row 0 should have selection background")
	}
	// Row 1 should also have selection (covers cols 20-25 mapped to row 1)
	if !strings.Contains(lines[1], mouseSelBg) {
		t.Error("visual row 1 should have selection background")
	}
}

func TestRender_WrappedLine_SelectionOnlyOnFirstRow(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 20
	m.height = 15
	m.vp = lightViewport{width: 20, height: m.height - m.footerHeight()}
	m.padLineCache = strings.Repeat(" ", 20)
	m.rawLines = []string{"ABCDEFGHIJKLMNOPQRSTUVWXYZ1234"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()

	// Selection only on first visual row (rawCol 2 to 8)
	m.mouseSelAnchor = visualPosition{line: 0, col: 2}
	m.mouseSelEnd = visualPosition{line: 0, col: 8}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)

	// Row 0 should have selection
	if !strings.Contains(lines[0], mouseSelBg) {
		t.Error("visual row 0 should have selection background")
	}
	// Row 1 should NOT have selection
	if strings.Contains(lines[1], mouseSelBg) {
		t.Error("visual row 1 should NOT have selection background")
	}

	// Both rows should have exact width
	for i := 0; i < 2; i++ {
		w := lipgloss.Width(lines[i])
		if w != 20 {
			t.Errorf("visual row %d: width = %d, want 20", i, w)
		}
	}
}

func TestRender_WrappedCJKLine_Width(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 20
	m.height = 15
	m.vp = lightViewport{width: 20, height: m.height - m.footerHeight()}
	m.padLineCache = strings.Repeat(" ", 20)
	// 10 CJK chars = 20 display cols, fits exactly in one row
	// 11 CJK chars = 22 display cols, wraps to 2 rows
	m.rawLines = []string{"日本語テストABC全角文字"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()

	// Select the entire line
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	maxCol := m.maxCursorColForLine(0)
	m.mouseSelEnd = visualPosition{line: 0, col: maxCol}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)

	// All visible rows should have exact width
	nRows := len(m.layout.Rows)
	for i := 0; i < nRows && i < len(lines); i++ {
		w := lipgloss.Width(lines[i])
		if w != 20 {
			t.Errorf("visual row %d: width = %d, want 20 (line=%q)", i, w, lines[i])
		}
	}

	// Plain content should be preserved
	var allPlain strings.Builder
	for i := 0; i < nRows && i < len(lines); i++ {
		allPlain.WriteString(strings.TrimRight(stripANSI(lines[i]), " "))
	}
	expectedPlain := stripANSI(m.rawLines[0])
	if allPlain.String() != expectedPlain {
		t.Errorf("concatenated plain = %q, want %q", allPlain.String(), expectedPlain)
	}
}

func TestRender_IntermediateLineFull(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"first", "INTERMEDIATE", "last"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.vp.gotoTop()
	// Selection from first to last: intermediate should be fully highlighted
	m.mouseSelAnchor = visualPosition{line: 0, col: 2}
	m.mouseSelEnd = visualPosition{line: 2, col: 2}

	view := m.renderViewportWithMouseSelection()
	lines := helperSplitViewLines(view, m.vp.height)

	// Intermediate line (index 1) should be fully highlighted
	plain1 := stripANSI(lines[1])
	trimmed1 := strings.TrimRight(plain1, " ")
	if trimmed1 != "INTERMEDIATE" {
		t.Errorf("intermediate plain = %q, want %q", trimmed1, "INTERMEDIATE")
	}
	// Should have selection bg
	if !strings.Contains(lines[1], mouseSelBg) {
		t.Error("intermediate line should have selection background")
	}
	// Width should be correct
	w := lipgloss.Width(lines[1])
	if w != m.vp.width {
		t.Errorf("intermediate width = %d, want %d", w, m.vp.width)
	}
}

func TestStylePlainTextRange_RangeBasedANSI(t *testing.T) {
	// stylePlainTextRange should emit bg once at range start and reset once at end,
	// NOT per-character wrapping that causes flickering between characters.
	styled := stylePlainTextRange("ABCDE", 1, 4, "\033[48;5;240m")
	// Expected: A + bg + BCD + reset + E (single bg span)
	// NOT: A + bg+B+reset + bg+C+reset + bg+D+reset + E

	bgCount := strings.Count(styled, "\033[48;5;240m")
	if bgCount != 1 {
		t.Errorf("expected 1 bg open, got %d (styled=%q) — per-character wrapping causes flicker", bgCount, styled)
	}
	resetCount := strings.Count(styled, "\033[0m")
	if resetCount != 1 {
		t.Errorf("expected 1 reset, got %d (styled=%q)", resetCount, styled)
	}
	// Width must still match
	if w := lipgloss.Width(styled); w != 5 {
		t.Errorf("width = %d, want 5", w)
	}
	// Plain content preserved
	if p := stripANSI(styled); p != "ABCDE" {
		t.Errorf("plain = %q, want %q", p, "ABCDE")
	}
}

func TestStylePlainTextRange_CJKRangeBased(t *testing.T) {
	// CJK: selecting 本語 from 日本語テスト (cols 2-5 inclusive, endCol=6)
	styled := stylePlainTextRange("日本語テスト", 2, 6, "\033[48;5;240m")
	bgCount := strings.Count(styled, "\033[48;5;240m")
	if bgCount != 1 {
		t.Errorf("expected 1 bg open for CJK range, got %d (styled=%q)", bgCount, styled)
	}
	if p := stripANSI(styled); p != "日本語テスト" {
		t.Errorf("plain = %q, want 日本語テスト", p)
	}
}

func TestStylePlainTextRange_PreservesWidth(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"ASCII", "Hello, World!"},
		{"CJK", "日本語テスト完了"},
		{"Mixed", "abc日本語def"},
		{"Fullwidth", "ＡＢＣ１２３"},
		{"Empty", ""},
		{"SingleCJK", "日"},
		{"Space", "  spaces  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputWidth := lipgloss.Width(tt.input)
			// Full highlight
			styled := stylePlainTextRange(tt.input, 0, 9999, mouseSelBg)
			styledWidth := lipgloss.Width(styled)
			if styledWidth != inputWidth {
				t.Errorf("full highlight: width %d -> %d (styled=%q)", inputWidth, styledWidth, styled)
			}
			// Partial highlight
			mid := inputWidth / 2
			if mid > 0 {
				styled2 := stylePlainTextRange(tt.input, mid, mid+1, mouseSelBg)
				styled2Width := lipgloss.Width(styled2)
				if styled2Width != inputWidth {
					t.Errorf("partial highlight: width %d -> %d", inputWidth, styled2Width)
				}
			}
		})
	}
}

func TestFillANSITextWidth_StyledOutputExactWidth(t *testing.T) {
	tests := []struct {
		name  string
		input string
		width int
	}{
		{"ASCII short", "Hello", 80},
		{"CJK short", "日本語", 80},
		{"CJK exact", "日本語テスト日本語テ", 20}, // 10 CJK = 20 cols
		{"ANSI styled", stylePlainTextRange("abcdef", 2, 5, mouseSelBg), 80},
		{"CJK styled", stylePlainTextRange("日本語テスト", 2, 8, mouseSelBg), 80},
		{"Empty styled", stylePlainTextRange("", 0, 5, mouseSelBg), 80},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fillANSITextWidth(tt.input, tt.width, "")
			w := lipgloss.Width(result)
			if w != tt.width {
				t.Errorf("fillANSITextWidth width = %d, want %d (result=%q)", w, tt.width, result)
			}
		})
	}
}

// --- clearMouseSelection ---

func TestClearMouseSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.mouseSelAnchor = visualPosition{line: 1, col: 2}
	m.mouseSelEnd = visualPosition{line: 3, col: 4}
	m.mouseDragging = true
	m.mouseAutoScrolling = true

	m.clearMouseSelection()

	if m.mouseSelAnchor.line != -1 {
		t.Fatal("expected anchor cleared")
	}
	if m.mouseSelEnd.line != -1 {
		t.Fatal("expected end cleared")
	}
	if m.mouseDragging {
		t.Fatal("expected dragging cleared")
	}
	if m.mouseAutoScrolling {
		t.Fatal("expected auto-scrolling cleared")
	}
}
