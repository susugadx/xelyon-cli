package ui

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// SelectOption は選択肢の情報
type SelectOption struct {
	Label       string // 表示ラベル（例: "Yes"）
	Description string // 説明文（例: "Execute the proposed change"）
	Value       string // 戻り値（例: "yes"）
}

// Selector は矢印キー選択UI
type Selector struct {
	Message  string         // 質問メッセージ
	Options  []SelectOption // 選択肢
	Selected int            // 現在選択中のインデックス
}

// ANSI escape codes
const (
	cursorUp       = "\033[A"
	cursorDown     = "\033[B"
	clearLine      = "\033[2K"
	cursorToStart  = "\033[G"
	hideCursor     = "\033[?25l"
	showCursor     = "\033[?25h"
	colorCyan      = "\033[36m"
	colorDim       = "\033[2m"
	colorReset     = "\033[0m"
	colorBold      = "\033[1m"
	colorHighlight = "\033[48;5;236m" // Dark gray background
)

// NewSelector は新しいSelectorを作成
func NewSelector(message string, options []SelectOption) *Selector {
	return &Selector{
		Message:  message,
		Options:  options,
		Selected: 0,
	}
}

// Run は選択UIを表示し、選択結果を返す
// 戻り値: 選択されたオプションのValue, エラー
func (s *Selector) Run() (string, error) {
	fd := int(os.Stdin.Fd())

	// ターミナルでない場合はフォールバック
	if !term.IsTerminal(fd) {
		return s.runFallback()
	}

	// Raw mode に切り替え
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return s.runFallback()
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	// カーソルを非表示
	fmt.Print(hideCursor)
	defer fmt.Print(showCursor)

	// 初期描画
	s.render()

	// 入力ループ
	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return "", err
		}

		// キー処理
		if n == 1 {
			switch buf[0] {
			case 'q', 0x1b: // q or Esc
				s.clearDisplay()
				return "", fmt.Errorf("cancelled")
			case 0x03: // Ctrl+C
				s.clearDisplay()
				return "", fmt.Errorf("interrupted")
			case '\r', '\n': // Enter
				s.clearDisplay()
				return s.Options[s.Selected].Value, nil
			case 'k', 'K': // vim style up
				s.moveUp()
				s.render()
			case 'j', 'J': // vim style down
				s.moveDown()
				s.render()
			case '1', '2', '3', '4', '5', '6', '7', '8', '9':
				idx := int(buf[0] - '1')
				if idx < len(s.Options) {
					s.Selected = idx
					s.clearDisplay()
					return s.Options[s.Selected].Value, nil
				}
			case 'y', 'Y': // shortcut for yes
				for i, opt := range s.Options {
					if opt.Value == "yes" {
						s.Selected = i
						s.clearDisplay()
						return "yes", nil
					}
				}
			case 'n', 'N': // shortcut for no
				for i, opt := range s.Options {
					if opt.Value == "no" {
						s.Selected = i
						s.clearDisplay()
						return "no", nil
					}
				}
			case 'c', 'C': // shortcut for comment
				for i, opt := range s.Options {
					if opt.Value == "comment" {
						s.Selected = i
						s.clearDisplay()
						return "comment", nil
					}
				}
			}
		} else if n == 3 && buf[0] == 0x1b && buf[1] == '[' {
			// Arrow keys
			switch buf[2] {
			case 'A': // Up
				s.moveUp()
				s.render()
			case 'B': // Down
				s.moveDown()
				s.render()
			}
		}
	}
}

// moveUp は選択を上に移動
func (s *Selector) moveUp() {
	if s.Selected > 0 {
		s.Selected--
	} else {
		s.Selected = len(s.Options) - 1 // ループ
	}
}

// moveDown は選択を下に移動
func (s *Selector) moveDown() {
	if s.Selected < len(s.Options)-1 {
		s.Selected++
	} else {
		s.Selected = 0 // ループ
	}
}

// render は選択UIを描画
func (s *Selector) render() {
	// カーソルを先頭に戻して再描画
	// 最初の描画時以外は上に移動
	totalLines := len(s.Options) + 3 // message + options + hint + blank
	for i := 0; i < totalLines; i++ {
		fmt.Print(cursorUp)
	}

	s.draw()
}

// draw は選択UIを描画（初回用）
func (s *Selector) draw() {
	// メッセージ
	fmt.Printf("%s? %s%s\r\n", colorCyan, s.Message, colorReset)
	fmt.Print("\r\n")

	// 選択肢
	for i, opt := range s.Options {
		if i == s.Selected {
			// 選択中: ハイライト + 矢印
			fmt.Printf("   %s▶  %s%s%s  %s%s%s\r\n",
				colorCyan,
				colorBold, opt.Label, colorReset,
				colorDim, opt.Description, colorReset)
		} else {
			// 非選択: インデント
			fmt.Printf("      %s  %s%s%s\r\n",
				opt.Label,
				colorDim, opt.Description, colorReset)
		}
	}

	// ヒント
	fmt.Printf("\r\n   %s↑/↓ navigate · Enter confirm · y/n/c shortcut%s\r\n", colorDim, colorReset)
}

// clearDisplay は表示をクリア
func (s *Selector) clearDisplay() {
	totalLines := len(s.Options) + 3
	for i := 0; i < totalLines; i++ {
		fmt.Print(cursorUp + clearLine)
	}
	fmt.Print(cursorToStart)
}

// runFallback はターミナルでない場合のフォールバック
func (s *Selector) runFallback() (string, error) {
	fmt.Printf("? %s\n", s.Message)
	for i, opt := range s.Options {
		fmt.Printf("  %d. %s - %s\n", i+1, opt.Label, opt.Description)
	}
	fmt.Print("Choice (1-", len(s.Options), "): ")

	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		return "", err
	}

	// 数字入力
	if len(input) == 1 && input[0] >= '1' && input[0] <= '9' {
		idx := int(input[0] - '1')
		if idx < len(s.Options) {
			return s.Options[idx].Value, nil
		}
	}

	// ショートカット
	switch input {
	case "y", "yes":
		return "yes", nil
	case "n", "no":
		return "no", nil
	case "c", "comment":
		return "comment", nil
	}

	return "", fmt.Errorf("invalid input")
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
