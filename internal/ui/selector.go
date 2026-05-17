package ui

import (
	"fmt"
	"strings"
)

// SelectOption は選択肢の情報
type SelectOption struct {
	Label       string // 表示ラベル（例: "Yes"）
	Description string // 説明文（例: "Execute the proposed change"）
	Value       string // 戻り値（例: "yes"）
}

// Selector は選択UI（数字キー + Enter方式）
type Selector struct {
	Message         string         // 質問メッセージ
	Options         []SelectOption // 選択肢
	RequireExplicit bool           // 空入力や不明な入力で先頭 option に倒さない
}

// ANSI escape codes
const (
	colorCyan  = "\033[36m"
	colorGreen = "\033[32m"
	colorDim   = "\033[2m"
	colorReset = "\033[0m"
)

// NewSelector は新しいSelectorを作成
func NewSelector(message string, options []SelectOption) *Selector {
	return &Selector{
		Message: message,
		Options: options,
	}
}

// Run は選択UIを表示し、選択結果を返す
func (s *Selector) Run() (string, error) {
	return s.RunWithIO(DefaultPromptIO())
}

// RunWithIO は選択UIを表示し、選択結果を返す。
func (s *Selector) RunWithIO(promptIO PromptIO) (string, error) {
	promptIO = normalizePromptIO(promptIO)

	// カーソルを表示（スピナー停止）
	stopSpinnerForPromptIO(promptIO)
	_, _ = fmt.Fprint(promptIO.Out, "\033[?25h")
	// メッセージを表示
	_, _ = fmt.Fprintf(promptIO.Out, "\n%s?%s %s\n\n", colorCyan, colorReset, s.Message)

	// 選択肢を表示
	for i, opt := range s.Options {
		marker := "  "
		if i == 0 {
			marker = "▶ "
		}
		_, _ = fmt.Fprintf(promptIO.Out, "  %s%d. %-8s%s - %s%s%s\n",
			marker, i+1, opt.Label, colorReset,
			colorDim, opt.Description, colorReset)
	}

	// ヒント表示
	if hint := selectorInputHint(s.Options, s.RequireExplicit); hint != "" {
		_, _ = fmt.Fprintf(promptIO.Out, "\n%s  (%s)%s\n", colorDim, hint, colorReset)
	}
	choicePrompt := "Choice [1]:"
	if s.RequireExplicit {
		choicePrompt = "Choice:"
	}
	for {
		_, _ = fmt.Fprintf(promptIO.Out, "%s%s%s ", colorCyan, choicePrompt, colorReset)

		input, err := promptIO.ReadSimpleLine()
		if err != nil {
			if s.RequireExplicit {
				return "", err
			}
			input = ""
		}
		input = strings.TrimSpace(strings.ToLower(input))

		selected, ok := resolveSelectorInput(s.Options, input, s.RequireExplicit)
		if !ok {
			_, _ = fmt.Fprintf(promptIO.Out, "%sPlease choose one of the listed options.%s\n", colorDim, colorReset)
			continue
		}
		if selected.defaulted {
			_, _ = fmt.Fprintf(promptIO.Out, "%s✓ %s (default)%s\n", colorGreen, selected.label, colorReset)
		} else {
			_, _ = fmt.Fprintf(promptIO.Out, "%s✓ %s%s\n", colorGreen, selected.label, colorReset)
		}
		return selected.value, nil
	}
}

type selectorResolvedInput struct {
	value     string
	label     string
	defaulted bool
}

func resolveSelectorInput(options []SelectOption, input string, requireExplicit bool) (selectorResolvedInput, bool) {
	if len(options) == 0 {
		return selectorResolvedInput{}, false
	}
	if input == "" {
		if requireExplicit {
			return selectorResolvedInput{}, false
		}
		return selectorResolvedInput{value: options[0].Value, label: options[0].Label, defaulted: true}, true
	}
	if len(input) == 1 && input[0] >= '1' && input[0] <= '9' {
		idx := int(input[0] - '1')
		if idx >= 0 && idx < len(options) {
			return selectorResolvedInput{value: options[idx].Value, label: options[idx].Label}, true
		}
	}

	for _, opt := range options {
		if selectorInputMatchesOption(input, opt) {
			return selectorResolvedInput{value: opt.Value, label: opt.Label}, true
		}
	}

	if requireExplicit {
		return selectorResolvedInput{}, false
	}
	return selectorResolvedInput{value: options[0].Value, label: options[0].Label, defaulted: true}, true
}

func selectorInputMatchesOption(input string, opt SelectOption) bool {
	return ConfirmPromptOptionMatchesInput(input, PromptOption(opt))
}

func selectorInputHint(options []SelectOption, requireExplicit bool) string {
	if len(options) == 0 {
		return ""
	}

	parts := make([]string, 0, len(options)+1)
	if !requireExplicit {
		parts = append(parts, "Enter="+options[0].Label)
	}
	for i, opt := range options {
		shortcuts := []string{fmt.Sprintf("%d", i+1)}
		if action, ok := ConfirmPromptActionFromValue(opt.Value); ok {
			switch action {
			case PromptActionYes:
				shortcuts = append(shortcuts, "y")
			case PromptActionNo:
				shortcuts = append(shortcuts, "n")
			case PromptActionComment:
				shortcuts = append(shortcuts, "c")
			}
		}
		parts = append(parts, strings.Join(shortcuts, "/")+"="+opt.Label)
	}
	return strings.Join(parts, ", ")
}
