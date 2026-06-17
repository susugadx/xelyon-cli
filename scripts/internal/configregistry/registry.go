package configregistry

import (
	"fmt"
	"slices"
	"sort"

	"github.com/susugadx/xelyon-cli/scripts/internal/configmeta"
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

func fieldTypeToConst(fieldType string) (string, error) {
	if constant, ok := fieldTypeToConstMap[fieldType]; ok {
		return constant, nil
	}
	return "", fmt.Errorf("unsupported field type %q", fieldType)
}

func uniqueStrings(values []string) []string {
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

func collectCategoryFields(category string) []string {
	var fields []string
	for _, sectionName := range configmeta.OrderedSectionsForCategory(category) {
		section, ok := configmeta.Sections[sectionName]
		if !ok {
			continue
		}
		for fieldName := range section.FieldTypes {
			if fieldName == sectionName {
				fields = append(fields, sectionName)
			}
		}
		for _, fieldName := range sortedKeys(section.Fields) {
			fields = append(fields, configmeta.CanonicalFieldPath(sectionName, fieldName))
		}
	}
	fields = uniqueStrings(fields)
	sort.Strings(fields)
	return fields
}

type registryFieldTypeEntry struct {
	Path      string
	FieldType string
}

type registrySelectEntry struct {
	Path    string
	Options []string
}

type registryDescriptionEntry struct {
	Path        string
	Description string
}

func buildRegistryFieldTypeEntries() ([]registryFieldTypeEntry, error) {
	var entries []registryFieldTypeEntry
	for _, sectionName := range sortedKeys(configmeta.Sections) {
		section := configmeta.Sections[sectionName]
		for _, fieldName := range sortedKeys(section.FieldTypes) {
			fieldType := section.FieldTypes[fieldName]
			constant, err := fieldTypeToConst(fieldType)
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", sectionName, fieldName, err)
			}
			entries = append(entries, registryFieldTypeEntry{
				Path:      configmeta.CanonicalFieldPath(sectionName, fieldName),
				FieldType: constant,
			})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Path < entries[j].Path
	})
	return entries, nil
}

func buildRegistrySelectEntries() []registrySelectEntry {
	var entries []registrySelectEntry
	for _, sectionName := range sortedKeys(configmeta.Sections) {
		section := configmeta.Sections[sectionName]
		for _, fieldName := range sortedKeys(section.SelectOpts) {
			path := configmeta.CanonicalFieldPath(sectionName, fieldName)
			opts := slices.Clone(section.SelectOpts[fieldName])
			entries = append(entries, registrySelectEntry{
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

func buildRegistryDescriptionEntries() []registryDescriptionEntry {
	var entries []registryDescriptionEntry
	for _, sectionName := range sortedKeys(configmeta.Sections) {
		section := configmeta.Sections[sectionName]
		for _, fieldName := range sortedKeys(section.Fields) {
			path := configmeta.CanonicalFieldPath(sectionName, fieldName)
			entries = append(entries, registryDescriptionEntry{
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

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
