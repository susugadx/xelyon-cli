package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/uiprompt"
)

type promptModalMode int

const (
	promptModalChoice promptModalMode = iota
	promptModalText
)

type promptModalState struct {
	id       uint64
	req      uiprompt.PromptRequest
	respond  chan<- uiprompt.PromptResponse
	mode     promptModalMode
	selected int
	values   map[string]bool
	text     promptTextState
}

func newPromptModalState(msg OpenPromptMsg) *promptModalState {
	state := &promptModalState{
		id:       msg.ID,
		req:      msg.Request,
		respond:  msg.Respond,
		mode:     promptModalChoice,
		selected: initialPromptSelectedIndex(msg.Request),
		values:   map[string]bool{},
		text:     newPromptTextState(msg.Request),
	}

	if msg.Request.Kind == uiprompt.PromptKindText {
		state.mode = promptModalText
		state.text.focus()
	}

	for _, value := range msg.Request.DefaultValues {
		state.values[value] = true
	}
	if msg.Request.Kind == uiprompt.PromptKindSingleChoice && msg.Request.DefaultValue != "" {
		for i, opt := range promptOptions(msg.Request) {
			if opt.value == msg.Request.DefaultValue {
				state.selected = i
				break
			}
		}
	}

	return state
}

func (m Model) handleOpenPromptMsg(msg OpenPromptMsg) (Model, tea.Cmd) {
	m.switchToComposerInput()
	m.prompt = newPromptModalState(msg)
	if m.screen == screenChat && m.ready {
		m.vp.gotoBottom()
		m.newOutput = false
	}
	m.chromeDirty = true
	m.rebuildChromeAfterPromptRootChange()
	return m, nil
}

func (m Model) handleCancelPromptMsg(msg CancelPromptMsg) (Model, tea.Cmd) {
	if m.prompt == nil || m.prompt.id != msg.ID {
		return m, nil
	}
	m.prompt = nil
	m.chromeDirty = true
	m.rebuildChromeAfterPromptRootChange()
	return m, nil
}

func (m Model) handlePromptKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.prompt == nil {
		return m, nil
	}
	if m.prompt.mode == promptModalText {
		return m.handlePromptTextKeyMsg(msg)
	}
	return m.handlePromptChoiceKeyMsg(msg), nil
}

func (m Model) handlePromptChoiceKeyMsg(msg tea.KeyMsg) Model {
	options := promptOptions(m.prompt.req)
	if len(options) == 0 {
		m.finishPrompt(uiprompt.PromptResponse{Cancelled: true})
		return m
	}
	if action, ok := promptConfirmShortcutAction(m.prompt.req, options, msg); ok {
		m.submitPromptConfirmAction(action)
		return m
	}

	switch {
	case msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC:
		m.finishPrompt(cancelPromptResponse(m.prompt.req))
	case msg.Type == tea.KeyUp || msg.String() == "k":
		m.prompt.moveChoiceSelection(-1, len(options))
	case msg.Type == tea.KeyDown || msg.String() == "j":
		m.prompt.moveChoiceSelection(1, len(options))
	case msg.Type == tea.KeySpace && m.prompt.req.Kind == uiprompt.PromptKindMultiChoice:
		if opt, ok := m.prompt.selectedChoice(options); ok {
			m.prompt.values[opt.value] = !m.prompt.values[opt.value]
		}
	case isEnterKey(msg):
		if opt, ok := m.prompt.selectedChoice(options); ok {
			m.submitPromptChoice(opt, options)
		}
	}
	return m
}

func (m Model) handlePromptTextKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC:
		m.finishPrompt(cancelPromptResponse(m.prompt.req))
		return m, nil
	case isEnterKey(msg):
		m.finishPrompt(m.prompt.text.response())
		return m, nil
	default:
		return m, m.prompt.text.update(msg)
	}
}

func (m *Model) submitPromptChoice(opt promptOptionView, options []promptOptionView) {
	switch m.prompt.req.Kind {
	case uiprompt.PromptKindConfirm:
		m.submitPromptConfirmAction(opt.action)
	case uiprompt.PromptKindSingleChoice:
		m.finishPrompt(uiprompt.PromptResponse{Value: opt.value})
	case uiprompt.PromptKindMultiChoice:
		values := make([]string, 0, len(options))
		for _, option := range options {
			if m.prompt.values[option.value] {
				values = append(values, option.value)
			}
		}
		m.finishPrompt(uiprompt.PromptResponse{Values: values})
	}
}

func (m *Model) submitPromptConfirmAction(action uiprompt.PromptAction) {
	if action == uiprompt.PromptActionComment {
		m.prompt.mode = promptModalText
		m.prompt.text.beginComment(m.prompt.req.Placeholder)
		return
	}
	m.finishPrompt(uiprompt.PromptResponse{Action: action})
}

func (m *Model) finishPrompt(resp uiprompt.PromptResponse) {
	if m.prompt == nil {
		return
	}
	if m.prompt.respond != nil {
		select {
		case m.prompt.respond <- resp:
		default:
		}
	}
	m.prompt = nil
	m.chromeDirty = true
	m.rebuildChromeAfterPromptRootChange()
}

func (m *Model) rebuildChromeAfterPromptRootChange() {
	if !m.ready {
		return
	}
	m.rebuildChrome()
	m.chromeDirty = false
}

func cancelPromptResponse(req uiprompt.PromptRequest) uiprompt.PromptResponse {
	if req.Kind == uiprompt.PromptKindConfirm {
		return uiprompt.PromptResponse{Action: uiprompt.PromptActionNo, Cancelled: true}
	}
	return uiprompt.PromptResponse{Cancelled: true}
}
