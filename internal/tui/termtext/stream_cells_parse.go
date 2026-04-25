package termtext

import (
	"strings"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

func parseStreamCells(s string) []streamCell {
	cells := make([]streamCell, 0, len(s))
	inEscape := false
	var token strings.Builder
	currentStyle := ""

	flushSegment := func(segment string) {
		appendSegmentToStreamCells(&cells, segment, currentStyle)
	}

	segmentStart := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			size = 1
		}

		if inEscape {
			token.WriteRune(r)
			if isANSITerminator(r) {
				inEscape = false
				currentStyle = applyANSICode(currentStyle, token.String())
				token.Reset()
				segmentStart = i + size
			}
			i += size
			continue
		}

		if r == '\033' {
			if segmentStart < i {
				flushSegment(s[segmentStart:i])
			}
			inEscape = true
			token.WriteRune(r)
			i += size
			continue
		}

		i += size
	}

	if !inEscape && segmentStart < len(s) {
		flushSegment(s[segmentStart:])
	}

	return cells
}

func appendSegmentToStreamCells(cells *[]streamCell, segment, style string) {
	gr := uniseg.NewGraphemes(segment)
	for gr.Next() {
		cluster := gr.Str()
		if cluster == "\t" {
			appendStreamCell(cells, cluster, VisualTabWidth, style)
			continue
		}
		appendStreamCell(cells, cluster, PlainTextDisplayWidth(cluster), style)
	}
}

func appendStreamCell(cells *[]streamCell, text string, width int, style string) {
	if width <= 0 {
		return
	}
	*cells = append(*cells, streamCell{text: text, style: style, span: width})
	for i := 1; i < width; i++ {
		*cells = append(*cells, streamCell{style: style, span: 0})
	}
}
