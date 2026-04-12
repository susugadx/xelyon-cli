package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

func mergeStreamFragment(currentLine, fragment string, cursorCol int, currentANSI, pendingPartialANSI string) (string, int, string, string) {
	if fragment == "" {
		if cursorCol < 0 {
			cursorCol = 0
		}
		return currentLine, cursorCol, currentANSI, pendingPartialANSI
	}
	if cursorCol < 0 {
		cursorCol = 0
	}

	fragment = pendingPartialANSI + fragment
	cells := parseStreamCells(currentLine)
	inEscape := false
	var partialANSI strings.Builder
	segmentStart := 0

	flushSegment := func(segment string) {
		if segment == "" {
			return
		}
		gr := uniseg.NewGraphemes(segment)
		for gr.Next() {
			cluster := gr.Str()
			width := plainTextDisplayWidth(cluster)
			cells = applyClusterToStreamCells(cells, cursorCol, cluster, width, currentANSI)
			cursorCol += width
		}
	}

	for i := 0; i < len(fragment); {
		r, size := utf8.DecodeRuneInString(fragment[i:])
		if r == utf8.RuneError && size == 1 {
			size = 1
		}

		if inEscape {
			partialANSI.WriteRune(r)
			if isANSITerminator(r) {
				inEscape = false
				currentANSI = applyANSICode(currentANSI, partialANSI.String())
				partialANSI.Reset()
				segmentStart = i + size
			}
			i += size
			continue
		}

		if r == '\033' {
			if segmentStart < i {
				flushSegment(fragment[segmentStart:i])
			}
			inEscape = true
			partialANSI.WriteRune(r)
			i += size
			continue
		}

		if r == '\r' {
			if segmentStart < i {
				flushSegment(fragment[segmentStart:i])
			}
			cursorCol = 0
			segmentStart = i + size
			i += size
			continue
		}

		i += size
	}

	line := rebuildStreamLine(cells)
	if inEscape {
		return line, cursorCol, currentANSI, partialANSI.String()
	}
	if segmentStart < len(fragment) {
		flushSegment(fragment[segmentStart:])
		line = rebuildStreamLine(cells)
	}

	return line, cursorCol, currentANSI, ""
}
