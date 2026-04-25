package confirmdialog

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

// Render は指定 option を持つ確認ダイアログを描画する。
func Render(width, height int, title string, options []string, selected int, palette theme.ConfigPalette) string {
	var sb strings.Builder
	midY := height / 2
	for row := 0; row < height; row++ {
		if row > 0 {
			sb.WriteByte('\n')
		}
		lineOffset := row - midY + 3

		switch {
		case lineOffset == -2:
			text := palette.BgHeader + palette.Bold + palette.FgBright + "  " + title + palette.Reset
			sb.WriteString(termtext.FillANSITextWidth(text, width, palette.BgHeader))
		case lineOffset >= 0 && lineOffset < len(options):
			sb.WriteString(renderOption(width, options[lineOffset], lineOffset == selected, palette))
		default:
			sb.WriteString(strings.Repeat(" ", maxInt(0, width)))
		}
	}
	return sb.String()
}

func renderOption(width int, label string, selected bool, palette theme.ConfigPalette) string {
	bg := palette.BgNormal
	fg := palette.FgNormal
	prefix := "  ( ) "
	if selected {
		bg = palette.BgSelected
		fg = palette.FgBright
		prefix = "  (*) "
	}
	text := bg + fg + prefix + label + palette.Reset
	return termtext.FillANSITextWidth(text, width, bg)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
