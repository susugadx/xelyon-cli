package viewfmt

import (
	"fmt"
	"strings"
)

func Number(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%s,%03d", Number(n/1000), n%1000)
}

func USD(value float64) string {
	return fmt.Sprintf("$%.4f", value)
}

func USDWithSuffix(value float64) string {
	return fmt.Sprintf("%s USD", USD(value))
}

func FirstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
