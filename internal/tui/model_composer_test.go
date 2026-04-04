package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func pasteKey(text string) tea.KeyMsg {
	return tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune(text),
		Paste: true,
	}
}

func stubClipboardRead(t *testing.T, text string, err error) {
	t.Helper()
	prev := readClipboardText
	readClipboardText = func() (string, error) {
		return text, err
	}
	t.Cleanup(func() {
		readClipboardText = prev
	})
}

func TestComposer_ShortSingleLinePasteStaysInInput(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	updated, _ := m.Update(pasteKey("short paste"))
	m = updated.(Model)

	if got := m.textInput.Value(); got != "short paste" {
		t.Fatalf("textInput.Value() = %q, want %q", got, "short paste")
	}
	if len(m.pasteBlocks) != 0 {
		t.Fatalf("pasteBlocks length = %d, want 0", len(m.pasteBlocks))
	}
	if got := m.footerHeight(); got != statusBarHeight+inputHeight {
		t.Fatalf("footerHeight() = %d, want %d", got, statusBarHeight+inputHeight)
	}
}

func TestComposer_MultilinePasteCreatesFoldedBlock(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	updated, _ := m.Update(pasteKey("line1\nline2"))
	m = updated.(Model)

	if got := len(m.pasteBlocks); got != 1 {
		t.Fatalf("pasteBlocks length = %d, want 1", got)
	}
	if got := m.textInput.Value(); got != "" {
		t.Fatalf("textInput.Value() = %q, want empty", got)
	}
	if got := m.footerHeight(); got != statusBarHeight+inputHeight+1 {
		t.Fatalf("footerHeight() = %d, want %d", got, statusBarHeight+inputHeight+1)
	}
	if got := m.vp.height; got != m.height-m.footerHeight() {
		t.Fatalf("vp.height = %d, want %d", got, m.height-m.footerHeight())
	}

	dock := m.renderInputDock()
	if !strings.Contains(stripANSI(dock), "[Pasted Content 11 chars, 2 lines] #1") {
		t.Fatalf("renderInputDock() should contain folded paste summary, got %q", dock)
	}
	if strings.Contains(dock, "line1\nline2") {
		t.Fatalf("renderInputDock() should not contain raw pasted content, got %q", dock)
	}
}

func TestComposer_LongSingleLinePasteCreatesFoldedBlock(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	longLine := strings.Repeat("x", pasteBlockFoldThreshold)

	updated, _ := m.Update(pasteKey(longLine))
	m = updated.(Model)

	if got := len(m.pasteBlocks); got != 1 {
		t.Fatalf("pasteBlocks length = %d, want 1", got)
	}
	if got := m.textInput.Value(); got != "" {
		t.Fatalf("textInput.Value() = %q, want empty", got)
	}
	if strings.Contains(m.renderInputDock(), longLine) {
		t.Fatal("renderInputDock() should not inline a folded long paste")
	}
}

func TestComposer_PrefixTextStaysVisibleWithFoldedPaste(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	m.textInput.SetValue("Explain this:")
	updated, _ := m.Update(pasteKey("line1\nline2"))
	m = updated.(Model)

	dock := stripANSI(m.renderInputDock())
	prefixIndex := strings.Index(dock, "Explain this:")
	pasteIndex := strings.Index(dock, "[Pasted Content 11 chars, 2 lines] #1")
	if prefixIndex < 0 {
		t.Fatalf("renderInputDock() should contain the prefix text, got %q", dock)
	}
	if pasteIndex < 0 {
		t.Fatalf("renderInputDock() should contain the folded paste summary, got %q", dock)
	}
	if prefixIndex >= pasteIndex {
		t.Fatalf("prefix text should render before folded paste summary, got %q", dock)
	}
	if got := m.buildComposerPayload(); got != "Explain this:line1\nline2" {
		t.Fatalf("buildComposerPayload() = %q, want %q", got, "Explain this:line1\nline2")
	}
}

func TestComposer_CtrlVPasteCreatesFoldedBlockAndPreservesContent(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	stubClipboardRead(t, "line1\tvalue\nline2", nil)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlV})
	m = updated.(Model)

	if got := len(m.pasteBlocks); got != 1 {
		t.Fatalf("pasteBlocks length = %d, want 1", got)
	}
	if got := m.textInput.Value(); got != "" {
		t.Fatalf("textInput.Value() = %q, want empty", got)
	}
	if got := m.buildComposerPayload(); got != "line1\tvalue\nline2" {
		t.Fatalf("buildComposerPayload() = %q, want %q", got, "line1\tvalue\nline2")
	}
	if !strings.Contains(stripANSI(m.renderInputDock()), "[Pasted Content 17 chars, 2 lines] #1") {
		t.Fatalf("renderInputDock() should contain folded paste summary, got %q", m.renderInputDock())
	}
}

func TestComposer_SendBuildsPayloadInOrder(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	m.textInput.SetValue("alpha")
	updated, _ := m.Update(pasteKey("one\ntwo"))
	m = updated.(Model)

	m.textInput.SetValue("beta")
	updated, _ = m.Update(pasteKey("three\nfour"))
	m = updated.(Model)

	m.textInput.SetValue("gamma")
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("Enter should return a send command")
	}
	_ = cmd()

	want := "alphaone\ntwobetathree\nfourgamma"
	if got := agent.lastChatInput(); got != want {
		t.Fatalf("lastChatInput() = %q, want %q", got, want)
	}
	if got := m.textInput.Value(); got != "" {
		t.Fatalf("textInput.Value() = %q after send, want empty", got)
	}
	if len(m.pasteBlocks) != 0 {
		t.Fatalf("pasteBlocks length = %d after send, want 0", len(m.pasteBlocks))
	}
	if len(m.composerParts) != 0 {
		t.Fatalf("composerParts length = %d after send, want 0", len(m.composerParts))
	}
	if len(m.messages) == 0 {
		t.Fatal("messages should contain the user message after send")
	}
	if got := m.messages[len(m.messages)-1].Content; got != want {
		t.Fatalf("last message content = %q, want %q", got, want)
	}
	for i, line := range m.rawLines {
		if strings.Contains(line, "\n") {
			t.Fatalf("rawLines[%d] should not contain embedded newline, got %q", i, line)
		}
	}
}

func TestComposer_BackspaceRemovesLastPasteBlockWhenInputIsEmpty(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	m.textInput.SetValue("alpha")
	updated, _ := m.Update(pasteKey("one\ntwo"))
	m = updated.(Model)

	m.textInput.SetValue("beta")
	updated, _ = m.Update(pasteKey("three\nfour"))
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(Model)

	if got := len(m.pasteBlocks); got != 1 {
		t.Fatalf("pasteBlocks length = %d after removing last block, want 1", got)
	}
	if got := m.textInput.Value(); got != "beta" {
		t.Fatalf("textInput.Value() = %q after removing last block, want %q", got, "beta")
	}
	if got := m.buildComposerPayload(); got != "alphaone\ntwobeta" {
		t.Fatalf("buildComposerPayload() = %q, want %q", got, "alphaone\ntwobeta")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(Model)

	if got := len(m.pasteBlocks); got != 1 {
		t.Fatalf("pasteBlocks length = %d after backspacing text, want 1", got)
	}
	if got := m.textInput.Value(); got != "bet" {
		t.Fatalf("textInput.Value() = %q after backspacing text, want %q", got, "bet")
	}
}

func TestComposer_BackspaceRemovesLastPasteBlockAtInputStartWithTrailingText(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	m.textInput.SetValue("beforeafter")
	m.textInput.SetCursor(len("before"))
	updated, _ := m.Update(pasteKey("one\ntwo"))
	m = updated.(Model)

	if got := m.textInput.Value(); got != "after" {
		t.Fatalf("textInput.Value() after paste = %q, want %q", got, "after")
	}
	if got := m.textInput.Position(); got != 0 {
		t.Fatalf("textInput.Position() after paste = %d, want 0", got)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	m = updated.(Model)

	if got := len(m.pasteBlocks); got != 0 {
		t.Fatalf("pasteBlocks length = %d after removing block, want 0", got)
	}
	if got := m.textInput.Value(); got != "beforeafter" {
		t.Fatalf("textInput.Value() after removing block = %q, want %q", got, "beforeafter")
	}
	if got := m.textInput.Position(); got != len("before") {
		t.Fatalf("textInput.Position() after removing block = %d, want %d", got, len("before"))
	}
	if got := m.buildComposerPayload(); got != "beforeafter" {
		t.Fatalf("buildComposerPayload() = %q, want %q", got, "beforeafter")
	}
}

func TestComposer_EscDoesNotEnterNavigationWithFoldedPaste(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	updated, _ := m.Update(pasteKey("line1\nline2"))
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.navigationMode {
		t.Fatal("navigationMode should stay false while composer has folded paste blocks")
	}
}

func TestComposer_EscTreatsWhitespaceOnlyInputAsEmpty(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.textInput.SetValue(" \t ")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if !m.navigationMode {
		t.Fatal("navigationMode should become true for whitespace-only input")
	}
}

func TestComposer_EnterHandlesSlashCommandWithFoldedPaste(t *testing.T) {
	agent := &stubAgent{
		statusLine:      "ready",
		handledCommands: map[string]bool{"/clear": true},
	}
	m := newModelWithViewport(agent)

	updated, _ := m.Update(pasteKey("line1\nline2"))
	m = updated.(Model)
	m.textInput.SetValue("/clear")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("command Enter should not return sendChat cmd, got %v", cmd)
	}
	if got := len(agent.handledInputs); got != 1 {
		t.Fatalf("handledInputs length = %d, want 1", got)
	}
	if got := agent.handledInputs[0]; got != "/clear" {
		t.Fatalf("handledInputs[0] = %q, want %q", got, "/clear")
	}
	if got := agent.lastChatInput(); got != "" {
		t.Fatalf("lastChatInput() = %q, want empty", got)
	}
	if got := m.textInput.Value(); got != "" {
		t.Fatalf("textInput.Value() after handled command = %q, want empty", got)
	}
	if len(m.pasteBlocks) != 1 || len(m.composerParts) != 1 {
		t.Fatalf("composer state should be preserved after handled command, got pasteBlocks=%d composerParts=%d", len(m.pasteBlocks), len(m.composerParts))
	}
	if got := m.buildComposerPayload(); got != "line1\nline2" {
		t.Fatalf("buildComposerPayload() after handled command = %q, want %q", got, "line1\nline2")
	}
	if len(m.messages) == 0 || m.messages[len(m.messages)-1].Content != "/clear" {
		t.Fatalf("last message should be command text, got %#v", m.messages)
	}
}

func TestComposer_EnterHandlesQuitAliasWithFoldedPaste(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	updated, _ := m.Update(pasteKey("line1\nline2"))
	m = updated.(Model)
	m.textInput.SetValue("/q")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd == nil {
		t.Fatal("quit alias should return tea.Quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("quit alias command = %T, want tea.QuitMsg", cmd())
	}
	if !m.quitting {
		t.Fatal("model should enter quitting state")
	}
	if agent.cleanupCalls != 1 {
		t.Fatalf("cleanupCalls = %d, want 1", agent.cleanupCalls)
	}
	if len(m.messages) == 0 || m.messages[len(m.messages)-1].Content != "/q" {
		t.Fatalf("last message should preserve original alias input, got %#v", m.messages)
	}
}

func TestComposer_EnterHandlesCopyAliasWithSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"Hello, World!"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 4}
	m.textInput.SetValue("/cp")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("copy alias should not return sendChat cmd, got %v", cmd)
	}
	if len(agent.copyTexts) != 1 {
		t.Fatalf("copyTexts length = %d, want 1", len(agent.copyTexts))
	}
	if got := agent.copyTexts[0]; got != "Hello" {
		t.Fatalf("copyTexts[0] = %q, want %q", got, "Hello")
	}
	if got := agent.lastChatInput(); got != "" {
		t.Fatalf("lastChatInput() = %q, want empty", got)
	}
}

func TestComposer_UnhandledSlashSendsFullComposerPayload(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	m.textInput.SetValue("prefix ")
	updated, _ := m.Update(pasteKey("line1\nline2"))
	m = updated.(Model)
	m.textInput.SetValue("/tmp/log")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("unhandled slash input should return send command")
	}
	_ = cmd()

	want := "prefix line1\nline2/tmp/log"
	if got := agent.lastChatInput(); got != want {
		t.Fatalf("lastChatInput() = %q, want %q", got, want)
	}
	if got := len(agent.handledInputs); got != 1 {
		t.Fatalf("handledInputs length = %d, want 1", got)
	}
	if got := agent.handledInputs[0]; got != "/tmp/log" {
		t.Fatalf("handledInputs[0] = %q, want %q", got, "/tmp/log")
	}
	if len(m.pasteBlocks) != 0 || len(m.composerParts) != 0 {
		t.Fatalf("composer state should be cleared after sending payload, got pasteBlocks=%d composerParts=%d", len(m.pasteBlocks), len(m.composerParts))
	}
	if len(m.messages) == 0 || m.messages[len(m.messages)-1].Content != want {
		t.Fatalf("last message content = %#v, want %q", m.messages, want)
	}
}

func TestComposer_MultilineUserMessageKeepsViewStructure(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)

	m.appendContentLines("context line")
	m.appendMessage(ChatMessage{
		Role:    "user",
		Content: "alpha\nbeta\ngamma",
	})
	m.rebuildChrome()

	wantTail := []string{"", "> alpha", "> beta", "> gamma", ""}
	if got := m.rawLines[len(m.rawLines)-len(wantTail):]; !equalStringSlices(got, wantTail) {
		t.Fatalf("rawLines tail = %#v, want %#v", got, wantTail)
	}
	for i, line := range m.rawLines {
		if strings.Contains(line, "\n") {
			t.Fatalf("rawLines[%d] should not contain embedded newline, got %q", i, line)
		}
	}

	lines := strings.Split(m.View(), "\n")
	if got := len(lines); got != m.height {
		t.Fatalf("View() line count = %d, want %d", got, m.height)
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
