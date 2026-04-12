package tui

import "github.com/rivo/uniseg"

// nextClusterStart は現在のカーソル列の次の grapheme cluster 先頭列を返す。
func (m Model) nextClusterStart(line, col int) int {
	if line < 0 || line >= len(m.rawLines) {
		return col
	}
	plain := stripANSI(m.rawLines[line])
	width := 0
	gr := uniseg.NewGraphemes(plain)
	for gr.Next() {
		cw := plainTextDisplayWidth(gr.Str())
		if width+cw > col {
			return width + cw
		}
		width += cw
	}
	return col
}

// prevClusterStart は現在のカーソル列の前の grapheme cluster 先頭列を返す。
func (m Model) prevClusterStart(line, col int) int {
	if line < 0 || line >= len(m.rawLines) {
		return col
	}
	plain := stripANSI(m.rawLines[line])
	prevStart := 0
	width := 0
	gr := uniseg.NewGraphemes(plain)
	for gr.Next() {
		cw := plainTextDisplayWidth(gr.Str())
		if width+cw >= col {
			if width < col {
				return width
			}
			return prevStart
		}
		prevStart = width
		width += cw
	}
	return prevStart
}
