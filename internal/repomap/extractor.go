//go:build !norepomap
// +build !norepomap

package repomap

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// ExtractSymbols はファイルからシンボルを抽出
func ExtractSymbols(filePath string) (*FileSymbols, error) {
	// 正規表現ベース抽出が必要なファイル（Makefile, Jenkinsfile等）
	if IsConfigFile(filePath) {
		return extractConfigFileSymbols(filePath)
	}

	// Issue #63: Vue/Svelte SFC（ハイブリッド方式）
	if IsSFCFile(filePath) {
		return extractSFCSymbols(filePath)
	}

	// Issue #65: Tailwind CSS config
	if IsTailwindConfigFile(filePath) {
		return extractTailwindConfigSymbols(filePath)
	}

	lang := GetLanguage(filePath)
	if lang == nil {
		return nil, nil // サポートされていない言語
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	root := tree.RootNode()
	symbols := extractFromNode(root, content, filePath)

	return &FileSymbols{
		Path:    filePath,
		Symbols: symbols,
	}, nil
}

// extractFromNode はノードからシンボルを再帰的に抽出
func extractFromNode(node *sitter.Node, content []byte, filePath string) []Symbol {
	var symbols []Symbol

	// ノードタイプに応じてシンボルを抽出
	// 拡張子で言語を判定
	ext := strings.ToLower(filepath.Ext(filePath))
	nodeType := node.Type()

	switch nodeType {
	// Go
	case "function_declaration":
		switch ext {
		case ".go":
			symbols = append(symbols, extractGoFunction(node, content, filePath))
		case ".kt", ".kts":
			symbols = append(symbols, extractKotlinFunction(node, content, filePath))
		case ".swift":
			symbols = append(symbols, extractSwiftFunction(node, content, filePath))
		default:
			symbols = append(symbols, extractJSFunction(node, content, filePath))
		}

	case "method_declaration":
		switch ext {
		case ".go":
			symbols = append(symbols, extractGoMethod(node, content, filePath))
		case ".java":
			symbols = append(symbols, extractJavaMethod(node, content, filePath))
		case ".cs":
			symbols = append(symbols, extractCSharpMethod(node, content, filePath))
		}

	case "type_declaration":
		symbols = append(symbols, extractGoType(node, content, filePath)...)

	// JavaScript/TypeScript
	// "function" ノードは匿名関数式 (function expression)
	// 注意: "function" キーワードノードではない（それは ChildCount=0）
	case "function":
		// ChildCount > 0 の場合のみ処理（キーワードノードではなく関数式ノード）
		if node.ChildCount() > 0 {
			symbols = append(symbols, extractJSFunction(node, content, filePath))
		}

	// Issue #62: Arrow Function Components / Hooks
	case "lexical_declaration", "variable_declaration":
		if isJSFamily(ext) {
			if sym := extractArrowFunctionOrHook(node, content, filePath); sym != nil {
				symbols = append(symbols, *sym)
			}
		}

	case "class_declaration":
		switch ext {
		case ".java":
			symbols = append(symbols, extractJavaClass(node, content, filePath))
		case ".kt", ".kts":
			symbols = append(symbols, extractKotlinClass(node, content, filePath))
		case ".scala":
			symbols = append(symbols, extractScalaClass(node, content, filePath))
		case ".php":
			symbols = append(symbols, extractPHPClass(node, content, filePath))
		case ".cs":
			symbols = append(symbols, extractCSharpClass(node, content, filePath))
		default:
			symbols = append(symbols, extractJSClass(node, content, filePath))
		}

	case "method_definition":
		symbols = append(symbols, extractJSMethod(node, content, filePath))

	// Python / C / PHP / Bash
	case "function_definition":
		if isCFamily(ext) {
			symbols = append(symbols, extractCFunction(node, content, filePath))
		} else if ext == ".php" {
			symbols = append(symbols, extractPHPFunction(node, content, filePath))
		} else if ext == ".scala" {
			symbols = append(symbols, extractScalaFunction(node, content, filePath))
		} else if ext == ".sh" || ext == ".bash" || ext == ".zsh" {
			symbols = append(symbols, extractBashFunction(node, content, filePath))
		} else {
			symbols = append(symbols, extractPyFunction(node, content, filePath))
		}

	case "class_definition":
		if ext == ".swift" {
			symbols = append(symbols, extractSwiftClass(node, content, filePath))
		} else {
			symbols = append(symbols, extractPyClass(node, content, filePath))
		}

	// Rust
	case "function_item":
		symbols = append(symbols, extractRustFunction(node, content, filePath))
	case "struct_item":
		symbols = append(symbols, extractRustStruct(node, content, filePath))
	case "impl_item":
		symbols = append(symbols, extractRustImpl(node, content, filePath)...)
	case "trait_item":
		symbols = append(symbols, extractRustTrait(node, content, filePath))
	case "enum_item":
		symbols = append(symbols, extractRustEnum(node, content, filePath))

	// Java / C#
	case "interface_declaration":
		switch ext {
		case ".java":
			symbols = append(symbols, extractJavaInterface(node, content, filePath))
		case ".cs":
			symbols = append(symbols, extractCSharpInterface(node, content, filePath))
		}

	// Ruby
	case "method":
		if ext == ".rb" {
			symbols = append(symbols, extractRubyMethod(node, content, filePath))
		}
	case "class":
		if ext == ".rb" {
			symbols = append(symbols, extractRubyClass(node, content, filePath))
		}
	case "module":
		if ext == ".rb" {
			symbols = append(symbols, extractRubyModule(node, content, filePath))
		}

	// Kotlin
	case "object_declaration":
		if ext == ".kt" || ext == ".kts" {
			symbols = append(symbols, extractKotlinObject(node, content, filePath))
		}

	// Swift
	case "struct_declaration":
		switch ext {
		case ".swift":
			symbols = append(symbols, extractSwiftStruct(node, content, filePath))
		case ".cs":
			symbols = append(symbols, extractCSharpStruct(node, content, filePath))
		}
	case "protocol_declaration":
		if ext == ".swift" {
			symbols = append(symbols, extractSwiftProtocol(node, content, filePath))
		}

	// Scala
	case "object_definition":
		if ext == ".scala" {
			symbols = append(symbols, extractScalaObject(node, content, filePath))
		}
	case "trait_definition":
		if ext == ".scala" {
			symbols = append(symbols, extractScalaTrait(node, content, filePath))
		}

	// CSS/SCSS (#59)
	case "rule_set":
		if ext == ".css" || ext == ".scss" {
			symbols = append(symbols, extractCSSRuleSet(node, content, filePath))
		}
	case "keyframes_statement":
		if ext == ".css" || ext == ".scss" {
			symbols = append(symbols, extractCSSKeyframes(node, content, filePath))
		}
	case "media_statement":
		if ext == ".css" || ext == ".scss" {
			symbols = append(symbols, extractCSSMediaQuery(node, content, filePath))
		}
	case "declaration":
		if (ext == ".css" || ext == ".scss") && isCSSCustomProperty(node, content) {
			symbols = append(symbols, extractCSSCustomProperty(node, content, filePath))
		}

	// YAML (#60)
	case "block_mapping_pair":
		if ext == ".yaml" || ext == ".yml" {
			// トップレベルのキーのみ抽出
			if isTopLevelYAMLKey(node) {
				symbols = append(symbols, extractYAMLKey(node, content, filePath))
			}
		}

	// TOML (#60)
	case "table":
		if ext == ".toml" {
			symbols = append(symbols, extractTOMLTable(node, content, filePath))
		}
	case "table_array_element":
		if ext == ".toml" {
			symbols = append(symbols, extractTOMLTableArray(node, content, filePath))
		}

	// SQL (#60)
	case "create_table_statement":
		if ext == ".sql" {
			symbols = append(symbols, extractSQLCreateTable(node, content, filePath))
		}
	case "create_function_statement":
		if ext == ".sql" {
			symbols = append(symbols, extractSQLCreateFunction(node, content, filePath))
		}
	case "create_view_statement":
		if ext == ".sql" {
			symbols = append(symbols, extractSQLCreateView(node, content, filePath))
		}
	case "create_index_statement":
		if ext == ".sql" {
			symbols = append(symbols, extractSQLCreateIndex(node, content, filePath))
		}

	// Dockerfile (#60)
	case "from_instruction":
		baseName := filepath.Base(filePath)
		if baseName == "Dockerfile" || strings.HasPrefix(baseName, "Dockerfile.") {
			symbols = append(symbols, extractDockerFrom(node, content, filePath))
		}
	case "run_instruction":
		baseName := filepath.Base(filePath)
		if baseName == "Dockerfile" || strings.HasPrefix(baseName, "Dockerfile.") {
			symbols = append(symbols, extractDockerRun(node, content, filePath))
		}
	case "cmd_instruction", "entrypoint_instruction":
		baseName := filepath.Base(filePath)
		if baseName == "Dockerfile" || strings.HasPrefix(baseName, "Dockerfile.") {
			symbols = append(symbols, extractDockerCmd(node, content, filePath))
		}

	// Markdown (#60)
	case "atx_heading", "setext_heading":
		if ext == ".md" || ext == ".markdown" {
			symbols = append(symbols, extractMarkdownHeading(node, content, filePath))
		}

	// HTML (#64)
	case "element":
		if ext == ".html" || ext == ".htm" {
			htmlSymbols := extractHTMLElement(node, content, filePath)
			symbols = append(symbols, htmlSymbols...)
		}

	// Elixir
	case "call":
		if ext == ".ex" || ext == ".exs" {
			if sym := extractElixirCall(node, content, filePath); sym != nil {
				symbols = append(symbols, *sym)
			}
		}

	// Lua
	case "function_statement":
		if ext == ".lua" {
			symbols = append(symbols, extractLuaFunction(node, content, filePath))
		}
	}

	// 子ノードを再帰的に処理
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		symbols = append(symbols, extractFromNode(child, content, filePath)...)
	}

	return symbols
}
