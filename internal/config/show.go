package config

import (
	"fmt"
	"reflect"
	"strings"
)

// ConfigDiff は設定の差分情報
type ConfigDiff struct {
	Key      string
	Current  string
	Default  string
	IsDiff   bool
	IsNested bool
}

// ShowConfig は現在の設定とデフォルトの差分を含む表示文字列を生成
func ShowConfig(current *Config) string {
	defaults := DefaultConfig()
	diffs := compareConfigs(current, defaults, "")

	var sb strings.Builder

	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("📋 Current Configuration / 現在の設定\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// グループごとに表示
	currentGroup := ""
	for _, diff := range diffs {
		// トップレベルのグループ名を抽出
		parts := strings.SplitN(diff.Key, ".", 2)
		group := parts[0]

		// 新しいグループなら見出し表示
		if group != currentGroup {
			if currentGroup != "" {
				sb.WriteString("\n")
			}
			fmt.Fprintf(&sb, "[%s]\n", group)
			currentGroup = group
		}

		// キー名（グループ名を除去）
		keyName := diff.Key
		if len(parts) > 1 {
			keyName = "  " + parts[1]
		}

		// 値と差分マーク
		if diff.IsDiff {
			fmt.Fprintf(&sb, "%-35s = %-20s (default: %s) ⚡\n", keyName, diff.Current, diff.Default)
		} else {
			fmt.Fprintf(&sb, "%-35s = %s\n", keyName, diff.Current)
		}
	}

	sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("⚡ = differs from default / デフォルトと異なる\n")
	sb.WriteString("📖 Full docs: docs/config.md\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return sb.String()
}

// hiddenShowSections は /config show で非表示にする内部専用セクション
var hiddenShowSections = map[string]bool{
	"loop_detection":  true,
	"api_retry":       true,
	"diff":            true,
	"command_aliases": true,
	"prompt_cache":    true,
	"responses":       true, // Responses API retention policy（YAML 互換は維持）
	"streaming":       true,
	"tool_confirm":    true,
	"bash":            true,
	"git_stage":       true,
	"list_dir":        true,
	"openai":          true, // 内部ルーティング（YAML 互換は維持）
	"thinking":        true, // /think コマンドが正規ルート（YAML 互換は維持）
}

// compareConfigs はリフレクションで2つの設定を比較
func compareConfigs(current, defaults *Config, prefix string) []ConfigDiff {
	var diffs []ConfigDiff

	currentVal := reflect.ValueOf(current).Elem()
	defaultVal := reflect.ValueOf(defaults).Elem()
	currentType := currentVal.Type()

	for i := 0; i < currentVal.NumField(); i++ {
		field := currentType.Field(i)

		// yaml タグからキー名を取得
		yamlTag := field.Tag.Get("yaml")
		if yamlTag == "" || yamlTag == "-" {
			continue
		}
		// "omitempty" を除去
		yamlKey := strings.Split(yamlTag, ",")[0]

		// 内部専用セクションは非表示
		if hiddenShowSections[yamlKey] {
			continue
		}

		// プレフィックス付きキー
		fullKey := yamlKey
		if prefix != "" {
			fullKey = prefix + "." + yamlKey
		}

		currentField := currentVal.Field(i)
		defaultField := defaultVal.Field(i)

		// ネストされた構造体を再帰的に処理
		if currentField.Kind() == reflect.Struct {
			// 新しい構造体インスタンスを作成して比較
			nested := compareStructFields(currentField, defaultField, fullKey)
			diffs = append(diffs, nested...)
		} else if currentField.Kind() == reflect.Map {
			// マップは特別処理（ProviderModels等）
			mapDiffs := compareMapFields(currentField, defaultField, fullKey)
			diffs = append(diffs, mapDiffs...)
		} else {
			// プリミティブ型
			currentStr := formatValue(currentField)
			defaultStr := formatValue(defaultField)
			isDiff := currentStr != defaultStr

			diffs = append(diffs, ConfigDiff{
				Key:     fullKey,
				Current: currentStr,
				Default: defaultStr,
				IsDiff:  isDiff,
			})
		}
	}

	return diffs
}

// compareStructFields はネストされた構造体のフィールドを比較
func compareStructFields(current, defaults reflect.Value, prefix string) []ConfigDiff {
	var diffs []ConfigDiff

	currentType := current.Type()

	for i := 0; i < current.NumField(); i++ {
		field := currentType.Field(i)

		yamlTag := field.Tag.Get("yaml")
		if yamlTag == "" || yamlTag == "-" {
			continue
		}
		yamlKey := strings.Split(yamlTag, ",")[0]
		fullKey := prefix + "." + yamlKey

		currentField := current.Field(i)
		defaultField := defaults.Field(i)

		if currentField.Kind() == reflect.Struct {
			nested := compareStructFields(currentField, defaultField, fullKey)
			diffs = append(diffs, nested...)
		} else if currentField.Kind() == reflect.Slice {
			// スライスは特別処理
			currentStr := formatSlice(currentField)
			defaultStr := formatSlice(defaultField)
			isDiff := currentStr != defaultStr

			diffs = append(diffs, ConfigDiff{
				Key:     fullKey,
				Current: currentStr,
				Default: defaultStr,
				IsDiff:  isDiff,
			})
		} else {
			currentStr := formatValue(currentField)
			defaultStr := formatValue(defaultField)
			isDiff := currentStr != defaultStr

			diffs = append(diffs, ConfigDiff{
				Key:     fullKey,
				Current: currentStr,
				Default: defaultStr,
				IsDiff:  isDiff,
			})
		}
	}

	return diffs
}

// compareMapFields はマップフィールドを比較
func compareMapFields(current, defaults reflect.Value, prefix string) []ConfigDiff {
	var diffs []ConfigDiff

	// マップが空または異なるキー数
	if current.Len() != defaults.Len() {
		diffs = append(diffs, ConfigDiff{
			Key:     prefix,
			Current: fmt.Sprintf("(%d items)", current.Len()),
			Default: fmt.Sprintf("(%d items)", defaults.Len()),
			IsDiff:  true,
		})
		return diffs
	}

	// 各キーを比較
	for _, key := range current.MapKeys() {
		keyStr := fmt.Sprintf("%v", key.Interface())
		fullKey := prefix + "." + keyStr

		currentVal := current.MapIndex(key)
		defaultVal := defaults.MapIndex(key)

		if !defaultVal.IsValid() {
			// デフォルトにないキー
			diffs = append(diffs, ConfigDiff{
				Key:     fullKey,
				Current: formatValue(currentVal),
				Default: "(not in default)",
				IsDiff:  true,
			})
			continue
		}

		// ネストされた構造体（ProviderModelConfig等）
		if currentVal.Kind() == reflect.Struct {
			nested := compareStructFields(currentVal, defaultVal, fullKey)
			diffs = append(diffs, nested...)
		} else {
			currentStr := formatValue(currentVal)
			defaultStr := formatValue(defaultVal)
			isDiff := currentStr != defaultStr

			diffs = append(diffs, ConfigDiff{
				Key:     fullKey,
				Current: currentStr,
				Default: defaultStr,
				IsDiff:  isDiff,
			})
		}
	}

	return diffs
}

// formatValue は値を文字列に変換
func formatValue(v reflect.Value) string {
	if !v.IsValid() {
		return "<nil>"
	}

	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return fmt.Sprintf("%v", v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%.2f", v.Float())
	case reflect.Slice:
		return formatSlice(v)
	case reflect.Map:
		return fmt.Sprintf("(%d items)", v.Len())
	case reflect.Struct:
		return "<struct>"
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// formatSlice はスライスを文字列に変換
func formatSlice(v reflect.Value) string {
	if v.Len() == 0 {
		return "[]"
	}

	var items []string
	for i := 0; i < v.Len(); i++ {
		items = append(items, formatValue(v.Index(i)))
	}
	return "[" + strings.Join(items, ", ") + "]"
}
