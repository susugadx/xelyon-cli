package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newPromptTestModel(req ui.PromptRequest, ch chan ui.PromptResponse) Model {
	m := newSizedPromptTestModel(&stubAgent{}, 60, 16)
	updated, _ := m.Update(OpenPromptMsg{ID: 1, Request: req, Respond: ch})
	return updated.(Model)
}

func newSizedPromptTestModel(agent *stubAgent, width int, height int) Model {
	m := NewModel(agent, "")
	m.applyChatWindowSize(width, height)
	m.rebuildChrome()
	m.chromeDirty = false
	return m
}

func appendPromptTestLines(m *Model, count int) {
	for i := 0; i < count; i++ {
		m.appendContentLines(fmt.Sprintf("preview line %02d", i))
	}
	m.rebuildChrome()
	m.chromeDirty = false
}

func promptRuneKey(input string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(input)}
}

func TestPromptModal_ConfirmNavigationAndSubmit(t *testing.T) {
	ch := make(chan ui.PromptResponse, 1)
	m := newPromptTestModel(ui.PromptRequest{
		Kind:         ui.PromptKindConfirm,
		Message:      "Run tool?",
		AllowComment: true,
	}, ch)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)
	if m.prompt.selected != 2 {
		t.Fatalf("selected = %d, want comment option", m.prompt.selected)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.prompt != nil {
		t.Fatal("prompt should close after submit")
	}
	resp := <-ch
	if resp.Action != ui.PromptActionNo {
		t.Fatalf("Action = %q, want no", resp.Action)
	}
}

func TestPromptModal_EscCancelsConfirm(t *testing.T) {
	ch := make(chan ui.PromptResponse, 1)
	m := newPromptTestModel(ui.PromptRequest{Kind: ui.PromptKindConfirm, Message: "Proceed?"}, ch)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	if m.prompt != nil {
		t.Fatal("prompt should close after Esc")
	}
	resp := <-ch
	if resp.Action != ui.PromptActionNo || !resp.Cancelled {
		t.Fatalf("response = %#v, want cancelled no", resp)
	}
}

func TestPromptModal_CtrlCWhileProcessingCancelsAgent(t *testing.T) {
	agent := &stubAgent{processing: true, statusLine: "running"}
	ch := make(chan ui.PromptResponse, 1)
	m := newSizedPromptTestModel(agent, 60, 16)
	updated, _ := m.Update(OpenPromptMsg{
		ID:      1,
		Request: ui.PromptRequest{Kind: ui.PromptKindConfirm, Message: "Proceed?"},
		Respond: ch,
	})
	m = updated.(Model)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if cmd != nil {
		t.Fatal("Ctrl+C during processing should not quit while prompt is open")
	}
	if agent.cancelCalls != 1 {
		t.Fatalf("cancelCalls = %d, want 1", agent.cancelCalls)
	}
	if m.prompt == nil {
		t.Fatal("prompt should remain open until runtime cancellation closes it")
	}
	select {
	case resp := <-ch:
		t.Fatalf("Ctrl+C during processing should not answer prompt, got %#v", resp)
	default:
	}
}

func TestPromptModal_CtrlCWhenIdleCancelsPrompt(t *testing.T) {
	agent := &stubAgent{statusLine: "ready"}
	ch := make(chan ui.PromptResponse, 1)
	m := newSizedPromptTestModel(agent, 60, 16)
	updated, _ := m.Update(OpenPromptMsg{
		ID:      1,
		Request: ui.PromptRequest{Kind: ui.PromptKindConfirm, Message: "Proceed?"},
		Respond: ch,
	})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if agent.cancelCalls != 0 {
		t.Fatalf("cancelCalls = %d, want 0", agent.cancelCalls)
	}
	if m.prompt != nil {
		t.Fatal("idle Ctrl+C should close prompt")
	}
	resp := <-ch
	if resp.Action != ui.PromptActionNo || !resp.Cancelled {
		t.Fatalf("response = %#v, want cancelled no", resp)
	}
}

func TestPromptModal_ConfirmActionShortcuts(t *testing.T) {
	tests := []struct {
		input string
		want  ui.PromptAction
	}{
		{input: "y", want: ui.PromptActionYes},
		{input: "yes", want: ui.PromptActionYes},
		{input: "1", want: ui.PromptActionYes},
		{input: "n", want: ui.PromptActionNo},
		{input: "no", want: ui.PromptActionNo},
		{input: "2", want: ui.PromptActionNo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ch := make(chan ui.PromptResponse, 1)
			m := newPromptTestModel(ui.PromptRequest{Kind: ui.PromptKindConfirm, Message: "Proceed?"}, ch)

			updated, _ := m.Update(promptRuneKey(tt.input))
			m = updated.(Model)
			if m.prompt != nil {
				t.Fatal("prompt should close after confirm shortcut")
			}
			resp := <-ch
			if resp.Action != tt.want {
				t.Fatalf("Action = %q, want %q", resp.Action, tt.want)
			}
		})
	}
}

func TestPromptModal_ExplicitConfirmInitialEnterDoesNotSubmit(t *testing.T) {
	ch := make(chan ui.PromptResponse, 1)
	m := newPromptTestModel(ui.PromptRequest{
		Kind:                ui.PromptKindConfirm,
		Message:             "Proceed?",
		ConfirmSubmitPolicy: ui.PromptConfirmSubmitExplicit,
	}, ch)

	if m.prompt.selected != -1 {
		t.Fatalf("selected = %d, want no initial selection", m.prompt.selected)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.prompt == nil {
		t.Fatal("prompt should remain open when explicit confirm has no selection")
	}
	select {
	case resp := <-ch:
		t.Fatalf("initial Enter should not submit explicit confirm, got %#v", resp)
	default:
	}
}

func TestPromptModal_ExplicitConfirmShortcutStillSubmits(t *testing.T) {
	ch := make(chan ui.PromptResponse, 1)
	m := newPromptTestModel(ui.PromptRequest{
		Kind:                ui.PromptKindConfirm,
		Message:             "Proceed?",
		ConfirmSubmitPolicy: ui.PromptConfirmSubmitExplicit,
	}, ch)

	updated, _ := m.Update(promptRuneKey("y"))
	m = updated.(Model)
	if m.prompt != nil {
		t.Fatal("prompt should close after explicit yes shortcut")
	}
	resp := <-ch
	if resp.Action != ui.PromptActionYes {
		t.Fatalf("Action = %q, want yes", resp.Action)
	}
}

func TestPromptModal_ExplicitConfirmMoveThenEnterSubmits(t *testing.T) {
	ch := make(chan ui.PromptResponse, 1)
	m := newPromptTestModel(ui.PromptRequest{
		Kind:                ui.PromptKindConfirm,
		Message:             "Proceed?",
		ConfirmSubmitPolicy: ui.PromptConfirmSubmitExplicit,
	}, ch)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.prompt.selected != 0 {
		t.Fatalf("selected = %d, want yes after first Down", m.prompt.selected)
	}
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.prompt != nil {
		t.Fatal("prompt should close after moved selection submit")
	}
	resp := <-ch
	if resp.Action != ui.PromptActionYes {
		t.Fatalf("Action = %q, want yes", resp.Action)
	}
}

func TestPromptModal_ConfirmCommentShortcutsEnterTextMode(t *testing.T) {
	for _, input := range []string{"c", "comment", "3"} {
		t.Run(input, func(t *testing.T) {
			ch := make(chan ui.PromptResponse, 1)
			m := newPromptTestModel(ui.PromptRequest{
				Kind:         ui.PromptKindConfirm,
				Message:      "Proceed?",
				AllowComment: true,
			}, ch)

			updated, _ := m.Update(promptRuneKey(input))
			m = updated.(Model)
			if m.prompt == nil || m.prompt.mode != promptModalText || m.prompt.text.responseAction != ui.PromptActionComment {
				t.Fatalf("prompt state = %#v, want comment text mode", m.prompt)
			}
			select {
			case resp := <-ch:
				t.Fatalf("comment shortcut should wait for text, got %#v", resp)
			default:
			}
		})
	}
}

func TestPromptModal_ConfirmCommentShortcutsIgnoredWhenCommentDisabled(t *testing.T) {
	for _, input := range []string{"c", "comment", "3"} {
		t.Run(input, func(t *testing.T) {
			ch := make(chan ui.PromptResponse, 1)
			m := newPromptTestModel(ui.PromptRequest{Kind: ui.PromptKindConfirm, Message: "Proceed?"}, ch)
			selected := m.prompt.selected

			updated, _ := m.Update(promptRuneKey(input))
			m = updated.(Model)
			if m.prompt == nil {
				t.Fatal("prompt should remain open when comment is disabled")
			}
			if m.prompt.mode != promptModalChoice {
				t.Fatalf("mode = %v, want choice", m.prompt.mode)
			}
			if m.prompt.selected != selected {
				t.Fatalf("selected = %d, want unchanged %d", m.prompt.selected, selected)
			}
			select {
			case resp := <-ch:
				t.Fatalf("disabled comment shortcut should not respond, got %#v", resp)
			default:
			}
		})
	}
}

func TestPromptModal_TextDefaultIsFallbackOnly(t *testing.T) {
	tests := []struct {
		name string
		keys []tea.KeyMsg
		want string
	}{
		{name: "empty submit uses default", want: "default answer"},
		{
			name: "typed answer replaces default",
			keys: []tea.KeyMsg{promptRuneKey("custom answer")},
			want: "custom answer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan ui.PromptResponse, 1)
			m := newPromptTestModel(ui.PromptRequest{
				Kind:         ui.PromptKindText,
				Message:      "Describe it",
				DefaultValue: "default answer",
				Placeholder:  "Type answer",
			}, ch)

			if m.prompt == nil || m.prompt.mode != promptModalText {
				t.Fatalf("prompt state = %#v, want text mode", m.prompt)
			}
			if got := m.prompt.text.input.Value(); got != "" {
				t.Fatalf("initial text input value = %q, want empty default fallback", got)
			}
			if got := m.prompt.text.input.Placeholder; got != "Type answer" {
				t.Fatalf("placeholder = %q, want request placeholder", got)
			}

			for _, key := range tt.keys {
				updated, _ := m.Update(key)
				m = updated.(Model)
			}
			updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m = updated.(Model)

			if m.prompt != nil {
				t.Fatal("prompt should close after text submit")
			}
			resp := <-ch
			if resp.Text != tt.want {
				t.Fatalf("Text = %q, want %q", resp.Text, tt.want)
			}
		})
	}
}

func TestPromptModal_MultiChoiceToggleAndSubmit(t *testing.T) {
	ch := make(chan ui.PromptResponse, 1)
	m := newPromptTestModel(ui.PromptRequest{
		Kind:    ui.PromptKindMultiChoice,
		Message: "Pick",
		Options: []ui.PromptOption{
			{Label: "Alpha", Value: "a"},
			{Label: "Beta", Value: "b"},
		},
	}, ch)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	resp := <-ch
	if strings.Join(resp.Values, ",") != "a,b" {
		t.Fatalf("Values = %#v, want [a b]", resp.Values)
	}
	if m.prompt != nil {
		t.Fatal("prompt should close after multi submit")
	}
}

func TestPromptModal_CommentTextAndCancel(t *testing.T) {
	ch := make(chan ui.PromptResponse, 1)
	m := newPromptTestModel(ui.PromptRequest{
		Kind:         ui.PromptKindConfirm,
		Message:      "Approve?",
		AllowComment: true,
	}, ch)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.prompt == nil || m.prompt.mode != promptModalText {
		t.Fatal("comment option should switch to text mode")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("needs context")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	resp := <-ch
	if resp.Action != ui.PromptActionComment || resp.Text != "needs context" {
		t.Fatalf("response = %#v, want comment text", resp)
	}

	cancelCh := make(chan ui.PromptResponse, 1)
	m = newPromptTestModel(ui.PromptRequest{
		Kind:         ui.PromptKindConfirm,
		Message:      "Approve?",
		AllowComment: true,
	}, cancelCh)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)
	cancelResp := <-cancelCh
	if cancelResp.Action != ui.PromptActionNo || !cancelResp.Cancelled {
		t.Fatalf("cancel response = %#v, want cancelled no", cancelResp)
	}
}

func TestPromptModal_PastedCommentTextRendersSingleLineAndSubmitsRaw(t *testing.T) {
	ch := make(chan ui.PromptResponse, 1)
	m := newPromptTestModel(ui.PromptRequest{
		Kind:         ui.PromptKindConfirm,
		Message:      "Approve?",
		AllowComment: true,
	}, ch)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	pasted := "needs context\nimage:/tmp/a.png"
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(pasted), Paste: true})
	m = updated.(Model)

	if got := m.prompt.text.value; got != pasted {
		t.Fatalf("raw prompt text value = %q, want pasted text", got)
	}
	if got := m.prompt.text.input.Value(); got != `needs context\nimage:/tmp/a.png` {
		t.Fatalf("display text input value = %q, want literal newline marker", got)
	}
	view := stripANSI(m.View())
	if !strings.Contains(view, `needs context\nimage:/tmp/a.png`) {
		t.Fatalf("prompt view should render pasted newline as literal marker, got %q", view)
	}
	if strings.Contains(view, "needs context\nimage:/tmp/a.png") {
		t.Fatalf("prompt view contains raw pasted newline: %q", view)
	}
	verifyViewStructure(t, m, "prompt pasted multiline comment")

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	resp := <-ch
	if resp.Action != ui.PromptActionComment || resp.Text != pasted {
		t.Fatalf("response = %#v, want raw pasted comment text", resp)
	}
}

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
