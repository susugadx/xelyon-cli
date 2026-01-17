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

	// Python / C / PHP
	case "function_definition":
		if isCFamily(ext) {
			symbols = append(symbols, extractCFunction(node, content, filePath))
		} else if ext == ".php" {
			symbols = append(symbols, extractPHPFunction(node, content, filePath))
		} else if ext == ".scala" {
			symbols = append(symbols, extractScalaFunction(node, content, filePath))
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

	return Symbol{
		Name:      name,
		Kind:      "function",
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
