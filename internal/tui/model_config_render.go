package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/config"
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

// configPaneWidths は3ペインの幅を計算する。
func configPaneWidths(totalWidth int) (left, mid, right int) {
	// 最小幅確保
	if totalWidth < 40 {
		return totalWidth, 0, 0
	}
	if totalWidth < 80 {
		left = totalWidth * 30 / 100
		mid = totalWidth - left
		right = 0
		return
	}
	left = max(18, totalWidth*20/100)
	mid = max(25, totalWidth*30/100)
	right = totalWidth - left - mid
	return
}

// configView は config screen の View を構築する。
func (m Model) configView() string {
	if m.configScreen == nil {
		return "Loading..."
	}
	cs := m.configScreen

	// 終了確認ダイアログ
	if cs.confirmQuit {
		return m.renderConfigConfirmDialog()
	}

	leftW, midW, rightW := configPaneWidths(m.width)
	bodyHeight := m.height - 2 // ヘッダー1行 + ステータス1行

	if bodyHeight < 1 {
		bodyHeight = 1
	}

	// ヘッダー
	header := m.renderConfigHeader(m.width)

	var body strings.Builder
	switch {
	case rightW > 0:
		leftLines := m.renderConfigCategoryPane(leftW, bodyHeight)
		midLines := m.renderConfigFieldPane(midW, bodyHeight)
		rightLines := m.renderConfigDetailPane(rightW, bodyHeight)
		for i := 0; i < bodyHeight; i++ {
			if i > 0 {
				body.WriteByte('\n')
			}
			var line string
			if i < len(leftLines) {
				line += leftLines[i]
			} else {
				line += fillANSITextWidth("", leftW, cfgBgNormal)
			}
			if i < len(midLines) {
				line += midLines[i]
			} else {
				line += fillANSITextWidth("", midW, cfgBgNormal)
			}
			if i < len(rightLines) {
				line += rightLines[i]
			} else {
				line += fillANSITextWidth("", rightW, cfgBgNormal)
			}
			body.WriteString(line)
		}

	case midW > 0:
		leftLines := m.renderConfigCategoryPane(leftW, bodyHeight)
		midLines := m.renderConfigFieldPane(midW, bodyHeight)
		if cs.editMode != editNone || cs.activePane == paneDetail {
			midLines = m.renderConfigDetailPane(midW, bodyHeight)
		}
		for i := 0; i < bodyHeight; i++ {
			if i > 0 {
				body.WriteByte('\n')
			}
			var line string
			if i < len(leftLines) {
				line += leftLines[i]
			} else {
				line += fillANSITextWidth("", leftW, cfgBgNormal)
			}
			if i < len(midLines) {
				line += midLines[i]
			} else {
				line += fillANSITextWidth("", midW, cfgBgNormal)
			}
			body.WriteString(line)
		}

	default:
		lines := m.renderConfigCategoryPane(leftW, bodyHeight)
		switch {
		case cs.editMode != editNone || cs.activePane == paneDetail:
			lines = m.renderConfigDetailPane(leftW, bodyHeight)
		case cs.activePane == paneField:
			lines = m.renderConfigFieldPane(leftW, bodyHeight)
		}
		for i := 0; i < bodyHeight; i++ {
			if i > 0 {
				body.WriteByte('\n')
			}
			if i < len(lines) {
				body.WriteString(lines[i])
			} else {
				body.WriteString(fillANSITextWidth("", leftW, cfgBgNormal))
			}
		}
	}

	// ステータスバー
	status := m.renderConfigStatus(m.width)

	return header + "\n" + body.String() + "\n" + status
}

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

	// 左: 保存状態
	var statusColor string
	statusMsg := cs.statusText()
	switch cs.saveStatus {
	case statusModified:
		statusColor = cfgFgYellow
	case statusFailed:
		statusColor = cfgFgRed
	case statusSaving:
		statusColor = cfgFgCyan
	default:
		statusColor = cfgFgGreen
	}
	left := " " + statusColor + statusMsg + cfgReset

	// 右: コンテキストヒント
	var hint string
	switch {
	case cs.filterMode:
		hint = "Enter:apply  Esc:cancel"
	case cs.editMode == editSelect:
		hint = "j/k:move  Enter:select  Esc:cancel"
	case cs.editMode == editInput:
		hint = "Enter:confirm  Esc:cancel"
	case cs.editMode == editSlice:
		if cs.editSliceAdding || cs.editSliceEditing {
			hint = "Enter:confirm  Esc:cancel"
		} else {
			hint = "a:add  d:delete  Enter:edit  Esc:done"
		}
	case cs.editMode == editStructMap:
		switch {
		case cs.editStructAdding:
			hint = "Enter:confirm  Esc:cancel"
		case cs.editEntryActive && cs.editEntryFieldEdit == "input":
			hint = "Enter:confirm  Esc:cancel"
		case cs.editEntryActive && cs.editEntryFieldEdit == "slice":
			if cs.editSliceAdding || cs.editSliceEditing {
				hint = "Enter:confirm  Esc:cancel"
			} else {
				hint = "a:add  d:delete  Enter:edit  Esc:done"
			}
		case cs.editEntryActive:
			hint = "j/k:move  Enter:edit  Space:toggle  Esc:back"
		default:
			hint = "j/k:move  Enter:edit entry  a:add  d:delete  Esc:done"
		}
	default:
		hint = "j/k:move  h/l:pane  Enter:edit  Space:toggle"
	}

	right := cfgFgDim + hint + cfgReset + " "
	padding := width - lipgloss.Width(statusMsg) - lipgloss.Width(hint) - 3
	if padding < 1 {
		return fillANSITextWidth(left, width, "")
	}
	return fitANSITextWidth(left+strings.Repeat(" ", padding)+right, width)
}

// renderConfigCategoryPane は左ペイン（カテゴリ一覧）を構築する。
func (m Model) renderConfigCategoryPane(width, height int) []string {
	cs := m.configScreen
	isActive := cs.activePane == paneCategory

	lines := make([]string, 0, height)
	for i, cat := range cs.categories {
		selected := i == cs.catIndex
		var bg string
		switch {
		case selected && isActive:
			bg = cfgBgSelected
		case selected:
			bg = cfgBgInactive
		default:
			bg = cfgBgNormal
		}
		fg := cfgFgNormal
		if selected && isActive {
			fg = cfgFgBright
		}

		name := cat.DisplayName
		prefix := "  "
		if selected {
			prefix = "> "
		}
		line := bg + fg + prefix + truncateWithANSI(name, width-3) + cfgReset
		lines = append(lines, fillANSITextWidth(line, width, bg))
	}

	// 高さを埋める
	for len(lines) < height {
		lines = append(lines, fillANSITextWidth("", width, cfgBgNormal))
	}
	return lines
}

// renderConfigFieldPane は中ペイン（フィールド一覧）を構築する。
func (m Model) renderConfigFieldPane(width, height int) []string {
	if width <= 0 {
		return nil
	}
	cs := m.configScreen
	isActive := cs.activePane == paneField
	fields := cs.filteredFields()

	hasFilter := cs.filterMode || cs.filterText != ""
	itemRows := height
	if hasFilter {
		itemRows-- // フィルタ行が最下部を使う
	}
	if itemRows < 0 {
		itemRows = 0
	}

	// fieldScroll を基準にスライス
	start := cs.fieldScroll
	if start > len(fields) {
		start = len(fields)
	}
	end := start + itemRows
	if end > len(fields) {
		end = len(fields)
	}

	lines := make([]string, 0, height)
	for idx := start; idx < end; idx++ {
		field := fields[idx]
		selected := idx == cs.fieldIndex
		var bg string
		switch {
		case selected && isActive:
			bg = cfgBgSelected
		case selected:
			bg = cfgBgInactive
		default:
			bg = cfgBgNormal
		}
		fg := cfgFgNormal
		if selected && isActive {
			fg = cfgFgBright
		}

		// フィールド名と現在値
		name := field.DisplayName
		val := formatConfigValue(field.Current, field.FieldType)

		prefix := "  "
		if selected {
			prefix = "> "
		}

		// bool は [x]/[ ] で表示
		if field.FieldType == config.FieldTypeBool {
			b, _ := field.Current.(bool)
			if b {
				val = "[x]"
			} else {
				val = "[ ]"
			}
			line := bg + fg + prefix + truncateWithANSI(name+" "+val, width-3) + cfgReset
			lines = append(lines, fillANSITextWidth(line, width, bg))
			continue
		}

		valColor := cfgFgDim
		maxNameW := width - 3
		nameW := lipgloss.Width(name) + 1
		valW := max(0, maxNameW-nameW)
		truncVal := truncateWithANSI(val, valW)

		line := bg + fg + prefix + truncateWithANSI(name, maxNameW) + " " + valColor + truncVal + cfgReset
		lines = append(lines, fillANSITextWidth(line, width, bg))
	}

	// 残りを空行で埋める
	for len(lines) < itemRows {
		lines = append(lines, fillANSITextWidth("", width, cfgBgNormal))
	}

	// フィルタ行
	if hasFilter {
		filterLine := cfgBgHeader + cfgFgCyan + " /" + cs.filterText + cfgReset
		if cs.filterMode {
			filterLine = cfgBgHeader + cfgFgCyan + " " + cs.filterInput.View() + cfgReset
		}
		lines = append(lines, fillANSITextWidth(filterLine, width, cfgBgHeader))
	}
	return lines
}

// renderConfigDetailPane は右ペイン（詳細/エディタ）を構築する。
func (m Model) renderConfigDetailPane(width, height int) []string {
	if width <= 0 {
		return nil
	}
	cs := m.configScreen
	field := cs.selectedField()

	lines := make([]string, 0, height)

	if field == nil {
		lines = append(lines, fillANSITextWidth(cfgBgNormal+cfgFgDim+"  No field selected"+cfgReset, width, cfgBgNormal))
		for len(lines) < height {
			lines = append(lines, fillANSITextWidth("", width, cfgBgNormal))
		}
		return lines
	}

	// フィールド詳細
	addLine := func(s string) {
		if len(lines) < height {
			lines = append(lines, fillANSITextWidth(cfgBgNormal+s+cfgReset, width, cfgBgNormal))
		}
	}

	addLine(cfgBold + cfgFgBright + "  " + field.DisplayName)
	addLine("")
	addLine(cfgFgDim + "  " + field.Description)
	addLine(cfgFgDim + "  path: " + field.Path)
	addLine(cfgFgDim + "  type: " + field.FieldType.String())
	addLine("")

	// 現在値
	addLine(cfgFgNormal + "  current: " + cfgFgBright + formatConfigValue(field.Current, field.FieldType))

	// デフォルト値
	if field.Default != nil {
		addLine(cfgFgDim + "  default: " + formatConfigValue(field.Default, field.FieldType))
	}
	addLine("")

	// 編集モードごとの表示
	switch cs.editMode {
	case editSelect:
		addLine(cfgFgCyan + "  Select:")
		for i, opt := range field.Options {
			marker := "  ( ) "
			if i == cs.editSelect {
				marker = "  (*) "
			}
			if i == cs.editSelect {
				addLine(cfgBgSelected + cfgFgBright + marker + opt)
			} else {
				addLine(cfgFgNormal + marker + opt)
			}
		}

	case editInput:
		addLine(cfgFgCyan + "  Edit:")
		addLine("  " + cfgFgBright + cs.editInput.View())

	case editSlice:
		addLine(cfgFgCyan + "  Items: (" + fmt.Sprintf("%d", len(cs.editSliceItems)) + ")")
		for i, item := range cs.editSliceItems {
			prefix := "    "
			bg := cfgBgNormal
			if i == cs.editSliceIndex {
				prefix = "  > "
				bg = cfgBgInactive
			}
			if cs.editSliceEditing && i == cs.editSliceIndex {
				addLine(bg + cfgFgBright + "  > " + cs.editSliceInput.View())
			} else {
				addLine(bg + cfgFgNormal + prefix + item)
			}
		}
		if cs.editSliceAdding {
			addLine(cfgFgCyan + "  + " + cs.editSliceInput.View())
		}

	case editStructMap:
		if cs.editEntryActive {
			// entry value 編集ビュー
			addLine(cfgFgCyan + "  " + cs.editEntryKey + ":")
			addLine("")
			for i, ef := range cs.editEntryFields {
				prefix := "    "
				bg := cfgBgNormal
				if i == cs.editEntryIndex {
					prefix = "  > "
					bg = cfgBgInactive
				}
				// 個別フィールド編集中はその行をエディタ表示
				if i == cs.editEntryIndex && cs.editEntryFieldEdit == "input" {
					addLine(bg + cfgFgBright + "  > " + ef.Name + ": " + cs.editInput.View())
					continue
				}
				if i == cs.editEntryIndex && cs.editEntryFieldEdit == "slice" {
					addLine(bg + cfgFgBright + "  > " + ef.Name + ":")
					// slice サブエディタ
					for si, item := range cs.editSliceItems {
						sp := "      "
						sbg := cfgBgNormal
						if si == cs.editSliceIndex {
							sp = "    > "
							sbg = cfgBgInactive
						}
						if cs.editSliceEditing && si == cs.editSliceIndex {
							addLine(sbg + cfgFgBright + "    > " + cs.editSliceInput.View())
						} else {
							addLine(sbg + cfgFgNormal + sp + item)
						}
					}
					if cs.editSliceAdding {
						addLine(cfgFgCyan + "    + " + cs.editSliceInput.View())
					}
					continue
				}
				// 通常表示
				valStr := entryFieldValueStr(ef)
				addLine(bg + cfgFgNormal + prefix + ef.Name + ": " + cfgFgDim + valStr)
			}
		} else {
			// key 一覧 + selected key の value summary
			addLine(cfgFgCyan + "  Keys: (" + fmt.Sprintf("%d", len(cs.editStructKeys)) + ")")
			for i, key := range cs.editStructKeys {
				prefix := "    "
				bg := cfgBgNormal
				if i == cs.editStructIndex {
					prefix = "  > "
					bg = cfgBgInactive
				}
				addLine(bg + cfgFgNormal + prefix + key)
			}
			if cs.editStructAdding {
				addLine(cfgFgCyan + "  + " + cs.editStructInput.View())
			}
			// selected key の value summary
			if cs.editStructIndex >= 0 && cs.editStructIndex < len(cs.editStructKeys) {
				key := cs.editStructKeys[cs.editStructIndex]
				summary := cs.loadEntryFields(field.Path, key)
				if len(summary) > 0 {
					addLine("")
					addLine(cfgFgCyan + "  " + key + ":")
					for _, ef := range summary {
						addLine(cfgFgDim + "    " + ef.Name + ": " + cfgFgNormal + entryFieldValueStr(ef))
					}
				}
			}
		}

	default:
		// 編集ヒント
		switch field.FieldType {
		case config.FieldTypeBool:
			addLine(cfgFgDim + "  Press Space or Enter to toggle")
		case config.FieldTypeSelect:
			addLine(cfgFgDim + "  Press Enter to select from options")
		case config.FieldTypeStringSlice:
			addLine(cfgFgDim + "  Press Enter to edit items")
		case config.FieldTypeStructMap:
			addLine(cfgFgDim + "  Press Enter to manage entries")
		default:
			addLine(cfgFgDim + "  Press Enter to edit")
		}
	}

	for len(lines) < height {
		lines = append(lines, fillANSITextWidth("", width, cfgBgNormal))
	}
	return lines
}

// renderConfigConfirmDialog は終了確認ダイアログを構築する。
func (m Model) renderConfigConfirmDialog() string {
	cs := m.configScreen
	width := m.width
	height := m.height

	options := []string{"Save and quit", "Discard and quit", "Cancel"}

	var sb strings.Builder
	midY := height / 2

	for i := 0; i < height; i++ {
		if i > 0 {
			sb.WriteByte('\n')
		}
		lineOffset := i - midY + 3

		switch {
		case lineOffset == -2:
			text := cfgBgHeader + cfgBold + cfgFgBright + "  Unsaved changes" + cfgReset
			sb.WriteString(fillANSITextWidth(text, width, cfgBgHeader))

		case lineOffset >= 0 && lineOffset < len(options):
			prefix := "  ( ) "
			bg := cfgBgNormal
			fg := cfgFgNormal
			if lineOffset == cs.confirmIdx {
				prefix = "  (*) "
				bg = cfgBgSelected
				fg = cfgFgBright
			}
			text := bg + fg + prefix + options[lineOffset] + cfgReset
			sb.WriteString(fillANSITextWidth(text, width, bg))

		default:
			sb.WriteString(strings.Repeat(" ", max(0, width)))
		}
	}
	return sb.String()
}

// entryFieldValueStr は structEntryField の値を簡潔な文字列にする。
func entryFieldValueStr(ef structEntryField) string {
	switch v := ef.Value.(type) {
	case bool:
		if v {
			return "[x]"
		}
		return "[ ]"
	case string:
		if v == "" {
			return "(empty)"
		}
		return v
	case int:
		return fmt.Sprintf("%d", v)
	case []string:
		if len(v) == 0 {
			return "[]"
		}
		return fmt.Sprintf("%d items", len(v))
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatConfigValue はフィールド値を表示用文字列に変換する。
func formatConfigValue(v interface{}, ft config.ConfigFieldType) string {
	if v == nil {
		return "(nil)"
	}
	switch val := v.(type) {
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case string:
		if val == "" {
			return "(empty)"
		}
		return val
	case []string:
		if len(val) == 0 {
			return "[]"
		}
		return fmt.Sprintf("%d items", len(val))
	case map[string]string:
		return fmt.Sprintf("%d entries", len(val))
	case map[string]config.ProviderModelConfig:
		return fmt.Sprintf("%d entries", len(val))
	case map[string]config.LSPServerConfig:
		return fmt.Sprintf("%d entries", len(val))
	default:
		return fmt.Sprintf("%v", val)
	}
}
