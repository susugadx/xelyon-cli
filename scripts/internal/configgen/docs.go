package configgen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// StructInfo stores parsed config struct information for docs generation.
type StructInfo struct {
	Name    string
	Comment string
	Fields  []FieldInfo
}

// FieldInfo stores parsed field information for docs generation.
type FieldInfo struct {
	Name       string
	Type       string
	YAMLTag    string
	Comment    string
	IsOptional bool
}

// ReplaceMarkerContent replaces the content between two markers.
func ReplaceMarkerContent(content, startMarker, endMarker, newContent string) (string, bool) {
	if !strings.Contains(content, startMarker) || !strings.Contains(content, endMarker) {
		return content, false
	}
	pattern := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(startMarker) + `.*?` + regexp.QuoteMeta(endMarker))
	replacement := startMarker + "\n" + newContent + "\n" + endMarker
	return pattern.ReplaceAllString(content, replacement), true
}

// FormatConfigExample strips the file header and wraps the example in a YAML code block.
func FormatConfigExample(example string) string {
	lines := strings.Split(example, "\n")
	var yamlLines []string
	inYAML := false
	for _, line := range lines {
		if !strings.HasPrefix(line, "#") && strings.TrimSpace(line) != "" {
			inYAML = true
		}
		if inYAML {
			yamlLines = append(yamlLines, line)
		}
	}
	return "```yaml\n" + strings.Join(yamlLines, "\n") + "```"
}

// ParseConfigTypes parses struct comments and YAML-tagged fields from a config package dir.
func ParseConfigTypes(dir string) ([]StructInfo, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	structMap := make(map[string]StructInfo)
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				typeSpec, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					return true
				}
				if typeSpec.Name.Name == "Config" {
					return true
				}

				var structComment string
				if typeSpec.Doc != nil {
					structComment = strings.TrimSpace(typeSpec.Doc.Text())
				}

				var fields []FieldInfo
				for _, field := range structType.Fields.List {
					if len(field.Names) == 0 {
						continue
					}
					fieldName := field.Names[0].Name
					fieldType := getTypeString(field.Type)

					var yamlTag string
					var isOptional bool
					if field.Tag != nil {
						tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
						yamlTag = tag.Get("yaml")
						if strings.Contains(yamlTag, ",omitempty") {
							isOptional = true
							yamlTag = strings.Split(yamlTag, ",")[0]
						}
					}

					var comment string
					if field.Doc != nil {
						comment = strings.TrimSpace(field.Doc.Text())
					} else if field.Comment != nil {
						comment = strings.TrimSpace(field.Comment.Text())
					}

					fields = append(fields, FieldInfo{
						Name:       fieldName,
						Type:       fieldType,
						YAMLTag:    yamlTag,
						Comment:    comment,
						IsOptional: isOptional,
					})
				}

				structMap[typeSpec.Name.Name] = StructInfo{
					Name:    typeSpec.Name.Name,
					Comment: structComment,
					Fields:  fields,
				}
				return true
			})
		}
	}

	var structs []StructInfo
	for _, info := range structMap {
		structs = append(structs, info)
	}
	sort.Slice(structs, func(i, j int) bool {
		return structs[i].Name < structs[j].Name
	})
	return structs, nil
}

// GenerateConfigDetails builds the docs section details from parsed struct info and defaults.
func GenerateConfigDetails(structs []StructInfo, defaults map[string]interface{}) string {
	var sb strings.Builder

	structMap := make(map[string]StructInfo)
	for _, s := range structs {
		structMap[s.Name] = s
	}

	for _, sectionKey := range SectionOrder {
		section, ok := Sections[sectionKey]
		if !ok || strings.TrimSpace(section.StructName) == "" {
			continue
		}
		s, ok := structMap[section.StructName]
		if !ok {
			continue
		}

		sb.WriteString(fmt.Sprintf("### %s (`%s`)\n\n", section.Title, sectionKey))
		if s.Comment != "" {
			lines := strings.Split(s.Comment, "\n")
			if len(lines) > 0 {
				sb.WriteString(lines[0] + "\n\n")
			}
		}

		sb.WriteString("```yaml\n")
		sb.WriteString(fmt.Sprintf("%s:\n", sectionKey))
		for _, f := range s.Fields {
			if f.YAMLTag == "" || f.YAMLTag == "-" {
				continue
			}
			if strings.HasPrefix(f.Comment, "内部:") || strings.HasPrefix(f.Comment, "内部既定値") {
				continue
			}
			if strings.HasPrefix(f.Type, "map[") {
				sb.WriteString(fmt.Sprintf("  %s: { ... }\n", f.YAMLTag))
				continue
			}
			defaultVal := getDefaultValue(defaults, sectionKey, f.YAMLTag)
			sb.WriteString(fmt.Sprintf("  %s: %s\n", f.YAMLTag, formatYAMLValue(defaultVal)))
		}
		sb.WriteString("```\n\n")

		for _, f := range s.Fields {
			if f.YAMLTag == "" || f.YAMLTag == "-" {
				continue
			}
			if strings.HasPrefix(f.Comment, "内部:") || strings.HasPrefix(f.Comment, "内部既定値") {
				continue
			}

			sb.WriteString(fmt.Sprintf("#### `%s`\n", f.YAMLTag))
			sb.WriteString(fmt.Sprintf("- **型**: %s\n", mapGoTypeToDisplay(f.Type)))

			defaultVal := getDefaultValue(defaults, sectionKey, f.YAMLTag)
			if defaultVal != nil && !strings.HasPrefix(f.Type, "map[") {
				sb.WriteString(fmt.Sprintf("- **デフォルト**: `%v`\n", formatDefaultValue(defaultVal)))
			}

			desc := extractDescription(f.Comment)
			if desc != "" {
				sb.WriteString(fmt.Sprintf("- **説明**: %s\n", desc))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.ArrayType:
		return "[]" + getTypeString(t.Elt)
	case *ast.MapType:
		return "map[" + getTypeString(t.Key) + "]" + getTypeString(t.Value)
	case *ast.SelectorExpr:
		return getTypeString(t.X) + "." + t.Sel.Name
	default:
		return "unknown"
	}
}

func getDefaultValue(defaults map[string]interface{}, section, field string) interface{} {
	if sectionMap, ok := defaults[section].(map[string]interface{}); ok {
		return sectionMap[field]
	}
	return nil
}

func formatYAMLValue(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case bool:
		return fmt.Sprintf("%v", val)
	case int, int64, float64:
		return fmt.Sprintf("%v", val)
	case string:
		if val == "" {
			return `""`
		}
		return val
	case []interface{}:
		if len(val) == 0 {
			return "[]"
		}
		return "[...]"
	default:
		return "..."
	}
}

func formatDefaultValue(v interface{}) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case bool:
		return fmt.Sprintf("%v", val)
	case int, int64, float64:
		return fmt.Sprintf("%v", val)
	case string:
		if val == "" {
			return `""`
		}
		return val
	case []interface{}:
		if len(val) == 0 {
			return "[]"
		}
		var items []string
		for _, item := range val {
			items = append(items, fmt.Sprintf("%v", item))
		}
		return "[" + strings.Join(items, ", ") + "]"
	default:
		return fmt.Sprintf("%v", val)
	}
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
	re := regexp.MustCompile(`\s*[\(（]デフォルト[：:][^)）]+[)）]`)
	desc := re.ReplaceAllString(comment, "")
	return strings.TrimSpace(desc)
}
