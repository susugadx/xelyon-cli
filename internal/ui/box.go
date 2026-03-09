package ui

import (
	"fmt"
	"io"
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

// ToolConfirmBoxWithRuntime は runtime の出力先にツール確認用ボックスを表示する。
func ToolConfirmBoxWithRuntime(runtime *Runtime, toolName string, details []string) {
	ToolConfirmBoxToWriter(runtimeOrDefault(runtime).Output(), toolName, details)
}

// ToolConfirmBoxToWriter は指定 writer にツール確認用ボックスを表示する。
func ToolConfirmBoxToWriter(out io.Writer, toolName string, details []string) {
	if out == nil {
		return
	}

	width := 45

	// 上辺
	Cyan.Fprintf(out, "%s%s%s\n", DefaultBoxStyle.TopLeft, strings.Repeat(DefaultBoxStyle.Horizontal, width-2), DefaultBoxStyle.TopRight)

	// ツール名（アイコン付き）
	icon := getToolIcon(toolName)
	title := fmt.Sprintf(" %s %s", icon, toolName)
	padding := width - 2 - len([]rune(title))
	if padding < 0 {
		padding = 0
	}
	Cyan.Fprintf(out, "%s", DefaultBoxStyle.Vertical)
	Yellow.Fprintf(out, "%s%s", title, strings.Repeat(" ", padding))
	Cyan.Fprintf(out, "%s\n", DefaultBoxStyle.Vertical)

	// 詳細行
	for _, detail := range details {
		printBoxLineToWriter(out, detail, width)
	}

	// 区切り線
	Cyan.Fprintf(out, "%s%s%s\n", DefaultBoxStyle.LeftT, strings.Repeat(DefaultBoxStyle.Horizontal, width-2), DefaultBoxStyle.RightT)

	// 選択肢
	options := " [y] Approve  [n] Reject  [c] Comment"
	optPadding := width - 2 - len(options)
	if optPadding < 0 {
		optPadding = 0
	}
	Cyan.Fprintf(out, "%s", DefaultBoxStyle.Vertical)
	_, _ = fmt.Fprintf(out, "%s%s", options, strings.Repeat(" ", optPadding))
	Cyan.Fprintf(out, "%s\n", DefaultBoxStyle.Vertical)

	// 下辺
	Cyan.Fprintf(out, "%s%s%s\n", DefaultBoxStyle.BottomLeft, strings.Repeat(DefaultBoxStyle.Horizontal, width-2), DefaultBoxStyle.BottomRight)
}

func printBoxLineToWriter(out io.Writer, text string, width int) {
	if out == nil {
		return
	}

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

	Cyan.Fprintf(out, "%s", DefaultBoxStyle.Vertical)
	_, _ = fmt.Fprintf(out, " %s%s", displayText, strings.Repeat(" ", padding))
	Cyan.Fprintf(out, "%s\n", DefaultBoxStyle.Vertical)
}

// getToolIcon はツール名に対応するアイコンを返す
func getToolIcon(toolName string) string {
	icons := map[string]string{
		"read_file":   "📖",
		"write_file":  "📝",
		"str_replace": "✏️",
		"delete_file": "🗑️",
		"copy_file":   "📋",
		"list_dir":    "📁",
		"bash":        "💻",
		"git_add":     "➕",
		"git_push":    "🚀",
		"git_status":  "📊",
		"git_diff":    "📄",
		"git_log":     "📜",
		"git_branch":  "🌿",
		"git_stash":   "📥",
		"web_search":  "🌐",
	}

	if icon, ok := icons[toolName]; ok {
		return icon
	}
	return "🔧"
}

// SimpleDividerWithRuntime は runtime の出力先に区切り線を表示する。
func SimpleDividerWithRuntime(runtime *Runtime, width int) {
	SimpleDividerToWriter(runtimeOrDefault(runtime).Output(), width)
}

// SimpleDividerToWriter は指定 writer に区切り線を表示する。
func SimpleDividerToWriter(out io.Writer, width int) {
	if out == nil {
		return
	}
	Cyan.Fprintln(out, strings.Repeat("─", width))
}

// ConfirmPromptBoxWithRuntime は runtime の出力先に確認プロンプト用ボックスを表示する。
func ConfirmPromptBoxWithRuntime(runtime *Runtime, message string) {
	ConfirmPromptBoxToWriter(runtimeOrDefault(runtime).Output(), message)
}

// ConfirmPromptBoxToWriter は指定 writer に確認プロンプト用ボックスを表示する。
func ConfirmPromptBoxToWriter(out io.Writer, message string) {
	if out == nil {
		return
	}

	width := 45

	// 上辺
	Cyan.Fprintf(out, "%s%s%s\n", DefaultBoxStyle.TopLeft, strings.Repeat(DefaultBoxStyle.Horizontal, width-2), DefaultBoxStyle.TopRight)

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
	Cyan.Fprintf(out, "%s", DefaultBoxStyle.Vertical)
	Yellow.Fprintf(out, " %s%s", displayMsg, strings.Repeat(" ", padding))
	Cyan.Fprintf(out, "%s\n", DefaultBoxStyle.Vertical)

	// 区切り線
	Cyan.Fprintf(out, "%s%s%s\n", DefaultBoxStyle.LeftT, strings.Repeat(DefaultBoxStyle.Horizontal, width-2), DefaultBoxStyle.RightT)

	// 選択肢
	options := " [y] Approve  [n] Reject  [c] Comment"
	optPadding := width - 2 - len(options)
	if optPadding < 0 {
		optPadding = 0
	}
	Cyan.Fprintf(out, "%s", DefaultBoxStyle.Vertical)
	_, _ = fmt.Fprintf(out, "%s%s", options, strings.Repeat(" ", optPadding))
	Cyan.Fprintf(out, "%s\n", DefaultBoxStyle.Vertical)

	// 下辺
	Cyan.Fprintf(out, "%s%s%s\n", DefaultBoxStyle.BottomLeft, strings.Repeat(DefaultBoxStyle.Horizontal, width-2), DefaultBoxStyle.BottomRight)
}
