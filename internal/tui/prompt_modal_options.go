package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type promptOptionView struct {
	label       string
	description string
	value       string
	action      ui.PromptAction
}

func promptOptions(req ui.PromptRequest) []promptOptionView {
	switch req.Kind {
	case ui.PromptKindConfirm:
		return promptConfirmOptions(req)
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

func promptConfirmOptions(req ui.PromptRequest) []promptOptionView {
	options := ui.ConfirmPromptOptions(req, defaultPromptConfirmOptions())
	views := make([]promptOptionView, 0, len(options))
	for _, opt := range options {
		action, ok := ui.ConfirmPromptActionFromValue(opt.Value)
		if !ok {
			continue
		}
		views = append(views, promptOptionView{
			label:       opt.Label,
			description: opt.Description,
			action:      action,
		})
	}
	return views
}

func defaultPromptConfirmOptions() []ui.PromptOption {
	return []ui.PromptOption{
		{Label: "Yes", Description: "Approve", Value: string(ui.PromptActionYes)},
		{Label: "No", Description: "Cancel", Value: string(ui.PromptActionNo)},
		{Label: "Comment", Description: "Send feedback", Value: string(ui.PromptActionComment)},
	}
}

func promptConfirmShortcutAction(req ui.PromptRequest, options []promptOptionView, msg tea.KeyMsg) (ui.PromptAction, bool) {
	if req.Kind != ui.PromptKindConfirm {
		return "", false
	}
	input := strings.ToLower(strings.TrimSpace(msg.String()))
	if len(input) == 1 && input[0] >= '1' && input[0] <= '9' {
		idx := int(input[0] - '1')
		if idx >= 0 && idx < len(options) && options[idx].action != "" {
			return options[idx].action, true
		}
		return "", false
	}
	if action, ok := ui.ConfirmPromptActionShortcut(input); ok {
		return promptConfirmActionIfAvailable(options, action)
	}
	return "", false
}

func promptConfirmActionIfAvailable(options []promptOptionView, action ui.PromptAction) (ui.PromptAction, bool) {
	for _, opt := range options {
		if opt.action == action {
			return action, true
		}
	}
	return "", false
}
