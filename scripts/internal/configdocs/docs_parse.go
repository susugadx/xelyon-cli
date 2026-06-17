package configdocs

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// parseConfigTypes は config package dir から struct コメントと YAML tag 付き field を解析する。
func parseConfigTypes(dir string) ([]structInfo, error) {
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	structMap := make(map[string]structInfo)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		path := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		if err := collectStructInfoFromFile(file, fset, structMap); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}

	var structs []structInfo
	for _, info := range structMap {
		structs = append(structs, info)
	}
	sort.Slice(structs, func(i, j int) bool {
		return structs[i].Name < structs[j].Name
	})
	return structs, nil
}

func collectStructInfoFromFile(file *ast.File, fset *token.FileSet, structMap map[string]structInfo) error {
	var parseErr error
	ast.Inspect(file, func(n ast.Node) bool {
		if parseErr != nil {
			return false
		}
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok || typeSpec.Name.Name == "Config" {
			return true
		}

		var structComment string
		if typeSpec.Doc != nil {
			structComment = strings.TrimSpace(typeSpec.Doc.Text())
		}

		fields, err := collectFieldInfo(structType, fset)
		if err != nil {
			parseErr = fmt.Errorf("struct %s: %w", typeSpec.Name.Name, err)
			return false
		}
		structMap[typeSpec.Name.Name] = structInfo{
			Name:    typeSpec.Name.Name,
			Comment: structComment,
			Fields:  fields,
		}
		return true
	})
	return parseErr
}

func collectFieldInfo(structType *ast.StructType, fset *token.FileSet) ([]fieldInfo, error) {
	fields := make([]fieldInfo, 0, len(structType.Fields.List))
	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue
		}
		fieldType, err := getTypeString(field.Type)
		if err != nil {
			pos := fset.Position(field.Type.Pos())
			return nil, fmt.Errorf("field declaration at %s: %w", pos.String(), err)
		}

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

		for _, nameNode := range field.Names {
			if nameNode == nil {
				continue
			}
			fieldName := nameNode.Name
			if strings.TrimSpace(fieldName) == "" {
				continue
			}
			fields = append(fields, fieldInfo{
				Name:       fieldName,
				Type:       fieldType,
				YAMLTag:    yamlTag,
				Comment:    comment,
				IsOptional: isOptional,
			})
		}
	}
	return fields, nil
}

func getTypeString(expr ast.Expr) (string, error) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, nil
	case *ast.ArrayType:
		eltType, err := getTypeString(t.Elt)
		if err != nil {
			return "", err
		}
		return "[]" + eltType, nil
	case *ast.MapType:
		keyType, err := getTypeString(t.Key)
		if err != nil {
			return "", err
		}
		valueType, err := getTypeString(t.Value)
		if err != nil {
			return "", err
		}
		return "map[" + keyType + "]" + valueType, nil
	case *ast.SelectorExpr:
		pkg, err := getTypeString(t.X)
		if err != nil {
			return "", err
		}
		return pkg + "." + t.Sel.Name, nil
	case *ast.StarExpr:
		baseType, err := getTypeString(t.X)
		if err != nil {
			return "", err
		}
		return "*" + baseType, nil
	case *ast.InterfaceType:
		return "interface{}", nil
	default:
		return "", fmt.Errorf("unsupported type expression %T", expr)
	}
}
