package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type stubAgent struct {
	processing   bool
	cancelCalls  int
	cleanupCalls int
	copyCalls    int
	statusLine   string
}

func (s *stubAgent) Chat(input string)             {}
func (s *stubAgent) HandleCommand(cmd string) bool { return false }
func (s *stubAgent) GetStatusLine() string         { return s.statusLine }
func (s *stubAgent) Cancel()                       { s.cancelCalls++ }
func (s *stubAgent) Cleanup()                      { s.cleanupCalls++ }
func (s *stubAgent) IsProcessing() bool            { return s.processing }
func (s *stubAgent) CopyLastOutput() (string, error) {
	s.copyCalls++
	return "Copied 5 lines", nil
}
func (s *stubAgent) CopyText(text string) error { return nil }

// TestModel_KeyDownScrollsViewport は Alternate Scroll Mode (1007) で
// ホイールがカーソルキーに変換された場合に viewport がスクロールすることを検証。
func TestModel_KeyDownScrollsViewport(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.vp = lightViewport{width: 10, height: 5}
	m.vp.setContent(strings.Repeat("line\n", 20))

	// KeyDown → 1行スクロール
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
	m.vp.setContent(strings.Repeat("line\n", 20))

	updated, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	got := updated.(Model)

	if got.vp.yOffset != 3 {
		t.Fatalf("yOffset = %d, want 3", got.vp.yOffset)
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

	// プロンプトが1つだけ表示される
	if strings.Count(view, inputPrompt) != 1 {
		t.Fatalf("prompt count = %d, want 1; view=%q", strings.Count(view, inputPrompt), view)
	}

	// ステータスバーにステータス文字列が含まれる
	if !strings.Contains(view, "ready") {
		t.Fatalf("view should contain status line, got %q", view)
	}

	// 入力欄に入力値が含まれる
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

	if got := m.renderedLines[len(m.renderedLines)-1]; got != "12345" {
		t.Fatalf("narrow render = %q, want %q", got, "12345")
	}

	updated, _ = m.Update(tea.WindowSizeMsg{Width: 12, Height: 8})
	m = updated.(Model)

	if got := m.rawLines[len(m.rawLines)-1]; got != "123456789" {
		t.Fatalf("raw line = %q, want %q", got, "123456789")
	}
	if got := m.renderedLines[len(m.renderedLines)-1]; got != "123456789" {
		t.Fatalf("wide render = %q, want %q", got, "123456789")
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

// --- Navigation Mode tests ---

func TestNavMode_EscEntersNavWhenInputEmpty(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEsc})
	got := updated.(Model)
	if !got.navigationMode {
		t.Fatal("Esc with empty input should enter navigation mode")
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
	m.vp.setContent(strings.Repeat("line\n", 20))

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got := updated.(Model)
	if got.vp.yOffset != 1 {
		t.Fatalf("j: yOffset = %d, want 1", got.vp.yOffset)
	}

	updated, _ = got.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	got = updated.(Model)
	if got.vp.yOffset != 0 {
		t.Fatalf("k: yOffset = %d, want 0", got.vp.yOffset)
	}
}

func TestNavMode_DUHalfPage(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 10}
	m.vp.setContent(strings.Repeat("line\n", 40))

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	got := updated.(Model)
	if got.vp.yOffset != 5 {
		t.Fatalf("d: yOffset = %d, want 5", got.vp.yOffset)
	}

	updated, _ = got.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	got = updated.(Model)
	if got.vp.yOffset != 0 {
		t.Fatalf("u: yOffset = %d, want 0", got.vp.yOffset)
	}
}

func TestNavMode_GGAndG(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := NewModel(agent, "")
	m.ready = true
	m.navigationMode = true
	m.vp = lightViewport{width: 10, height: 5}
	m.vp.setContent(strings.Repeat("line\n", 20))

	// G → gotoBottom
	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	got := updated.(Model)
	if !got.vp.atBottom() {
		t.Fatal("G should go to bottom")
	}

	// gg → gotoTop
	updated, _ = got.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	got = updated.(Model)
	if !got.gPressed {
		t.Fatal("first g should set gPressed")
	}
	updated, _ = got.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	got = updated.(Model)
	if got.vp.yOffset != 0 {
		t.Fatalf("gg: yOffset = %d, want 0", got.vp.yOffset)
	}
	if got.gPressed {
		t.Fatal("gPressed should be reset after gg")
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

	updated, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	got := updated.(Model)
	if agent.copyCalls != 1 {
		t.Fatalf("copyCalls = %d, want 1", agent.copyCalls)
	}
	if got.transientStatus == "" {
		t.Fatal("transientStatus should be set after copy")
	}
}

// --- Tool Block tests ---

func newModelWithViewport(agent AgentInterface) Model {
	m := NewModel(agent, "")
	m.ready = true
	m.width = 80
	m.height = 30
	m.vp = lightViewport{width: 80, height: 26}
	m.padLineCache = strings.Repeat(" ", 80)
	return m
}

func TestToolBlock_AppendToolResultTracksLineStart(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})

	// 先にテキスト行を追加
	m.appendContentLines("line1", "line2", "line3")
	baseLines := len(m.rawLines)

	m.appendToolResult(ToolResult{
		Name:      "search_code",
		Summary:   "🔍 search_code: test",
		Detail:    "match1\nmatch2",
		Collapsed: true,
	})

	if len(m.toolBlocks) != 1 {
		t.Fatalf("toolBlocks len = %d, want 1", len(m.toolBlocks))
	}
	block := m.toolBlocks[0]
	if block.lineStart != baseLines {
		t.Fatalf("lineStart = %d, want %d", block.lineStart, baseLines)
	}
	if block.lineCount != 1 {
		t.Fatalf("collapsed lineCount = %d, want 1", block.lineCount)
	}
}

func TestToolBlock_ToggleExpandsAndCollapses(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})

	m.appendToolResult(ToolResult{
		Name:      "search_code",
		Summary:   "🔍 search_code: test",
		Detail:    "match1\nmatch2\nmatch3",
		Collapsed: true,
	})

	initialLineCount := len(m.rawLines)
	if m.toolBlocks[0].lineCount != 1 {
		t.Fatalf("collapsed lineCount = %d, want 1", m.toolBlocks[0].lineCount)
	}

	// 展開
	m.toggleToolBlock(0)
	if m.toolBlocks[0].tool.Collapsed {
		t.Fatal("block should be expanded after toggle")
	}
	expandedLineCount := m.toolBlocks[0].lineCount
	if expandedLineCount != 4 { // summary + 3 detail lines
		t.Fatalf("expanded lineCount = %d, want 4", expandedLineCount)
	}
	if len(m.rawLines) != initialLineCount+(expandedLineCount-1) {
		t.Fatalf("rawLines len = %d, want %d", len(m.rawLines), initialLineCount+(expandedLineCount-1))
	}

	// 折りたたみ
	m.toggleToolBlock(0)
	if !m.toolBlocks[0].tool.Collapsed {
		t.Fatal("block should be collapsed after second toggle")
	}
	if len(m.rawLines) != initialLineCount {
		t.Fatalf("rawLines len after re-collapse = %d, want %d", len(m.rawLines), initialLineCount)
	}
}

func TestToolBlock_MultipleBlocksLineStartUpdated(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})

	m.appendToolResult(ToolResult{
		Name: "read_file", Summary: "📄 read_file: a.go",
		Detail: "content1\ncontent2", Collapsed: true,
	})
	m.appendToolResult(ToolResult{
		Name: "search_code", Summary: "🔍 search_code: test",
		Detail: "match1", Collapsed: true,
	})

	block1Start := m.toolBlocks[1].lineStart
	if block1Start != m.toolBlocks[0].lineStart+1 {
		t.Fatalf("second block lineStart = %d, want %d", block1Start, m.toolBlocks[0].lineStart+1)
	}

	// 最初のブロックを展開 → 2番目のブロックの lineStart が更新される
	m.toggleToolBlock(0)
	delta := m.toolBlocks[0].lineCount - 1 // 1行 → N行
	expectedStart := block1Start + delta
	if m.toolBlocks[1].lineStart != expectedStart {
		t.Fatalf("after expand: second block lineStart = %d, want %d", m.toolBlocks[1].lineStart, expectedStart)
	}

	// 折りたたみ → 元に戻る
	m.toggleToolBlock(0)
	if m.toolBlocks[1].lineStart != block1Start {
		t.Fatalf("after collapse: second block lineStart = %d, want %d", m.toolBlocks[1].lineStart, block1Start)
	}
}

func TestToolBlock_MoveBlockFocusClampsRange(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})

	m.appendToolResult(ToolResult{Name: "a", Summary: "a", Detail: "a", Collapsed: true})
	m.appendToolResult(ToolResult{Name: "b", Summary: "b", Detail: "b", Collapsed: true})
	m.appendToolResult(ToolResult{Name: "c", Summary: "c", Detail: "c", Collapsed: true})

	m.setBlockFocus(1)
	if m.focusedBlock != 1 {
		t.Fatalf("focusedBlock = %d, want 1", m.focusedBlock)
	}

	// 範囲下限クランプ
	m.moveBlockFocus(-1)
	if m.focusedBlock != 0 {
		t.Fatalf("after move to -1: focusedBlock = %d, want 0", m.focusedBlock)
	}
	m.moveBlockFocus(-1)
	if m.focusedBlock != 0 {
		t.Fatalf("after move to -1 again: focusedBlock = %d, want 0 (clamped)", m.focusedBlock)
	}

	// 範囲上限クランプ
	m.moveBlockFocus(100)
	if m.focusedBlock != 2 {
		t.Fatalf("after move to 100: focusedBlock = %d, want 2 (clamped)", m.focusedBlock)
	}
}

func TestToolBlock_FocusIndicatorReflected(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})

	m.appendToolResult(ToolResult{Name: "a", Summary: "test-summary", Detail: "d", Collapsed: true})

	// フォーカスなし → スペースインジケータ
	firstLine := m.rawLines[m.toolBlocks[0].lineStart]
	if firstLine[0] != ' ' {
		t.Fatalf("unfocused indicator = %q, want space", string(firstLine[0]))
	}

	// フォーカス設定 → → インジケータ
	m.setBlockFocus(0)
	firstLine = m.rawLines[m.toolBlocks[0].lineStart]
	if !strings.HasPrefix(firstLine, "→") {
		t.Fatalf("focused line = %q, want → prefix", firstLine)
	}

	// フォーカス解除 → スペースに戻る
	m.clearBlockFocus()
	firstLine = m.rawLines[m.toolBlocks[0].lineStart]
	if firstLine[0] != ' ' {
		t.Fatalf("after clear: indicator = %q, want space", string(firstLine[0]))
	}
}

func TestToolBlock_TabKeyEntersFocusAndToggles(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})
	m.navigationMode = true

	m.appendToolResult(ToolResult{
		Name: "search_code", Summary: "🔍 search_code: test",
		Detail: "match1\nmatch2", Collapsed: true,
	})

	// Tab → 最後のブロックにフォーカス
	updated, _ := m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.focusedBlock != 0 {
		t.Fatalf("after first Tab: focusedBlock = %d, want 0", m.focusedBlock)
	}

	// Tab → トグル（展開）
	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.toolBlocks[0].tool.Collapsed {
		t.Fatal("after second Tab: block should be expanded")
	}
}

func TestToolBlock_EscClearsBlockFocusBeforeExitingNav(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})
	m.navigationMode = true

	m.appendToolResult(ToolResult{Name: "a", Summary: "a", Detail: "d", Collapsed: true})
	m.setBlockFocus(0)

	// Esc → フォーカス解除（NAVモードは維持）
	updated, _ := m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.focusedBlock != -1 {
		t.Fatalf("after Esc: focusedBlock = %d, want -1", m.focusedBlock)
	}
	if !m.navigationMode {
		t.Fatal("after Esc with focus: should still be in NAV mode")
	}

	// もう一度 Esc → NAVモード終了
	updated, _ = m.handleNavigationKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.navigationMode {
		t.Fatal("after second Esc: should exit NAV mode")
	}
}

func TestToolBlock_ScrollToBlock(t *testing.T) {
	m := newModelWithViewport(&stubAgent{statusLine: "ready"})

	// 多数の行を追加してスクロール可能にする
	for i := 0; i < 50; i++ {
		m.appendContentLines("padding line")
	}
	m.appendToolResult(ToolResult{Name: "a", Summary: "target", Detail: "d", Collapsed: true})
	m.vp.setLines(m.renderedLines)

	// 先頭にスクロール
	m.vp.gotoTop()
	if m.vp.yOffset != 0 {
		t.Fatalf("yOffset after gotoTop = %d, want 0", m.vp.yOffset)
	}

	// ブロックにスクロール
	m.scrollToBlock(0)
	blockStart := m.toolBlocks[0].lineStart
	target := max(0, blockStart-2)
	maxY := m.vp.maxYOffset()
	if target > maxY {
		target = maxY
	}
	if m.vp.yOffset != target {
		t.Fatalf("yOffset after scrollToBlock = %d, want %d", m.vp.yOffset, target)
	}
	// ブロック先頭行がビューポート内に表示されていることを確認
	if m.vp.yOffset > blockStart || m.vp.yOffset+m.vp.height <= blockStart {
		t.Fatalf("block at line %d not visible in viewport [%d, %d)", blockStart, m.vp.yOffset, m.vp.yOffset+m.vp.height)
	}
}
