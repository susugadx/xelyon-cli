package uiconfig

import (
	"strconv"
	"strings"
)

func readConfigMenuInput(promptIO *PromptIO) string {
	return strings.TrimSpace(strings.ToLower(readLineWithIO(promptIO)))
}

func parseMenuNumberWithZeroAsTen(input string, max int) (int, bool) {
	num, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil {
		return 0, false
	}
	if num == 0 {
		num = 10
	}
	if num < 1 || num > max {
		return 0, false
	}
	return num - 1, true
}
