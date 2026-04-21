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

type fieldAdapter struct {
	get        func(*Config) (interface{}, error)
	set        func(*Config, interface{}) error
	getDefault func() interface{}
}

func getProviderModelsFieldValue(cfg *Config) (interface{}, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	return cfg.ProviderModelsForEdit(), nil
}

func setProviderModelsFieldValue(cfg *Config, value interface{}) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	providerModels, ok := value.(map[string]ProviderModelConfig)
	if !ok {
		return fmt.Errorf("type mismatch: expected map[string]ProviderModelConfig, got %T", value)
	}
	if providerModels == nil {
		cfg.ResetProviderModelsForEdit()
		return nil
	}
	cfg.SetProviderModelsForEdit(providerModels)
	return nil
}

func getLSPServersFieldValue(cfg *Config) (interface{}, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	return cfg.LSP.Servers, nil
}

func setLSPServersFieldValue(cfg *Config, value interface{}) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	servers, ok := value.(map[string]LSPServerConfig)
	if !ok {
		return fmt.Errorf("type mismatch: expected map[string]LSPServerConfig, got %T", value)
	}
	cfg.LSP.Servers = servers
	return nil
}

var fieldAdapters = map[string]fieldAdapter{
	"provider_models": {
		get: getProviderModelsFieldValue,
		set: setProviderModelsFieldValue,
		getDefault: func() interface{} {
			return map[string]ProviderModelConfig(nil)
		},
	},
	"lsp.servers": {
		get: getLSPServersFieldValue,
		set: setLSPServersFieldValue,
	},
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
			if adapter, ok := fieldAdapters[fieldPath]; ok && adapter.getDefault != nil {
				defaultVal = adapter.getDefault()
			}

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
