package configgen

import "sort"

// FieldTypeToConst converts a field type string to the generated enum constant.
func FieldTypeToConst(t string) string {
	switch t {
	case "bool":
		return "FieldTypeBool"
	case "int":
		return "FieldTypeInt"
	case "string":
		return "FieldTypeString"
	case "select":
		return "FieldTypeSelect"
	case "float":
		return "FieldTypeFloat"
	case "[]string":
		return "FieldTypeStringSlice"
	case "map":
		return "FieldTypeStringMap"
	case "structmap":
		return "FieldTypeStructMap"
	default:
		return "FieldTypeString"
	}
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
		if _, ok := section.FieldTypes[sectionName]; ok {
			fields = append(fields, sectionName)
		}
		for fieldName := range section.Fields {
			if fieldName == sectionName {
				fields = append(fields, fieldName)
			} else {
				fields = append(fields, sectionName+"."+fieldName)
			}
		}
	}
	fields = UniqueStrings(fields)
	sort.Strings(fields)
	return fields
}
