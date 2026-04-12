package tui

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// entryFieldValueStr は structEntryField の値を簡潔な文字列にする。
func entryFieldValueStr(ef structEntryField) string {
	switch v := ef.Value.(type) {
	case bool:
		if v {
			return "[x]"
		}
		return "[ ]"
	case string:
		if v == "" {
			return "(empty)"
		}
		return v
	case int:
		return fmt.Sprintf("%d", v)
	case []string:
		if len(v) == 0 {
			return "[]"
		}
		return fmt.Sprintf("%d items", len(v))
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatConfigValue はフィールド値を表示用文字列に変換する。
func formatConfigValue(v interface{}, _ config.ConfigFieldType) string {
	if v == nil {
		return "(nil)"
	}
	switch val := v.(type) {
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case string:
		if val == "" {
			return "(empty)"
		}
		return val
	case []string:
		if len(val) == 0 {
			return "[]"
		}
		return fmt.Sprintf("%d items", len(val))
	case map[string]string:
		return fmt.Sprintf("%d entries", len(val))
	case map[string]config.ProviderModelConfig:
		return fmt.Sprintf("%d entries", len(val))
	case map[string]config.LSPServerConfig:
		return fmt.Sprintf("%d entries", len(val))
	default:
		return fmt.Sprintf("%v", val)
	}
}
