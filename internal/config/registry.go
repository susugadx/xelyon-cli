package config

import (
	"fmt"
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

func getReflectFieldValue(cfg *Config, path string) (interface{}, error) {
	v, err := resolveConfigValueByPath(cfg, strings.Split(path, "."))
	if err != nil {
		return nil, err
	}
	return v.Interface(), nil
}

func setReflectFieldValue(cfg *Config, path string, value interface{}) error {
	parent, lastPart, err := resolveConfigParentForSet(cfg, strings.Split(path, "."))
	if err != nil {
		return err
	}
	return assignValueByPathPart(parent, lastPart, value)
}

// GetFieldValue はパスを指定して設定値を取得する
func GetFieldValue(cfg *Config, path string) (interface{}, error) {
	if adapter, ok := fieldAdapters[path]; ok {
		return adapter.get(cfg)
	}
	return getReflectFieldValue(cfg, path)
}

// SetFieldValue はパスを指定して設定値を設定する
func SetFieldValue(cfg *Config, path string, value interface{}) error {
	if adapter, ok := fieldAdapters[path]; ok {
		return adapter.set(cfg, value)
	}
	return setReflectFieldValue(cfg, path, value)
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
	case FieldTypeFloat:
		return "float"
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
