package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/uiprompt"
)

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
			ch := make(chan uiprompt.PromptResponse, 1)
			m := newPromptTestModel(uiprompt.PromptRequest{
				Kind:         uiprompt.PromptKindText,
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
	ch := make(chan uiprompt.PromptResponse, 1)
	m := newPromptTestModel(uiprompt.PromptRequest{
		Kind:    uiprompt.PromptKindMultiChoice,
		Message: "Pick",
		Options: []uiprompt.PromptOption{
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
	ch := make(chan uiprompt.PromptResponse, 1)
	m := newPromptTestModel(uiprompt.PromptRequest{
		Kind:         uiprompt.PromptKindConfirm,
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
	if resp.Action != uiprompt.PromptActionComment || resp.Text != "needs context" {
		t.Fatalf("response = %#v, want comment text", resp)
	}

	cancelCh := make(chan uiprompt.PromptResponse, 1)
	m = newPromptTestModel(uiprompt.PromptRequest{
		Kind:         uiprompt.PromptKindConfirm,
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
	if cancelResp.Action != uiprompt.PromptActionNo || !cancelResp.Cancelled {
		t.Fatalf("cancel response = %#v, want cancelled no", cancelResp)
	}
}

func TestPromptModal_PastedCommentTextRendersSingleLineAndSubmitsRaw(t *testing.T) {
	ch := make(chan uiprompt.PromptResponse, 1)
	m := newPromptTestModel(uiprompt.PromptRequest{
		Kind:         uiprompt.PromptKindConfirm,
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
	if resp.Action != uiprompt.PromptActionComment || resp.Text != pasted {
		t.Fatalf("response = %#v, want raw pasted comment text", resp)
	}
}
