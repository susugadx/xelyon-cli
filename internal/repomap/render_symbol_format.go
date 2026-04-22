package repomap

import (
	"fmt"
	"strconv"
	"strings"
)

func writeRenderedSymbol(b *strings.Builder, symbolPrefix string, symbol Symbol) {
	location := strconv.Itoa(symbol.Line)
	if symbol.EndLine > 0 && symbol.EndLine != symbol.Line {
		location = fmt.Sprintf("%d-%d", symbol.Line, symbol.EndLine)
	}

	lines := strings.Split(symbol.Signature, "\n")
	if len(lines) == 0 {
		fmt.Fprintf(b, "%s%s:\n", symbolPrefix, location)
		return
	}

	fmt.Fprintf(b, "%s%s: %s\n", symbolPrefix, location, lines[0])
	if len(lines) > 1 {
		padding := strings.Repeat(" ", len(location)+2)
		for _, line := range lines[1:] {
			fmt.Fprintf(b, "%s%s%s\n", symbolPrefix, padding, line)
		}
	}
}
