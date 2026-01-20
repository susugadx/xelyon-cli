//go:build !norepomap
// +build !norepomap

package repomap

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

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

// extractJSFunction はJS/TS用の抽出関数（TypeScript型注釈対応版）
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
	var kind string
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
