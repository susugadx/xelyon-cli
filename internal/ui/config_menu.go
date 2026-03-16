package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

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

// Run はメインメニューを表示し、選択されたカテゴリを返す
func (m *ConfigMenu) Run() (*config.ConfigCategory, error) {
	runtime := runtimeOrDefault(m.Runtime)
	runtime.StopSpinner()
	runtime.ResetTerminalState()
	promptIO := runtime.PromptIO()
	out := promptIO.Out

	// ページング（10カテゴリ/ページ）
	pageSize := 10
	totalPages := (len(m.Categories) + pageSize - 1) / pageSize
	currentPage := 0

	for {
		start := currentPage * pageSize
		end := start + pageSize
		if end > len(m.Categories) {
			end = len(m.Categories)
		}

		pageCategories := m.Categories[start:end]

		// ヘッダー
		pageInfo := ""
		if totalPages > 1 {
			pageInfo = fmt.Sprintf(" (%d/%d)", currentPage+1, totalPages)
		}
		_, _ = fmt.Fprintf(out, "\n%s── Configuration%s ──────────────────────%s\n\n", colorCyan, pageInfo, colorReset)

		// カテゴリ一覧
		for i, cat := range pageCategories {
			num := i + 1
			if num == 10 {
				num = 0
			}
			_, _ = fmt.Fprintf(out, "  [%d] %s %s\n", num, cat.Icon, cat.DisplayName)
		}

		// ナビゲーション
		_, _ = fmt.Fprintln(out)
		if totalPages > 1 {
			if currentPage < totalPages-1 {
				_, _ = fmt.Fprintln(out, "  [n] Next page")
			}
			if currentPage > 0 {
				_, _ = fmt.Fprintln(out, "  [p] Previous page")
			}
		}
		_, _ = fmt.Fprintln(out, "  [q] Cancel")
		_, _ = fmt.Fprintf(out, "\n%sSelect category:%s ", colorCyan, colorReset)

		input := readLineWithIO(&promptIO)
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "q", "quit", "cancel":
			return nil, nil

		case "n", "next":
			if currentPage < totalPages-1 {
				currentPage++
			}

		case "p", "prev", "previous":
			if currentPage > 0 {
				currentPage--
			}

		default:
			// 数字入力
			num, err := strconv.Atoi(input)
			if err != nil {
				continue
			}
			if num == 0 {
				num = 10
			}
			idx := start + num - 1
			if idx >= 0 && idx < len(m.Categories) {
				return &m.Categories[idx], nil
			}
		}
	}
}

// ShowFieldList はカテゴリ内のフィールドリストを表示
func (m *ConfigMenu) ShowFieldList(cat *config.ConfigCategory) (*config.ConfigField, error) {
	runtime := runtimeOrDefault(m.Runtime)
	runtime.StopSpinner()
	runtime.ResetTerminalState()
	promptIO := runtime.PromptIO()
	out := promptIO.Out

	for {
		_, _ = fmt.Fprintf(out, "\n%s── %s %s ───────────────────────────────%s\n\n",
			colorCyan, cat.Icon, cat.DisplayName, colorReset)

		for i, field := range cat.Fields {
			num := i + 1
			if num == 10 {
				num = 0
			}

			// 現在の値を取得
			currentVal := formatValue(field.Current)
			displayName := field.DisplayName

			_, _ = fmt.Fprintf(out, "  [%d] %-20s = %s\n", num, displayName, truncateString(currentVal, 15))
		}

		_, _ = fmt.Fprintln(out)
		_, _ = fmt.Fprintln(out, "  [b] Back")
		_, _ = fmt.Fprintf(out, "\n%sSelect field:%s ", colorCyan, colorReset)

		input := readLineWithIO(&promptIO)
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "b", "back":
			return nil, fmt.Errorf("back")

		default:
			num, err := strconv.Atoi(input)
			if err != nil {
				continue
			}
			if num == 0 {
				num = 10
			}
			idx := num - 1
			if idx >= 0 && idx < len(cat.Fields) {
				return &cat.Fields[idx], nil
			}
		}
	}
}

// EditField はフィールドを編集する
func (m *ConfigMenu) EditField(field *config.ConfigField) (interface{}, bool, error) {
	runtime := runtimeOrDefault(m.Runtime)
	runtime.StopSpinner()
	runtime.ResetTerminalState()
	promptIO := runtime.PromptIO()
	out := promptIO.Out

	_, _ = fmt.Fprintf(out, "\n%s%s%s\n", colorCyan, field.Description, colorReset)
	_, _ = fmt.Fprintf(out, "Path: %s\n", field.Path)
	_, _ = fmt.Fprintf(out, "Type: %s\n", field.FieldType.String())
	_, _ = fmt.Fprintf(out, "Current: %v\n", formatValue(field.Current))
	if field.Default != nil {
		_, _ = fmt.Fprintf(out, "Default: %v\n", formatValue(field.Default))
	}
	_, _ = fmt.Fprintln(out)

	switch field.FieldType {
	case config.FieldTypeBool:
		return m.editBool(promptIO, field)
	case config.FieldTypeInt:
		return m.editInt(promptIO, field)
	case config.FieldTypeFloat:
		return m.editFloat(promptIO, field)
	case config.FieldTypeString:
		return m.editString(promptIO, field)
	case config.FieldTypeSelect:
		return m.editSelect(promptIO, field)
	case config.FieldTypeStringSlice:
		return m.editStringSlice(field)
	case config.FieldTypeStringMap:
		return m.editStringMap(field)
	case config.FieldTypeStructMap:
		return m.editStructMap(field)
	default:
		_, _ = fmt.Fprintf(out, "%sUnsupported field type%s\n", colorDim, colorReset)
		return nil, false, nil
	}
}

func (m *ConfigMenu) editBool(promptIO PromptIO, field *config.ConfigField) (interface{}, bool, error) {
	current, _ := field.Current.(bool)
	out := promptIO.Out

	_, _ = fmt.Fprintf(out, "Current value: %v\n", current)
	_, _ = fmt.Fprint(out, "Enter new value (y/n, or Enter to keep): ")

	input := strings.TrimSpace(strings.ToLower(readLineWithIO(&promptIO)))

	switch input {
	case "":
		return current, false, nil
	case "y", "yes", "true", "1", "on":
		return true, true, nil
	case "n", "no", "false", "0", "off":
		return false, true, nil
	default:
		_, _ = fmt.Fprintf(out, "%sInvalid input, keeping current value%s\n", colorDim, colorReset)
		return current, false, nil
	}
}

func (m *ConfigMenu) editInt(promptIO PromptIO, field *config.ConfigField) (interface{}, bool, error) {
	current := 0
	out := promptIO.Out
	switch v := field.Current.(type) {
	case int:
		current = v
	case int64:
		current = int(v)
	case float64:
		current = int(v)
	}

	_, _ = fmt.Fprintf(out, "Current value: %d\n", current)
	_, _ = fmt.Fprint(out, "Enter new value (or Enter to keep): ")

	input := strings.TrimSpace(readLineWithIO(&promptIO))
	if input == "" {
		return current, false, nil
	}

	newVal, err := strconv.Atoi(input)
	if err != nil {
		_, _ = fmt.Fprintf(out, "%sInvalid number, keeping current value%s\n", colorDim, colorReset)
		return current, false, nil
	}

	return newVal, true, nil
}

func (m *ConfigMenu) editFloat(promptIO PromptIO, field *config.ConfigField) (interface{}, bool, error) {
	current := 0.0
	out := promptIO.Out
	switch v := field.Current.(type) {
	case float64:
		current = v
	case float32:
		current = float64(v)
	case int:
		current = float64(v)
	}

	_, _ = fmt.Fprintf(out, "Current value: %g\n", current)
	_, _ = fmt.Fprint(out, "Enter new value (or Enter to keep): ")

	input := strings.TrimSpace(readLineWithIO(&promptIO))
	if input == "" {
		return current, false, nil
	}

	newVal, err := strconv.ParseFloat(input, 64)
	if err != nil || math.IsNaN(newVal) || math.IsInf(newVal, 0) {
		_, _ = fmt.Fprintf(out, "%sInvalid number, keeping current value%s\n", colorDim, colorReset)
		return current, false, nil
	}
	if field.Path == "project_map.context_ratio" &&
		(newVal < config.ProjectMapContextRatioMin || newVal > config.ProjectMapContextRatioMax) {
		_, _ = fmt.Fprintf(out, "%sValue must be between %.2f and %.2f, keeping current value%s\n",
			colorDim, config.ProjectMapContextRatioMin, config.ProjectMapContextRatioMax, colorReset)
		return current, false, nil
	}

	return newVal, true, nil
}

func (m *ConfigMenu) editString(promptIO PromptIO, field *config.ConfigField) (interface{}, bool, error) {
	current, _ := field.Current.(string)
	out := promptIO.Out

	_, _ = fmt.Fprintf(out, "Current value: %s\n", current)
	_, _ = fmt.Fprint(out, "Enter new value (or Enter to keep): ")

	input := readLineWithIO(&promptIO)
	if strings.TrimSpace(input) == "" {
		return current, false, nil
	}

	return strings.TrimSpace(input), true, nil
}

func (m *ConfigMenu) editSelect(promptIO PromptIO, field *config.ConfigField) (interface{}, bool, error) {
	current, _ := field.Current.(string)
	out := promptIO.Out

	_, _ = fmt.Fprintf(out, "Current value: %s\n", current)
	_, _ = fmt.Fprintln(out, "Available options:")

	for i, opt := range field.Options {
		marker := "  "
		if opt == current {
			marker = "▶ "
		}
		_, _ = fmt.Fprintf(out, "  %s%d. %s\n", marker, i+1, opt)
	}

	_, _ = fmt.Fprintf(out, "\nEnter number (1-%d) or Enter to keep: ", len(field.Options))

	input := strings.TrimSpace(readLineWithIO(&promptIO))
	if input == "" {
		return current, false, nil
	}

	num, err := strconv.Atoi(input)
	if err != nil || num < 1 || num > len(field.Options) {
		_, _ = fmt.Fprintf(out, "%sInvalid selection, keeping current value%s\n", colorDim, colorReset)
		return current, false, nil
	}

	return field.Options[num-1], true, nil
}

func (m *ConfigMenu) editStringSlice(field *config.ConfigField) (interface{}, bool, error) {
	current := []string{}
	if slice, ok := field.Current.([]string); ok {
		current = slice
	}

	editor := NewStringSliceEditorWithRuntime(field.Path, current, m.Runtime)
	result, changed, err := editor.Run()
	if err != nil {
		return nil, false, err
	}

	return result, changed, nil
}

func (m *ConfigMenu) editStringMap(field *config.ConfigField) (interface{}, bool, error) {
	current := map[string]string{}
	if mp, ok := field.Current.(map[string]string); ok {
		current = mp
	}

	editor := NewStringMapEditorWithRuntime(field.Path, current, m.Runtime)
	result, changed, err := editor.Run()
	if err != nil {
		return nil, false, err
	}

	return result, changed, nil
}

func (m *ConfigMenu) editStructMap(field *config.ConfigField) (interface{}, bool, error) {
	editor := NewStructMapEditorWithRuntime(field.Path, field.FieldType, m.Runtime)
	changed, err := editor.Run(m.Config)
	if err != nil {
		return nil, false, err
	}

	// StructMapEditorは直接Configを編集するので、変更されたかどうかのみ返す
	return nil, changed, nil
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
