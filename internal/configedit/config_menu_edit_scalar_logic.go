package configedit

import (
	"math"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// FloatInputStatus は float 入力の parse 結果分類を表す。
type FloatInputStatus int

type floatInputStatus = FloatInputStatus

const (
	// FloatInputKeep は空入力により現在値を維持することを表す。
	FloatInputKeep FloatInputStatus = iota
	// FloatInputValid は有効な float 入力を表す。
	FloatInputValid
	// FloatInputInvalid は数値として解釈できない入力を表す。
	FloatInputInvalid
	// FloatInputOutOfRange は設定項目固有の範囲外入力を表す。
	FloatInputOutOfRange
)

const (
	floatInputKeep       = FloatInputKeep
	floatInputValid      = FloatInputValid
	floatInputInvalid    = FloatInputInvalid
	floatInputOutOfRange = FloatInputOutOfRange
)

// ParseBoolInput は bool 設定値の入力を parse し、変更有無と妥当性を返す。
func ParseBoolInput(input string, current bool) (value bool, changed bool, valid bool) {
	return parseBoolInput(input, current)
}

// ExtractIntCurrentValue は config field の現在値を int として取り出す。
func ExtractIntCurrentValue(value interface{}) int {
	return extractIntCurrentValue(value)
}

// ParseIntInput は int 設定値の入力を parse し、変更有無と妥当性を返す。
func ParseIntInput(input string, current int) (value int, changed bool, valid bool) {
	return parseIntInput(input, current)
}

// ExtractFloatCurrentValue は config field の現在値を float64 として取り出す。
func ExtractFloatCurrentValue(value interface{}) float64 {
	return extractFloatCurrentValue(value)
}

// ParseFloatInput は float 設定値の入力を parse し、変更有無と分類を返す。
func ParseFloatInput(input string, current float64, path string) (value float64, changed bool, status FloatInputStatus) {
	return parseFloatInput(input, current, path)
}

// ParseStringInput は string 設定値の入力を parse し、変更有無を返す。
func ParseStringInput(input, current string) (value string, changed bool) {
	return parseStringInput(input, current)
}

// ParseSelectInput は select 設定値の番号入力を parse し、変更有無と妥当性を返す。
func ParseSelectInput(input, current string, options []string) (value string, changed bool, valid bool) {
	return parseSelectInput(input, current, options)
}

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

func parseFloatInput(input string, current float64, path string) (value float64, changed bool, status FloatInputStatus) {
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
