package promptmodal

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/uiprompt"
)

// Mode は prompt modal の入力モードを表す。
type Mode int

const (
	// ModeChoice は choice/confirm prompt を表す。
	ModeChoice Mode = iota
	// ModeText は text/comment prompt を表す。
	ModeText
)

type promptModalMode = Mode

const (
	promptModalChoice = ModeChoice
	promptModalText   = ModeText
)

// Screen は prompt modal の state/input/render を保持する。
type Screen struct {
	id       uint64
	req      uiprompt.PromptRequest
	respond  chan<- uiprompt.PromptResponse
	mode     promptModalMode
	selected int
	values   map[string]bool
	text     promptTextState
}

// Snapshot は prompt modal の読み取り専用状態を返す。
type Snapshot struct {
	ID                 uint64
	Mode               Mode
	Selected           int
	Values             map[string]bool
	TextValue          string
	TextDisplayValue   string
	TextPlaceholder    string
	TextResponseAction uiprompt.PromptAction
}

// KeyResult はキー入力処理の結果を表す。
type KeyResult struct {
	Response *uiprompt.PromptResponse
}

// New は prompt modal screen を構築する。
func New(id uint64, req uiprompt.PromptRequest, respond chan<- uiprompt.PromptResponse) *Screen {
	state := &Screen{
		id:       id,
		req:      req,
		respond:  respond,
		mode:     promptModalChoice,
		selected: initialPromptSelectedIndex(req),
		values:   map[string]bool{},
		text:     newPromptTextState(req),
	}

	if req.Kind == uiprompt.PromptKindText {
		state.mode = promptModalText
		state.text.focus()
	}

	for _, value := range req.DefaultValues {
		state.values[value] = true
	}
	if req.Kind == uiprompt.PromptKindSingleChoice && req.DefaultValue != "" {
		for i, opt := range promptOptions(req) {
			if opt.value == req.DefaultValue {
				state.selected = i
				break
			}
		}
	}

	return state
}

// ID は prompt modal の識別子を返す。
func (p *Screen) ID() uint64 {
	if p == nil {
		return 0
	}
	return p.id
}

// Respond は prompt response の送信先 channel を返す。
func (p *Screen) Respond() chan<- uiprompt.PromptResponse {
	if p == nil {
		return nil
	}
	return p.respond
}

// Snapshot は prompt modal の公開状態を返す。
func (p *Screen) Snapshot() Snapshot {
	if p == nil {
		return Snapshot{}
	}
	values := make(map[string]bool, len(p.values))
	for key, value := range p.values {
		values[key] = value
	}
	return Snapshot{
		ID:                 p.id,
		Mode:               p.mode,
		Selected:           p.selected,
		Values:             values,
		TextValue:          p.text.value,
		TextDisplayValue:   p.text.input.Value(),
		TextPlaceholder:    p.text.input.Placeholder,
		TextResponseAction: p.text.responseAction,
	}
}

// HandleKey は prompt modal のキー入力を処理する。
func (p *Screen) HandleKey(msg tea.KeyMsg) (KeyResult, tea.Cmd) {
	if p == nil {
		return KeyResult{}, nil
	}
	if p.mode == promptModalText {
		return p.handleTextKey(msg)
	}
	return p.handleChoiceKey(msg), nil
}

func (p *Screen) handleChoiceKey(msg tea.KeyMsg) KeyResult {
	options := promptOptions(p.req)
	if len(options) == 0 {
		resp := uiprompt.PromptResponse{Cancelled: true}
		return KeyResult{Response: &resp}
	}
	if action, ok := promptConfirmShortcutAction(p.req, options, msg); ok {
		return p.submitConfirmAction(action)
	}

	switch {
	case msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC:
		resp := cancelPromptResponse(p.req)
		return KeyResult{Response: &resp}
	case msg.Type == tea.KeyUp || msg.String() == "k":
		p.moveChoiceSelection(-1, len(options))
	case msg.Type == tea.KeyDown || msg.String() == "j":
		p.moveChoiceSelection(1, len(options))
	case msg.Type == tea.KeySpace && p.req.Kind == uiprompt.PromptKindMultiChoice:
		if opt, ok := p.selectedChoice(options); ok {
			p.values[opt.value] = !p.values[opt.value]
		}
	case isEnterKey(msg):
		if opt, ok := p.selectedChoice(options); ok {
			return p.submitChoice(opt, options)
		}
	}
	return KeyResult{}
}

func (p *Screen) handleTextKey(msg tea.KeyMsg) (KeyResult, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC:
		resp := cancelPromptResponse(p.req)
		return KeyResult{Response: &resp}, nil
	case isEnterKey(msg):
		resp := p.text.response()
		return KeyResult{Response: &resp}, nil
	default:
		return KeyResult{}, p.text.update(msg)
	}
}

func (p *Screen) submitChoice(opt promptOptionView, options []promptOptionView) KeyResult {
	switch p.req.Kind {
	case uiprompt.PromptKindConfirm:
		return p.submitConfirmAction(opt.action)
	case uiprompt.PromptKindSingleChoice:
		resp := uiprompt.PromptResponse{Value: opt.value}
		return KeyResult{Response: &resp}
	case uiprompt.PromptKindMultiChoice:
		values := make([]string, 0, len(options))
		for _, option := range options {
			if p.values[option.value] {
				values = append(values, option.value)
			}
		}
		resp := uiprompt.PromptResponse{Values: values}
		return KeyResult{Response: &resp}
	}
	return KeyResult{}
}

func (p *Screen) submitConfirmAction(action uiprompt.PromptAction) KeyResult {
	if action == uiprompt.PromptActionComment {
		p.mode = promptModalText
		p.text.beginComment(p.req.Placeholder)
		return KeyResult{}
	}
	resp := uiprompt.PromptResponse{Action: action}
	return KeyResult{Response: &resp}
}

func cancelPromptResponse(req uiprompt.PromptRequest) uiprompt.PromptResponse {
	if req.Kind == uiprompt.PromptKindConfirm {
		return uiprompt.PromptResponse{Action: uiprompt.PromptActionNo, Cancelled: true}
	}
	return uiprompt.PromptResponse{Cancelled: true}
}

func isEnterKey(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyEnter || msg.String() == "enter"
}
