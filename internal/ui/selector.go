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
	Message string         // 質問メッセージ
	Options []SelectOption // 選択肢
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
	StopGlobalSpinner()
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
	_, _ = fmt.Fprintf(promptIO.Out, "\n%s  (Enter=Yes, 2/n=No, 3/c=Comment)%s\n", colorDim, colorReset)
	_, _ = fmt.Fprintf(promptIO.Out, "%sChoice [1]:%s ", colorCyan, colorReset)

	input, err := promptIO.ReadSimpleLine()
	if err != nil {
		input = ""
	}
	input = strings.TrimSpace(strings.ToLower(input))
	// 入力を解釈
	switch input {
	case "", "1", "y", "yes":
		_, _ = fmt.Fprintf(promptIO.Out, "%s✓ Yes%s\n", colorGreen, colorReset)
		return "yes", nil
	case "2", "n", "no":
		_, _ = fmt.Fprintf(promptIO.Out, "%s✓ No%s\n", colorGreen, colorReset)
		return "no", nil
	case "3", "c", "comment":
		_, _ = fmt.Fprintf(promptIO.Out, "%s✓ Comment%s\n", colorGreen, colorReset)
		return "comment", nil
	default:
		// 数字入力の場合
		if len(input) == 1 && input[0] >= '1' && input[0] <= '9' {
			idx := int(input[0] - '1')
			if idx < len(s.Options) {
				_, _ = fmt.Fprintf(promptIO.Out, "%s✓ %s%s\n", colorGreen, s.Options[idx].Label, colorReset)
				return s.Options[idx].Value, nil
			}
		}
		// 不明な入力はデフォルト（Yes）
		_, _ = fmt.Fprintf(promptIO.Out, "%s✓ Yes (default)%s\n", colorGreen, colorReset)
		return "yes", nil
	}
}

// ConfirmSelector は確認用の3択セレクター（Yes/No/Comment）
func ConfirmSelector(message string) (string, error) {
	return ConfirmSelectorWithIO(DefaultPromptIO(), message)
}

// ConfirmSelectorWithIO は入出力先を指定した確認用の3択セレクター。
func ConfirmSelectorWithIO(promptIO PromptIO, message string) (string, error) {
	selector := NewSelector(message, []SelectOption{
		{Label: "Yes", Description: "Execute the proposed change", Value: "yes"},
		{Label: "No", Description: "Skip this action", Value: "no"},
		{Label: "Comment", Description: "Provide feedback to AI", Value: "comment"},
	})
	return selector.RunWithIO(promptIO)
}
