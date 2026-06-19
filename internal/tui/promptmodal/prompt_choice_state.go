package promptmodal

import "github.com/susugadx/xelyon-cli/internal/uiprompt"

func initialPromptSelectedIndex(req uiprompt.PromptRequest) int {
	if promptConfirmRequiresExplicitSelection(req) {
		return -1
	}
	return 0
}

func promptConfirmRequiresExplicitSelection(req uiprompt.PromptRequest) bool {
	return req.Kind == uiprompt.PromptKindConfirm && req.ConfirmSubmitPolicy == uiprompt.PromptConfirmSubmitExplicit
}

func (p *Screen) moveChoiceSelection(delta int, optionCount int) {
	if p == nil || optionCount <= 0 {
		return
	}
	if p.selected < 0 {
		if delta < 0 {
			p.selected = optionCount - 1
			return
		}
		p.selected = 0
		return
	}
	p.selected += delta
	if p.selected < 0 {
		p.selected = 0
	}
	if p.selected >= optionCount {
		p.selected = optionCount - 1
	}
}

func (p *Screen) selectedChoice(options []promptOptionView) (promptOptionView, bool) {
	if p == nil || p.selected < 0 || p.selected >= len(options) {
		return promptOptionView{}, false
	}
	return options[p.selected], true
}
