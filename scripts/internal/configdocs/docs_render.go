package configdocs

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/scripts/internal/configmeta"
)

type valueSliceFormatMode int

const (
	valueSliceFormatPlaceholder valueSliceFormatMode = iota
	valueSliceFormatExpanded
)

var defaultValueAnnotationRe = regexp.MustCompile(`\s*[\(（]デフォルト[：:][^)）]+[)）]`)

// generateConfigDetails は解析済み struct 情報と default から docs の詳細 section を生成する。
func generateConfigDetails(structs []structInfo, defaults map[string]interface{}) string {
	var sb strings.Builder

	structMap := make(map[string]structInfo, len(structs))
	for _, structInfo := range structs {
		structMap[structInfo.Name] = structInfo
	}

	for _, sectionKey := range configmeta.SectionOrder {
		section, ok := configmeta.Sections[sectionKey]
		if !ok || strings.TrimSpace(section.StructName) == "" {
			continue
		}
		structInfo, ok := structMap[section.StructName]
		if !ok {
			continue
		}

		visibleFields := filterVisibleDocFields(structInfo.Fields)
		if len(visibleFields) == 0 {
			continue
		}

		fmt.Fprintf(&sb, "### %s (`%s`)\n\n", section.Title, sectionKey)
		appendStructSummary(&sb, structInfo.Comment)
		appendSectionYAMLExample(&sb, sectionKey, visibleFields, defaults)
		appendSectionFieldDetails(&sb, sectionKey, visibleFields, defaults)
	}

	return sb.String()
}

func filterVisibleDocFields(fields []fieldInfo) []fieldInfo {
	visible := make([]fieldInfo, 0, len(fields))
	for _, field := range fields {
		if field.YAMLTag == "" || field.YAMLTag == "-" {
			continue
		}
		if isInternalDocField(field.Comment) {
			continue
		}
		visible = append(visible, field)
	}
	return visible
}

func isInternalDocField(comment string) bool {
	return strings.HasPrefix(comment, "内部:") || strings.HasPrefix(comment, "内部既定値")
}

func appendStructSummary(builder *strings.Builder, structComment string) {
	if structComment == "" {
		return
	}
	lines := strings.Split(structComment, "\n")
	if len(lines) == 0 {
		return
	}
	builder.WriteString(lines[0] + "\n\n")
}

func appendSectionYAMLExample(builder *strings.Builder, sectionKey string, fields []fieldInfo, defaults map[string]interface{}) {
	builder.WriteString("```yaml\n")
	fmt.Fprintf(builder, "%s:\n", sectionKey)
	for _, field := range fields {
		if strings.HasPrefix(field.Type, "map[") {
			fmt.Fprintf(builder, "  %s: { ... }\n", field.YAMLTag)
			continue
		}
		defaultValue := getDefaultValue(defaults, sectionKey, field.YAMLTag)
		fmt.Fprintf(builder, "  %s: %s\n", field.YAMLTag, formatYAMLValue(defaultValue))
	}
	builder.WriteString("```\n\n")
}

func appendSectionFieldDetails(builder *strings.Builder, sectionKey string, fields []fieldInfo, defaults map[string]interface{}) {
	for _, field := range fields {
		fmt.Fprintf(builder, "#### `%s`\n", field.YAMLTag)
		fmt.Fprintf(builder, "- **型**: %s\n", mapGoTypeToDisplay(field.Type))

		defaultValue := getDefaultValue(defaults, sectionKey, field.YAMLTag)
		if defaultValue != nil && !strings.HasPrefix(field.Type, "map[") {
			fmt.Fprintf(builder, "- **デフォルト**: `%v`\n", formatDefaultValue(defaultValue))
		}

		description := extractDescription(field.Comment)
		if description != "" {
			fmt.Fprintf(builder, "- **説明**: %s\n", description)
		}
		builder.WriteString("\n")
	}
}

func getDefaultValue(defaults map[string]interface{}, section, field string) interface{} {
	if sectionMap, ok := defaults[section].(map[string]interface{}); ok {
		return sectionMap[field]
	}
	return nil
}

func formatYAMLValue(value interface{}) string {
	return formatConfigValue(value, valueSliceFormatPlaceholder)
}

func formatDefaultValue(value interface{}) string {
	return formatConfigValue(value, valueSliceFormatExpanded)
}

func formatConfigValue(value interface{}, sliceMode valueSliceFormatMode) string {
	if value == nil {
		return "null"
	}
	switch typed := value.(type) {
	case bool:
		return fmt.Sprintf("%v", typed)
	case int, int64, float64:
		return fmt.Sprintf("%v", typed)
	case string:
		if typed == "" {
			return `""`
		}
		return typed
	case []interface{}:
		return formatInterfaceSliceValue(typed, sliceMode)
	case []string:
		items := make([]interface{}, len(typed))
		for i, item := range typed {
			items[i] = item
		}
		return formatInterfaceSliceValue(items, sliceMode)
	default:
		if sliceMode == valueSliceFormatPlaceholder {
			return "..."
		}
		return fmt.Sprintf("%v", typed)
	}
}

func formatInterfaceSliceValue(items []interface{}, sliceMode valueSliceFormatMode) string {
	if len(items) == 0 {
		return "[]"
	}
	if sliceMode == valueSliceFormatPlaceholder {
		return "[...]"
	}
	formatted := make([]string, 0, len(items))
	for _, item := range items {
		formatted = append(formatted, fmt.Sprintf("%v", item))
	}
	return "[" + strings.Join(formatted, ", ") + "]"
}

func mapGoTypeToDisplay(goType string) string {
	switch goType {
	case "bool":
		return "boolean"
	case "int", "int64":
		return "integer"
	case "string":
		return "string"
	case "[]string":
		return "string[]"
	default:
		if strings.HasPrefix(goType, "map[") {
			return "map"
		}
		return goType
	}
}

func extractDescription(comment string) string {
	if comment == "" {
		return ""
	}
	description := defaultValueAnnotationRe.ReplaceAllString(comment, "")
	return strings.TrimSpace(description)
}
