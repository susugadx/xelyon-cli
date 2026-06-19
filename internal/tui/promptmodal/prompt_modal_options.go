package promptmodal

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/uiprompt"
)

type promptOptionView struct {
	label       string
	description string
	value       string
	action      uiprompt.PromptAction
}

func promptOptions(req uiprompt.PromptRequest) []promptOptionView {
	switch req.Kind {
	case uiprompt.PromptKindConfirm:
		return promptConfirmOptions(req)
	case uiprompt.PromptKindSingleChoice, uiprompt.PromptKindMultiChoice:
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

func promptConfirmOptions(req uiprompt.PromptRequest) []promptOptionView {
	options := uiprompt.ConfirmPromptOptions(req, defaultPromptConfirmOptions())
	views := make([]promptOptionView, 0, len(options))
	for _, opt := range options {
		action, ok := uiprompt.ConfirmPromptActionFromValue(opt.Value)
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

func defaultPromptConfirmOptions() []uiprompt.PromptOption {
	return []uiprompt.PromptOption{
		{Label: "Yes", Description: "Approve", Value: string(uiprompt.PromptActionYes)},
		{Label: "No", Description: "Cancel", Value: string(uiprompt.PromptActionNo)},
		{Label: "Comment", Description: "Send feedback", Value: string(uiprompt.PromptActionComment)},
	}
}

func promptConfirmShortcutAction(req uiprompt.PromptRequest, options []promptOptionView, msg tea.KeyMsg) (uiprompt.PromptAction, bool) {
	if req.Kind != uiprompt.PromptKindConfirm {
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
	if action, ok := uiprompt.ConfirmPromptActionShortcut(input); ok {
		return promptConfirmActionIfAvailable(options, action)
	}
	return "", false
}

func promptConfirmActionIfAvailable(options []promptOptionView, action uiprompt.PromptAction) (uiprompt.PromptAction, bool) {
	for _, opt := range options {
		if opt.action == action {
			return action, true
		}
	}
	return "", false
}
