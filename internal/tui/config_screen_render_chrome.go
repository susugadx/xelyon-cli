package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ANSI カラー定数（config screen 用）
const (
	cfgBgNormal   = "\033[48;5;235m" // ペイン背景
	cfgBgSelected = "\033[48;5;25m"  // 選択行（アクティブペイン）
	cfgBgInactive = "\033[48;5;238m" // 選択行（非アクティブペイン）
	cfgBgHeader   = "\033[48;5;236m" // ヘッダー背景
	cfgFgNormal   = "\033[38;5;252m" // 通常テキスト
	cfgFgDim      = "\033[38;5;244m" // 薄いテキスト
	cfgFgBright   = "\033[38;5;255m" // 明るいテキスト
	cfgFgGreen    = "\033[38;5;82m"  // 緑
	cfgFgYellow   = "\033[38;5;220m" // 黄
	cfgFgRed      = "\033[38;5;196m" // 赤
	cfgFgCyan     = "\033[38;5;87m"  // シアン
	cfgReset      = "\033[0m"
	cfgBold       = "\033[1m"
)

// renderConfigHeader はヘッダー行を構築する。
func (m Model) renderConfigHeader(width int) string {
	title := cfgBgHeader + cfgBold + cfgFgBright + " Configuration " + cfgReset
	hints := cfgFgDim + "q:close  s:save  /:filter  r:reset" + cfgReset
	padding := width - lipgloss.Width("Configuration") - 2 - lipgloss.Width("q:close  s:save  /:filter  r:reset")
	if padding < 2 {
		return fillANSITextWidth(title, width, cfgBgHeader)
	}
	return fillANSITextWidth(title+strings.Repeat(" ", padding)+hints, width, cfgBgHeader)
}

// renderConfigStatus はステータスバーを構築する。
func (m Model) renderConfigStatus(width int) string {
	cs := m.configScreen
	statusMsg := cs.statusText()
	left := " " + configStatusColor(cs.saveStatus) + statusMsg + cfgReset
	hint := configStatusHint(cs)
	right := cfgFgDim + hint + cfgReset + " "
	padding := width - lipgloss.Width(statusMsg) - lipgloss.Width(hint) - 3
	if padding < 1 {
		return fillANSITextWidth(left, width, "")
	}
	return fitANSITextWidth(left+strings.Repeat(" ", padding)+right, width)
}

func configStatusColor(status configSaveStatus) string {
	switch status {
	case statusModified:
		return cfgFgYellow
	case statusFailed:
		return cfgFgRed
	case statusSaving:
		return cfgFgCyan
	default:
		return cfgFgGreen
	}
}

func configStatusHint(cs *configScreen) string {
	switch {
	case cs.filterMode:
		return "Enter:apply  Esc:cancel"
	case cs.editMode == editSelect:
		return "j/k:move  Enter:select  Esc:cancel"
	case cs.editMode == editInput:
		return "Enter:confirm  Esc:cancel"
	case cs.editMode == editSlice:
		if cs.editSliceAdding || cs.editSliceEditing {
			return "Enter:confirm  Esc:cancel"
		}
		return "a:add  d:delete  Enter:edit  Esc:done"
	case cs.editMode == editStructMap:
		switch {
		case cs.editStructAdding:
			return "Enter:confirm  Esc:cancel"
		case cs.editEntryActive && cs.editEntryFieldEdit == "input":
			return "Enter:confirm  Esc:cancel"
		case cs.editEntryActive && cs.editEntryFieldEdit == "slice":
			if cs.editSliceAdding || cs.editSliceEditing {
				return "Enter:confirm  Esc:cancel"
			}
			return "a:add  d:delete  Enter:edit  Esc:done"
		case cs.editEntryActive:
			return "j/k:move  Enter:edit  Space:toggle  Esc:back"
		default:
			return "j/k:move  Enter:edit entry  a:add  d:delete  Esc:done"
		}
	default:
		return "j/k:move  h/l:pane  Enter:edit  Space:toggle"
	}
}

// renderConfigConfirmDialog は終了確認ダイアログを構築する。
func (m Model) renderConfigConfirmDialog() string {
	options := []string{"Save and quit", "Discard and quit", "Cancel"}

	var sb strings.Builder
	midY := m.height / 2
	for row := 0; row < m.height; row++ {
		if row > 0 {
			sb.WriteByte('\n')
		}
		lineOffset := row - midY + 3

		switch {
		case lineOffset == -2:
			text := cfgBgHeader + cfgBold + cfgFgBright + "  Unsaved changes" + cfgReset
			sb.WriteString(fillANSITextWidth(text, m.width, cfgBgHeader))
		case lineOffset >= 0 && lineOffset < len(options):
			sb.WriteString(m.renderConfigConfirmOption(lineOffset, options[lineOffset]))
		default:
			sb.WriteString(strings.Repeat(" ", max(0, m.width)))
		}
	}
	return sb.String()
}

func (m Model) renderConfigConfirmOption(index int, label string) string {
	bg := cfgBgNormal
	fg := cfgFgNormal
	prefix := "  ( ) "
	if index == m.configScreen.confirmIdx {
		bg = cfgBgSelected
		fg = cfgFgBright
		prefix = "  (*) "
	}
	text := bg + fg + prefix + label + cfgReset
	return fillANSITextWidth(text, m.width, bg)
}
