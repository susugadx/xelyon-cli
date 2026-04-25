package termtext

import (
	"strings"

	"github.com/rivo/uniseg"
)

// StylePlainTextRange は plain text の表示列範囲に背景 style を適用する。
func StylePlainTextRange(s string, startCol, endCol int, bg string) string {
	if bg == "" || startCol >= endCol {
		return s
	}
	var result ansiStyleBuilder
	width := 0
	inRange := false
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		cluster := gr.Str()
		cw := PlainTextDisplayWidth(cluster)
		highlight := width < endCol && width+cw > startCol
		if highlight && !inRange {
			result.write(bg)
			inRange = true
		} else if !highlight && inRange {
			result.reset()
			inRange = false
		}
		result.write(cluster)
		width += cw
	}
	if inRange {
		result.reset()
	}
	return result.String()
}

// StylePlainTextRangeWithCursor は選択範囲と cursor 位置を同時に style 付けする。
func StylePlainTextRangeWithCursor(s string, startCol, endCol int, rangeBg string, cursorCol int, cursorBg string, lineBg string) string {
	if s == "" {
		if cursorCol == 0 {
			if lineBg != "" {
				return lineBg + cursorBg + " " + "\033[0m"
			}
			return cursorBg + " " + "\033[0m"
		}
		if lineBg != "" {
			return lineBg + "\033[0m"
		}
		return ""
	}

	hasRange := rangeBg != "" && startCol < endCol
	hasLine := lineBg != ""

	var result ansiStyleBuilder
	width := 0
	cursorDone := false
	current := ansiStyleState{}

	transition := func(next ansiStyleState) {
		if next == current {
			return
		}
		if current.line || current.rng || current.cursor {
			result.reset()
		}
		if next.line {
			result.write(lineBg)
		}
		if next.rng {
			result.write(rangeBg)
		}
		if next.cursor {
			result.write(cursorBg)
		}
		current = next
	}

	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		cluster := gr.Str()
		cw := PlainTextDisplayWidth(cluster)
		state := ansiStyleState{
			line:   hasLine,
			rng:    hasRange && width < endCol && width+cw > startCol,
			cursor: !cursorDone && cursorCol >= 0 && cursorCol < width+cw,
		}

		transition(state)
		result.write(cluster)
		width += cw
		if state.cursor {
			cursorDone = true
		}
	}

	if current.line || current.rng || current.cursor {
		result.reset()
	}
	return result.String()
}

type ansiStyleBuilder struct {
	builder strings.Builder
}

func (b *ansiStyleBuilder) write(s string) {
	b.builder.WriteString(s)
}

func (b *ansiStyleBuilder) reset() {
	b.builder.WriteString("\033[0m")
}

func (b *ansiStyleBuilder) String() string {
	return b.builder.String()
}

type ansiStyleState struct {
	line   bool
	rng    bool
	cursor bool
}
