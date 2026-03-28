package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TestModel_KeyDownScrollsViewport は Alternate Scroll Mode (1007) で
// ホイールがカーソルキーに変換された場合に viewport がスクロールすることを検証。
func TestModel_KeyDownScrollsViewport(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.vp = lightViewport{width: 10, height: 5}
	m.vp.setContent(strings.Repeat("line\n", 20))

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyDown})
	got := updated.(Model)

	if got.vp.yOffset != 1 {
		t.Fatalf("yOffset = %d, want 1", got.vp.yOffset)
	}
}

func TestModel_MouseWheelScrollsViewport(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.vp = lightViewport{width: 10, height: 5}
	setModelRawLines(&m, 20)
	m.navigationMode = true
	m.cursorLine = 0

	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	got := updated.(Model)

	if got.vp.yOffset != 3 {
		t.Fatalf("yOffset = %d, want 3", got.vp.yOffset)
	}
	if got.cursorLine != 3 {
		t.Fatalf("cursorLine = %d, want 3", got.cursorLine)
	}
	if got.cursorLine < got.vp.yOffset || got.cursorLine >= got.vp.yOffset+got.vp.height {
		t.Fatalf("cursorLine = %d should stay visible in viewport [%d, %d)", got.cursorLine, got.vp.yOffset, got.vp.yOffset+got.vp.height)
	}
}

func TestModel_MouseWheelToBottomClearsNewOutputBadge(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	for i := 0; i < 40; i++ {
		m.appendContentLines(fmt.Sprintf("line %d", i))
	}

	m.vp.gotoTop()
	m.appendContentLines("new line")
	m.chromeDirty = true
	m.rebuildChrome()
	if !strings.Contains(m.chromeCache, "New output") {
		t.Fatalf("chromeCache should include new output badge, got %q", m.chromeCache)
	}

	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	m = updated.(Model)
	for !m.vp.atBottom() {
		updated, _ = m.Update(tea.MouseMsg{
			Button: tea.MouseButtonWheelDown,
			Action: tea.MouseActionPress,
		})
		m = updated.(Model)
	}

	if m.newOutput {
		t.Fatal("newOutput should be cleared after mouse scrolling to bottom")
	}
	if strings.Contains(m.chromeCache, "New output") {
		t.Fatalf("chromeCache should clear new output badge, got %q", m.chromeCache)
	}
}

func TestModel_HandleKeyMsg_CtrlCRequiresTwoPresses(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyCtrlC})
	m1 := updated.(Model)
	if cmd != nil {
		t.Fatalf("first ctrl+c returned unexpected command")
	}
	if agent.cancelCalls != 0 {
		t.Fatalf("cancelCalls = %d, want 0", agent.cancelCalls)
	}
	if agent.cleanupCalls != 0 {
		t.Fatalf("cleanupCalls = %d, want 0", agent.cleanupCalls)
	}
	if m1.lastInterrupt.IsZero() {
		t.Fatal("expected lastInterrupt to be recorded")
	}

	updated, cmd = m1.handleKeyMsg(tea.KeyMsg{Type: tea.KeyCtrlC})
	m2 := updated.(Model)
	if cmd == nil {
		t.Fatal("second ctrl+c returned nil command, want tea.Quit")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("second ctrl+c returned %T, want tea.QuitMsg", msg)
	}
	if agent.cleanupCalls != 1 {
		t.Fatalf("cleanupCalls = %d, want 1", agent.cleanupCalls)
	}
	if !m2.quitting {
		t.Fatal("expected model to enter quitting state")
	}
}

func TestModel_HandleKeyMsg_CtrlCRestartsWindow(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyCtrlC})
	m1 := updated.(Model)
	m1.lastInterrupt = time.Now().Add(-4 * time.Second)

	_, cmd := m1.handleKeyMsg(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd != nil {
		t.Fatalf("second ctrl+c after timeout returned unexpected command")
	}
	if agent.cleanupCalls != 0 {
		t.Fatalf("cleanupCalls = %d, want 0", agent.cleanupCalls)
	}
}

func TestNewModel_ClearsTextInputPrompt(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	if m.textInput.Prompt != "" {
		t.Fatalf("textInput.Prompt = %q, want empty string", m.textInput.Prompt)
	}
}

func TestModel_View_RendersSinglePromptAndContainsInput(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 80
	m.height = 3
	m.vp = lightViewport{width: m.width, height: 1}
	m.vp.setContent("body")
	m.textInput.SetValue("hello")
	m.padLineCache = strings.Repeat(" ", m.width)
	m.rebuildChrome()

	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) < 3 {
		t.Fatalf("view should have at least 3 lines, got %d: %q", len(lines), view)
	}

	if strings.Count(view, inputPrompt) != 1 {
		t.Fatalf("prompt count = %d, want 1; view=%q", strings.Count(view, inputPrompt), view)
	}
	if !strings.Contains(view, "ready") {
		t.Fatalf("view should contain status line, got %q", view)
	}
	if !strings.Contains(view, "hello") {
		t.Fatalf("view should contain input value, got %q", view)
	}
	if !strings.Contains(view, "/copy") {
		t.Fatalf("view should contain copy hint, got %q", view)
	}
}

func TestModel_WindowResizeRestoresFullLineFromRawContent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 5, Height: 8})
	m = updated.(Model)

	updated, _ = m.Update(AppendMessageMsg{
		Message: ChatMessage{Role: "assistant", Content: "123456789"},
	})
	m = updated.(Model)

	if got := m.getVisualRowContents()[len(m.getVisualRowContents())-1]; got != "6789" {
		t.Fatalf("narrow render last row = %q, want %q", got, "6789")
	}

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 12, Height: 8})
	m = updated.(Model)

	if got := m.rawLines[len(m.rawLines)-1]; got != "123456789" {
		t.Fatalf("raw line = %q, want %q", got, "123456789")
	}
	if got := m.getVisualRowContents()[len(m.getVisualRowContents())-1]; got != "123456789" {
		t.Fatalf("wide render = %q, want %q", got, "123456789")
	}
}

func TestModel_WindowResizeKeepsCharVisualSelectionState(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 6, Height: 8})
	m = updated.(Model)
	m.navigationMode = true
	m.rawLines = []string{"abcdefghij"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.cursorLine = 0
	m.cursorCol = 1

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	m = updated.(Model)
	m.cursorCol = 7
	m.rebuildChrome()

	before := m.View()
	if !strings.Contains(before, "\033[48;5;255;38;5;16mh") {
		t.Fatalf("narrow view should keep cursor visible before resize, got %q", before)
	}

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 12, Height: 8})
	m = updated.(Model)
	after := m.View()

	if m.visualMode != visualModeChar {
		t.Fatalf("visualMode = %d, want %d", m.visualMode, visualModeChar)
	}
	if m.visualStart != (visualPosition{line: 0, col: 1}) {
		t.Fatalf("visualStart = %+v, want {line:0 col:1}", m.visualStart)
	}
	if m.cursorCol != 7 {
		t.Fatalf("cursorCol = %d, want 7", m.cursorCol)
	}
	if !strings.Contains(stripANSI(after), "abcdefgh") {
		t.Fatalf("wide view should restore full visible selection context, got %q", after)
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
	if !strings.Contains(lines[0], "\033[48;5;240mc\033[0m") || !strings.Contains(lines[0], "\033[48;5;240me\033[0m") {
		t.Fatalf("first wrapped row should highlight c..e, got %q", lines[0])
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

func TestModel_WindowResizeKeepsLineVisualSelectionState(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 8, Height: 8})
	m = updated.(Model)
	m.navigationMode = true
	setModelRawLines(&m, 20)
	m.cursorLine = 2

	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'V'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	updated, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 20, Height: 10})
	m = updated.(Model)

	if m.visualMode != visualModeLine {
		t.Fatalf("visualMode = %d, want %d", m.visualMode, visualModeLine)
	}
	if m.visualStart.line != 2 {
		t.Fatalf("visualStart.line = %d, want 2", m.visualStart.line)
	}
	if m.cursorLine != 4 {
		t.Fatalf("cursorLine = %d, want 4", m.cursorLine)
	}
	view := m.View()
	if !strings.Contains(view, "\033[48;5;240m") {
		t.Fatalf("line visual selection should remain highlighted after resize, got %q", view)
	}
}

func TestModel_AppendMessageKeepsViewportAtBottom(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 10, Height: 8})
	m = updated.(Model)

	for i := 0; i < 8; i++ {
		updated, _ = m.Update(AppendMessageMsg{
			Message: ChatMessage{Role: "assistant", Content: "line"},
		})
		m = updated.(Model)
	}

	if !m.vp.atBottom() {
		t.Fatal("viewport should stay pinned to bottom after appends")
	}
}

func TestModel_StreamTextMergesChunksAcrossMessages(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 20, Height: 8})
	m = updated.(Model)

	updated, _ = m.Update(StreamTextMsg{Text: "hello", Done: false})
	m = updated.(Model)
	updated, _ = m.Update(StreamTextMsg{Text: "\nworld", Done: true})
	m = updated.(Model)

	if len(m.rawLines) != 2 {
		t.Fatalf("rawLines len = %d, want 2", len(m.rawLines))
	}
	if m.rawLines[0] != "hello" || m.rawLines[1] != "world" {
		t.Fatalf("rawLines = %#v, want [hello world]", m.rawLines)
	}
	if m.streamingActive {
		t.Fatal("streamingActive should be reset after done")
	}
}

func TestModel_UpdateKeyMsgRebuildsChrome(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rebuildChrome()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m = updated.(Model)

	if !strings.Contains(m.chromeCache, "a") {
		t.Fatalf("chromeCache should include typed input, got %q", m.chromeCache)
	}
}

func TestModel_StatusBarClampedToWidth(t *testing.T) {
	agent := &stubAgent{statusLine: strings.Repeat("status ", 20)}
	m := NewModel(agent, "")
	m.ready = true
	m.width = 20
	m.height = 8
	m.vp = lightViewport{width: 20, height: 4}
	m.padLineCache = strings.Repeat(" ", 20)
	m.rebuildChrome()

	lines := strings.Split(m.chromeCache, "\n")
	if len(lines) != 4 {
		t.Fatalf("chromeCache lines = %d, want 4", len(lines))
	}

	statusLine := stripANSI(lines[len(lines)-1])
	if got := lipgloss.Width(statusLine); got != m.width {
		t.Fatalf("status line width = %d, want %d; line=%q", got, m.width, statusLine)
	}
}

func TestTruncateWithANSI_AppendsResetWhenTruncated(t *testing.T) {
	got := truncateWithANSI("\033[31mabcdef", 3)

	if !strings.HasSuffix(got, "\033[0m") {
		t.Fatalf("truncated line should end with reset, got %q", got)
	}
	if width := lipgloss.Width(got); width != 3 {
		t.Fatalf("rendered width = %d, want 3", width)
	}
}

func TestLightViewport_ViewPadsRowsToWidth(t *testing.T) {
	v := lightViewport{
		width:  8,
		height: 3,
		lines: []string{
			"\033[31mabc\033[0m",
			"日本a",
		},
	}

	lines := strings.Split(v.view(), "\n")
	if len(lines) != 3 {
		t.Fatalf("view lines = %d, want 3", len(lines))
	}

	for i, line := range lines {
		if got := lipgloss.Width(line); got != v.width {
			t.Fatalf("line %d width = %d, want %d; line=%q", i, got, v.width, line)
		}
	}
}

func TestModel_ViewPadsScrolledRowsToViewportWidth(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 10, Height: 8})
	m = updated.(Model)

	m.appendContentLines(
		"0123456789",
		"shrt",
		"\033[32mgo test ./...\033[0m",
		"日本語mix",
	)
	m.vp.gotoTop()

	body := strings.Split(m.viewportView(), "\n")
	if len(body) != m.vp.height {
		t.Fatalf("body lines = %d, want %d", len(body), m.vp.height)
	}

	for i, line := range body {
		if got := lipgloss.Width(line); got != m.vp.width {
			t.Fatalf("body line %d width = %d, want %d; line=%q", i, got, m.vp.width, line)
		}
	}
}

func TestModel_ScrollingToBottomClearsNewOutputBadge(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	for i := 0; i < 40; i++ {
		m.appendContentLines(fmt.Sprintf("line %d", i))
	}

	m.vp.gotoTop()
	m.appendContentLines("new line")
	m.chromeDirty = true
	m.rebuildChrome()
	if !strings.Contains(m.chromeCache, "New output") {
		t.Fatalf("chromeCache should include new output badge, got %q", m.chromeCache)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = updated.(Model)
	for !m.vp.atBottom() {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
		m = updated.(Model)
	}

	if m.newOutput {
		t.Fatal("newOutput should be cleared after scrolling to bottom")
	}
	if strings.Contains(m.chromeCache, "New output") {
		t.Fatalf("chromeCache should clear new output badge, got %q", m.chromeCache)
	}
}
