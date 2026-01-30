package config

import (
	"fmt"
	"reflect"
	"strings"
)

// ConfigField は設定フィールドの情報
type ConfigField struct {
	Path        string          // "thinking.enabled"
	DisplayName string          // 表示名
	Description string          // 説明
	FieldType   ConfigFieldType // 型
	Options     []string        // FieldTypeSelect用の選択肢
	Category    string          // カテゴリ名
	Current     interface{}     // 現在の値
	Default     interface{}     // デフォルト値
}

// ConfigCategory は設定カテゴリの情報
type ConfigCategory struct {
	Name        string        // カテゴリ名
	DisplayName string        // 表示名
	Icon        string        // アイコン
	Fields      []ConfigField // フィールドリスト
}

// BuildConfigRegistry はConfig構造体からカテゴリリストを構築する
func BuildConfigRegistry(cfg *Config) []ConfigCategory {
	defaultCfg := DefaultConfig()
	var categories []ConfigCategory

	for _, catDef := range CategoryDefinitions {
		cat := ConfigCategory{
			Name:        catDef.Name,
			DisplayName: catDef.DisplayName,
			Icon:        catDef.Icon,
		}

		for _, fieldPath := range catDef.Fields {
			fieldType, ok := FieldTypeMap[fieldPath]
			if !ok {
				fieldType = FieldTypeString
			}

			desc := FieldDescriptions[fieldPath]
			opts := SelectOptions[fieldPath]

			currentVal, _ := GetFieldValue(cfg, fieldPath)
			defaultVal, _ := GetFieldValue(defaultCfg, fieldPath)

			// 表示名をパスから生成
			displayName := fieldPath
			if parts := strings.Split(fieldPath, "."); len(parts) > 1 {
				displayName = parts[len(parts)-1]
			}

			field := ConfigField{
				Path:        fieldPath,
				DisplayName: displayName,
				Description: desc,
				FieldType:   fieldType,
				Options:     opts,
				Category:    catDef.Name,
				Current:     currentVal,
				Default:     defaultVal,
			}
			cat.Fields = append(cat.Fields, field)
		}

		categories = append(categories, cat)
	}

	return categories
}

// GetFieldValue はパスを指定して設定値を取得する
func GetFieldValue(cfg *Config, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	v := reflect.ValueOf(cfg).Elem()

	for _, part := range parts {
		if v.Kind() == reflect.Map {
			// map型の場合
			key := reflect.ValueOf(part)
			mapVal := v.MapIndex(key)
			if !mapVal.IsValid() {
				return nil, fmt.Errorf("key not found: %s", part)
			}
			v = mapVal
			continue
		}

		if v.Kind() != reflect.Struct {
			return nil, fmt.Errorf("not a struct: %s", part)
		}

		// yamlタグでフィールドを検索
		t := v.Type()
		found := false
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			yamlTag := strings.Split(field.Tag.Get("yaml"), ",")[0]
			if yamlTag == part {
				v = v.Field(i)
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("field not found: %s", part)
		}
	}

	return v.Interface(), nil
}

// SetFieldValue はパスを指定して設定値を設定する
func SetFieldValue(cfg *Config, path string, value interface{}) error {
	parts := strings.Split(path, ".")
	v := reflect.ValueOf(cfg).Elem()

	// 最後の要素以外をたどる
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]

		if v.Kind() == reflect.Map {
			key := reflect.ValueOf(part)
			mapVal := v.MapIndex(key)
			if !mapVal.IsValid() {
				return fmt.Errorf("key not found: %s", part)
			}
			v = mapVal
			continue
		}

		if v.Kind() != reflect.Struct {
			return fmt.Errorf("not a struct: %s", part)
		}

		t := v.Type()
		found := false
		for j := 0; j < t.NumField(); j++ {
			field := t.Field(j)
			yamlTag := strings.Split(field.Tag.Get("yaml"), ",")[0]
			if yamlTag == part {
				v = v.Field(j)
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("field not found: %s", part)
		}
	}

	// 最後の要素を設定
	lastPart := parts[len(parts)-1]

	if v.Kind() == reflect.Map {
		// map型の場合
		key := reflect.ValueOf(lastPart)
		v.SetMapIndex(key, reflect.ValueOf(value))
		return nil
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("not a struct: %s", lastPart)
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		yamlTag := strings.Split(field.Tag.Get("yaml"), ",")[0]
		if yamlTag == lastPart {
			fieldVal := v.Field(i)
			if !fieldVal.CanSet() {
				return fmt.Errorf("cannot set field: %s", lastPart)
			}

			// 型変換
			newVal := reflect.ValueOf(value)
			if newVal.Type().ConvertibleTo(fieldVal.Type()) {
				fieldVal.Set(newVal.Convert(fieldVal.Type()))
				return nil
			}

			return fmt.Errorf("type mismatch: expected %s, got %s",
				fieldVal.Type(), newVal.Type())
		}
	}

	return fmt.Errorf("field not found: %s", lastPart)
}

// GetStringMapValue はmap[string]string型のフィールド値を取得する
func GetStringMapValue(cfg *Config, path string) (map[string]string, error) {
	val, err := GetFieldValue(cfg, path)
	if err != nil {
		return nil, err
	}

	if m, ok := val.(map[string]string); ok {
		return m, nil
	}

	return nil, fmt.Errorf("not a map[string]string: %s", path)
}

// SetStringMapValue はmap[string]string型のフィールド値を設定する
func SetStringMapValue(cfg *Config, path string, value map[string]string) error {
	return SetFieldValue(cfg, path, value)
}

// GetStringSliceValue は[]string型のフィールド値を取得する
func GetStringSliceValue(cfg *Config, path string) ([]string, error) {
	val, err := GetFieldValue(cfg, path)
	if err != nil {
		return nil, err
	}

	if s, ok := val.([]string); ok {
		return s, nil
	}

	return nil, fmt.Errorf("not a []string: %s", path)
}

// SetStringSliceValue は[]string型のフィールド値を設定する
func SetStringSliceValue(cfg *Config, path string, value []string) error {
	return SetFieldValue(cfg, path, value)
}

// FieldTypeString は ConfigFieldType を文字列に変換する
func (t ConfigFieldType) String() string {
	switch t {
	case FieldTypeBool:
		return "bool"
	case FieldTypeInt:
		return "int"
	case FieldTypeString:
		return "string"
	case FieldTypeSelect:
		return "select"
	case FieldTypeStringSlice:
		return "[]string"
	case FieldTypeStringMap:
		return "map[string]string"
	case FieldTypeStructMap:
		return "map[string]struct"
	default:
		return "unknown"
	}
}
