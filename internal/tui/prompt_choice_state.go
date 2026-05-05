package tui

import "github.com/susugadx/xelyon-cli/internal/ui"

func initialPromptSelectedIndex(req ui.PromptRequest) int {
	if promptConfirmRequiresExplicitSelection(req) {
		return -1
	}
	return 0
}

func promptConfirmRequiresExplicitSelection(req ui.PromptRequest) bool {
	return req.Kind == ui.PromptKindConfirm && req.ConfirmSubmitPolicy == ui.PromptConfirmSubmitExplicit
}

func (p *promptModalState) moveChoiceSelection(delta int, optionCount int) {
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

func (p *promptModalState) selectedChoice(options []promptOptionView) (promptOptionView, bool) {
	if p == nil || p.selected < 0 || p.selected >= len(options) {
		return promptOptionView{}, false
	}
	return options[p.selected], true
}
