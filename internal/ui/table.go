package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// Table はテーブル表示用の構造体
type Table struct {
	Headers []string     // ヘッダー行（オプション）
	Rows    [][]string   // データ行
	colors  []*ColorFunc // 各列の色（オプション）
}

// ColorFunc は色付け関数の型
type ColorFunc func(format string, a ...interface{})

// NewTable は新しいテーブルを作成
func NewTable() *Table {
	return &Table{
		Headers: nil,
		Rows:    nil,
		colors:  nil,
	}
}

// SetHeaders はヘッダーを設定
func (t *Table) SetHeaders(headers ...string) *Table {
	t.Headers = headers
	return t
}

// AddRow は行を追加
func (t *Table) AddRow(cells ...string) *Table {
	t.Rows = append(t.Rows, cells)
	return t
}

// SetColumnColors は列ごとの色を設定
func (t *Table) SetColumnColors(colors ...*ColorFunc) *Table {
	t.colors = colors
	return t
}

// Render はテーブルを罫線付きで描画して文字列を返す
func (t *Table) Render() string {
	if len(t.Rows) == 0 && len(t.Headers) == 0 {
		return ""
	}

	// 列数を決定
	colCount := t.getColumnCount()
	if colCount == 0 {
		return ""
	}

	// 各列の最大幅を計算
	widths := t.calculateColumnWidths(colCount)

	var sb strings.Builder

	// 上罫線
	sb.WriteString(t.drawLine(widths, "┌", "┬", "┐"))
	sb.WriteString("\n")

	// ヘッダー行（存在する場合）
	if len(t.Headers) > 0 {
		sb.WriteString(t.drawRow(t.Headers, widths, true))
		sb.WriteString("\n")
		// ヘッダーと本体の区切り線
		sb.WriteString(t.drawLine(widths, "├", "┼", "┤"))
		sb.WriteString("\n")
	}

	// データ行
	for i, row := range t.Rows {
		sb.WriteString(t.drawRow(row, widths, false))
		sb.WriteString("\n")

		// 行間の区切り線（最後の行以外）
		if i < len(t.Rows)-1 {
			sb.WriteString(t.drawLine(widths, "├", "┼", "┤"))
			sb.WriteString("\n")
		}
	}

	// 下罫線
	sb.WriteString(t.drawLine(widths, "└", "┴", "┘"))
	sb.WriteString("\n")

	return sb.String()
}

// RenderCompact はコンパクト形式（行間の区切り線なし）で描画
func (t *Table) RenderCompact() string {
	if len(t.Rows) == 0 && len(t.Headers) == 0 {
		return ""
	}

	colCount := t.getColumnCount()
	if colCount == 0 {
		return ""
	}

	widths := t.calculateColumnWidths(colCount)

	var sb strings.Builder

	// 上罫線
	sb.WriteString(t.drawLine(widths, "┌", "┬", "┐"))
	sb.WriteString("\n")

	// ヘッダー行
	if len(t.Headers) > 0 {
		sb.WriteString(t.drawRow(t.Headers, widths, true))
		sb.WriteString("\n")
		sb.WriteString(t.drawLine(widths, "├", "┼", "┤"))
		sb.WriteString("\n")
	}

	// データ行（区切り線なし）
	for _, row := range t.Rows {
		sb.WriteString(t.drawRow(row, widths, false))
		sb.WriteString("\n")
	}

	// 下罫線
	sb.WriteString(t.drawLine(widths, "└", "┴", "┘"))
	sb.WriteString("\n")

	return sb.String()
}

// getColumnCount は列数を取得
func (t *Table) getColumnCount() int {
	maxCols := len(t.Headers)
	for _, row := range t.Rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}
	return maxCols
}

// calculateColumnWidths は各列の最大幅を計算
func (t *Table) calculateColumnWidths(colCount int) []int {
	widths := make([]int, colCount)

	// ヘッダーの幅を考慮
	for i, h := range t.Headers {
		if i < colCount {
			w := stringWidth(h)
			if w > widths[i] {
				widths[i] = w
			}
		}
	}

	// データ行の幅を考慮
	for _, row := range t.Rows {
		for i, cell := range row {
			if i < colCount {
				w := stringWidth(cell)
				if w > widths[i] {
					widths[i] = w
				}
			}
		}
	}

	return widths
}

// drawLine は罫線を描画
func (t *Table) drawLine(widths []int, left, mid, right string) string {
	var sb strings.Builder
	sb.WriteString(left)
	for i, w := range widths {
		sb.WriteString(strings.Repeat("─", w+2)) // パディング分+2
		if i < len(widths)-1 {
			sb.WriteString(mid)
		}
	}
	sb.WriteString(right)
	return sb.String()
}

// drawRow は行を描画
func (t *Table) drawRow(cells []string, widths []int, isHeader bool) string {
	var sb strings.Builder
	sb.WriteString("│")

	for i, w := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}

		// セルの表示幅を計算
		cellWidth := stringWidth(cell)
		padding := w - cellWidth

		// 左パディング + セル内容 + 右パディング
		sb.WriteString(" ")
		sb.WriteString(cell)
		sb.WriteString(strings.Repeat(" ", padding+1))
		sb.WriteString("│")
	}

	return sb.String()
}

// stringWidth は文字列の表示幅を計算（全角=2、半角=1）
func stringWidth(s string) int {
	return runewidth.StringWidth(s)
}

// StringWidth は文字列の表示幅を計算（外部公開版）
func StringWidth(s string) int {
	return stringWidth(s)
}

// runeWidth は1文字の表示幅を計算
func runeWidth(r rune) int {
	return runewidth.RuneWidth(r)
}

// PadRight は文字列を右パディング（表示幅ベース）
func PadRight(s string, width int) string {
	w := stringWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// PadLeft は文字列を左パディング（表示幅ベース）
func PadLeft(s string, width int) string {
	w := stringWidth(s)
	if w >= width {
		return s
	}
	return strings.Repeat(" ", width-w) + s
}

// PadCenter は文字列を中央揃え（表示幅ベース）
func PadCenter(s string, width int) string {
	w := stringWidth(s)
	if w >= width {
		return s
	}
	leftPad := (width - w) / 2
	rightPad := width - w - leftPad
	return strings.Repeat(" ", leftPad) + s + strings.Repeat(" ", rightPad)
}

// TruncateWidth は文字列を指定幅で切り詰め（表示幅ベース）
func TruncateWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	sw := stringWidth(s)
	if sw <= maxWidth {
		return s
	}

	// 省略記号を付けるために幅を確保
	if maxWidth < 3 {
		// 幅が3未満の場合は省略記号なしで切り詰め
		currentWidth := 0
		var result strings.Builder
		for _, r := range s {
			rw := runeWidth(r)
			if currentWidth+rw > maxWidth {
				break
			}
			result.WriteRune(r)
			currentWidth += rw
		}
		return result.String()
	}

	// 省略記号分の幅を確保して切り詰め
	targetWidth := maxWidth - 3 // "..." の幅
	currentWidth := 0
	var result strings.Builder

	for _, r := range s {
		rw := runeWidth(r)
		if currentWidth+rw > targetWidth {
			break
		}
		result.WriteRune(r)
		currentWidth += rw
	}

	result.WriteString("...")
	return result.String()
}

// KeyValueTable はキー・バリュー形式のシンプルなテーブルを作成
func KeyValueTable(pairs ...string) string {
	if len(pairs) == 0 || len(pairs)%2 != 0 {
		return ""
	}

	t := NewTable()
	for i := 0; i < len(pairs); i += 2 {
		t.AddRow(pairs[i], pairs[i+1])
	}
	return t.Render()
}

// SimpleTable はヘッダー付きのシンプルなテーブルを作成
func SimpleTable(headers []string, rows [][]string) string {
	t := NewTable()
	t.SetHeaders(headers...)
	for _, row := range rows {
		t.AddRow(row...)
	}
	return t.RenderCompact()
}

// CountRunes は文字列のルーン数を返す（バイト数ではなく文字数）
func CountRunes(s string) int {
	return utf8.RuneCountInString(s)
}
