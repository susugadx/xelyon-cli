package navigation

import (
	"fmt"
	goast "go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

func findSymbolColumn(cand SymbolCandidate) (int, error) {
	absPath := candidateAbsPath(cand)
	if absPath == "" {
		return 1, fmt.Errorf("empty symbol path")
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return 1, err
	}

	lines := strings.Split(string(content), "\n")
	if cand.Line < 1 || cand.Line > len(lines) {
		return 1, fmt.Errorf("line %d out of range for %s", cand.Line, cand.File)
	}

	name := cand.Name
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	line := lines[cand.Line-1]
	if idx := strings.LastIndex(line, name); idx >= 0 {
		return idx + 1, nil
	}
	return 1, nil
}

// findEnclosingFunction returns the enclosing function name for a file/line pair.
func findEnclosingFunction(filePath string, line int) string {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	symbols, err := ast.ExtractSymbols(absPath)
	if err != nil {
		return "package-level"
	}
	for _, s := range symbols {
		if line < s.Line || line > s.EndLine {
			continue
		}
		switch s.Kind {
		case ast.SymbolFunction:
			return "func " + s.Name
		case ast.SymbolMethod:
			return "method " + s.Name
		}
	}
	return "package-level"
}

// findTypeNameAtLine returns the most likely type name declared at the provided location.
func findTypeNameAtLine(filePath string, line int) string {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	symbols, err := ast.ExtractSymbols(absPath)
	if err == nil {
		for _, s := range symbols {
			if s.Line != line {
				continue
			}
			switch s.Kind {
			case ast.SymbolType, ast.SymbolStruct, ast.SymbolInterface, ast.SymbolClass, ast.SymbolEnum, ast.SymbolTrait, ast.SymbolImpl:
				if s.Name != "" {
					return s.Name
				}
			}
		}
	}

	base := filepath.Base(absPath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// isTestFile reports whether the file is a Go test file.
func isTestFile(filePath string) bool {
	return strings.HasSuffix(filePath, "_test.go")
}

// readLineSnippet returns the trimmed contents of the given line.
func readLineSnippet(filePath string, line int) string {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	if line < 1 || line > len(lines) {
		return ""
	}
	return strings.TrimSpace(lines[line-1])
}

// classifyLineByAST classifies a single Go line using the existing AST heuristics.
func classifyLineByAST(filePath string, line int, symbol string) ast.MatchClass {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}

	src, err := os.ReadFile(absPath)
	if err != nil {
		return ast.ClassUnknown
	}

	info, err := ast.ClassifyLine(absPath, src, line, symbol)
	if err != nil || info == nil {
		return ast.ClassUnknown
	}
	return info.Class
}

type parsedRipgrepReferenceLine struct {
	AbsPath string
	RelPath string
	Line    int
	Snippet string
	IsTest  bool
}

type referenceClassification struct {
	Scope        string
	Class        ast.MatchClass
	NodeType     string
	SelectorKind string
	ReceiverType string
}

// parseRipgrepLine は "file:line:content" 形式の行をパースし、分類情報を付与する。
func parseRipgrepLine(line, symbol string, cache *referenceParseCache) *Reference {
	parsed, ok := parseRipgrepReferenceLine(line)
	if !ok {
		return nil
	}

	classification := classifyParsedReferenceLine(parsed, symbol, cache)
	classification = applySnippetCompletionHints(parsed.Snippet, symbol, classification)
	return buildReferenceFromParsedLine(parsed, classification)
}

// parseRipgrepReferenceLine は ripgrep 1行を参照情報の基本形に変換する。
func parseRipgrepReferenceLine(line string) (parsedRipgrepReferenceLine, bool) {
	firstColon := strings.Index(line, ":")
	if firstColon < 0 {
		return parsedRipgrepReferenceLine{}, false
	}
	rest := line[firstColon+1:]
	secondColon := strings.Index(rest, ":")
	if secondColon < 0 {
		return parsedRipgrepReferenceLine{}, false
	}

	filePath := line[:firstColon]
	lineNum, err := strconv.Atoi(rest[:secondColon])
	if err != nil || lineNum <= 0 {
		return parsedRipgrepReferenceLine{}, false
	}

	absPath := mustAbs(filePath)
	return parsedRipgrepReferenceLine{
		AbsPath: absPath,
		RelPath: toRelativePath(absPath),
		Line:    lineNum,
		Snippet: strings.TrimSpace(rest[secondColon+1:]),
		IsTest:  strings.HasSuffix(filePath, "_test.go"),
	}, true
}

// classifyParsedReferenceLine は parse 済み参照に AST ベース分類を適用する。
func classifyParsedReferenceLine(parsed parsedRipgrepReferenceLine, symbol string, cache *referenceParseCache) referenceClassification {
	classification := referenceClassification{
		Class: ast.ClassUnknown,
	}
	if symbol == "" || cache == nil || !ast.IsSupportedFile(parsed.AbsPath) {
		return classification
	}

	cf := cache.get(parsed.AbsPath)
	if cf == nil {
		return classification
	}

	classification = classifyParsedLineWithTreeSitter(cf, parsed, symbol, classification)
	classification = classifyParsedLineWithGoASTHints(cf, parsed, symbol, classification)
	return classification
}

// classifyParsedLineWithTreeSitter は tree-sitter 分類結果を反映する。
func classifyParsedLineWithTreeSitter(cf *cachedFile, parsed parsedRipgrepReferenceLine, symbol string, current referenceClassification) referenceClassification {
	if pf := cf.ensureTreeSitter(parsed.AbsPath); pf != nil {
		if info, err := ast.ClassifyLineWithParsed(pf, parsed.Line, symbol); err == nil && info != nil {
			current.Scope = info.Scope
			current.Class = info.Class
			current.NodeType = info.NodeType
			current.SelectorKind = info.SelectorKind
			current.ReceiverType = info.ReceiverType
		}
	}
	return current
}

// classifyParsedLineWithGoASTHints は go/parser 分類をヒントとして補完適用する。
func classifyParsedLineWithGoASTHints(cf *cachedFile, parsed parsedRipgrepReferenceLine, symbol string, current referenceClassification) referenceClassification {
	if goFile, goFSet, goImports := cf.ensureGoParser(parsed.AbsPath); goFile != nil {
		return applyGoParserReferenceHints(goFile, goFSet, goImports, parsed.Line, symbol, current)
	}
	return current
}

// applySnippetCompletionHints は snippet から不足した分類情報を補完する。
func applySnippetCompletionHints(snippet, symbol string, current referenceClassification) referenceClassification {
	current.Class, current.NodeType, current.SelectorKind, current.ReceiverType = applySnippetReferenceHints(
		snippet,
		symbol,
		current.Class,
		current.NodeType,
		current.SelectorKind,
		current.ReceiverType,
	)
	return current
}

// buildReferenceFromParsedLine は parse + classify 結果から最終参照構造体を組み立てる。
func buildReferenceFromParsedLine(parsed parsedRipgrepReferenceLine, classification referenceClassification) *Reference {
	return &Reference{
		File:         parsed.RelPath,
		ResolvedPath: cleanNavigationResolvedPath(parsed.AbsPath),
		Line:         parsed.Line,
		Scope:        classification.Scope,
		Snippet:      parsed.Snippet,
		IsTest:       parsed.IsTest,
		Class:        classification.Class,
		NodeType:     classification.NodeType,
		SelectorKind: classification.SelectorKind,
		ReceiverType: classification.ReceiverType,
	}
}

func cleanNavigationResolvedPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}

func applyGoParserReferenceHints(file *goast.File, fset *token.FileSet, imports map[string]bool, line int, symbol string, current referenceClassification) referenceClassification {
	fallback, ok := classifyLineWithGoAST(file, fset, imports, line, symbol)
	if !ok {
		return current
	}

	if (current.Scope == "" || current.Scope == "package-level") && fallback.Scope != "" {
		current.Scope = fallback.Scope
	}
	if fallback.Class == ast.ClassDef {
		current.Class = ast.ClassDef
	} else if current.Class == ast.ClassUnknown && fallback.Class != ast.ClassUnknown {
		current.Class = fallback.Class
	}
	if current.NodeType == "" && fallback.NodeType != "" {
		current.NodeType = fallback.NodeType
	}
	if (current.SelectorKind == "" || current.SelectorKind == "unknown") && fallback.SelectorKind != "" {
		current.SelectorKind = fallback.SelectorKind
	}
	if current.ReceiverType == "" && fallback.ReceiverType != "" {
		current.ReceiverType = fallback.ReceiverType
	}

	return current
}

func classifyLineWithGoAST(file *goast.File, fset *token.FileSet, imports map[string]bool, line int, symbol string) (Reference, bool) {
	if !isGoASTClassificationInputValid(file, fset, line, symbol) {
		return Reference{}, false
	}

	ctx := goASTLineClassificationContext{
		file:    file,
		fset:    fset,
		imports: imports,
		line:    line,
		symbol:  symbol,
	}
	result := Reference{
		Scope: enclosingScopeFromGoAST(file, fset, line),
		Class: ast.ClassUnknown,
	}
	if !classifyLineByGoASTNodes(ctx, &result) {
		return Reference{}, false
	}
	return result, true
}

type goASTLineClassificationContext struct {
	file    *goast.File
	fset    *token.FileSet
	imports map[string]bool
	line    int
	symbol  string
}

func isGoASTClassificationInputValid(file *goast.File, fset *token.FileSet, line int, symbol string) bool {
	return file != nil && fset != nil && line > 0 && symbol != ""
}

func classifyLineByGoASTNodes(ctx goASTLineClassificationContext, result *Reference) bool {
	matched := false
	goast.Inspect(ctx.file, func(n goast.Node) bool {
		if !nodeIncludesLine(ctx.fset, n, ctx.line) {
			return true
		}
		if classifyGoASTNode(ctx, result, n) {
			matched = true
		}
		return true
	})
	return matched
}

func nodeIncludesLine(fset *token.FileSet, n goast.Node, line int) bool {
	if fset == nil || n == nil {
		return false
	}
	startLine := fset.Position(n.Pos()).Line
	endLine := fset.Position(n.End()).Line
	return line >= startLine && line <= endLine
}

func classifyGoASTNode(ctx goASTLineClassificationContext, result *Reference, n goast.Node) bool {
	switch node := n.(type) {
	case *goast.FuncDecl:
		return classifyGoASTFuncDecl(ctx, result, node)
	case *goast.CallExpr:
		return classifyGoASTCallExpr(ctx, result, node)
	case *goast.SelectorExpr:
		return classifyGoASTSelectorExpr(ctx, result, node)
	case *goast.Ident:
		return classifyGoASTIdent(ctx, result, node)
	default:
		return false
	}
}

func classifyGoASTFuncDecl(ctx goASTLineClassificationContext, result *Reference, fn *goast.FuncDecl) bool {
	if !identMatchesLine(ctx.fset, fn.Name, ctx.symbol, ctx.line) {
		return false
	}
	applyGoASTDefinitionHint(result)
	return true
}

func classifyGoASTCallExpr(ctx goASTLineClassificationContext, result *Reference, call *goast.CallExpr) bool {
	switch fun := call.Fun.(type) {
	case *goast.Ident:
		if !identMatchesLine(ctx.fset, fun, ctx.symbol, ctx.line) {
			return false
		}
		applyGoASTIdentCallHint(result)
		return true
	case *goast.SelectorExpr:
		if !identMatchesLine(ctx.fset, fun.Sel, ctx.symbol, ctx.line) {
			return false
		}
		selectorKind := selectorKindFromGoExpr(fun.X, ctx.imports, ctx.file, ctx.fset, ctx.line)
		receiverType := ""
		if selectorKind == "method" {
			receiverType = receiverTypeFromGoExpr(fun.X)
		}
		applyGoASTSelectorCallHint(result, selectorKind, receiverType)
		return true
	default:
		return false
	}
}

func classifyGoASTSelectorExpr(ctx goASTLineClassificationContext, result *Reference, selector *goast.SelectorExpr) bool {
	if !identMatchesLine(ctx.fset, selector.Sel, ctx.symbol, ctx.line) {
		return false
	}
	selectorKind := result.SelectorKind
	if selectorKind == "" || selectorKind == "unknown" {
		selectorKind = selectorKindFromGoExpr(selector.X, ctx.imports, ctx.file, ctx.fset, ctx.line)
	}
	receiverType := result.ReceiverType
	if receiverType == "" && selectorKind == "method" {
		receiverType = receiverTypeFromGoExpr(selector.X)
	}
	applyGoASTSelectorRefHint(result, selectorKind, receiverType)
	return true
}

func classifyGoASTIdent(ctx goASTLineClassificationContext, result *Reference, ident *goast.Ident) bool {
	if !identMatchesLine(ctx.fset, ident, ctx.symbol, ctx.line) {
		return false
	}
	applyGoASTIdentRefHint(result)
	return true
}

func identMatchesLine(fset *token.FileSet, ident *goast.Ident, symbol string, line int) bool {
	return ident != nil && ident.Name == symbol && fset.Position(ident.Pos()).Line == line
}

func applyGoASTDefinitionHint(result *Reference) {
	result.Class = ast.ClassDef
	result.NodeType = "identifier"
}

func applyGoASTIdentCallHint(result *Reference) {
	result.Class = ast.ClassCall
	result.NodeType = "identifier"
}

func applyGoASTSelectorCallHint(result *Reference, selectorKind, receiverType string) {
	result.Class = ast.ClassCall
	result.NodeType = "field_identifier"
	result.SelectorKind = selectorKind
	if result.SelectorKind == "method" {
		result.ReceiverType = receiverType
	}
}

func applyGoASTSelectorRefHint(result *Reference, selectorKind, receiverType string) {
	if result.Class == ast.ClassUnknown {
		result.Class = ast.ClassRef
	}
	if result.NodeType == "" {
		result.NodeType = "field_identifier"
	}
	if result.SelectorKind == "" || result.SelectorKind == "unknown" {
		result.SelectorKind = selectorKind
	}
	if result.ReceiverType == "" && result.SelectorKind == "method" {
		result.ReceiverType = receiverType
	}
}

func applyGoASTIdentRefHint(result *Reference) {
	if result.Class == ast.ClassUnknown {
		result.Class = ast.ClassRef
	}
	if result.NodeType == "" {
		result.NodeType = "identifier"
	}
}

func enclosingScopeFromGoAST(file *goast.File, fset *token.FileSet, line int) string {
	for _, decl := range file.Decls {
		fn, ok := decl.(*goast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}
		startLine := fset.Position(fn.Pos()).Line
		endLine := fset.Position(fn.End()).Line
		if line < startLine || line > endLine {
			continue
		}
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			return "method " + fn.Name.Name
		}
		return "func " + fn.Name.Name
	}
	return "package-level"
}

func importedPackageNames(file *goast.File) map[string]bool {
	imports := make(map[string]bool)
	for _, spec := range file.Imports {
		if spec == nil {
			continue
		}
		if spec.Name != nil {
			name := strings.TrimSpace(spec.Name.Name)
			if name != "" && name != "." && name != "_" {
				imports[name] = true
			}
			continue
		}
		pathValue := strings.Trim(spec.Path.Value, "\"")
		if pathValue == "" {
			continue
		}
		if idx := strings.LastIndex(pathValue, "/"); idx >= 0 {
			pathValue = pathValue[idx+1:]
		}
		if pathValue != "" {
			imports[pathValue] = true
		}
	}
	return imports
}

func selectorKindFromGoExpr(expr goast.Expr, imports map[string]bool, file *goast.File, fset *token.FileSet, line int) string {
	ident, ok := expr.(*goast.Ident)
	if !ok {
		return "method"
	}
	if !imports[ident.Name] {
		return "method"
	}
	// ローカル変数がインポート名をシャドーイングしている場合はメソッド呼び出し
	if isIdentShadowedInGoFunc(file, fset, line, ident.Name) {
		return "method"
	}
	return "package"
}

func receiverTypeFromGoExpr(expr goast.Expr) string {
	switch node := expr.(type) {
	case *goast.CompositeLit:
		return receiverTypeFromGoExpr(node.Type)
	case *goast.Ident:
		return canonicalReceiver(node.Name)
	case *goast.StarExpr:
		return receiverTypeFromGoExpr(node.X)
	case *goast.ParenExpr:
		return receiverTypeFromGoExpr(node.X)
	case *goast.SelectorExpr:
		// フィールドチェーン（foo.Bar.Method()）では Sel は型ではなくフィールド名の可能性が高い。
		// 型チェッカーなしでは実型を解決できないため空文字を返す。
		return ""
	}
	return ""
}

// isIdentShadowedInGoFunc は Go 関数内でインポート名がローカル変数にシャドーイングされているかを判定する。
func isIdentShadowedInGoFunc(file *goast.File, fset *token.FileSet, line int, name string) bool {
	for _, decl := range file.Decls {
		fn, ok := decl.(*goast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		startLine := fset.Position(fn.Pos()).Line
		endLine := fset.Position(fn.End()).Line
		if line < startLine || line > endLine {
			continue
		}
		// 関数パラメータをチェック
		if fn.Type != nil && fn.Type.Params != nil {
			for _, param := range fn.Type.Params.List {
				for _, paramName := range param.Names {
					if paramName.Name == name {
						return true
					}
				}
			}
		}
		// レシーバをチェック
		if fn.Recv != nil {
			for _, param := range fn.Recv.List {
				for _, paramName := range param.Names {
					if paramName.Name == name {
						return true
					}
				}
			}
		}
		// 関数本体のローカル宣言をチェック
		return hasLocalDeclInBlock(fn.Body, fset, line, name)
	}
	return false
}

// hasLocalDeclInBlock はブロック文内で useLine のスコープから見える name のローカル宣言を検出する。
// ネストブロック（if/for/switch 等）内の宣言も再帰的に走査する。
func hasLocalDeclInBlock(block *goast.BlockStmt, fset *token.FileSet, useLine int, name string) bool {
	if block == nil {
		return false
	}
	return hasLocalDeclInStmts(block.List, fset, useLine, name)
}

func hasLocalDeclInStmts(stmts []goast.Stmt, fset *token.FileSet, useLine int, name string) bool {
	for _, stmt := range stmts {
		stmtLine := fset.Position(stmt.Pos()).Line
		stmtEndLine := fset.Position(stmt.End()).Line
		if stmtLine > useLine {
			break
		}
		// useLine より前の直接宣言をチェック
		if stmtLine < useLine && matchesDeclName(stmt, name) {
			return true
		}
		// useLine を含む文のネストブロックに再帰
		if stmtEndLine >= useLine {
			if checkNestedDeclInStmt(stmt, fset, useLine, name) {
				return true
			}
		}
	}
	return false
}

// matchesDeclName は文が name を直接宣言しているかを判定する。
func matchesDeclName(stmt goast.Stmt, name string) bool {
	switch s := stmt.(type) {
	case *goast.AssignStmt:
		if s.Tok == token.DEFINE {
			for _, lhs := range s.Lhs {
				if ident, ok := lhs.(*goast.Ident); ok && ident.Name == name {
					return true
				}
			}
		}
	case *goast.DeclStmt:
		if genDecl, ok := s.Decl.(*goast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if vs, ok := spec.(*goast.ValueSpec); ok {
					for _, n := range vs.Names {
						if n.Name == name {
							return true
						}
					}
				}
			}
		}
	}
	return false
}

// checkNestedDeclInStmt は複合文（if/for/switch 等）のサブブロック内で name の宣言を検出する。
func checkNestedDeclInStmt(stmt goast.Stmt, fset *token.FileSet, useLine int, name string) bool {
	switch s := stmt.(type) {
	case *goast.IfStmt:
		if s.Init != nil && matchesDeclName(s.Init, name) {
			return true
		}
		if hasLocalDeclInBlock(s.Body, fset, useLine, name) {
			return true
		}
		if s.Else != nil {
			switch e := s.Else.(type) {
			case *goast.BlockStmt:
				return hasLocalDeclInBlock(e, fset, useLine, name)
			case *goast.IfStmt:
				return checkNestedDeclInStmt(e, fset, useLine, name)
			}
		}
	case *goast.ForStmt:
		if s.Init != nil && matchesDeclName(s.Init, name) {
			return true
		}
		return hasLocalDeclInBlock(s.Body, fset, useLine, name)
	case *goast.RangeStmt:
		if s.Tok == token.DEFINE {
			if key, ok := s.Key.(*goast.Ident); ok && key.Name == name {
				return true
			}
			if s.Value != nil {
				if value, ok := s.Value.(*goast.Ident); ok && value.Name == name {
					return true
				}
			}
		}
		return hasLocalDeclInBlock(s.Body, fset, useLine, name)
	case *goast.SwitchStmt:
		if s.Init != nil && matchesDeclName(s.Init, name) {
			return true
		}
		return hasLocalDeclInBlock(s.Body, fset, useLine, name)
	case *goast.TypeSwitchStmt:
		if s.Init != nil && matchesDeclName(s.Init, name) {
			return true
		}
		if s.Assign != nil && matchesDeclName(s.Assign, name) {
			return true
		}
		return hasLocalDeclInBlock(s.Body, fset, useLine, name)
	case *goast.SelectStmt:
		return hasLocalDeclInBlock(s.Body, fset, useLine, name)
	case *goast.CaseClause:
		return hasLocalDeclInStmts(s.Body, fset, useLine, name)
	case *goast.CommClause:
		if s.Comm != nil && matchesDeclName(s.Comm, name) {
			return true
		}
		return hasLocalDeclInStmts(s.Body, fset, useLine, name)
	case *goast.BlockStmt:
		return hasLocalDeclInBlock(s, fset, useLine, name)
	}
	return false
}
