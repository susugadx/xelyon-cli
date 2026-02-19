package ui

import (
	"fmt"
	"strings"
)

// BoxStyle はボックスの罫線スタイル
type BoxStyle struct {
	TopLeft     string
	TopRight    string
	BottomLeft  string
	BottomRight string
	Horizontal  string
	Vertical    string
	LeftT       string
	RightT      string
}

// DefaultBoxStyle は標準のボックススタイル
var DefaultBoxStyle = BoxStyle{
	TopLeft:     "┌",
	TopRight:    "┐",
	BottomLeft:  "└",
	BottomRight: "┘",
	Horizontal:  "─",
	Vertical:    "│",
	LeftT:       "├",
	RightT:      "┤",
}

// ToolConfirmBox はツール確認用のボックスを表示
func ToolConfirmBox(toolName string, details []string) {
	width := 45

	// 上辺
	Cyan.Printf("%s%s%s\n", DefaultBoxStyle.TopLeft, strings.Repeat(DefaultBoxStyle.Horizontal, width-2), DefaultBoxStyle.TopRight)

	// ツール名（アイコン付き）
	icon := getToolIcon(toolName)
	title := fmt.Sprintf(" %s %s", icon, toolName)
	padding := width - 2 - len([]rune(title))
	if padding < 0 {
		padding = 0
	}
	Cyan.Printf("%s", DefaultBoxStyle.Vertical)
	Yellow.Printf("%s%s", title, strings.Repeat(" ", padding))
	Cyan.Printf("%s\n", DefaultBoxStyle.Vertical)

	// 詳細行
	for _, detail := range details {
		printBoxLine(detail, width)
	}

	// 区切り線
	Cyan.Printf("%s%s%s\n", DefaultBoxStyle.LeftT, strings.Repeat(DefaultBoxStyle.Horizontal, width-2), DefaultBoxStyle.RightT)

	// 選択肢
	options := " [y] Approve  [n] Reject  [c] Comment"
	optPadding := width - 2 - len(options)
	if optPadding < 0 {
		optPadding = 0
	}
	Cyan.Printf("%s", DefaultBoxStyle.Vertical)
	fmt.Printf("%s%s", options, strings.Repeat(" ", optPadding))
	Cyan.Printf("%s\n", DefaultBoxStyle.Vertical)

	// 下辺
	Cyan.Printf("%s%s%s\n", DefaultBoxStyle.BottomLeft, strings.Repeat(DefaultBoxStyle.Horizontal, width-2), DefaultBoxStyle.BottomRight)
}

// printBoxLine はボックス内の1行を表示
func printBoxLine(text string, width int) {
	// 長すぎる場合は切り詰め
	maxLen := width - 4
	displayText := text
	if len([]rune(text)) > maxLen {
		displayText = string([]rune(text)[:maxLen-3]) + "..."
	}

	padding := width - 2 - len([]rune(displayText)) - 1
	if padding < 0 {
		padding = 0
	}

	Cyan.Printf("%s", DefaultBoxStyle.Vertical)
	fmt.Printf(" %s%s", displayText, strings.Repeat(" ", padding))
	Cyan.Printf("%s\n", DefaultBoxStyle.Vertical)
}

// getToolIcon はツール名に対応するアイコンを返す
func getToolIcon(toolName string) string {
	icons := map[string]string{
		"read_file":     "📖",
		"write_file":    "📝",
		"str_replace":   "✏️",
		"delete_file":   "🗑️",
		"delete_lines":  "✂️",
		"move_file":     "📦",
		"copy_file":     "📋",
		"list_dir":      "📁",
		"bash":          "💻",
		"git_add":       "➕",
		"git_push":      "🚀",
		"git_status":    "📊",
		"git_diff":      "📄",
		"git_log":       "📜",
		"git_branch":    "🌿",
		"git_stash":     "📥",
		"web_search":    "🌐",
		"insert_after":  "⬇️",
		"insert_before": "⬆️",
	}

	if icon, ok := icons[toolName]; ok {
		return icon
	}
	return "🔧"
}

// SimpleDivider はシンプルな区切り線を表示
func SimpleDivider(width int) {
	Cyan.Println(strings.Repeat("─", width))
}

// ConfirmPromptBox は確認プロンプト用のボックスを表示
func ConfirmPromptBox(message string) {
	width := 45

	// 上辺
	Cyan.Printf("%s%s%s\n", DefaultBoxStyle.TopLeft, strings.Repeat(DefaultBoxStyle.Horizontal, width-2), DefaultBoxStyle.TopRight)

	// メッセージ
	displayMsg := message
	maxLen := width - 4
	if len([]rune(message)) > maxLen {
		displayMsg = string([]rune(message)[:maxLen-3]) + "..."
	}
	padding := width - 2 - len([]rune(displayMsg)) - 1
	if padding < 0 {
		padding = 0
	}
	Cyan.Printf("%s", DefaultBoxStyle.Vertical)
	Yellow.Printf(" %s%s", displayMsg, strings.Repeat(" ", padding))
	Cyan.Printf("%s\n", DefaultBoxStyle.Vertical)

	// 区切り線
	Cyan.Printf("%s%s%s\n", DefaultBoxStyle.LeftT, strings.Repeat(DefaultBoxStyle.Horizontal, width-2), DefaultBoxStyle.RightT)

	// 選択肢
	options := " [y] Approve  [n] Reject  [c] Comment"
	optPadding := width - 2 - len(options)
	if optPadding < 0 {
		optPadding = 0
	}
	Cyan.Printf("%s", DefaultBoxStyle.Vertical)
	fmt.Printf("%s%s", options, strings.Repeat(" ", optPadding))
	Cyan.Printf("%s\n", DefaultBoxStyle.Vertical)

	// 下辺
	Cyan.Printf("%s%s%s\n", DefaultBoxStyle.BottomLeft, strings.Repeat(DefaultBoxStyle.Horizontal, width-2), DefaultBoxStyle.BottomRight)
}
