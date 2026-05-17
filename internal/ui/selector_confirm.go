package ui

// ConfirmSelector は確認用の3択セレクター（Yes/No/Comment）
func ConfirmSelector(message string) (string, error) {
	return ConfirmSelectorWithIO(DefaultPromptIO(), message)
}

// ConfirmSelectorWithIO は入出力先を指定した確認用の3択セレクター。
func ConfirmSelectorWithIO(promptIO PromptIO, message string) (string, error) {
	return ConfirmSelectorRequestWithIO(promptIO, PromptRequest{
		Kind:         PromptKindConfirm,
		Message:      message,
		AllowComment: true,
	})
}

// ConfirmSelectorRequestWithIO は PromptRequest の confirm option を使って確認セレクターを実行する。
func ConfirmSelectorRequestWithIO(promptIO PromptIO, req PromptRequest) (string, error) {
	selector := NewSelector(req.Message, confirmSelectorOptions(req))
	selector.RequireExplicit = req.ConfirmSubmitPolicy == PromptConfirmSubmitExplicit
	return selector.RunWithIO(promptIO)
}

func confirmSelectorOptions(req PromptRequest) []SelectOption {
	options := ConfirmPromptOptions(req, defaultSelectorConfirmOptions())
	selectorOptions := make([]SelectOption, 0, len(options))
	for _, opt := range options {
		selectorOptions = append(selectorOptions, SelectOption(opt))
	}
	return selectorOptions
}

func defaultSelectorConfirmOptions() []PromptOption {
	return []PromptOption{
		{Label: "Yes", Description: "Execute the proposed change", Value: string(PromptActionYes)},
		{Label: "No", Description: "Skip this action", Value: string(PromptActionNo)},
		{Label: "Comment", Description: "Provide feedback to AI", Value: string(PromptActionComment)},
	}
}
