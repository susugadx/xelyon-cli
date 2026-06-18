package readtool

import (
	"fmt"
	"strings"
)

func generateLines(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d", i+1)
	}
	return strings.Join(lines, "\n")
}
