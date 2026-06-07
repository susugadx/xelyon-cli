package config

import (
	"reflect"
	"strings"
)

type registryFieldResolver struct {
	current *Config
	def     *Config
}

func newRegistryFieldResolver(current *Config) registryFieldResolver {
	return registryFieldResolver{
		current: current,
		def:     DefaultConfig(),
	}
}

func (r registryFieldResolver) resolve(path string) (current interface{}, def interface{}) {
	current, _ = GetFieldValue(r.current, path)
	def, _ = GetFieldValue(r.def, path)
	if adapter, ok := fieldAdapters[path]; ok && adapter.getDefault != nil {
		def = adapter.getDefault()
	}
	return current, def
}

type registryFieldMeta struct {
	path        string
	displayName string
	description string
	fieldType   ConfigFieldType
	options     []string
	category    string
}

func newRegistryFieldMeta(categoryName, fieldPath string) registryFieldMeta {
	return registryFieldMeta{
		path:        fieldPath,
		displayName: registryFieldDisplayName(fieldPath),
		description: FieldDescriptions[fieldPath],
		fieldType:   registryFieldType(fieldPath),
		options:     SelectOptions[fieldPath],
		category:    categoryName,
	}
}

func registryFieldDisplayName(fieldPath string) string {
	displayName := fieldPath
	if parts := strings.Split(fieldPath, "."); len(parts) > 1 {
		displayName = parts[len(parts)-1]
	}
	return displayName
}

func registryFieldType(fieldPath string) ConfigFieldType {
	fieldType, ok := FieldTypeMap[fieldPath]
	if !ok {
		return FieldTypeString
	}
	return fieldType
}

func buildRegistryField(meta registryFieldMeta, resolver registryFieldResolver) ConfigField {
	currentVal, defaultVal := resolver.resolve(meta.path)
	if meta.fieldType == FieldTypeSelect {
		currentVal = normalizeSelectRegistryValue(currentVal)
		defaultVal = normalizeSelectRegistryValue(defaultVal)
	}
	return ConfigField{
		Path:        meta.path,
		DisplayName: meta.displayName,
		Description: meta.description,
		FieldType:   meta.fieldType,
		Options:     meta.options,
		Category:    meta.category,
		Current:     currentVal,
		Default:     defaultVal,
	}
}

func normalizeSelectRegistryValue(value interface{}) interface{} {
	if value == nil {
		return ""
	}
	rv := reflect.ValueOf(value)
	if rv.IsValid() && rv.Kind() == reflect.String {
		return rv.String()
	}
	return value
}

func buildConfigCategory(catDef CategoryDef, resolver registryFieldResolver) ConfigCategory {
	cat := ConfigCategory{
		Name:        catDef.Name,
		DisplayName: catDef.DisplayName,
		Icon:        catDef.Icon,
	}
	for _, fieldPath := range catDef.Fields {
		meta := newRegistryFieldMeta(catDef.Name, fieldPath)
		cat.Fields = append(cat.Fields, buildRegistryField(meta, resolver))
	}
	return cat
}

// BuildConfigRegistry はConfig構造体からカテゴリリストを構築する
func BuildConfigRegistry(cfg *Config) []ConfigCategory {
	resolver := newRegistryFieldResolver(cfg)
	categories := make([]ConfigCategory, 0, len(CategoryDefinitions))
	for _, catDef := range CategoryDefinitions {
		categories = append(categories, buildConfigCategory(catDef, resolver))
	}
	return categories
}
