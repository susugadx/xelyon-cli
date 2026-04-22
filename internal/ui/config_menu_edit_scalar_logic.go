package ui

import (
	"math"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

type floatInputStatus int

const (
	floatInputKeep floatInputStatus = iota
	floatInputValid
	floatInputInvalid
	floatInputOutOfRange
)

func parseBoolInput(input string, current bool) (value bool, changed bool, valid bool) {
	switch strings.TrimSpace(strings.ToLower(input)) {
	case "":
		return current, false, true
	case "y", "yes", "true", "1", "on":
		return true, true, true
	case "n", "no", "false", "0", "off":
		return false, true, true
	default:
		return current, false, false
	}
}

func extractIntCurrentValue(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func parseIntInput(input string, current int) (value int, changed bool, valid bool) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return current, false, true
	}

	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return current, false, false
	}
	return parsed, true, true
}

func extractFloatCurrentValue(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	default:
		return 0
	}
}

func parseFloatInput(input string, current float64, path string) (value float64, changed bool, status floatInputStatus) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return current, false, floatInputKeep
	}

	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return current, false, floatInputInvalid
	}

	if path == "project_map.context_ratio" &&
		(parsed < config.ProjectMapContextRatioMin || parsed > config.ProjectMapContextRatioMax) {
		return current, false, floatInputOutOfRange
	}

	return parsed, true, floatInputValid
}

func parseStringInput(input, current string) (value string, changed bool) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return current, false
	}
	return trimmed, true
}

func parseSelectInput(input, current string, options []string) (value string, changed bool, valid bool) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return current, false, true
	}

	num, err := strconv.Atoi(trimmed)
	if err != nil || num < 1 || num > len(options) {
		return current, false, false
	}

	return options[num-1], true, true
}
