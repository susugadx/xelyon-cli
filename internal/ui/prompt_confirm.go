package ui

import "strings"

// ConfirmPromptOptions は PromptRequest の confirm 選択肢を正規化する。
func ConfirmPromptOptions(req PromptRequest, fallback []PromptOption) []PromptOption {
	if options := filterConfirmPromptOptions(req.Options, req.AllowComment); len(options) > 0 {
		return options
	}
	return filterConfirmPromptOptions(fallback, req.AllowComment)
}

// ConfirmPromptActionFromValue は PromptOption.Value を confirm action に変換する。
func ConfirmPromptActionFromValue(value string) (PromptAction, bool) {
	switch PromptAction(value) {
	case PromptActionYes:
		return PromptActionYes, true
	case PromptActionNo:
		return PromptActionNo, true
	case PromptActionComment:
		return PromptActionComment, true
	default:
		return "", false
	}
}

// ConfirmPromptActionShortcut は入力文字列を confirm action の shortcut として解釈する。
func ConfirmPromptActionShortcut(input string) (PromptAction, bool) {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes":
		return PromptActionYes, true
	case "n", "no":
		return PromptActionNo, true
	case "c", "comment":
		return PromptActionComment, true
	default:
		return "", false
	}
}

// ConfirmPromptOptionMatchesInput は入力が confirm option を選択しているかを返す。
func ConfirmPromptOptionMatchesInput(input string, opt PromptOption) bool {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == strings.ToLower(strings.TrimSpace(opt.Value)) || input == strings.ToLower(strings.TrimSpace(opt.Label)) {
		return true
	}
	if action, ok := ConfirmPromptActionFromValue(opt.Value); ok {
		if shortcut, ok := ConfirmPromptActionShortcut(input); ok && shortcut == action {
			return true
		}
	}
	return false
}

func filterConfirmPromptOptions(options []PromptOption, allowComment bool) []PromptOption {
	filtered := make([]PromptOption, 0, len(options))
	for _, opt := range options {
		action, ok := ConfirmPromptActionFromValue(opt.Value)
		if !ok {
			continue
		}
		if action == PromptActionComment && !allowComment {
			continue
		}
		opt.Value = string(action)
		filtered = append(filtered, opt)
	}
	return filtered
}
