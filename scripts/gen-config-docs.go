//go:build ignore

// gen-config-docs.go は config_types.go のコメントから docs/config.md を自動更新するスクリプト
//
// 使用方法:
//
//	go run scripts/gen-config-docs.go
//	make gen-docs
//
// docs/config.md 内の以下のマーカー間を自動更新:
//
//	<!-- CONFIG-EXAMPLE-START --> ... <!-- CONFIG-EXAMPLE-END -->
//	<!-- CONFIG-DETAILS-START --> ... <!-- CONFIG-DETAILS-END -->
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"gopkg.in/yaml.v3"
)

// StructInfo は構造体の情報
type StructInfo struct {
	Name    string
	Comment string
	Fields  []FieldInfo
}

// FieldInfo はフィールドの情報
type FieldInfo struct {
	Name       string
	Type       string
	YAMLTag    string
	Comment    string
	IsOptional bool
}

func main() {
	// 1. config_types.go をパース
	structs, err := parseConfigTypes("internal/config/config_types.go")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing config_types.go: %v\n", err)
		os.Exit(1)
	}

	// 2. デフォルト設定を取得
	defaultCfg := config.DefaultConfig()

	// 3. 設定例を生成（config.yaml.example から読み込み）
	configExample, err := os.ReadFile("config.yaml.example")
	if err != nil {
		// config.yaml.example がない場合は生成
		fmt.Println("config.yaml.example not found, generating...")
		configExample = generateConfigExample(defaultCfg)
	}

	// 4. 設定項目詳細を生成
	defaultYAML, _ := yaml.Marshal(defaultCfg)
	defaults := make(map[string]interface{})
	yaml.Unmarshal(defaultYAML, &defaults)
	configDetails := generateConfigDetails(structs, defaults)

	// 5. docs/config.md を読み込み
	configMdPath := "docs/config.md"
	content, err := os.ReadFile(configMdPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", configMdPath, err)
		os.Exit(1)
	}

	// 6. マーカー間を置換
	newContent := string(content)

	// 設定例を置換
	newContent = replaceMarkerContent(newContent,
		"<!-- CONFIG-EXAMPLE-START -->",
		"<!-- CONFIG-EXAMPLE-END -->",
		formatConfigExample(string(configExample)))

	// 設定項目詳細を置換
	newContent = replaceMarkerContent(newContent,
		"<!-- CONFIG-DETAILS-START -->",
		"<!-- CONFIG-DETAILS-END -->",
		configDetails)

	// 7. 保存
	if err := os.WriteFile(configMdPath, []byte(newContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", configMdPath, err)
		os.Exit(1)
	}

	fmt.Printf("Updated %s\n", configMdPath)
}

func replaceMarkerContent(content, startMarker, endMarker, newContent string) string {
	pattern := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(startMarker) + `.*?` + regexp.QuoteMeta(endMarker))
	replacement := startMarker + "\n" + newContent + "\n" + endMarker
	return pattern.ReplaceAllString(content, replacement)
}

func generateConfigExample(cfg *config.Config) []byte {
	data, _ := yaml.Marshal(cfg)
	header := `# XELYON CLI 設定例
# 設定ファイルの場所: ~/.xelyon/config.yaml
# 初回起動時に自動的に作成されます

`
	return []byte(header + string(data))
}

func formatConfigExample(example string) string {
	// ヘッダーコメントを除去（yamlブロック内に入れるため）
	lines := strings.Split(example, "\n")
	var yamlLines []string
	inYaml := false
	for _, line := range lines {
		if !strings.HasPrefix(line, "#") && strings.TrimSpace(line) != "" {
			inYaml = true
		}
		if inYaml {
			yamlLines = append(yamlLines, line)
		}
	}

	return "```yaml\n" + strings.Join(yamlLines, "\n") + "```"
}

func parseConfigTypes(filename string) ([]StructInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var structs []StructInfo

	ast.Inspect(node, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		// Config 構造体自体はスキップ（トップレベル）
		if typeSpec.Name.Name == "Config" {
			return true
		}

		// 構造体のコメントを取得
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

			// yaml タグを取得
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

			// フィールドのコメントを取得
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

		structs = append(structs, StructInfo{
			Name:    typeSpec.Name.Name,
			Comment: structComment,
			Fields:  fields,
		})

		return true
	})

	return structs, nil
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

func generateConfigDetails(structs []StructInfo, defaults map[string]interface{}) string {
	var sb strings.Builder

	// 構造体名からYAMLキーへのマッピング
	structToYAML := map[string]string{
		"GeneralConfig":      "general",
		"CompressionConfig":  "compression",
		"ToolConfirmConfig":  "tool_confirm",
		"PromptCacheConfig":  "prompt_cache",
		"PasteConfig":        "paste",
		"StreamingConfig":    "streaming",
		"BashConfig":         "bash",
		"GitStageConfig":     "git_stage",
		"PlanModeConfig":     "plan_mode",
		"OpenAIConfig":       "openai",
		"ThinkingConfig":     "thinking",
		"LSPConfig":          "lsp",
		"WebSearchConfig":    "web_search",
		"UtilityModelConfig": "utility_model",
	}

	// 順序を定義（内部専用: LoopDetection, APIRetry, Diff は除外）
	order := []string{
		"GeneralConfig",
		"CompressionConfig",
		"StreamingConfig",
		"ToolConfirmConfig",
		"PromptCacheConfig",
		"PasteConfig",
		"BashConfig",
		"GitStageConfig",
		"PlanModeConfig",
		"OpenAIConfig",
		"LSPConfig",
		"ThinkingConfig",
		"WebSearchConfig",
		"UtilityModelConfig",
	}

	structMap := make(map[string]StructInfo)
	for _, s := range structs {
		structMap[s.Name] = s
	}

	for _, structName := range order {
		s, ok := structMap[structName]
		if !ok {
			continue
		}

		yamlKey, ok := structToYAML[structName]
		if !ok {
			continue
		}

		// セクションタイトル
		title := formatTitle(structName)
		sb.WriteString(fmt.Sprintf("### %s (`%s`)\n\n", title, yamlKey))

		// 構造体のコメント（説明として使用）
		if s.Comment != "" {
			// 最初の1行だけ使用
			lines := strings.Split(s.Comment, "\n")
			if len(lines) > 0 {
				sb.WriteString(lines[0] + "\n\n")
			}
		}

		// YAML例（「内部:」コメントで始まるフィールドはスキップ）
		sb.WriteString("```yaml\n")
		sb.WriteString(fmt.Sprintf("%s:\n", yamlKey))
		for _, f := range s.Fields {
			if f.YAMLTag == "" || f.YAMLTag == "-" {
				continue
			}
			if strings.HasPrefix(f.Comment, "内部:") || strings.HasPrefix(f.Comment, "内部既定値") {
				continue
			}
			// map や複雑な型はスキップ
			if strings.HasPrefix(f.Type, "map[") {
				sb.WriteString(fmt.Sprintf("  %s: { ... }\n", f.YAMLTag))
				continue
			}
			defaultVal := getDefaultValue(defaults, yamlKey, f.YAMLTag)
			sb.WriteString(fmt.Sprintf("  %s: %s\n", f.YAMLTag, formatYAMLValue(defaultVal)))
		}
		sb.WriteString("```\n\n")

		// フィールド詳細（「内部:」コメントで始まるフィールドはスキップ）
		for _, f := range s.Fields {
			if f.YAMLTag == "" || f.YAMLTag == "-" {
				continue
			}
			if strings.HasPrefix(f.Comment, "内部:") || strings.HasPrefix(f.Comment, "内部既定値") {
				continue
			}

			sb.WriteString(fmt.Sprintf("#### `%s`\n", f.YAMLTag))

			// 型
			displayType := mapGoTypeToDisplay(f.Type)
			sb.WriteString(fmt.Sprintf("- **型**: %s\n", displayType))

			// デフォルト値
			defaultVal := getDefaultValue(defaults, yamlKey, f.YAMLTag)
			if defaultVal != nil && !strings.HasPrefix(f.Type, "map[") {
				sb.WriteString(fmt.Sprintf("- **デフォルト**: `%v`\n", formatDefaultValue(defaultVal)))
			}

			// 説明（コメントから抽出）
			desc := extractDescription(f.Comment)
			if desc != "" {
				sb.WriteString(fmt.Sprintf("- **説明**: %s\n", desc))
			}

			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func formatTitle(structName string) string {
	titles := map[string]string{
		"GeneralConfig":      "一般設定",
		"CompressionConfig":  "会話履歴圧縮設定",
		"ToolConfirmConfig":  "ツール確認設定",
		"PromptCacheConfig":  "プロンプトキャッシュ設定",
		"PasteConfig":        "ペーストモード設定",
		"StreamingConfig":    "ストリーミング設定",
		"BashConfig":         "bashツール設定",
		"GitStageConfig":     "git_add設定",
		"PlanModeConfig":     "Plan Mode設定",
		"OpenAIConfig":       "OpenAI設定",
		"ThinkingConfig":     "Extended Thinking設定",
		"LSPConfig":          "LSP連携設定",
		"WebSearchConfig":    "Web検索設定",
	}
	if title, ok := titles[structName]; ok {
		return title
	}
	return structName
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
		// 配列は省略
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
	// コメントからデフォルト値の記述を除去
	re := regexp.MustCompile(`\s*[\(（]デフォルト[：:][^)）]+[)）]`)
	desc := re.ReplaceAllString(comment, "")
	return strings.TrimSpace(desc)
}
