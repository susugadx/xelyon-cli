package ast

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// SymbolKind はシンボルの種別を表す。
type SymbolKind string

const (
	SymbolFunction  SymbolKind = "function"
	SymbolMethod    SymbolKind = "method"
	SymbolType      SymbolKind = "type"
	SymbolInterface SymbolKind = "interface"
	SymbolStruct    SymbolKind = "struct"
	SymbolConst     SymbolKind = "const"
	SymbolVar       SymbolKind = "var"
	SymbolClass     SymbolKind = "class"
	SymbolEnum      SymbolKind = "enum"
	SymbolTrait     SymbolKind = "trait"
	SymbolImpl      SymbolKind = "impl"
	SymbolUnknown   SymbolKind = "unknown"
)

// MatchClass は検索マッチの分類を表す。
type MatchClass string

const (
	ClassDef     MatchClass = "def"
	ClassCall    MatchClass = "call"
	ClassRef     MatchClass = "ref"
	ClassImport  MatchClass = "import"
	ClassComment MatchClass = "comment"
	ClassString  MatchClass = "string"
	ClassUnknown MatchClass = "unknown"
)

// Symbol はコード内の定義シンボルを表す。
type Symbol struct {
	Name      string
	Kind      SymbolKind
	Signature string
	Line      int
	EndLine   int
	Exported  bool
}

// MatchInfo は特定行のマッチ分類情報を表す。
type MatchInfo struct {
	Class    MatchClass
	Scope    string
	NodeType string
}

// IsSupportedFile は AST 解析に対応しているファイルかを返す。
// Phase 1: Go のみ。Phase 2 で grammars.DetectLanguage に差し替え予定。
func IsSupportedFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".go")
}

// ParseFile はファイルをパースして AST ツリーとソースコードを返す。
func ParseFile(path string) (*gotreesitter.Tree, []byte, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", path, err)
	}
	return ParseBytes(path, src)
}

// ParseBytes はソースコードバイト列をパースして AST ツリーとソースコードを返す。
func ParseBytes(path string, src []byte) (*gotreesitter.Tree, []byte, error) {
	if !IsSupportedFile(path) {
		return nil, nil, fmt.Errorf("unsupported language: %s", path)
	}

	parser := gotreesitter.NewParser(grammars.GoLanguage())
	tree, err := parser.Parse(src)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return tree, src, nil
}

// ExtractSymbols はファイルから定義シンボルを抽出する。
func ExtractSymbols(path string) ([]Symbol, error) {
	tree, src, err := ParseFile(path)
	if err != nil {
		return nil, err
	}
	return extractSymbolsFromTree(tree, src)
}

// ExtractSymbolsFromBytes はソースコードバイト列から定義シンボルを抽出する。
func ExtractSymbolsFromBytes(path string, src []byte) ([]Symbol, error) {
	tree, src, err := ParseBytes(path, src)
	if err != nil {
		return nil, err
	}
	return extractSymbolsFromTree(tree, src)
}

// ClassifyLine は指定行の targetName マッチを AST ベースで分類する。
func ClassifyLine(path string, src []byte, line int, targetName string) (*MatchInfo, error) {
	tree, src, err := ParseBytes(path, src)
	if err != nil {
		return nil, err
	}
	return classifyLineInTree(tree, src, line, targetName)
}

func extractSymbolsFromTree(tree *gotreesitter.Tree, src []byte) ([]Symbol, error) {
	query, err := getGoSymbolQuery()
	if err != nil {
		return nil, err
	}

	lang := tree.Language()
	cursor := query.Exec(tree.RootNode(), lang, src)
	var symbols []Symbol
	seen := make(map[string]int)

	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}

		var defNode *gotreesitter.Node
		var typeBody *gotreesitter.Node
		nameNodes := make([]*gotreesitter.Node, 0, len(match.Captures))
		for _, capture := range match.Captures {
			switch capture.Name {
			case "def":
				if defNode == nil {
					defNode = capture.Node
				}
			case "type_body":
				if typeBody == nil {
					typeBody = capture.Node
				}
			case "name":
				if capture.Node != nil {
					nameNodes = append(nameNodes, capture.Node)
				}
			}
		}
		if defNode == nil || len(nameNodes) == 0 {
			continue
		}

		kind := symbolKindForNode(defNode, typeBody, lang)
		signature := extractSignature(defNode, src, lang)
		line := int(defNode.StartPoint().Row) + 1
		endLine := int(defNode.EndPoint().Row) + 1
		for _, nameNode := range nameNodes {
			name := strings.TrimSpace(nameNode.Text(src))
			if name == "" {
				continue
			}
			symbol := Symbol{
				Name:      name,
				Kind:      kind,
				Signature: signature,
				Line:      line,
				EndLine:   endLine,
				Exported:  isExportedIdentifier(name),
			}
			appendSymbol(&symbols, seen, symbol)
		}
	}

	collectTypeAliasSymbols(tree.RootNode(), src, lang, &symbols, seen)

	sort.SliceStable(symbols, func(i, j int) bool {
		if symbols[i].Line != symbols[j].Line {
			return symbols[i].Line < symbols[j].Line
		}
		if symbols[i].EndLine != symbols[j].EndLine {
			return symbols[i].EndLine < symbols[j].EndLine
		}
		return symbols[i].Name < symbols[j].Name
	})
	return symbols, nil
}

func classifyLineInTree(tree *gotreesitter.Tree, src []byte, line int, targetName string) (*MatchInfo, error) {
	if line <= 0 {
		return nil, fmt.Errorf("line must be >= 1: %d", line)
	}
	if targetName == "" {
		return nil, fmt.Errorf("targetName is required")
	}

	startByte, endByte, ok := lineByteRange(src, line)
	if !ok {
		return &MatchInfo{Class: ClassUnknown, Scope: "package-level"}, nil
	}

	lang := tree.Language()
	root := tree.RootNode()
	occurrences := findIdentifierOccurrences(src[startByte:endByte], targetName)
	for _, offset := range occurrences {
		absStart := startByte + offset
		absEnd := absStart + uint32(len(targetName))
		node := root.DescendantForByteRange(absStart, absEnd)
		if node == nil {
			continue
		}
		class := classifyNode(node, absStart, absEnd, lang)
		return &MatchInfo{
			Class:    class,
			Scope:    findEnclosingScope(node, src, lang),
			NodeType: node.Type(lang),
		}, nil
	}

	return &MatchInfo{Class: ClassUnknown, Scope: "package-level"}, nil
}

func symbolKindForNode(defNode, typeBody *gotreesitter.Node, lang *gotreesitter.Language) SymbolKind {
	switch defNode.Type(lang) {
	case "function_declaration":
		return SymbolFunction
	case "method_declaration":
		return SymbolMethod
	case "type_alias":
		return SymbolType
	case "const_spec":
		return SymbolConst
	case "var_spec":
		return SymbolVar
	case "type_spec":
		if typeBody == nil {
			return SymbolType
		}
		switch typeBody.Type(lang) {
		case "struct_type":
			return SymbolStruct
		case "interface_type":
			return SymbolInterface
		default:
			return SymbolType
		}
	default:
		return SymbolUnknown
	}
}

func extractSignature(defNode *gotreesitter.Node, src []byte, lang *gotreesitter.Language) string {
	if defNode == nil {
		return ""
	}

	switch defNode.Type(lang) {
	case "function_declaration", "method_declaration":
		if body := defNode.ChildByFieldName("body", lang); body != nil {
			return strings.TrimSpace(string(src[defNode.StartByte():body.StartByte()]))
		}
	case "type_alias":
		return "type " + strings.TrimSpace(defNode.Text(src))
	case "type_spec":
		return "type " + strings.TrimSpace(defNode.Text(src))
	case "const_spec":
		return "const " + strings.TrimSpace(defNode.Text(src))
	case "var_spec":
		return "var " + strings.TrimSpace(defNode.Text(src))
	}

	return strings.TrimSpace(defNode.Text(src))
}

func classifyNode(node *gotreesitter.Node, startByte, endByte uint32, lang *gotreesitter.Language) MatchClass {
	if hasAncestorType(node, lang, "import_spec", "import_declaration") {
		return ClassImport
	}

	for current := node; current != nil; current = current.Parent() {
		typ := current.Type(lang)
		switch typ {
		case "comment", "line_comment", "block_comment":
			return ClassComment
		case "interpreted_string_literal", "raw_string_literal", "string", "template_string":
			return ClassString
		}

		switch typ {
		case "function_declaration", "method_declaration", "type_alias", "type_spec", "const_spec", "var_spec":
			if fieldContainsRange(current, "name", startByte, endByte, lang) {
				return ClassDef
			}
		case "call_expression":
			if fieldContainsRange(current, "function", startByte, endByte, lang) {
				return ClassCall
			}
		}
	}

	return ClassRef
}

func hasAncestorType(node *gotreesitter.Node, lang *gotreesitter.Language, types ...string) bool {
	for current := node; current != nil; current = current.Parent() {
		typ := current.Type(lang)
		for _, want := range types {
			if typ == want {
				return true
			}
		}
	}
	return false
}

func findEnclosingScope(node *gotreesitter.Node, src []byte, lang *gotreesitter.Language) string {
	for current := node; current != nil; current = current.Parent() {
		switch current.Type(lang) {
		case "function_declaration":
			if nameNode := current.ChildByFieldName("name", lang); nameNode != nil {
				return "func " + strings.TrimSpace(nameNode.Text(src))
			}
		case "method_declaration":
			if nameNode := current.ChildByFieldName("name", lang); nameNode != nil {
				return "method " + strings.TrimSpace(nameNode.Text(src))
			}
		}
	}
	return "package-level"
}

func fieldContainsRange(node *gotreesitter.Node, fieldName string, startByte, endByte uint32, lang *gotreesitter.Language) bool {
	if node == nil {
		return false
	}
	fieldNode := node.ChildByFieldName(fieldName, lang)
	if fieldNode == nil {
		return false
	}
	if startByte < fieldNode.StartByte() || fieldNode.EndByte() < endByte {
		return false
	}
	if fieldNode.Type(lang) != "selector_expression" {
		return true
	}
	for i := 0; i < fieldNode.NamedChildCount(); i++ {
		child := fieldNode.NamedChild(i)
		if child == nil {
			continue
		}
		typ := child.Type(lang)
		if typ != "identifier" && typ != "field_identifier" {
			continue
		}
		if child.StartByte() <= startByte && endByte <= child.EndByte() {
			return true
		}
	}
	return false
}

func isExportedIdentifier(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError {
		return false
	}
	return unicode.IsUpper(r)
}

func collectTypeAliasSymbols(node *gotreesitter.Node, src []byte, lang *gotreesitter.Language, symbols *[]Symbol, seen map[string]int) {
	if node == nil {
		return
	}

	stack := []*gotreesitter.Node{node}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == nil {
			continue
		}

		if current.Type(lang) == "type_alias" {
			nameNode := current.NamedChild(0)
			if nameNode != nil {
				name := strings.TrimSpace(nameNode.Text(src))
				if name != "" {
					symbol := Symbol{
						Name:      name,
						Kind:      SymbolType,
						Signature: extractSignature(current, src, lang),
						Line:      int(current.StartPoint().Row) + 1,
						EndLine:   int(current.EndPoint().Row) + 1,
						Exported:  isExportedIdentifier(name),
					}
					appendSymbol(symbols, seen, symbol)
				}
			}
		}

		for i := int(current.NamedChildCount()) - 1; i >= 0; i-- {
			child := current.NamedChild(i)
			if child != nil {
				stack = append(stack, child)
			}
		}
	}
}

func appendSymbol(symbols *[]Symbol, seen map[string]int, symbol Symbol) {
	key := fmt.Sprintf("%d:%d:%s", symbol.Line, symbol.EndLine, symbol.Name)
	if idx, ok := seen[key]; ok {
		if (*symbols)[idx].Kind == SymbolType && symbol.Kind != SymbolType {
			(*symbols)[idx] = symbol
		}
		return
	}
	seen[key] = len(*symbols)
	*symbols = append(*symbols, symbol)
}

// lineByteRange は 1 始まりの行番号に対応するバイト範囲を返す。
// 注意: O(n) スキャン。バッチ分類時は行オフセットテーブルの事前構築を検討。
func lineByteRange(src []byte, line int) (uint32, uint32, bool) {
	if line <= 0 {
		return 0, 0, false
	}
	currentLine := 1
	start := 0
	for i, b := range src {
		if currentLine == line {
			end := i
			for end < len(src) && src[end] != '\n' {
				end++
			}
			return uint32(start), uint32(end), true
		}
		if b == '\n' {
			currentLine++
			start = i + 1
		}
	}
	if currentLine == line {
		return uint32(start), uint32(len(src)), true
	}
	return 0, 0, false
}

func findIdentifierOccurrences(line []byte, target string) []uint32 {
	if len(line) == 0 || target == "" {
		return nil
	}

	lineStr := string(line)
	var offsets []uint32
	searchFrom := 0
	for {
		idx := strings.Index(lineStr[searchFrom:], target)
		if idx < 0 {
			break
		}
		idx += searchFrom
		start := idx
		end := idx + len(target)
		if isIdentifierBoundary(lineStr, start-1) && isIdentifierBoundary(lineStr, end) {
			offsets = append(offsets, uint32(start))
		}
		searchFrom = idx + len(target)
	}
	return offsets
}

func isIdentifierBoundary(s string, pos int) bool {
	if pos < 0 || pos >= len(s) {
		return true
	}
	b := s[pos]
	return b != '_' && (b < '0' || b > '9') && (b < 'A' || b > 'Z') && (b < 'a' || b > 'z')
}
