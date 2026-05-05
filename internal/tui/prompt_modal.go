package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type promptModalMode int

const (
	promptModalChoice promptModalMode = iota
	promptModalText
)

type promptModalState struct {
	id       uint64
	req      ui.PromptRequest
	respond  chan<- ui.PromptResponse
	mode     promptModalMode
	selected int
	values   map[string]bool
	text     promptTextState
}

type promptOptionView struct {
	label       string
	description string
	value       string
	action      ui.PromptAction
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

	if msg.Request.Kind == ui.PromptKindText {
		state.mode = promptModalText
		state.text.focus()
	}

	for _, value := range msg.Request.DefaultValues {
		state.values[value] = true
	}
	if msg.Request.Kind == ui.PromptKindSingleChoice && msg.Request.DefaultValue != "" {
		for i, opt := range promptOptions(msg.Request) {
			if opt.value == msg.Request.DefaultValue {
				state.selected = i
				break
			}
		}
	}

	return state
}

func promptOptions(req ui.PromptRequest) []promptOptionView {
	switch req.Kind {
	case ui.PromptKindConfirm:
		options := []promptOptionView{
			{label: "Yes", description: "Approve", action: ui.PromptActionYes},
			{label: "No", description: "Cancel", action: ui.PromptActionNo},
		}
		if req.AllowComment {
			options = append(options, promptOptionView{label: "Comment", description: "Send feedback", action: ui.PromptActionComment})
		}
		return options
	case ui.PromptKindSingleChoice, ui.PromptKindMultiChoice:
		options := make([]promptOptionView, 0, len(req.Options))
		for _, opt := range req.Options {
			value := opt.Value
			if value == "" {
				value = opt.Label
			}
			options = append(options, promptOptionView{
				label:       opt.Label,
				description: opt.Description,
				value:       value,
			})
		}
		return options
	default:
		return nil
	}
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
		m.finishPrompt(ui.PromptResponse{Cancelled: true})
		return m
	}
	if action, ok := promptConfirmShortcutAction(m.prompt.req, msg); ok {
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
	case msg.Type == tea.KeySpace && m.prompt.req.Kind == ui.PromptKindMultiChoice:
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

func promptConfirmShortcutAction(req ui.PromptRequest, msg tea.KeyMsg) (ui.PromptAction, bool) {
	if req.Kind != ui.PromptKindConfirm {
		return "", false
	}
	input := strings.ToLower(strings.TrimSpace(msg.String()))
	switch input {
	case "y", "yes", "1":
		return ui.PromptActionYes, true
	case "n", "no", "2":
		return ui.PromptActionNo, true
	case "c", "comment", "3":
		if req.AllowComment {
			return ui.PromptActionComment, true
		}
		return "", false
	default:
		return "", false
	}
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
	case ui.PromptKindConfirm:
		m.submitPromptConfirmAction(opt.action)
	case ui.PromptKindSingleChoice:
		m.finishPrompt(ui.PromptResponse{Value: opt.value})
	case ui.PromptKindMultiChoice:
		values := make([]string, 0, len(options))
		for _, option := range options {
			if m.prompt.values[option.value] {
				values = append(values, option.value)
			}
		}
		m.finishPrompt(ui.PromptResponse{Values: values})
	}
}

func (m *Model) submitPromptConfirmAction(action ui.PromptAction) {
	if action == ui.PromptActionComment {
		m.prompt.mode = promptModalText
		m.prompt.text.beginComment()
		return
	}
	m.finishPrompt(ui.PromptResponse{Action: action})
}

func (m *Model) finishPrompt(resp ui.PromptResponse) {
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

func cancelPromptResponse(req ui.PromptRequest) ui.PromptResponse {
	if req.Kind == ui.PromptKindConfirm {
		return ui.PromptResponse{Action: ui.PromptActionNo, Cancelled: true}
	}
	return ui.PromptResponse{Cancelled: true}
}
