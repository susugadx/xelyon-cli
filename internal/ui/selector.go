package ui

import (
	"fmt"
	"os"
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
	// カーソルを表示（スピナー停止）
	StopGlobalSpinner()
	fmt.Print("\033[?25h")
	// メッセージを表示
	fmt.Printf("\n%s?%s %s\n\n", colorCyan, colorReset, s.Message)

	// 選択肢を表示
	for i, opt := range s.Options {
		marker := "  "
		if i == 0 {
			marker = "▶ "
		}
		fmt.Printf("  %s%d. %-8s%s - %s%s%s\n",
			marker, i+1, opt.Label, colorReset,
			colorDim, opt.Description, colorReset)
	}

	// ヒント表示
	fmt.Printf("\n%s  (Enter=Yes, 2/n=No, 3/c=Comment)%s\n", colorDim, colorReset)
	fmt.Printf("%sChoice [1]:%s ", colorCyan, colorReset)

	// selector.go
	input := ""
	if reader := GetGlobalReader(); reader != nil {
		line, err := reader.ReadSimpleLine()
		if err == nil {
			input = line
		}
	} else {
		input = readLineFromStdin()
	}
	input = strings.TrimSpace(strings.ToLower(input))
	// 入力を解釈
	switch input {
	case "", "1", "y", "yes":
		fmt.Printf("%s✓ Yes%s\n", colorGreen, colorReset)
		return "yes", nil
	case "2", "n", "no":
		fmt.Printf("%s✓ No%s\n", colorGreen, colorReset)
		return "no", nil
	case "3", "c", "comment":
		fmt.Printf("%s✓ Comment%s\n", colorGreen, colorReset)
		return "comment", nil
	default:
		// 数字入力の場合
		if len(input) == 1 && input[0] >= '1' && input[0] <= '9' {
			idx := int(input[0] - '1')
			if idx < len(s.Options) {
				fmt.Printf("%s✓ %s%s\n", colorGreen, s.Options[idx].Label, colorReset)
				return s.Options[idx].Value, nil
			}
		}
		// 不明な入力はデフォルト（Yes）
		fmt.Printf("%s✓ Yes (default)%s\n", colorGreen, colorReset)
		return "yes", nil
	}
}

// readLineFromStdin は os.Stdin から1行読み取る（フォールバック用）
func readLineFromStdin() string {
	var buf []byte
	b := make([]byte, 1)

	for {
		n, err := os.Stdin.Read(b)
		if err != nil || n == 0 {
			break
		}
		if b[0] == '\n' {
			break
		}
		if b[0] == '\r' {
			continue
		}
		buf = append(buf, b[0])
	}

	return string(buf)
}

// ConfirmSelector は確認用の3択セレクター（Yes/No/Comment）
func ConfirmSelector(message string) (string, error) {
	selector := NewSelector(message, []SelectOption{
		{Label: "Yes", Description: "Execute the proposed change", Value: "yes"},
		{Label: "No", Description: "Skip this action", Value: "no"},
		{Label: "Comment", Description: "Provide feedback to AI", Value: "comment"},
	})
	return selector.Run()
}
