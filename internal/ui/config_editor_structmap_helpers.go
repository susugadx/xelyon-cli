package ui

import (
	"strconv"
	"strings"
)

func readConfigEditorChoice(promptIO *PromptIO) string {
	return strings.TrimSpace(strings.ToLower(readLineWithIO(promptIO)))
}

func parseConfigEditorIndex(input string, size int) (int, bool) {
	num, err := strconv.Atoi(strings.TrimSpace(input))
	if err != nil || num < 1 || num > size {
		return 0, false
	}
	return num - 1, true
}
