package ui

import (
	"strconv"
	"strings"
)

type projectMainMenuAction int

const (
	projectMainMenuUnknown projectMainMenuAction = iota
	projectMainMenuShowContext
	projectMainMenuEditRules
	projectMainMenuEditFinalChecks
	projectMainMenuSave
	projectMainMenuCancel
)

type projectFinalChecksAction int

const (
	projectFinalChecksUnknown projectFinalChecksAction = iota
	projectFinalChecksEditCommands
	projectFinalChecksEditTimeout
	projectFinalChecksBack
)

func readProjectMenuInput(promptIO *PromptIO) string {
	return strings.TrimSpace(strings.ToLower(readLineWithIO(promptIO)))
}

func resolveProjectMainMenuAction(input string) projectMainMenuAction {
	switch input {
	case "1":
		return projectMainMenuShowContext
	case "2":
		return projectMainMenuEditRules
	case "3":
		return projectMainMenuEditFinalChecks
	case "s", "save":
		return projectMainMenuSave
	case "c", "cancel":
		return projectMainMenuCancel
	default:
		return projectMainMenuUnknown
	}
}

func resolveProjectFinalChecksAction(input string) projectFinalChecksAction {
	switch input {
	case "1":
		return projectFinalChecksEditCommands
	case "2":
		return projectFinalChecksEditTimeout
	case "b", "back":
		return projectFinalChecksBack
	default:
		return projectFinalChecksUnknown
	}
}

func parsePositiveIntInput(raw string) (int, bool) {
	num, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || num <= 0 {
		return 0, false
	}
	return num, true
}
