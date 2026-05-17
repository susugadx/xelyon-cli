package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestPromptModal_ViewOverlaysTranscript(t *testing.T) {
	ch := make(chan ui.PromptResponse, 1)
	m := newSizedPromptTestModel(&stubAgent{statusLine: "ready"}, 80, 20)
	appendPromptTestLines(&m, 24)

	updated, _ := m.Update(OpenPromptMsg{
		ID: 1,
		Request: ui.PromptRequest{
			Kind:    ui.PromptKindConfirm,
			Message: "Run preview?",
		},
		Respond: ch,
	})
	m = updated.(Model)

	view := stripANSI(m.View())
	if !strings.Contains(view, "Run preview?") {
		t.Fatalf("view should contain prompt message, got %q", view)
	}
	if !strings.Contains(view, "preview line 23") {
		t.Fatalf("view should keep transcript preview behind prompt, got %q", view)
	}
	verifyViewStructure(t, m, "prompt overlay")
}

func TestPromptModal_ForwardsBackgroundStreamMessages(t *testing.T) {
	ch := make(chan ui.PromptResponse, 1)
	m := newPromptTestModel(ui.PromptRequest{Kind: ui.PromptKindConfirm, Message: "Proceed?"}, ch)

	updated, _ := m.Update(AppendMessageMsg{
		Message: ChatMessage{Role: "assistant", Content: "background assistant"},
	})
	m = updated.(Model)
	updated, _ = m.Update(StreamTextMsg{Text: "streamed output", Done: false})
	m = updated.(Model)
	updated, _ = m.Update(AppendToolResultMsg{
		Tool: ToolResult{Name: "bash", Summary: "command preview", Detail: "diff --git", Collapsed: true},
	})
	m = updated.(Model)

	if m.prompt == nil {
		t.Fatal("prompt should remain open while background messages are forwarded")
	}
	if len(m.messages) != 1 || m.messages[0].Content != "background assistant" {
		t.Fatalf("messages = %#v, want forwarded assistant message", m.messages)
	}
	raw := stripANSI(strings.Join(m.rawLines, "\n"))
	if !strings.Contains(raw, "streamed output") {
		t.Fatalf("rawLines should contain forwarded stream text, got %q", raw)
	}
	if len(m.toolBlocks) != 1 {
		t.Fatalf("toolBlocks len = %d, want 1", len(m.toolBlocks))
	}
	if !strings.Contains(raw, "command preview") {
		t.Fatalf("rawLines should contain forwarded tool preview, got %q", raw)
	}
}

func TestPromptModal_ForwardsStatusAndAgentDone(t *testing.T) {
	agent := &stubAgent{statusLine: "ready", processing: true}
	ch := make(chan ui.PromptResponse, 1)
	m := newSizedPromptTestModel(agent, 60, 16)
	updated, _ := m.Update(OpenPromptMsg{
		ID:      1,
		Request: ui.PromptRequest{Kind: ui.PromptKindConfirm, Message: "Proceed?"},
		Respond: ch,
	})
	m = updated.(Model)
	m.streamingActive = true
	m.streamCursorCol = 4

	updated, _ = m.Update(UpdateStatusMsg{Line: "running"})
	m = updated.(Model)
	if m.prompt == nil {
		t.Fatal("prompt should remain open after status update")
	}
	if m.streamingActive || m.streamCursorCol != 0 {
		t.Fatalf("stream state should reset, active=%v cursor=%d", m.streamingActive, m.streamCursorCol)
	}
	if m.statusLine != "running" {
		t.Fatalf("statusLine = %q, want running", m.statusLine)
	}

	agent.statusLine = "thinking"
	updated, _ = m.Update(spinner.TickMsg{})
	m = updated.(Model)
	if m.statusLine != "thinking" {
		t.Fatalf("statusLine after spinner tick = %q, want thinking", m.statusLine)
	}

	agent.statusLine = "done"
	m.streamingActive = true
	updated, _ = m.Update(AgentDoneMsg{})
	m = updated.(Model)
	if m.streamingActive {
		t.Fatal("AgentDoneMsg should reset streaming state behind prompt")
	}
	if m.statusLine != "done" {
		t.Fatalf("statusLine after AgentDoneMsg = %q, want done", m.statusLine)
	}
}

func TestPromptModal_OpenScrollsChatToBottom(t *testing.T) {
	ch := make(chan ui.PromptResponse, 1)
	m := newSizedPromptTestModel(&stubAgent{statusLine: "ready"}, 60, 16)
	appendPromptTestLines(&m, 40)
	m.vp.scrollUp(8)
	m.afterViewportScroll()
	if m.vp.atBottom() {
		t.Fatal("test setup should be scrolled away from bottom")
	}

	updated, _ := m.Update(OpenPromptMsg{
		ID:      1,
		Request: ui.PromptRequest{Kind: ui.PromptKindConfirm, Message: "Approve?"},
		Respond: ch,
	})
	m = updated.(Model)

	if !m.vp.atBottom() {
		t.Fatalf("viewport should move to bottom when prompt opens, yOffset=%d max=%d", m.vp.yOffset, m.vp.maxYOffset())
	}
	if m.newOutput {
		t.Fatal("newOutput badge should clear when prompt opens at bottom")
	}
	if !strings.Contains(stripANSI(m.View()), "preview line 39") {
		t.Fatalf("bottom preview should remain visible after prompt opens, view=%q", stripANSI(m.View()))
	}
}

func TestPromptModal_BackgroundScrollInputDoesNotMovePromptSelection(t *testing.T) {
	ch := make(chan ui.PromptResponse, 1)
	m := newSizedPromptTestModel(&stubAgent{statusLine: "ready"}, 60, 16)
	appendPromptTestLines(&m, 40)
	updated, _ := m.Update(OpenPromptMsg{
		ID: 1,
		Request: ui.PromptRequest{
			Kind:         ui.PromptKindConfirm,
			Message:      "Approve?",
			AllowComment: true,
		},
		Respond: ch,
	})
	m = updated.(Model)
	beforeOffset := m.vp.yOffset
	beforeSelected := m.prompt.selected

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = updated.(Model)
	if m.vp.yOffset >= beforeOffset {
		t.Fatalf("PgUp should scroll background up, offset %d -> %d", beforeOffset, m.vp.yOffset)
	}
	if m.prompt.selected != beforeSelected {
		t.Fatalf("prompt selected = %d, want unchanged %d after PgUp", m.prompt.selected, beforeSelected)
	}

	offsetAfterPgUp := m.vp.yOffset
	updated, _ = m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	m = updated.(Model)
	if m.vp.yOffset <= offsetAfterPgUp {
		t.Fatalf("wheel down should scroll background down, offset %d -> %d", offsetAfterPgUp, m.vp.yOffset)
	}
	if m.prompt.selected != beforeSelected {
		t.Fatalf("prompt selected = %d, want unchanged %d after wheel", m.prompt.selected, beforeSelected)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.prompt.selected != beforeSelected+1 {
		t.Fatalf("Down should still move prompt selection, selected=%d want %d", m.prompt.selected, beforeSelected+1)
	}
}

func TestPromptModal_OpenFromNavigationModeRestoresComposerInput(t *testing.T) {
	ch := make(chan ui.PromptResponse, 1)
	m := newSizedPromptTestModel(&stubAgent{statusLine: "ready"}, 60, 16)
	appendPromptTestLines(&m, 12)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if !m.navigationMode {
		t.Fatal("test setup should enter navigation mode")
	}
	if m.textInput.Focused() {
		t.Fatal("test setup should blur composer in navigation mode")
	}

	updated, _ = m.Update(OpenPromptMsg{
		ID: 1,
		Request: ui.PromptRequest{
			Kind:    ui.PromptKindConfirm,
			Message: "Approve?",
		},
		Respond: ch,
	})
	m = updated.(Model)

	if m.navigationMode {
		t.Fatal("prompt open should exit navigation mode")
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	resp := <-ch
	if resp.Action != ui.PromptActionYes {
		t.Fatalf("Action = %q, want yes", resp.Action)
	}
	if m.prompt != nil {
		t.Fatal("prompt should close after submit")
	}
	if !m.textInput.Focused() {
		t.Fatal("composer should be focused after prompt closes")
	}

	updated, _ = m.Update(promptRuneKey("after prompt"))
	m = updated.(Model)
	if got := m.textInput.Value(); got != "after prompt" {
		t.Fatalf("composer input = %q, want typed text after prompt", got)
	}
}
