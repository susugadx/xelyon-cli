//go:build !norepomap
// +build !norepomap

package repomap

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// ================== Elixir ==================

// extractElixirCall は Elixir の call ノードからモジュール/関数を抽出
func extractElixirCall(node *sitter.Node, content []byte, filePath string) *Symbol {
	// 最初の identifier を取得（defmodule, def, defp など）
	var keyword string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "identifier" {
			keyword = child.Content(content)
			break
		}
	}

	switch keyword {
	case "defmodule":
		return extractElixirModule(node, content, filePath)
	case "def":
		return extractElixirFunction(node, content, filePath, false)
	case "defp":
		return extractElixirFunction(node, content, filePath, true)
	}

	return nil
}

// extractElixirModule は Elixir モジュールを抽出
func extractElixirModule(node *sitter.Node, content []byte, filePath string) *Symbol {
	// arguments ノードから alias（モジュール名）を取得
	argsNode := findChildByType(node, "arguments")
	if argsNode == nil {
		return nil
	}

	aliasNode := findChildByType(argsNode, "alias")
	if aliasNode == nil {
		return nil
	}

	name := aliasNode.Content(content)
	return &Symbol{
		Name:      name,
		Kind:      "module",
		Signature: "defmodule " + name,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// extractElixirFunction は Elixir 関数を抽出
func extractElixirFunction(node *sitter.Node, content []byte, filePath string, isPrivate bool) *Symbol {
	// arguments ノードから関数呼び出し（名前とパラメータ）を取得
	argsNode := findChildByType(node, "arguments")
	if argsNode == nil {
		return nil
	}

	// 関数名と引数を取得
	var name string
	var params string
	for i := 0; i < int(argsNode.ChildCount()); i++ {
		child := argsNode.Child(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "call":
			// def hello(name) のパターン
			for j := 0; j < int(child.ChildCount()); j++ {
				subChild := child.Child(j)
				if subChild == nil {
					continue
				}
				if subChild.Type() == "identifier" && name == "" {
					name = subChild.Content(content)
				}
				if subChild.Type() == "arguments" {
					params = subChild.Content(content)
				}
			}
		case "identifier":
			// def private_func のパターン（引数なし）
			if name == "" {
				name = child.Content(content)
			}
		}
	}

	if name == "" {
		return nil
	}

	kind := "function"
	keyword := "def"
	if isPrivate {
		kind = "private_function"
		keyword = "defp"
	}

	sig := keyword + " " + name
	if params != "" {
		sig += params
	}

	return &Symbol{
		Name:      name,
		Kind:      kind,
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}

// ================== Lua ==================

// extractLuaFunction は Lua 関数を抽出
func extractLuaFunction(node *sitter.Node, content []byte, filePath string) Symbol {
	// local キーワードの有無を確認
	isLocal := false
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && child.Type() == "local" {
			isLocal = true
			break
		}
	}

	// 関数名を取得
	var name string
	var params string

	// function_name ノードを探す（グローバル関数やメソッド）
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "function_name":
			// M.moduleFunc のような形式
			name = child.Content(content)
		case "identifier":
			// local function greet のような形式
			if name == "" {
				name = child.Content(content)
			}
		case "parameter_list":
			params = child.Content(content)
		}
	}

	if name == "" {
		return Symbol{}
	}

	kind := "function"
	sig := "function " + name
	if params != "" {
		sig += "(" + params + ")"
	} else {
		sig += "()"
	}
	if isLocal {
		sig = "local " + sig
	}

	return Symbol{
		Name:      name,
		Kind:      kind,
		Signature: sig,
		FilePath:  filePath,
		Line:      int(node.StartPoint().Row) + 1,
	}
}
