//go:build !norepomap
// +build !norepomap

package repomap

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
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

// isCFamily は C系言語かどうかを判定
func isCFamily(ext string) bool {
	return ext == ".c" || ext == ".h" || ext == ".cpp" || ext == ".hpp" || ext == ".cc"
}

// isJSFamily は JavaScript/TypeScript系言語かどうかを判定
func isJSFamily(ext string) bool {
	return ext == ".js" || ext == ".jsx" || ext == ".ts" || ext == ".tsx" || ext == ".mjs"
}

// isPascalCase は PascalCase かどうかを判定（Reactコンポーネント規約）
func isPascalCase(name string) bool {
	if len(name) == 0 {
		return false
	}
	r := rune(name[0])
	return r >= 'A' && r <= 'Z'
}

// isHookName は React Hook 命名規約（useXxx）かどうかを判定
func isHookName(name string) bool {
	if len(name) < 4 {
		return false
	}
	if !strings.HasPrefix(name, "use") {
		return false
	}
	// use の後の文字が大文字であること（useAuth, useCounter など）
	r := rune(name[3])
	return r >= 'A' && r <= 'Z'
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
		if ext == ".go" {
			symbols = append(symbols, extractGoFunction(node, content, filePath))
		} else if ext == ".kt" || ext == ".kts" {
			symbols = append(symbols, extractKotlinFunction(node, content, filePath))
		} else if ext == ".swift" {
			symbols = append(symbols, extractSwiftFunction(node, content, filePath))
		} else {
			symbols = append(symbols, extractJSFunction(node, content, filePath))
		}

	case "method_declaration":
		if ext == ".go" {
			symbols = append(symbols, extractGoMethod(node, content, filePath))
		} else if ext == ".java" {
			symbols = append(symbols, extractJavaMethod(node, content, filePath))
		} else if ext == ".cs" {
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
		if ext == ".java" {
			symbols = append(symbols, extractJavaClass(node, content, filePath))
		} else if ext == ".kt" || ext == ".kts" {
			symbols = append(symbols, extractKotlinClass(node, content, filePath))
		} else if ext == ".scala" {
			symbols = append(symbols, extractScalaClass(node, content, filePath))
		} else if ext == ".php" {
			symbols = append(symbols, extractPHPClass(node, content, filePath))
		} else if ext == ".cs" {
			symbols = append(symbols, extractCSharpClass(node, content, filePath))
		} else {
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
		if ext == ".java" {
			symbols = append(symbols, extractJavaInterface(node, content, filePath))
		} else if ext == ".cs" {
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
		if ext == ".swift" {
			symbols = append(symbols, extractSwiftStruct(node, content, filePath))
		} else if ext == ".cs" {
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

	// ================== CSS/SCSS (#59) ==================
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

	// ================== YAML (#60) ==================
	case "block_mapping_pair":
		if ext == ".yaml" || ext == ".yml" {
			// トップレベルのキーのみ抽出
			if isTopLevelYAMLKey(node) {
				symbols = append(symbols, extractYAMLKey(node, content, filePath))
			}
		}

	// ================== TOML (#60) ==================
	case "table":
		if ext == ".toml" {
			symbols = append(symbols, extractTOMLTable(node, content, filePath))
		}
	case "table_array_element":
		if ext == ".toml" {
			symbols = append(symbols, extractTOMLTableArray(node, content, filePath))
		}

	// ================== SQL (#60) ==================
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

	// ================== Dockerfile (#60) ==================
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

	// ================== Markdown (#60) ==================
	case "atx_heading", "setext_heading":
		if ext == ".md" || ext == ".markdown" {
			symbols = append(symbols, extractMarkdownHeading(node, content, filePath))
		}

	// ================== HTML (#64) ==================
	case "element":
		if ext == ".html" || ext == ".htm" {
			htmlSymbols := extractHTMLElement(node, content, filePath)
			symbols = append(symbols, htmlSymbols...)
		}
	}

	// 子ノードを再帰的に処理
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		symbols = append(symbols, extractFromNode(child, content, filePath)...)
	}

	return symbols
}

// extractGoFunction はGo関数を抽出
func extractGoFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	// シグネチャは関数定義行全体
	startByte := node.StartByte()
	// 最初の { まで取得
	sig := ""
	for i := startByte; i < node.EndByte(); i++ {
		if content[i] == '{' {
			sig = strings.TrimSpace(string(content[startByte:i]))
			break
		}
	}
	if sig == "" {
		sig = strings.TrimSpace(string(content[startByte:node.EndByte()]))
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// extractGoMethod はGoメソッドを抽出
func extractGoMethod(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	startByte := node.StartByte()
	sig := ""
	for i := startByte; i < node.EndByte(); i++ {
		if content[i] == '{' {
			sig = strings.TrimSpace(string(content[startByte:i]))
			break
		}
	}

	return Symbol{
		Name:      name,
		Kind:      "method",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// extractGoType はGo型定義を抽出
func extractGoType(node *sitter.Node, content []byte, filePath string) []Symbol {
	var symbols []Symbol

	// type_spec を探す
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "type_spec" {
			nameNode := child.ChildByFieldName("name")
			typeNode := child.ChildByFieldName("type")

			name := ""
			if nameNode != nil {
				name = nameNode.Content(content)
			}

			kind := "type"
			if typeNode != nil {
				switch typeNode.Type() {
				case "struct_type":
					kind = "struct"
				case "interface_type":
					kind = "interface"
				}
			}

			sig := "type " + name
			if typeNode != nil {
				sig = strings.TrimSpace(string(content[child.StartByte():child.EndByte()]))
				// 長すぎる場合は省略
				if len(sig) > 100 {
					sig = sig[:100] + "..."
				}
			}

			symbols = append(symbols, Symbol{
				Name:      name,
				Kind:      kind,
				Signature: sig,
				FilePath:  filePath,
				Line:      int(child.StartPoint().Row) + 1,
			})
		}
	}

	return symbols
}

// JS/TS用の抽出関数（TypeScript型注釈対応版）
func extractJSFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	// 子ノードを直接探索（TypeScriptパーサーではフィールド名が異なる場合がある）
	var nameNode, paramsNode, returnTypeNode *sitter.Node
	isAsync := false

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "identifier":
			nameNode = child
		case "formal_parameters", "parameters":
			paramsNode = child
		case "type_annotation":
			returnTypeNode = child
		case "async":
			isAsync = true
		}
	}

	// フィールド名でも試す（一部のパーサー用）
	if nameNode == nil {
		nameNode = node.ChildByFieldName("name")
	}
	if paramsNode == nil {
		paramsNode = node.ChildByFieldName("parameters")
	}
	if returnTypeNode == nil {
		returnTypeNode = node.ChildByFieldName("return_type")
	}

	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	// TypeScript: パラメータの型注釈を含む完全なシグネチャを構築
	params := "()"
	if paramsNode != nil {
		params = extractTSParams(paramsNode, content)
	}

	prefix := "function"
	if isAsync {
		prefix = "async function"
	}

	sig := prefix + " " + name + params

	// TypeScript: 戻り値型注釈（type_annotationノードは ": Type" 形式）
	if returnTypeNode != nil {
		typeContent := returnTypeNode.Content(content)
		// 先頭の ":" を除去
		typeContent = strings.TrimSpace(strings.TrimPrefix(typeContent, ":"))
		sig += ": " + typeContent
	}

	// Issue #62: Hook判定（useXxx パターン）
	kind := "function"
	if isHookName(name) {
		kind = "hook"
	}

	return Symbol{
		Name:      name,
		Kind:      kind,
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// extractTSParams はTypeScriptパラメータから型注釈を含むシグネチャを抽出
func extractTSParams(paramsNode *sitter.Node, content []byte) string {
	// パラメータノードをそのまま使う（型注釈を含む）
	params := paramsNode.Content(content)
	// 改行を除去してコンパクトに
	params = strings.ReplaceAll(params, "\n", " ")
	params = strings.ReplaceAll(params, "\t", "")
	// 連続スペースを単一に
	for strings.Contains(params, "  ") {
		params = strings.ReplaceAll(params, "  ", " ")
	}
	return params
}

func extractJSClass(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "class",
		Signature: "class " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractJSMethod(node *sitter.Node, content []byte, filePath string) Symbol {
	// 子ノードを直接探索（TypeScriptパーサー対応）
	var nameNode, paramsNode, returnTypeNode *sitter.Node
	isAsync := false

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "property_identifier", "identifier":
			if nameNode == nil { // 最初のidentifierを名前として使用
				nameNode = child
			}
		case "formal_parameters", "parameters":
			paramsNode = child
		case "type_annotation":
			returnTypeNode = child
		case "async":
			isAsync = true
		}
	}

	// フィールド名でも試す
	if nameNode == nil {
		nameNode = node.ChildByFieldName("name")
	}
	if paramsNode == nil {
		paramsNode = node.ChildByFieldName("parameters")
	}
	if returnTypeNode == nil {
		returnTypeNode = node.ChildByFieldName("return_type")
	}

	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	// TypeScript: パラメータの型注釈を含む
	params := "()"
	if paramsNode != nil {
		params = extractTSParams(paramsNode, content)
	}

	sig := name + params
	if isAsync {
		sig = "async " + sig
	}

	// TypeScript: 戻り値型注釈
	if returnTypeNode != nil {
		typeContent := returnTypeNode.Content(content)
		typeContent = strings.TrimSpace(strings.TrimPrefix(typeContent, ":"))
		sig += ": " + typeContent
	}

	return Symbol{
		Name:      name,
		Kind:      "method",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractPyFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	paramsNode := node.ChildByFieldName("parameters")
	returnTypeNode := node.ChildByFieldName("return_type")

	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	// パラメータから型注釈を含むシグネチャを抽出
	params := "()"
	if paramsNode != nil {
		params = extractPyParams(paramsNode, content)
	}

	// async def 検出: 親ノードまたは前の兄弟ノードをチェック
	isAsync := isPythonAsyncFunction(node)

	prefix := "def"
	if isAsync {
		prefix = "async def"
	}

	sig := prefix + " " + name + params
	if returnTypeNode != nil {
		sig += " -> " + returnTypeNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// isPythonAsyncFunction はPython関数がasyncかどうかを判定
func isPythonAsyncFunction(node *sitter.Node) bool {
	// function_definition の前の兄弟ノードをチェック
	parent := node.Parent()
	if parent != nil {
		for i := 0; i < int(parent.ChildCount()); i++ {
			child := parent.Child(i)
			if child == node {
				// 自分自身の前をチェック
				if i > 0 {
					prev := parent.Child(i - 1)
					if prev != nil && prev.Type() == "async" {
						return true
					}
				}
				break
			}
		}
	}

	// ノード自身の子ノードをチェック（一部のパーサーバージョン用）
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "async" {
			return true
		}
	}

	return false
}

// extractPyParams はPythonパラメータから型注釈を含むシグネチャを抽出
func extractPyParams(paramsNode *sitter.Node, content []byte) string {
	params := paramsNode.Content(content)
	// 改行を除去してコンパクトに
	params = strings.ReplaceAll(params, "\n", " ")
	params = strings.ReplaceAll(params, "\t", "")
	// 連続スペースを単一に
	for strings.Contains(params, "  ") {
		params = strings.ReplaceAll(params, "  ", " ")
	}
	return params
}

func extractPyClass(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "class",
		Signature: "class " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== Rust ==================

func extractRustFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	// シグネチャは関数定義行全体（最初の { まで）
	startByte := node.StartByte()
	sig := ""
	for i := startByte; i < node.EndByte(); i++ {
		if content[i] == '{' {
			sig = strings.TrimSpace(string(content[startByte:i]))
			break
		}
	}
	if sig == "" {
		sig = "fn " + name + "(...)"
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractRustStruct(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "struct",
		Signature: "struct " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractRustImpl(node *sitter.Node, content []byte, filePath string) []Symbol {
	var symbols []Symbol

	// impl対象の型名を取得
	typeNode := node.ChildByFieldName("type")
	typeName := ""
	if typeNode != nil {
		typeName = typeNode.Content(content)
	}

	// impl ブロック内のメソッドを抽出
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "declaration_list" {
			for j := 0; j < int(child.ChildCount()); j++ {
				item := child.Child(j)
				if item.Type() == "function_item" {
					nameNode := item.ChildByFieldName("name")
					name := ""
					if nameNode != nil {
						name = nameNode.Content(content)
					}

					startByte := item.StartByte()
					sig := ""
					for k := startByte; k < item.EndByte(); k++ {
						if content[k] == '{' {
							sig = strings.TrimSpace(string(content[startByte:k]))
							break
						}
					}
					if sig == "" {
						sig = "fn " + name + "(...)"
					}

					symbols = append(symbols, Symbol{
						Name:      typeName + "::" + name,
						Kind:      "method",
						Signature: sig,
						FilePath:  filePath,
						Line:      int(item.StartPoint().Row) + 1,
					})
				}
			}
		}
	}

	return symbols
}

func extractRustTrait(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "trait",
		Signature: "trait " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractRustEnum(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "enum",
		Signature: "enum " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== Java ==================

func extractJavaClass(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "class",
		Signature: "class " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractJavaMethod(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	// シグネチャ抽出（最初の { まで）
	startByte := node.StartByte()
	sig := ""
	for i := startByte; i < node.EndByte(); i++ {
		if content[i] == '{' {
			sig = strings.TrimSpace(string(content[startByte:i]))
			break
		}
	}
	if sig == "" {
		sig = name + "(...)"
	}

	return Symbol{
		Name:      name,
		Kind:      "method",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractJavaInterface(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "interface",
		Signature: "interface " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== C/C++ ==================

func extractCFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	// C/C++では declarator から関数名を取得
	declaratorNode := node.ChildByFieldName("declarator")
	name := ""
	if declaratorNode != nil {
		// function_declarator の場合
		if declaratorNode.Type() == "function_declarator" {
			nameNode := declaratorNode.ChildByFieldName("declarator")
			if nameNode != nil {
				name = nameNode.Content(content)
			}
		} else {
			name = declaratorNode.Content(content)
		}
	}

	// シグネチャ抽出（最初の { まで）
	startByte := node.StartByte()
	sig := ""
	for i := startByte; i < node.EndByte(); i++ {
		if content[i] == '{' {
			sig = strings.TrimSpace(string(content[startByte:i]))
			break
		}
	}
	if sig == "" {
		sig = name + "(...)"
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== Ruby ==================

func extractRubyMethod(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "method",
		Signature: "def " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractRubyClass(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "class",
		Signature: "class " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractRubyModule(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "module",
		Signature: "module " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== Kotlin ==================

func extractKotlinFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	// シグネチャ抽出
	startByte := node.StartByte()
	sig := ""
	for i := startByte; i < node.EndByte(); i++ {
		if content[i] == '{' || content[i] == '=' {
			sig = strings.TrimSpace(string(content[startByte:i]))
			break
		}
	}
	if sig == "" {
		sig = "fun " + name + "(...)"
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractKotlinClass(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "class",
		Signature: "class " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractKotlinObject(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "object",
		Signature: "object " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== Swift ==================

func extractSwiftFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	// シグネチャ抽出
	startByte := node.StartByte()
	sig := ""
	for i := startByte; i < node.EndByte(); i++ {
		if content[i] == '{' {
			sig = strings.TrimSpace(string(content[startByte:i]))
			break
		}
	}
	if sig == "" {
		sig = "func " + name + "(...)"
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractSwiftClass(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "class",
		Signature: "class " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractSwiftStruct(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "struct",
		Signature: "struct " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractSwiftProtocol(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "protocol",
		Signature: "protocol " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== C# ==================

func extractCSharpClass(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "class",
		Signature: "class " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractCSharpMethod(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	// シグネチャ抽出
	startByte := node.StartByte()
	sig := ""
	for i := startByte; i < node.EndByte(); i++ {
		if content[i] == '{' {
			sig = strings.TrimSpace(string(content[startByte:i]))
			break
		}
	}
	if sig == "" {
		sig = name + "(...)"
	}

	return Symbol{
		Name:      name,
		Kind:      "method",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractCSharpInterface(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "interface",
		Signature: "interface " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractCSharpStruct(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "struct",
		Signature: "struct " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== Scala ==================

func extractScalaClass(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "class",
		Signature: "class " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractScalaFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	// シグネチャ抽出
	startByte := node.StartByte()
	sig := ""
	for i := startByte; i < node.EndByte(); i++ {
		if content[i] == '{' || content[i] == '=' {
			sig = strings.TrimSpace(string(content[startByte:i]))
			break
		}
	}
	if sig == "" {
		sig = "def " + name + "(...)"
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractScalaObject(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "object",
		Signature: "object " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractScalaTrait(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "trait",
		Signature: "trait " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== PHP ==================

func extractPHPClass(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	return Symbol{
		Name:      name,
		Kind:      "class",
		Signature: "class " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractPHPFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	nameNode := node.ChildByFieldName("name")
	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	// シグネチャ抽出
	startByte := node.StartByte()
	sig := ""
	for i := startByte; i < node.EndByte(); i++ {
		if content[i] == '{' {
			sig = strings.TrimSpace(string(content[startByte:i]))
			break
		}
	}
	if sig == "" {
		sig = "function " + name + "(...)"
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== CSS/SCSS (#59) ==================

func extractCSSRuleSet(node *sitter.Node, content []byte, filePath string) Symbol {
	// セレクタを取得
	var selector string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "selectors" {
			selector = child.Content(content)
			break
		}
	}

	// セレクタが取れなかった場合は最初の部分
	if selector == "" {
		selector = strings.TrimSpace(strings.Split(node.Content(content), "{")[0])
	}

	// 長すぎるセレクタは省略
	if len(selector) > 80 {
		selector = selector[:80] + "..."
	}

	kind := "selector"
	// セレクタタイプの判定
	if strings.HasPrefix(selector, "#") {
		kind = "id"
	} else if strings.HasPrefix(selector, ".") {
		kind = "class"
	} else if strings.HasPrefix(selector, "@") {
		kind = "at-rule"
	}

	return Symbol{
		Name:      selector,
		Kind:      kind,
		Signature: selector,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractCSSKeyframes(node *sitter.Node, content []byte, filePath string) Symbol {
	// @keyframes name を取得
	name := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "keyframes_name" {
			name = child.Content(content)
			break
		}
	}

	return Symbol{
		Name:      name,
		Kind:      "keyframes",
		Signature: "@keyframes " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractCSSMediaQuery(node *sitter.Node, content []byte, filePath string) Symbol {
	// @media クエリを取得
	nodeContent := node.Content(content)
	// 最初の { まで
	sig := strings.TrimSpace(strings.Split(nodeContent, "{")[0])
	if len(sig) > 100 {
		sig = sig[:100] + "..."
	}

	return Symbol{
		Name:      sig,
		Kind:      "media",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func isCSSCustomProperty(node *sitter.Node, content []byte) bool {
	// CSS カスタムプロパティ（--xxx）かどうか
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "property_name" {
			name := child.Content(content)
			return strings.HasPrefix(name, "--")
		}
	}
	return false
}

func extractCSSCustomProperty(node *sitter.Node, content []byte, filePath string) Symbol {
	var name, value string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "property_name":
			name = child.Content(content)
		case "plain_value", "integer_value", "color_value":
			value = child.Content(content)
		}
	}

	sig := name
	if value != "" {
		sig = name + ": " + value
	}

	return Symbol{
		Name:      name,
		Kind:      "custom-property",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== Bash (#60) ==================

func extractBashFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	name := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "word" {
			name = child.Content(content)
			break
		}
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: "function " + name + "()",
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== YAML (#60) ==================

func isTopLevelYAMLKey(node *sitter.Node) bool {
	// 親がdocument または stream の場合トップレベル
	parent := node.Parent()
	if parent == nil {
		return true
	}
	parentType := parent.Type()
	return parentType == "document" || parentType == "stream" || parentType == "block_mapping"
}

func extractYAMLKey(node *sitter.Node, content []byte, filePath string) Symbol {
	// block_mapping_pair の最初の子がキー
	key := ""
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "flow_node" || child.Type() == "block_node" || strings.Contains(child.Type(), "scalar") {
			key = child.Content(content)
			break
		}
	}

	return Symbol{
		Name:      key,
		Kind:      "key",
		Signature: key + ":",
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== TOML (#60) ==================

func extractTOMLTable(node *sitter.Node, content []byte, filePath string) Symbol {
	// [section.name] 形式
	nodeContent := strings.TrimSpace(node.Content(content))
	// 最初の行を取得
	lines := strings.Split(nodeContent, "\n")
	sig := lines[0]
	name := strings.Trim(strings.Trim(sig, "["), "]")

	return Symbol{
		Name:      name,
		Kind:      "table",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractTOMLTableArray(node *sitter.Node, content []byte, filePath string) Symbol {
	// [[array.name]] 形式
	nodeContent := strings.TrimSpace(node.Content(content))
	lines := strings.Split(nodeContent, "\n")
	sig := lines[0]
	name := strings.Trim(strings.Trim(sig, "["), "]")

	return Symbol{
		Name:      name,
		Kind:      "table-array",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== SQL (#60) ==================

func extractSQLCreateTable(node *sitter.Node, content []byte, filePath string) Symbol {
	// CREATE TABLE name を抽出
	nodeContent := node.Content(content)
	sig := strings.TrimSpace(strings.Split(nodeContent, "(")[0])
	if len(sig) > 100 {
		sig = sig[:100] + "..."
	}

	// テーブル名を抽出
	name := ""
	parts := strings.Fields(sig)
	for i, p := range parts {
		if strings.ToUpper(p) == "TABLE" && i+1 < len(parts) {
			name = parts[i+1]
			break
		}
	}

	return Symbol{
		Name:      name,
		Kind:      "table",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractSQLCreateFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	nodeContent := node.Content(content)
	// 最初の行を取得
	lines := strings.Split(nodeContent, "\n")
	sig := strings.TrimSpace(lines[0])
	if len(sig) > 100 {
		sig = sig[:100] + "..."
	}

	// 関数名を抽出
	name := ""
	parts := strings.Fields(sig)
	for i, p := range parts {
		if strings.ToUpper(p) == "FUNCTION" && i+1 < len(parts) {
			name = strings.Split(parts[i+1], "(")[0]
			break
		}
	}

	return Symbol{
		Name:      name,
		Kind:      "function",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractSQLCreateView(node *sitter.Node, content []byte, filePath string) Symbol {
	nodeContent := node.Content(content)
	// 最初の行を取得
	lines := strings.Split(nodeContent, "\n")
	sig := strings.TrimSpace(lines[0])
	if len(sig) > 100 {
		sig = sig[:100] + "..."
	}

	// ビュー名を抽出
	name := ""
	parts := strings.Fields(sig)
	for i, p := range parts {
		if strings.ToUpper(p) == "VIEW" && i+1 < len(parts) {
			name = parts[i+1]
			break
		}
	}

	return Symbol{
		Name:      name,
		Kind:      "view",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractSQLCreateIndex(node *sitter.Node, content []byte, filePath string) Symbol {
	nodeContent := node.Content(content)
	sig := strings.TrimSpace(strings.Split(nodeContent, "\n")[0])
	if len(sig) > 100 {
		sig = sig[:100] + "..."
	}

	// インデックス名を抽出
	name := ""
	parts := strings.Fields(sig)
	for i, p := range parts {
		if strings.ToUpper(p) == "INDEX" && i+1 < len(parts) {
			name = parts[i+1]
			break
		}
	}

	return Symbol{
		Name:      name,
		Kind:      "index",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== Dockerfile (#60) ==================

func extractDockerFrom(node *sitter.Node, content []byte, filePath string) Symbol {
	nodeContent := strings.TrimSpace(node.Content(content))
	// FROM image:tag AS alias
	name := ""
	parts := strings.Fields(nodeContent)
	if len(parts) >= 2 {
		name = parts[1]
	}

	return Symbol{
		Name:      name,
		Kind:      "from",
		Signature: nodeContent,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractDockerRun(node *sitter.Node, content []byte, filePath string) Symbol {
	nodeContent := strings.TrimSpace(node.Content(content))
	// RUN コマンド（長い場合省略）
	sig := nodeContent
	if len(sig) > 80 {
		sig = sig[:80] + "..."
	}

	return Symbol{
		Name:      "RUN",
		Kind:      "run",
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

func extractDockerCmd(node *sitter.Node, content []byte, filePath string) Symbol {
	nodeContent := strings.TrimSpace(node.Content(content))
	// CMD または ENTRYPOINT
	kind := "cmd"
	if strings.HasPrefix(strings.ToUpper(nodeContent), "ENTRYPOINT") {
		kind = "entrypoint"
	}

	return Symbol{
		Name:      strings.ToUpper(kind),
		Kind:      kind,
		Signature: nodeContent,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== Markdown (#60) ==================

func extractMarkdownHeading(node *sitter.Node, content []byte, filePath string) Symbol {
	nodeContent := strings.TrimSpace(node.Content(content))
	// # heading → level 1, ## heading → level 2, etc.
	level := 0
	for _, c := range nodeContent {
		if c == '#' {
			level++
		} else {
			break
		}
	}

	heading := strings.TrimLeft(nodeContent, "# ")
	if len(heading) > 80 {
		heading = heading[:80] + "..."
	}

	kind := "h1"
	if level >= 1 && level <= 6 {
		kind = "h" + string(rune('0'+level))
	}

	return Symbol{
		Name:      heading,
		Kind:      kind,
		Signature: nodeContent,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== Makefile (#60) - 正規表現ベース ==================

// extractConfigFileSymbols は正規表現ベースでシンボルを抽出（Makefile, Jenkinsfile等）
func extractConfigFileSymbols(filePath string) (*FileSymbols, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	baseName := filepath.Base(filePath)
	ext := strings.ToLower(filepath.Ext(filePath))

	var symbols []Symbol
	scanner := bufio.NewScanner(file)
	lineNum := 0

	// Makefile用正規表現（ターゲット定義: name:）
	makeTargetRe := regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_-]*)\s*:`)
	// Jenkinsfile用正規表現（stage('name')）
	jenkinsStageRe := regexp.MustCompile(`stage\s*\(\s*['"]([^'"]+)['"]\s*\)`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		switch {
		case baseName == "Makefile" || ext == ".mk":
			if matches := makeTargetRe.FindStringSubmatch(line); len(matches) > 1 {
				// .PHONY などの特殊ターゲットをスキップ
				target := matches[1]
				if !strings.HasPrefix(target, ".") {
					symbols = append(symbols, Symbol{
						Name:      target,
						Kind:      "target",
						Signature: target + ":",
						FilePath:  filePath,
						Line:      lineNum,
					})
				}
			}
		case baseName == "Jenkinsfile":
			if matches := jenkinsStageRe.FindStringSubmatch(line); len(matches) > 1 {
				symbols = append(symbols, Symbol{
					Name:      matches[1],
					Kind:      "stage",
					Signature: "stage('" + matches[1] + "')",
					FilePath:  filePath,
					Line:      lineNum,
				})
			}
		}
	}

	return &FileSymbols{
		Path:    filePath,
		Symbols: symbols,
	}, nil
}

// ================== Issue #62: Arrow Function Components / Hooks ==================

// extractArrowFunctionOrHook は Arrow Function Component または Hook を抽出
// const Header = () => {} → component
// const useAuth = () => {} → hook
func extractArrowFunctionOrHook(node *sitter.Node, content []byte, filePath string) *Symbol {
	// lexical_declaration / variable_declaration の子ノードを探索
	// 構造: lexical_declaration -> variable_declarator -> (name, arrow_function)
	var declarator *sitter.Node

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		if child.Type() == "variable_declarator" {
			declarator = child
			break
		}
	}

	if declarator == nil {
		return nil
	}

	// variable_declarator から name と value を取得
	var nameNode, valueNode, typeAnnotation *sitter.Node

	for i := 0; i < int(declarator.ChildCount()); i++ {
		child := declarator.Child(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "identifier":
			nameNode = child
		case "arrow_function":
			valueNode = child
		case "type_annotation":
			typeAnnotation = child
		}
	}

	// Arrow Function でなければスキップ
	if valueNode == nil {
		return nil
	}

	name := ""
	if nameNode != nil {
		name = nameNode.Content(content)
	}

	// 名前がなければスキップ
	if name == "" {
		return nil
	}

	// Kind を判定
	kind := "function"
	if isHookName(name) {
		kind = "hook"
	} else if isPascalCase(name) {
		kind = "component"
	} else {
		// 通常の変数（小文字始まり、hookでもない）はスキップ
		// 例: const handler = () => {} はスキップ
		return nil
	}

	// シグネチャを構築
	// const Header: React.FC<Props> = () => ...
	declKeyword := "const"
	if node.Type() == "variable_declaration" {
		// var を使用している場合
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child != nil && child.Type() == "var" {
				declKeyword = "var"
				break
			}
		}
	}

	// パラメータを取得
	params := "()"
	for i := 0; i < int(valueNode.ChildCount()); i++ {
		child := valueNode.Child(i)
		if child != nil && (child.Type() == "formal_parameters" || child.Type() == "parameters") {
			params = extractTSParams(child, content)
			break
		}
	}

	sig := declKeyword + " " + name
	if typeAnnotation != nil {
		sig += typeAnnotation.Content(content)
	}
	sig += " = " + params + " => ..."

	return &Symbol{
		Name:      name,
		Kind:      kind,
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== Issue #64: HTML id/class属性抽出 ==================

// semanticHTMLTags はセマンティックHTML要素のリスト
var semanticHTMLTags = map[string]bool{
	"header":  true,
	"footer":  true,
	"nav":     true,
	"main":    true,
	"aside":   true,
	"article": true,
	"section": true,
	"figure":  true,
	"form":    true,
}

// extractHTMLElement は HTML 要素から id/class 属性を抽出
func extractHTMLElement(node *sitter.Node, content []byte, filePath string) []Symbol {
	var symbols []Symbol

	// start_tag を探す
	var startTag *sitter.Node
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "start_tag" {
			startTag = child
			break
		}
	}

	if startTag == nil {
		return symbols
	}

	// タグ名と属性を抽出
	var tagName string
	var idValue, classValue string

	for i := 0; i < int(startTag.ChildCount()); i++ {
		child := startTag.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "tag_name":
			tagName = child.Content(content)
		case "attribute":
			attrName, attrValue := extractHTMLAttribute(child, content)
			switch attrName {
			case "id":
				idValue = attrValue
			case "class":
				classValue = attrValue
			}
		}
	}

	line := int(node.StartPoint().Row) + 1

	// id属性があれば抽出
	if idValue != "" {
		symbols = append(symbols, Symbol{
			Name:      "#" + idValue,
			Kind:      "id",
			Signature: "<" + tagName + " id=\"" + idValue + "\">",
			FilePath:  filePath,
			Line:      line,
		})
	}

	// class属性があれば各クラスを個別に抽出
	if classValue != "" {
		classes := strings.Fields(classValue)
		for _, class := range classes {
			symbols = append(symbols, Symbol{
				Name:      "." + class,
				Kind:      "class",
				Signature: "<" + tagName + " class=\"" + class + "\">",
				FilePath:  filePath,
				Line:      line,
			})
		}
	}

	// セマンティックHTML要素（id/classがない場合でも抽出）
	if semanticHTMLTags[tagName] && idValue == "" && classValue == "" {
		symbols = append(symbols, Symbol{
			Name:      "<" + tagName + ">",
			Kind:      "semantic",
			Signature: "<" + tagName + ">",
			FilePath:  filePath,
			Line:      line,
		})
	}

	return symbols
}

// extractHTMLAttribute は属性ノードから名前と値を抽出
func extractHTMLAttribute(node *sitter.Node, content []byte) (name, value string) {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "attribute_name":
			name = child.Content(content)
		case "quoted_attribute_value":
			// quoted_attribute_value の中から attribute_value を探す
			for j := 0; j < int(child.ChildCount()); j++ {
				grandchild := child.Child(j)
				if grandchild != nil && grandchild.Type() == "attribute_value" {
					value = grandchild.Content(content)
					break
				}
			}
		case "attribute_value":
			// クォートなしの属性値
			value = child.Content(content)
		}
	}
	return
}

// ================== Issue #63: Vue/Svelte SFC ==================

// SFCSection は SFC の各セクション情報
type SFCSection struct {
	Tag       string // "script", "style", "template"
	Content   string
	StartLine int
	Lang      string // "ts", "scss" など
}

// extractSFCSymbols は Vue/Svelte SFC からシンボルを抽出
func extractSFCSymbols(filePath string) (*FileSymbols, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	sections := parseSFCSections(string(content))

	var symbols []Symbol

	for _, section := range sections {
		switch section.Tag {
		case "script":
			// <script> セクションを JS/TS パーサーで解析
			scriptSymbols := extractSFCScriptSymbols(section, filePath, ext)
			symbols = append(symbols, scriptSymbols...)

		case "style":
			// <style> セクションを CSS パーサーで解析
			styleSymbols := extractSFCStyleSymbols(section, filePath)
			symbols = append(symbols, styleSymbols...)
		}
	}

	return &FileSymbols{
		Path:    filePath,
		Symbols: symbols,
	}, nil
}

// parseSFCSections は SFC の内容をセクションごとに分割
func parseSFCSections(content string) []SFCSection {
	var sections []SFCSection

	// <script>, <style>, <template> タグを正規表現で抽出
	// <script lang="ts" setup> のような属性も対応
	scriptRe := regexp.MustCompile(`(?is)<script([^>]*)>(.*?)</script>`)
	styleRe := regexp.MustCompile(`(?is)<style([^>]*)>(.*?)</style>`)

	// Script セクション
	for _, match := range scriptRe.FindAllStringSubmatchIndex(content, -1) {
		attrStart, attrEnd := match[2], match[3]
		contentStart, contentEnd := match[4], match[5]

		attrs := ""
		if attrStart >= 0 && attrEnd >= 0 {
			attrs = content[attrStart:attrEnd]
		}

		sectionContent := ""
		if contentStart >= 0 && contentEnd >= 0 {
			sectionContent = content[contentStart:contentEnd]
		}

		// 開始行を計算
		startLine := strings.Count(content[:contentStart], "\n") + 1

		// lang属性を抽出
		lang := extractSFCLangAttr(attrs, "js")

		sections = append(sections, SFCSection{
			Tag:       "script",
			Content:   sectionContent,
			StartLine: startLine,
			Lang:      lang,
		})
	}

	// Style セクション
	for _, match := range styleRe.FindAllStringSubmatchIndex(content, -1) {
		attrStart, attrEnd := match[2], match[3]
		contentStart, contentEnd := match[4], match[5]

		attrs := ""
		if attrStart >= 0 && attrEnd >= 0 {
			attrs = content[attrStart:attrEnd]
		}

		sectionContent := ""
		if contentStart >= 0 && contentEnd >= 0 {
			sectionContent = content[contentStart:contentEnd]
		}

		startLine := strings.Count(content[:contentStart], "\n") + 1
		lang := extractSFCLangAttr(attrs, "css")

		sections = append(sections, SFCSection{
			Tag:       "style",
			Content:   sectionContent,
			StartLine: startLine,
			Lang:      lang,
		})
	}

	return sections
}

// extractSFCLangAttr は lang="xxx" 属性を抽出
func extractSFCLangAttr(attrs, defaultLang string) string {
	langRe := regexp.MustCompile(`lang\s*=\s*["']([^"']+)["']`)
	if match := langRe.FindStringSubmatch(attrs); len(match) > 1 {
		return match[1]
	}
	return defaultLang
}

// extractSFCScriptSymbols は <script> セクションからシンボルを抽出
func extractSFCScriptSymbols(section SFCSection, filePath, sfcExt string) []Symbol {
	var symbols []Symbol

	// 言語に応じたパーサーを選択
	var lang *sitter.Language
	switch section.Lang {
	case "ts", "typescript":
		lang = SupportedLanguages[".ts"]
	default:
		lang = SupportedLanguages[".js"]
	}

	if lang == nil {
		return symbols
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lang)

	content := []byte(section.Content)
	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return symbols
	}
	defer tree.Close()

	// スクリプト用の仮想ファイルパス（.ts/.js として解析）
	virtualExt := ".js"
	if section.Lang == "ts" || section.Lang == "typescript" {
		virtualExt = ".ts"
	}
	virtualPath := filePath + virtualExt

	root := tree.RootNode()
	rawSymbols := extractFromNode(root, content, virtualPath)

	// 行番号をオフセットで調整し、実際のファイルパスに戻す
	for _, sym := range rawSymbols {
		sym.Line += section.StartLine - 1
		sym.FilePath = filePath

		// Svelte 固有: $: ラベルはリアクティブステートメント
		if sfcExt == ".svelte" && strings.HasPrefix(sym.Name, "$:") {
			sym.Kind = "reactive"
		}

		symbols = append(symbols, sym)
	}

	return symbols
}

// extractSFCStyleSymbols は <style> セクションからシンボルを抽出
func extractSFCStyleSymbols(section SFCSection, filePath string) []Symbol {
	var symbols []Symbol

	lang := SupportedLanguages[".css"]
	if lang == nil {
		return symbols
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lang)

	content := []byte(section.Content)
	tree, err := parser.ParseCtx(context.Background(), nil, content)
	if err != nil {
		return symbols
	}
	defer tree.Close()

	virtualPath := filePath + ".css"
	root := tree.RootNode()
	rawSymbols := extractFromNode(root, content, virtualPath)

	// 行番号をオフセットで調整
	for _, sym := range rawSymbols {
		sym.Line += section.StartLine - 1
		sym.FilePath = filePath
		symbols = append(symbols, sym)
	}

	return symbols
}
