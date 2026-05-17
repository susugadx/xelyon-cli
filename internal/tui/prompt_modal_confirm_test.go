package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

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

func TestPromptModal_ConfirmCustomOptionsUseRequestOrder(t *testing.T) {
	ch := make(chan ui.PromptResponse, 1)
	m := newPromptTestModel(ui.NewPlanApprovalPromptRequest(), ch)

	updated, _ := m.Update(promptRuneKey("2"))
	m = updated.(Model)
	if m.prompt == nil || m.prompt.mode != promptModalText || m.prompt.text.responseAction != ui.PromptActionComment {
		t.Fatalf("prompt state = %#v, want second custom option to enter feedback mode", m.prompt)
	}
	if got := m.prompt.text.input.Placeholder; got != "Describe what should change before implementation..." {
		t.Fatalf("comment placeholder = %q, want plan feedback placeholder", got)
	}
	select {
	case resp := <-ch:
		t.Fatalf("custom comment option should wait for feedback text, got %#v", resp)
	default:
	}
}

func TestPromptModal_ConfirmCustomOptionsKeepNamedShortcuts(t *testing.T) {
	tests := []struct {
		input string
		want  ui.PromptAction
	}{
		{input: "y", want: ui.PromptActionYes},
		{input: "n", want: ui.PromptActionNo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ch := make(chan ui.PromptResponse, 1)
			m := newPromptTestModel(ui.NewPlanApprovalPromptRequest(), ch)

			updated, _ := m.Update(promptRuneKey(tt.input))
			m = updated.(Model)
			if m.prompt != nil {
				t.Fatal("prompt should close after named shortcut")
			}
			resp := <-ch
			if resp.Action != tt.want {
				t.Fatalf("Action = %q, want %q", resp.Action, tt.want)
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
