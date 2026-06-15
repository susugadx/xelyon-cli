package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

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
	if len(m.composer.PasteBlocks) != 0 {
		t.Fatalf("pasteBlocks length = %d after send, want 0", len(m.composer.PasteBlocks))
	}
	if len(m.composer.Parts) != 0 {
		t.Fatalf("composerParts length = %d after send, want 0", len(m.composer.Parts))
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

func TestComposer_EnterHandlesSlashCommandWithFoldedPaste(t *testing.T) {
	agent := &stubAgent{
		statusLine: "ready",
	}
	m := newModelWithViewport(agent)

	updated, _ := m.Update(pasteKey("line1\nline2"))
	m = updated.(Model)
	m.textInput.SetValue("/clear")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("cmd = %T, want nil for local clear", cmd)
	}
	if got := len(agent.handledInputs); got != 0 {
		t.Fatalf("handledInputs length = %d, want 0", got)
	}
	if got := len(agent.startedSessionIDs); got != 1 {
		t.Fatalf("startedSessionIDs length = %d, want 1", got)
	}
	if got := agent.lastChatInput(); got != "" {
		t.Fatalf("lastChatInput() = %q, want empty", got)
	}
	if got := m.textInput.Value(); got != "" {
		t.Fatalf("textInput.Value() after handled command = %q, want empty", got)
	}
	if len(m.composer.PasteBlocks) != 1 || len(m.composer.Parts) != 1 {
		t.Fatalf("composer state should be preserved after handled command, got pasteBlocks=%d composerParts=%d", len(m.composer.PasteBlocks), len(m.composer.Parts))
	}
	if got := m.buildComposerPayload(); got != "line1\nline2" {
		t.Fatalf("buildComposerPayload() after handled command = %q, want %q", got, "line1\nline2")
	}
	if len(m.messages) == 0 || !strings.Contains(m.messages[len(m.messages)-1].Content, "Started new session") {
		t.Fatalf("last message should be new session notice, got %#v", m.messages)
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

func TestComposer_EnterHandlesQuitWithArgsViaTUI(t *testing.T) {
	tests := []string{"/quit now", "/exit now"}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			agent := &stubAgent{statusLine: "ready"}
			m := newModelWithViewport(agent)
			m.textInput.SetValue(input)

			updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = updated.(Model)

			if cmd == nil {
				t.Fatal("quit with args should return tea.Quit command")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("quit with args command = %T, want tea.QuitMsg", cmd())
			}
			if !m.quitting {
				t.Fatal("model should enter quitting state")
			}
			if agent.cleanupCalls != 1 {
				t.Fatalf("cleanupCalls = %d, want 1", agent.cleanupCalls)
			}
			if len(agent.handledInputs) != 0 {
				t.Fatalf("handledInputs = %#v, want no agent command dispatch", agent.handledInputs)
			}
			if len(m.messages) == 0 || m.messages[len(m.messages)-1].Content != input {
				t.Fatalf("last message should preserve original input %q, got %#v", input, m.messages)
			}
		})
	}
}

func TestComposer_EnterHandlesCopyCommandWithSelection(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	m := newModelWithViewport(agent)
	m.rawLines = []string{"Hello, World!"}
	m.rebuildLayout()
	m.vp.setLines(m.getVisualRowContents())
	m.mouseSelAnchor = visualPosition{line: 0, col: 0}
	m.mouseSelEnd = visualPosition{line: 0, col: 4}
	m.textInput.SetValue("/copy")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if cmd != nil {
		t.Fatalf("copy command should not return sendChat cmd, got %v", cmd)
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
	if len(m.composer.PasteBlocks) != 0 || len(m.composer.Parts) != 0 {
		t.Fatalf("composer state should be cleared after sending payload, got pasteBlocks=%d composerParts=%d", len(m.composer.PasteBlocks), len(m.composer.Parts))
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

	wantTail := []string{"━━ user · " + m.messages[len(m.messages)-1].Timestamp.Format("15:04") + " · now ━━", "┃ > alpha", "┃ > beta", "┃ > gamma"}
	gotTail := make([]string, len(wantTail))
	for i, line := range m.rawLines[len(m.rawLines)-len(wantTail):] {
		gotTail[i] = stripANSI(line)
	}
	if !equalStringSlices(gotTail, wantTail) {
		t.Fatalf("rawLines tail = %#v, want %#v", gotTail, wantTail)
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
