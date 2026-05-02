package configgen

import (
	"fmt"
	"slices"
	"sort"
)

var fieldTypeToConstMap = map[string]string{
	"bool":      "FieldTypeBool",
	"int":       "FieldTypeInt",
	"string":    "FieldTypeString",
	"select":    "FieldTypeSelect",
	"float":     "FieldTypeFloat",
	"[]string":  "FieldTypeStringSlice",
	"map":       "FieldTypeStringMap",
	"structmap": "FieldTypeStructMap",
}

// FieldTypeToConst はフィールド型のメタデータを生成コード側の enum 名へ変換する。
func FieldTypeToConst(fieldType string) (string, error) {
	if constant, ok := fieldTypeToConstMap[fieldType]; ok {
		return constant, nil
	}
	return "", fmt.Errorf("unsupported field type %q", fieldType)
}

// UniqueStrings removes duplicates while preserving the first occurrence order.
func UniqueStrings(values []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

// CollectCategoryFields returns the generated field paths for a category.
func CollectCategoryFields(category string) []string {
	var fields []string
	for _, sectionName := range OrderedSectionsForCategory(category) {
		section, ok := Sections[sectionName]
		if !ok {
			continue
		}
		for fieldName := range section.FieldTypes {
			if fieldName == sectionName {
				fields = append(fields, sectionName)
			}
		}
		for _, fieldName := range sortedKeys(section.Fields) {
			fields = append(fields, CanonicalFieldPath(sectionName, fieldName))
		}
	}
	fields = UniqueStrings(fields)
	sort.Strings(fields)
	return fields
}

// RegistryFieldTypeEntry は registry_generated.go の FieldTypeMap 1行分。
type RegistryFieldTypeEntry struct {
	Path      string
	FieldType string
}

// RegistrySelectEntry は registry_generated.go の SelectOptions 1行分。
type RegistrySelectEntry struct {
	Path    string
	Options []string
}

// RegistryDescriptionEntry は registry_generated.go の FieldDescriptions 1行分。
type RegistryDescriptionEntry struct {
	Path        string
	Description string
}

// BuildRegistryFieldTypeEntries は FieldTypeMap 用エントリを生成する。
func BuildRegistryFieldTypeEntries() ([]RegistryFieldTypeEntry, error) {
	var entries []RegistryFieldTypeEntry
	for _, sectionName := range sortedKeys(Sections) {
		section := Sections[sectionName]
		for _, fieldName := range sortedKeys(section.FieldTypes) {
			fieldType := section.FieldTypes[fieldName]
			constant, err := FieldTypeToConst(fieldType)
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", sectionName, fieldName, err)
			}
			entries = append(entries, RegistryFieldTypeEntry{
				Path:      CanonicalFieldPath(sectionName, fieldName),
				FieldType: constant,
			})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return entries, nil
}

// BuildRegistrySelectEntries は SelectOptions 用エントリを生成する。
func BuildRegistrySelectEntries() []RegistrySelectEntry {
	var entries []RegistrySelectEntry
	for _, sectionName := range sortedKeys(Sections) {
		section := Sections[sectionName]
		for _, fieldName := range sortedKeys(section.SelectOpts) {
			path := CanonicalFieldPath(sectionName, fieldName)
			opts := slices.Clone(section.SelectOpts[fieldName])
			entries = append(entries, RegistrySelectEntry{
				Path:    path,
				Options: opts,
			})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return entries
}

// BuildRegistryDescriptionEntries は FieldDescriptions 用エントリを生成する。
func BuildRegistryDescriptionEntries() []RegistryDescriptionEntry {
	var entries []RegistryDescriptionEntry
	for _, sectionName := range sortedKeys(Sections) {
		section := Sections[sectionName]
		for _, fieldName := range sortedKeys(section.Fields) {
			path := CanonicalFieldPath(sectionName, fieldName)
			entries = append(entries, RegistryDescriptionEntry{
				Path:        path,
				Description: section.Fields[fieldName],
			})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return entries
}
