package uiruntime

import "github.com/susugadx/xelyon-cli/internal/uiprompt"

// ConfirmSelector は確認用の3択セレクター（Yes/No/Comment）
func ConfirmSelector(message string) (string, error) {
	return ConfirmSelectorWithIO(DefaultPromptIO(), message)
}

// ConfirmSelectorWithIO は入出力先を指定した確認用の3択セレクター。
func ConfirmSelectorWithIO(promptIO PromptIO, message string) (string, error) {
	return ConfirmSelectorRequestWithIO(promptIO, uiprompt.PromptRequest{
		Kind:         uiprompt.PromptKindConfirm,
		Message:      message,
		AllowComment: true,
	})
}

// ConfirmSelectorRequestWithIO は PromptRequest の confirm option を使って確認セレクターを実行する。
func ConfirmSelectorRequestWithIO(promptIO PromptIO, req uiprompt.PromptRequest) (string, error) {
	selector := NewSelector(req.Message, confirmSelectorOptions(req))
	selector.RequireExplicit = req.ConfirmSubmitPolicy == uiprompt.PromptConfirmSubmitExplicit
	return selector.RunWithIO(promptIO)
}

func confirmSelectorOptions(req uiprompt.PromptRequest) []SelectOption {
	options := uiprompt.ConfirmPromptOptions(req, defaultSelectorConfirmOptions())
	selectorOptions := make([]SelectOption, 0, len(options))
	for _, opt := range options {
		selectorOptions = append(selectorOptions, SelectOption(opt))
	}
	return selectorOptions
}

func defaultSelectorConfirmOptions() []uiprompt.PromptOption {
	return []uiprompt.PromptOption{
		{Label: "Yes", Description: "Execute the proposed change", Value: string(uiprompt.PromptActionYes)},
		{Label: "No", Description: "Skip this action", Value: string(uiprompt.PromptActionNo)},
		{Label: "Comment", Description: "Provide feedback to AI", Value: string(uiprompt.PromptActionComment)},
	}
}
