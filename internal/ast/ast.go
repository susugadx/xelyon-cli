package ast

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
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
	Class        MatchClass
	Scope        string
	NodeType     string
	SelectorKind string
	ReceiverType string
}

// ParsedFile はパース済みファイルの情報を保持する。再利用のためにキャッシュできる。
type ParsedFile struct {
	tree *gotreesitter.Tree
	src  []byte
}

// SyntaxError は AST 構文検証で検出した構文エラー情報を表す。
type SyntaxError struct {
	Line    int
	Column  int
	Message string
}

var goTreeSitterMu sync.Mutex

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

	goTreeSitterMu.Lock()
	defer goTreeSitterMu.Unlock()

	parser := gotreesitter.NewParser(grammars.GoLanguage())
	tree, err := parser.Parse(src)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return tree, src, nil
}

// ValidateSyntax はソースコードを AST パースし、構文エラー情報を返す。
// 構文エラーがない場合と対応外ファイルの場合は nil を返す。
func ValidateSyntax(path string, src []byte) []SyntaxError {
	if !IsSupportedFile(path) {
		return nil
	}

	goTreeSitterMu.Lock()
	defer goTreeSitterMu.Unlock()

	parser := gotreesitter.NewParser(grammars.GoLanguage())
	tree, err := parser.Parse(src)
	if err != nil {
		return []SyntaxError{{Message: fmt.Sprintf("parse error: %v", err)}}
	}

	root := tree.RootNode()
	if root == nil || !root.HasError() {
		return nil
	}

	errors := collectSyntaxErrors(root, tree.Language(), src)
	if len(errors) > 0 {
		return errors
	}

	return []SyntaxError{{
		Line:    1,
		Column:  1,
		Message: "syntax error",
	}}
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

// ParseBytesForReuse はファイルをパースして再利用可能な結果を返す。
// 同一ファイルの複数行分類でパースコストを削減する。
func ParseBytesForReuse(path string, src []byte) (*ParsedFile, error) {
	tree, src, err := ParseBytes(path, src)
	if err != nil {
		return nil, err
	}
	return &ParsedFile{tree: tree, src: src}, nil
}

// ClassifyLineWithParsed は事前パース済みのファイルを使って行を分類する。
func ClassifyLineWithParsed(pf *ParsedFile, line int, targetName string) (*MatchInfo, error) {
	if pf == nil {
		return nil, fmt.Errorf("ParsedFile is nil")
	}
	return classifyLineInTree(pf.tree, pf.src, line, targetName)
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
		selectorKind, receiverType := analyzeSelectorMatch(root, node, src, lang, absStart)
		return &MatchInfo{
			Class:        class,
			Scope:        findEnclosingScope(node, src, lang),
			NodeType:     node.Type(lang),
			SelectorKind: selectorKind,
			ReceiverType: receiverType,
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

func analyzeSelectorMatch(root, node *gotreesitter.Node, src []byte, lang *gotreesitter.Language, matchByte uint32) (string, string) {
	if node == nil || node.Type(lang) != "field_identifier" {
		return "", ""
	}

	selector := findAncestorByType(node, lang, "selector_expression")
	if selector == nil {
		return "unknown", ""
	}

	operand := selector.ChildByFieldName("operand", lang)
	if operand == nil {
		operand = firstNamedChild(selector)
	}
	if operand == nil {
		return "unknown", ""
	}

	if selectorOperandIsImportedPackage(root, operand, src, lang, matchByte) {
		return "package", ""
	}
	if receiverType := inferReceiverTypeFromOperand(node, operand, src, lang, matchByte); receiverType != "" {
		return "method", receiverType
	}
	return "unknown", ""
}

func findAncestorByType(node *gotreesitter.Node, lang *gotreesitter.Language, want string) *gotreesitter.Node {
	for current := node; current != nil; current = current.Parent() {
		if current.Type(lang) == want {
			return current
		}
	}
	return nil
}

func selectorOperandIsImportedPackage(root, operand *gotreesitter.Node, src []byte, lang *gotreesitter.Language, matchByte uint32) bool {
	if root == nil || operand == nil || operand.Type(lang) != "identifier" {
		return false
	}

	name := strings.TrimSpace(operand.Text(src))
	if name == "" {
		return false
	}
	if !collectImportedPackageNames(root, src, lang)[name] {
		return false
	}
	// 型推論でローカル変数を検出
	if inferIdentifierType(operand, src, lang, matchByte) != "" {
		return false
	}
	// 型が不明でもローカル宣言が存在すればインポートのシャドーイング
	scope := findEnclosingCallable(operand, lang)
	if scope != nil && matchByte > scope.StartByte() && matchByte <= uint32(len(src)) {
		if detectLocalDeclaration(string(src[scope.StartByte():matchByte]), name) {
			return false
		}
	}
	return true
}

// detectLocalDeclaration はプレフィックスコード内に name のローカル宣言が存在するか検出する。
// inferIdentifierType が型を特定できない場合の補完チェック用。
func detectLocalDeclaration(prefix, name string) bool {
	quotedName := regexp.QuoteMeta(name)
	// 複数値の短変数宣言: name, x := or x, name :=
	shortVarRe := regexp.MustCompile(
		`(?m)(?:\b\w+[ \t]*,[ \t]*)*\b` + quotedName + `\b(?:[ \t]*,[ \t]*\w+)*[ \t]*:=`)
	if shortVarRe.MatchString(prefix) {
		return true
	}
	// グループ化 var 宣言: var name, other Type or var ( ... name Type ... )
	groupedVarRe := regexp.MustCompile(
		`(?m)\bvar\s+(?:\w+\s*,\s*)*` + quotedName + `\b`)
	return groupedVarRe.MatchString(prefix)
}

func collectImportedPackageNames(root *gotreesitter.Node, src []byte, lang *gotreesitter.Language) map[string]bool {
	packages := make(map[string]bool)
	if root == nil {
		return packages
	}

	stack := []*gotreesitter.Node{root}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == nil {
			continue
		}
		if current.Type(lang) == "import_spec" {
			if name := importNameFromSpec(strings.TrimSpace(current.Text(src))); name != "" {
				packages[name] = true
			}
		}
		for i := int(current.NamedChildCount()) - 1; i >= 0; i-- {
			child := current.NamedChild(i)
			if child != nil {
				stack = append(stack, child)
			}
		}
	}
	return packages
}

func importNameFromSpec(spec string) string {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return ""
	}

	fields := strings.Fields(spec)
	if len(fields) == 0 {
		return ""
	}

	if len(fields) == 1 {
		return defaultImportName(fields[0])
	}

	alias := strings.TrimSpace(fields[0])
	if alias == "" || alias == "." || alias == "_" {
		return ""
	}
	return alias
}

func defaultImportName(pathLiteral string) string {
	pathLiteral = strings.Trim(pathLiteral, "`\"")
	if pathLiteral == "" {
		return ""
	}
	if idx := strings.LastIndex(pathLiteral, "/"); idx >= 0 && idx < len(pathLiteral)-1 {
		return pathLiteral[idx+1:]
	}
	return pathLiteral
}

func inferReceiverTypeFromOperand(node, operand *gotreesitter.Node, src []byte, lang *gotreesitter.Language, matchByte uint32) string {
	if operand == nil {
		return ""
	}

	switch operand.Type(lang) {
	case "parenthesized_expression":
		return inferReceiverTypeFromOperand(node, firstNamedChild(operand), src, lang, matchByte)
	case "unary_expression":
		return inferReceiverTypeFromOperand(node, lastNamedChild(operand), src, lang, matchByte)
	case "composite_literal":
		if typeNode := operand.ChildByFieldName("type", lang); typeNode != nil {
			return strings.TrimSpace(typeNode.Text(src))
		}
		return inferTypeFromCompositeLiteralText(strings.TrimSpace(operand.Text(src)))
	case "identifier":
		return inferIdentifierType(operand, src, lang, matchByte)
	case "type_identifier", "generic_type", "pointer_type", "array_type", "slice_type", "map_type":
		return strings.TrimSpace(operand.Text(src))
	default:
		return inferTypeFromCompositeLiteralText(strings.TrimSpace(operand.Text(src)))
	}
}

func inferTypeFromCompositeLiteralText(expr string) string {
	expr = strings.TrimSpace(strings.TrimPrefix(expr, "&"))
	if expr == "" {
		return ""
	}
	if idx := strings.Index(expr, "{"); idx > 0 {
		return strings.TrimSpace(expr[:idx])
	}
	return ""
}

func inferIdentifierType(node *gotreesitter.Node, src []byte, lang *gotreesitter.Language, matchByte uint32) string {
	if node == nil {
		return ""
	}

	name := strings.TrimSpace(node.Text(src))
	if name == "" {
		return ""
	}

	scope := findEnclosingCallable(node, lang)
	if scope == nil {
		return ""
	}

	signature := extractSignature(scope, src, lang)
	if typ := inferIdentifierTypeFromSignature(signature, name); typ != "" {
		return typ
	}

	if matchByte <= scope.StartByte() || matchByte > uint32(len(src)) {
		return ""
	}
	return inferIdentifierTypeFromPrefix(string(src[scope.StartByte():matchByte]), name)
}

func findEnclosingCallable(node *gotreesitter.Node, lang *gotreesitter.Language) *gotreesitter.Node {
	for current := node; current != nil; current = current.Parent() {
		switch current.Type(lang) {
		case "function_declaration", "method_declaration":
			return current
		}
	}
	return nil
}

func inferIdentifierTypeFromSignature(signature, name string) string {
	signature = strings.TrimSpace(signature)
	if !strings.HasPrefix(signature, "func") {
		return ""
	}

	rest := strings.TrimSpace(strings.TrimPrefix(signature, "func"))
	if strings.HasPrefix(rest, "(") {
		receiverSpec, after := splitLeadingGroup(rest)
		if typ := inferNamedTypeFromSection(receiverSpec, name); typ != "" {
			return typ
		}
		rest = strings.TrimSpace(after)
	}

	openIdx := strings.Index(rest, "(")
	if openIdx < 0 || openIdx >= len(rest) {
		return ""
	}
	paramsSpec, _ := splitLeadingGroup(rest[openIdx:])
	return inferNamedTypeFromSection(paramsSpec, name)
}

func splitLeadingGroup(s string) (string, string) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "(") {
		return "", s
	}

	depth := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return strings.TrimSpace(s[1:i]), strings.TrimSpace(s[i+1:])
			}
		}
	}
	return "", ""
}

func inferNamedTypeFromSection(section, name string) string {
	section = strings.TrimSpace(section)
	if section == "" || name == "" {
		return ""
	}

	// カンマで分割してパラメータグループを解析する。
	// Go の "a, b Config" のようなグループ化パラメータに対応する。
	entries := splitTopLevelCommas(section)
	var pendingNames []string
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		fields := strings.Fields(entry)
		if len(fields) == 1 {
			// 型なし: グループ内の名前
			pendingNames = append(pendingNames, fields[0])
		} else {
			// "paramName Type" 形式
			paramName := fields[0]
			typeName := strings.TrimSpace(entry[len(paramName):])
			if paramName == name {
				return typeName
			}
			for _, pn := range pendingNames {
				if pn == name {
					return typeName
				}
			}
			pendingNames = nil
		}
	}
	return ""
}

// splitTopLevelCommas は括弧のネストを考慮してトップレベルのカンマで分割する。
func splitTopLevelCommas(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, r := range s {
		switch r {
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func inferIdentifierTypeFromPrefix(prefix, name string) string {
	if prefix == "" || name == "" {
		return ""
	}

	quotedName := regexp.QuoteMeta(name)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?m)\b` + quotedName + `\s*:?=\s*&?\s*([A-Za-z_][A-Za-z0-9_]*(?:\[[^\]\n]+\])?)\s*\{`),
		regexp.MustCompile(`(?m)\bvar\s+` + quotedName + `\s*=\s*&?\s*([A-Za-z_][A-Za-z0-9_]*(?:\[[^\]\n]+\])?)\s*\{`),
		regexp.MustCompile(`(?m)\bvar\s+` + quotedName + `\s+([*A-Za-z_][A-Za-z0-9_\.\[\]\*]*)`),
	}

	latestPos := -1
	latestType := ""
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatchIndex(prefix, -1)
		for _, loc := range matches {
			if len(loc) < 4 {
				continue
			}
			if loc[0] >= latestPos {
				latestPos = loc[0]
				latestType = strings.TrimSpace(prefix[loc[2]:loc[3]])
			}
		}
	}

	// グループ化 var 宣言: "var a, name Type"
	if latestType == "" {
		latestType = inferTypeFromGroupedVarDecl(prefix, name)
	}

	return latestType
}

// inferTypeFromGroupedVarDecl はグループ化 var 宣言 ("var a, b Type") から型を推論する。
func inferTypeFromGroupedVarDecl(prefix, name string) string {
	groupedVarRe := regexp.MustCompile(`(?m)\bvar\s+(\w+(?:\s*,\s*\w+)+)\s+([*A-Za-z_][A-Za-z0-9_.\[\]*]*)`)
	for _, match := range groupedVarRe.FindAllStringSubmatch(prefix, -1) {
		if len(match) < 3 {
			continue
		}
		for _, n := range strings.Split(match[1], ",") {
			if strings.TrimSpace(n) == name {
				return strings.TrimSpace(match[2])
			}
		}
	}
	return ""
}

func firstNamedChild(node *gotreesitter.Node) *gotreesitter.Node {
	if node == nil || node.NamedChildCount() == 0 {
		return nil
	}
	return node.NamedChild(0)
}

func lastNamedChild(node *gotreesitter.Node) *gotreesitter.Node {
	if node == nil || node.NamedChildCount() == 0 {
		return nil
	}
	return node.NamedChild(int(node.NamedChildCount()) - 1)
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

func collectSyntaxErrors(node *gotreesitter.Node, lang *gotreesitter.Language, src []byte) []SyntaxError {
	if node == nil {
		return nil
	}

	errors := make([]SyntaxError, 0, 5)
	seen := make(map[string]struct{})
	stack := []*gotreesitter.Node{node}

	for len(stack) > 0 && len(errors) < 5 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == nil {
			continue
		}

		if current.IsError() || current.IsMissing() {
			syntaxError := buildSyntaxError(current, lang, src)
			key := fmt.Sprintf("%d:%d:%s", syntaxError.Line, syntaxError.Column, syntaxError.Message)
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				errors = append(errors, syntaxError)
			}
		}

		for i := current.ChildCount() - 1; i >= 0; i-- {
			child := current.Child(i)
			if child != nil {
				stack = append(stack, child)
			}
		}
	}

	return errors
}

func buildSyntaxError(node *gotreesitter.Node, lang *gotreesitter.Language, src []byte) SyntaxError {
	line := int(node.StartPoint().Row) + 1
	column := int(node.StartPoint().Column) + 1
	snippet := syntaxErrorSnippet(node, src)

	message := fmt.Sprintf("syntax error at L%d:%d", line, column)
	if node.IsMissing() {
		missingType := strings.TrimSpace(node.Type(lang))
		if missingType != "" {
			message = fmt.Sprintf("missing %s at L%d:%d", missingType, line, column)
		} else {
			message = fmt.Sprintf("missing syntax element at L%d:%d", line, column)
		}
	}
	if snippet != "" {
		message += fmt.Sprintf(" near %q", snippet)
	}

	return SyntaxError{
		Line:    line,
		Column:  column,
		Message: message,
	}
}

func syntaxErrorSnippet(node *gotreesitter.Node, src []byte) string {
	start := int(node.StartByte())
	end := int(node.EndByte())
	if start < 0 {
		start = 0
	}
	if end > len(src) {
		end = len(src)
	}
	if start >= end {
		return ""
	}

	snippet := string(src[start:end])
	snippet = strings.NewReplacer("\n", "\\n", "\r", "", "\t", "\\t").Replace(snippet)
	snippet = strings.TrimSpace(snippet)
	if snippet == "" {
		return ""
	}
	if len(snippet) > 30 {
		return snippet[:30] + "..."
	}
	return snippet
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
