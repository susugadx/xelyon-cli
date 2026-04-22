package ui

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// ConfigMenu は対話式設定メニュー
type ConfigMenu struct {
	Config     *config.Config
	Categories []config.ConfigCategory
	Runtime    *Runtime
}

// NewConfigMenu は新しいConfigMenuを作成
func NewConfigMenu(cfg *config.Config, categories []config.ConfigCategory) *ConfigMenu {
	return NewConfigMenuWithRuntime(cfg, categories, DefaultRuntime())
}

// NewConfigMenuWithRuntime は UI runtime を指定して新しい ConfigMenu を作成する。
func NewConfigMenuWithRuntime(cfg *config.Config, categories []config.ConfigCategory, runtime *Runtime) *ConfigMenu {
	return &ConfigMenu{
		Config:     cfg,
		Categories: categories,
		Runtime:    runtimeOrDefault(runtime),
	}
}

// formatValue は値を表示用文字列に変換
func formatValue(v interface{}) string {
	if v == nil {
		return "null"
	}

	switch val := v.(type) {
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int, int64, float64:
		return fmt.Sprintf("%v", val)
	case string:
		if val == "" {
			return "(empty)"
		}
		return val
	case []string:
		if len(val) == 0 {
			return "[]"
		}
		return fmt.Sprintf("[%d items]", len(val))
	case map[string]string:
		if len(val) == 0 {
			return "{}"
		}
		return fmt.Sprintf("{%d entries}", len(val))
	default:
		return fmt.Sprintf("%v", val)
	}
}
