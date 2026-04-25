package viewport

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
)

// LinesFromContent は newline 区切りの content を viewport 用 line slice に変換する。
func LinesFromContent(content string) []string {
	return strings.Split(content, "\n")
}

// VisibleLines は offset と height に対応する可視行を返す。
func VisibleLines(lines []string, yOffset int, height int) []string {
	if len(lines) == 0 || height <= 0 {
		return nil
	}
	top := maxInt(0, yOffset)
	end := minInt(len(lines), top+height)
	return lines[top:end]
}

// Render は viewport の表示文字列を構築する。
func Render(lines []string, yOffset int, height int, width int) string {
	visible := VisibleLines(lines, yOffset, height)
	var sb strings.Builder
	sb.Grow(height * (width + 1))
	for i := 0; i < height; i++ {
		if i > 0 {
			sb.WriteByte('\n')
		}
		if i < len(visible) {
			sb.WriteString(termtext.FillANSITextWidth(visible[i], width, ""))
			continue
		}
		sb.WriteString(strings.Repeat(" ", maxInt(0, width)))
	}
	return sb.String()
}

// MaxYOffset は指定行数と高さで取り得る最大 yOffset を返す。
func MaxYOffset(lineCount int, height int) int {
	maxOffset := lineCount - height
	if maxOffset < 0 {
		return 0
	}
	return maxOffset
}

// ClampYOffset は yOffset を有効範囲に丸める。
func ClampYOffset(yOffset int, lineCount int, height int) int {
	if yOffset < 0 {
		return 0
	}
	maxOffset := MaxYOffset(lineCount, height)
	if yOffset > maxOffset {
		return maxOffset
	}
	return yOffset
}

// ScrollUp は yOffset を上方向へ移動した結果を返す。
func ScrollUp(yOffset int, n int, lineCount int, height int) int {
	return ClampYOffset(yOffset-n, lineCount, height)
}

// ScrollDown は yOffset を下方向へ移動した結果を返す。
func ScrollDown(yOffset int, n int, lineCount int, height int) int {
	return ClampYOffset(yOffset+n, lineCount, height)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
