package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// ConfigMenu は対話式設定メニュー
type ConfigMenu struct {
	Config     *config.Config
	Categories []config.ConfigCategory
}

// NewConfigMenu は新しいConfigMenuを作成
func NewConfigMenu(cfg *config.Config, categories []config.ConfigCategory) *ConfigMenu {
	return &ConfigMenu{
		Config:     cfg,
		Categories: categories,
	}
}

// Run はメインメニューを表示し、選択されたカテゴリを返す
func (m *ConfigMenu) Run() (*config.ConfigCategory, error) {
	StopGlobalSpinner()
	fmt.Print("\033[?25h") // カーソル表示

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
		fmt.Printf("\n%s── Configuration%s ──────────────────────%s\n\n", colorCyan, pageInfo, colorReset)

		// カテゴリ一覧
		for i, cat := range pageCategories {
			num := i + 1
			if num == 10 {
				num = 0
			}
			fmt.Printf("  [%d] %s %s\n", num, cat.Icon, cat.DisplayName)
		}

		// ナビゲーション
		fmt.Println()
		if totalPages > 1 {
			if currentPage < totalPages-1 {
				fmt.Println("  [n] Next page")
			}
			if currentPage > 0 {
				fmt.Println("  [p] Previous page")
			}
		}
		fmt.Println("  [q] Cancel")
		fmt.Printf("\n%sSelect category:%s ", colorCyan, colorReset)

		input := readLine()
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
	StopGlobalSpinner()
	fmt.Print("\033[?25h")

	for {
		fmt.Printf("\n%s── %s %s ───────────────────────────────%s\n\n",
			colorCyan, cat.Icon, cat.DisplayName, colorReset)

		for i, field := range cat.Fields {
			num := i + 1
			if num == 10 {
				num = 0
			}

			// 現在の値を取得
			currentVal := formatValue(field.Current)
			displayName := field.DisplayName

			fmt.Printf("  [%d] %-20s = %s\n", num, displayName, truncateString(currentVal, 15))
		}

		fmt.Println()
		fmt.Println("  [b] Back")
		fmt.Printf("\n%sSelect field:%s ", colorCyan, colorReset)

		input := readLine()
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
	StopGlobalSpinner()
	fmt.Print("\033[?25h")

	fmt.Printf("\n%s%s%s\n", colorCyan, field.Description, colorReset)
	fmt.Printf("Path: %s\n", field.Path)
	fmt.Printf("Type: %s\n", field.FieldType.String())
	fmt.Printf("Current: %v\n", formatValue(field.Current))
	if field.Default != nil {
		fmt.Printf("Default: %v\n", formatValue(field.Default))
	}
	fmt.Println()

	switch field.FieldType {
	case config.FieldTypeBool:
		return m.editBool(field)
	case config.FieldTypeInt:
		return m.editInt(field)
	case config.FieldTypeString:
		return m.editString(field)
	case config.FieldTypeSelect:
		return m.editSelect(field)
	case config.FieldTypeStringSlice:
		return m.editStringSlice(field)
	case config.FieldTypeStringMap:
		return m.editStringMap(field)
	case config.FieldTypeStructMap:
		return m.editStructMap(field)
	default:
		fmt.Printf("%sUnsupported field type%s\n", colorDim, colorReset)
		return nil, false, nil
	}
}

func (m *ConfigMenu) editBool(field *config.ConfigField) (interface{}, bool, error) {
	current, _ := field.Current.(bool)

	fmt.Printf("Current value: %v\n", current)
	fmt.Printf("Enter new value (y/n, or Enter to keep): ")

	input := strings.TrimSpace(strings.ToLower(readLine()))

	switch input {
	case "":
		return current, false, nil
	case "y", "yes", "true", "1", "on":
		return true, true, nil
	case "n", "no", "false", "0", "off":
		return false, true, nil
	default:
		fmt.Printf("%sInvalid input, keeping current value%s\n", colorDim, colorReset)
		return current, false, nil
	}
}

func (m *ConfigMenu) editInt(field *config.ConfigField) (interface{}, bool, error) {
	current := 0
	switch v := field.Current.(type) {
	case int:
		current = v
	case int64:
		current = int(v)
	case float64:
		current = int(v)
	}

	fmt.Printf("Current value: %d\n", current)
	fmt.Printf("Enter new value (or Enter to keep): ")

	input := strings.TrimSpace(readLine())
	if input == "" {
		return current, false, nil
	}

	newVal, err := strconv.Atoi(input)
	if err != nil {
		fmt.Printf("%sInvalid number, keeping current value%s\n", colorDim, colorReset)
		return current, false, nil
	}

	return newVal, true, nil
}

func (m *ConfigMenu) editString(field *config.ConfigField) (interface{}, bool, error) {
	current, _ := field.Current.(string)

	fmt.Printf("Current value: %s\n", current)
	fmt.Printf("Enter new value (or Enter to keep): ")

	input := readLine()
	if strings.TrimSpace(input) == "" {
		return current, false, nil
	}

	return strings.TrimSpace(input), true, nil
}

func (m *ConfigMenu) editSelect(field *config.ConfigField) (interface{}, bool, error) {
	current, _ := field.Current.(string)

	fmt.Printf("Current value: %s\n", current)
	fmt.Println("Available options:")

	for i, opt := range field.Options {
		marker := "  "
		if opt == current {
			marker = "▶ "
		}
		fmt.Printf("  %s%d. %s\n", marker, i+1, opt)
	}

	fmt.Printf("\nEnter number (1-%d) or Enter to keep: ", len(field.Options))

	input := strings.TrimSpace(readLine())
	if input == "" {
		return current, false, nil
	}

	num, err := strconv.Atoi(input)
	if err != nil || num < 1 || num > len(field.Options) {
		fmt.Printf("%sInvalid selection, keeping current value%s\n", colorDim, colorReset)
		return current, false, nil
	}

	return field.Options[num-1], true, nil
}

func (m *ConfigMenu) editStringSlice(field *config.ConfigField) (interface{}, bool, error) {
	current := []string{}
	if slice, ok := field.Current.([]string); ok {
		current = slice
	}

	editor := NewStringSliceEditor(field.Path, current)
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

	editor := NewStringMapEditor(field.Path, current)
	result, changed, err := editor.Run()
	if err != nil {
		return nil, false, err
	}

	return result, changed, nil
}

func (m *ConfigMenu) editStructMap(field *config.ConfigField) (interface{}, bool, error) {
	editor := NewStructMapEditor(field.Path, field.FieldType)
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
