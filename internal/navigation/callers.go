package navigation

import (
	"bufio"
	"context"
	"fmt"
	goast "go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// maxRipgrepResults は findReferences が返す参照の上限。
// これを超えた場合は truncated=true を返す。
const maxRipgrepResults = 500

const (
	ripgrepScannerInitialBufferSize = 64 * 1024
	ripgrepScannerMaxBufferSize     = 1024 * 1024
	lspReferenceTimeout             = 5 * time.Second
)

// referenceSearchResult は ripgrep 参照検索の内部状態を保持する。
type referenceSearchResult struct {
	Refs          []Reference
	Truncated     bool
	Incomplete    bool
	StopRequested bool
}

// referenceParseCache は同一ファイルの重複パースを防ぐキャッシュ。
type referenceParseCache struct {
	files map[string]*cachedFile
}

// cachedFile はファイルごとのパース済みデータを保持する。
type cachedFile struct {
	src         []byte
	tsParsed    *ast.ParsedFile
	tsAttempted bool
	goFile      *goast.File
	goFSet      *token.FileSet
	goImports   map[string]bool
	goAttempted bool
}

func newReferenceParseCache() *referenceParseCache {
	return &referenceParseCache{files: make(map[string]*cachedFile)}
}

func (c *referenceParseCache) get(absPath string) *cachedFile {
	cf, exists := c.files[absPath]
	if exists {
		return cf
	}
	src, err := os.ReadFile(absPath)
	if err != nil {
		c.files[absPath] = nil
		return nil
	}
	cf = &cachedFile{src: src}
	c.files[absPath] = cf
	return cf
}

func (cf *cachedFile) ensureTreeSitter(absPath string) *ast.ParsedFile {
	if cf.tsAttempted {
		return cf.tsParsed
	}
	cf.tsAttempted = true
	pf, err := ast.ParseBytesForReuse(absPath, cf.src)
	if err != nil {
		return nil
	}
	cf.tsParsed = pf
	return pf
}

func (cf *cachedFile) ensureGoParser(absPath string) (*goast.File, *token.FileSet, map[string]bool) {
	if cf.goAttempted {
		return cf.goFile, cf.goFSet, cf.goImports
	}
	cf.goAttempted = true
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absPath, cf.src, parser.SkipObjectResolution)
	if err != nil {
		return nil, nil, nil
	}
	cf.goFile = file
	cf.goFSet = fset
	cf.goImports = importedPackageNames(file)
	return cf.goFile, cf.goFSet, cf.goImports
}

// findReferences は ripgrep でシンボル名を検索し、全参照を返す。
// StdoutPipe + scanner で逐次読み取りし、201件目を検出したら早期停止する。
// truncated が true の場合、上流の検索結果が上限を超えたことを示す。
// incomplete が true の場合、読み取り失敗や異常終了により結果が不完全であることを示す。
func findReferences(symbol string) (refs []Reference, truncated bool, incomplete bool) {
	if !common.IsRipgrepAvailable() {
		return nil, false, false
	}

	args := []string{
		"-n",
		"--no-heading",
		"--color", "never",
		"-w", // 単語境界
		"--type", "go",
		"--glob", "!vendor/",
		symbol,
		".",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, common.RipgrepPath(), args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, false, true
	}
	if err := cmd.Start(); err != nil {
		return nil, false, true
	}

	return runReferenceSearch(stdout, symbol, cancel, cmd.Wait)
}

// collectReferenceSearchResult は ripgrep の標準出力を読み取り、参照一覧を構築する。
func collectReferenceSearchResult(reader io.Reader, symbol string) referenceSearchResult {
	result := referenceSearchResult{}
	if reader == nil {
		result.Incomplete = true
		return result
	}

	cache := newReferenceParseCache()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, ripgrepScannerInitialBufferSize), ripgrepScannerMaxBufferSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		ref := parseRipgrepLine(line, symbol, cache)
		if ref == nil {
			continue
		}
		result.Refs = append(result.Refs, *ref)

		// 201件目を検出したら truncated=true にし、先頭200件のみ保持して早期停止
		if len(result.Refs) > maxRipgrepResults {
			result.Truncated = true
			result.Refs = result.Refs[:maxRipgrepResults]
			result.StopRequested = true
			break
		}
	}

	if err := scanner.Err(); err != nil {
		result.Incomplete = true
	}

	return result
}

// runReferenceSearch は参照ストリームの読み取りと終了待機をまとめて処理する。
func runReferenceSearch(reader io.Reader, symbol string, cancel func(), wait func() error) ([]Reference, bool, bool) {
	result := collectReferenceSearchResult(reader, symbol)
	if result.StopRequested && cancel != nil {
		cancel()
	}
	if wait != nil {
		if err := wait(); err != nil && !result.StopRequested {
			result.Incomplete = true
		}
	}
	return result.Refs, result.Truncated, result.Incomplete
}

func findReferencesWithFallbackRuntime(baseName string, cand SymbolCandidate, runtime GoSymbolRuntime) ([]Reference, bool, bool) {
	ambiguousFiles := findAmbiguousFilesWithRuntime(baseName, cand, runtime)
	allRefs, truncated, incomplete := findReferences(baseName)
	return filterRefsByCandidate(allRefs, cand, ambiguousFiles), truncated, incomplete
}

// findReferencesViaLSP resolves references through the LSP client and converts them to Reference values.
func findReferencesViaLSP(client LSPClient, cand SymbolCandidate, invocationCWD string) ([]Reference, error) {
	ctx, cancel := context.WithTimeout(context.Background(), lspReferenceTimeout)
	defer cancel()

	col, err := findSymbolColumn(cand)
	if err != nil {
		return nil, err
	}

	locations, err := client.FindReferences(ctx, cand.File, cand.Line, col, false)
	if err != nil {
		return nil, err
	}

	refs := make([]Reference, 0, len(locations))
	for _, loc := range locations {
		filePath := lspLocationFilePath(loc.File, cand.RootPath, invocationCWD)
		refs = append(refs, Reference{
			File:    filePath,
			Line:    loc.Line,
			Scope:   findEnclosingFunction(filePath, loc.Line),
			Snippet: readLineSnippet(filePath, loc.Line),
			IsTest:  isTestFile(filePath),
			Class:   classifyLineByAST(filePath, loc.Line, cand.Name),
		})
	}
	return refs, nil
}

// findImplementationsViaLSP resolves interface implementations through the LSP client.
func findImplementationsViaLSP(client LSPClient, cand SymbolCandidate, invocationCWD string) ([]ImplementationRef, error) {
	ctx, cancel := context.WithTimeout(context.Background(), lspReferenceTimeout)
	defer cancel()

	col, err := findSymbolColumn(cand)
	if err != nil {
		return nil, err
	}

	locations, err := client.GotoImplementation(ctx, cand.File, cand.Line, col)
	if err != nil {
		return nil, err
	}

	impls := make([]ImplementationRef, 0, len(locations))
	for _, loc := range locations {
		filePath := lspLocationFilePath(loc.File, cand.RootPath, invocationCWD)
		impls = append(impls, ImplementationRef{
			File: filePath,
			Line: loc.Line,
			Name: findTypeNameAtLine(filePath, loc.Line),
		})
	}
	return impls, nil
}

func lspLocationFilePath(file, rootPath, invocationCWD string) string {
	file = strings.TrimSpace(file)
	if file == "" || filepath.IsAbs(file) {
		return file
	}
	file = filepath.Clean(filepath.FromSlash(file))
	if file == "." {
		if cwd := strings.TrimSpace(invocationCWD); cwd != "" {
			return cwd
		}
		if cwd, err := os.Getwd(); err == nil {
			return cwd
		}
		return file
	}
	if file == ".." || strings.HasPrefix(file, ".."+string(filepath.Separator)) || strings.HasPrefix(file, "."+string(filepath.Separator)) {
		if cwd := strings.TrimSpace(invocationCWD); cwd != "" {
			return filepath.Join(cwd, file)
		}
		if cwd, err := os.Getwd(); err == nil {
			return filepath.Join(cwd, file)
		}
		return file
	}
	rootPath = strings.TrimSpace(rootPath)
	if rootPath == "" {
		return file
	}
	return filepath.Join(rootPath, file)
}

// findSymbolColumn returns the 1-indexed column of the symbol name on the candidate line.
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

// parseRipgrepLine は "file:line:content" 形式の行をパースする。
func parseRipgrepLine(line, symbol string, cache *referenceParseCache) *Reference {
	// file:line:content
	firstColon := strings.Index(line, ":")
	if firstColon < 0 {
		return nil
	}
	rest := line[firstColon+1:]
	secondColon := strings.Index(rest, ":")
	if secondColon < 0 {
		return nil
	}

	filePath := line[:firstColon]
	lineNumStr := rest[:secondColon]
	content := rest[secondColon+1:]

	lineNum, err := strconv.Atoi(lineNumStr)
	if err != nil || lineNum <= 0 {
		return nil
	}

	relPath := toRelativePath(mustAbs(filePath))
	isTest := strings.HasSuffix(filePath, "_test.go")

	// AST 分類を試みる（キャッシュで同一ファイルの重複パースを回避）
	scope := ""
	class := ast.ClassUnknown
	nodeType := ""
	selectorKind := ""
	receiverType := ""
	absPath := mustAbs(filePath)
	if ast.IsSupportedFile(absPath) {
		if cf := cache.get(absPath); cf != nil {
			if pf := cf.ensureTreeSitter(absPath); pf != nil {
				if info, err := ast.ClassifyLineWithParsed(pf, lineNum, symbol); err == nil && info != nil {
					scope = info.Scope
					class = info.Class
					nodeType = info.NodeType
					selectorKind = info.SelectorKind
					receiverType = info.ReceiverType
				}
			}
			if goFile, goFSet, goImports := cf.ensureGoParser(absPath); goFile != nil {
				scope, class, nodeType, selectorKind, receiverType = applyGoParserReferenceHints(
					goFile,
					goFSet,
					goImports,
					lineNum,
					symbol,
					scope,
					class,
					nodeType,
					selectorKind,
					receiverType,
				)
			}
		}
	}
	class, nodeType, selectorKind, receiverType = applySnippetReferenceHints(strings.TrimSpace(content), symbol, class, nodeType, selectorKind, receiverType)

	return &Reference{
		File:         relPath,
		Line:         lineNum,
		Scope:        scope,
		Snippet:      strings.TrimSpace(content),
		IsTest:       isTest,
		Class:        class,
		NodeType:     nodeType,
		SelectorKind: selectorKind,
		ReceiverType: receiverType,
	}
}

func applyGoParserReferenceHints(file *goast.File, fset *token.FileSet, imports map[string]bool, line int, symbol, scope string, class ast.MatchClass, nodeType, selectorKind, receiverType string) (string, ast.MatchClass, string, string, string) {
	fallback, ok := classifyLineWithGoAST(file, fset, imports, line, symbol)
	if !ok {
		return scope, class, nodeType, selectorKind, receiverType
	}

	if (scope == "" || scope == "package-level") && fallback.Scope != "" {
		scope = fallback.Scope
	}
	if fallback.Class == ast.ClassDef {
		class = ast.ClassDef
	} else if class == ast.ClassUnknown && fallback.Class != ast.ClassUnknown {
		class = fallback.Class
	}
	if nodeType == "" && fallback.NodeType != "" {
		nodeType = fallback.NodeType
	}
	if (selectorKind == "" || selectorKind == "unknown") && fallback.SelectorKind != "" {
		selectorKind = fallback.SelectorKind
	}
	if receiverType == "" && fallback.ReceiverType != "" {
		receiverType = fallback.ReceiverType
	}

	return scope, class, nodeType, selectorKind, receiverType
}

func classifyLineWithGoAST(file *goast.File, fset *token.FileSet, imports map[string]bool, line int, symbol string) (Reference, bool) {
	if file == nil || line <= 0 || symbol == "" {
		return Reference{}, false
	}

	result := Reference{}
	if scope := enclosingScopeFromGoAST(file, fset, line); scope != "" {
		result.Scope = scope
	}

	matched := false
	goast.Inspect(file, func(n goast.Node) bool {
		if n == nil {
			return true
		}
		startLine := fset.Position(n.Pos()).Line
		endLine := fset.Position(n.End()).Line
		if line < startLine || line > endLine {
			return true
		}

		switch node := n.(type) {
		case *goast.FuncDecl:
			if node.Name != nil && node.Name.Name == symbol && fset.Position(node.Name.Pos()).Line == line {
				result.Class = ast.ClassDef
				result.NodeType = "identifier"
				matched = true
			}
		case *goast.CallExpr:
			switch fun := node.Fun.(type) {
			case *goast.Ident:
				if fun.Name == symbol && fset.Position(fun.Pos()).Line == line {
					result.Class = ast.ClassCall
					result.NodeType = "identifier"
					matched = true
				}
			case *goast.SelectorExpr:
				if fun.Sel != nil && fun.Sel.Name == symbol && fset.Position(fun.Sel.Pos()).Line == line {
					result.Class = ast.ClassCall
					result.NodeType = "field_identifier"
					result.SelectorKind = selectorKindFromGoExpr(fun.X, imports, file, fset, line)
					if result.SelectorKind == "method" {
						result.ReceiverType = receiverTypeFromGoExpr(fun.X)
					}
					matched = true
				}
			}
		case *goast.SelectorExpr:
			if node.Sel != nil && node.Sel.Name == symbol && fset.Position(node.Sel.Pos()).Line == line {
				if result.Class == ast.ClassUnknown {
					result.Class = ast.ClassRef
				}
				if result.NodeType == "" {
					result.NodeType = "field_identifier"
				}
				if result.SelectorKind == "" || result.SelectorKind == "unknown" {
					result.SelectorKind = selectorKindFromGoExpr(node.X, imports, file, fset, line)
				}
				if result.ReceiverType == "" && result.SelectorKind == "method" {
					result.ReceiverType = receiverTypeFromGoExpr(node.X)
				}
				matched = true
			}
		case *goast.Ident:
			if node.Name == symbol && fset.Position(node.Pos()).Line == line {
				if result.Class == ast.ClassUnknown {
					result.Class = ast.ClassRef
				}
				if result.NodeType == "" {
					result.NodeType = "identifier"
				}
				matched = true
			}
		}
		return true
	})

	if !matched {
		return Reference{}, false
	}
	return result, true
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

func applySnippetReferenceHints(snippet, symbol string, class ast.MatchClass, nodeType, selectorKind, receiverType string) (ast.MatchClass, string, string, string) {
	snippet = strings.TrimSpace(snippet)
	if snippet == "" || symbol == "" {
		return class, nodeType, selectorKind, receiverType
	}

	if nodeType == "" {
		switch {
		case strings.Contains(snippet, "."+symbol):
			nodeType = "field_identifier"
		case strings.Contains(snippet, symbol):
			nodeType = "identifier"
		}
	}

	operand := selectorOperandFromSnippet(snippet, symbol)
	if selectorKind == "" && operand != "" {
		if looksLikePackageOperand(operand) {
			selectorKind = "package"
		} else {
			selectorKind = "method"
		}
	}
	if receiverType == "" && selectorKind == "method" {
		receiverType = inferReceiverTypeFromSnippetOperand(operand)
	}

	if class != ast.ClassDef && class != ast.ClassImport && class != ast.ClassComment && class != ast.ClassString {
		switch {
		case class == ast.ClassUnknown && isDefinitionSnippet(snippet, symbol):
			class = ast.ClassDef
		case strings.Contains(snippet, "."+symbol+"("):
			class = ast.ClassCall
		case containsBareSymbolCall(snippet, symbol):
			class = ast.ClassCall
		case class == ast.ClassUnknown && strings.Contains(snippet, "."+symbol):
			class = ast.ClassRef
		case class == ast.ClassUnknown && strings.Contains(snippet, symbol):
			class = ast.ClassRef
		}
	}

	return class, nodeType, selectorKind, receiverType
}

func isDefinitionSnippet(snippet, symbol string) bool {
	snippet = strings.TrimSpace(snippet)
	return strings.Contains(snippet, "func "+symbol+"(") || strings.Contains(snippet, ") "+symbol+"(")
}

func containsBareSymbolCall(snippet, symbol string) bool {
	idx := strings.Index(snippet, symbol+"(")
	if idx < 0 {
		return false
	}
	if idx > 0 && snippet[idx-1] == '.' {
		return false
	}
	return true
}

func selectorOperandFromSnippet(snippet, symbol string) string {
	idx := strings.Index(snippet, "."+symbol)
	if idx <= 0 {
		return ""
	}
	end := idx
	start := end - 1
	for start >= 0 {
		if !isSnippetOperandChar(snippet[start]) {
			break
		}
		start--
	}
	return strings.TrimSpace(snippet[start+1 : end])
}

func isSnippetOperandChar(ch byte) bool {
	switch {
	case ch >= 'a' && ch <= 'z':
		return true
	case ch >= 'A' && ch <= 'Z':
		return true
	case ch >= '0' && ch <= '9':
		return true
	}
	switch ch {
	case '_', '.', '*', '&', '(', ')', '[', ']', '{', '}':
		return true
	default:
		return false
	}
}

func looksLikePackageOperand(operand string) bool {
	operand = strings.TrimSpace(strings.TrimLeft(operand, "*&("))
	operand = strings.TrimRight(operand, ")")
	if operand == "" {
		return false
	}
	if strings.ContainsAny(operand, "{}[].") {
		return false
	}
	for _, r := range operand {
		return r >= 'a' && r <= 'z'
	}
	return false
}

func inferReceiverTypeFromSnippetOperand(operand string) string {
	operand = strings.TrimSpace(operand)
	if operand == "" {
		return ""
	}
	operand = strings.TrimSpace(strings.TrimLeft(operand, "*&"))
	operand = strings.Trim(operand, "()")
	if idx := strings.Index(operand, "{"); idx > 0 {
		return canonicalReceiver(strings.TrimSpace(operand[:idx]))
	}
	if operand != "" && operand[0] >= 'A' && operand[0] <= 'Z' && !strings.ContainsAny(operand, ".[]") {
		return canonicalReceiver(operand)
	}
	return ""
}

// classifyCallers は参照一覧からコール箇所を抽出する。
// AST 分類（ClassCall）を使い、定義行自体・テスト・定義宣言を除外する。
func classifyCallers(refs []Reference, def SymbolCandidate, limit int) ([]Reference, int, bool) {
	var callers []Reference

	for _, ref := range refs {
		// 定義行自体を除外
		if ref.File == def.File && ref.Line >= def.Line && ref.Line <= def.EndLine {
			continue
		}
		// テストファイルは tests で扱う
		if ref.IsTest {
			continue
		}
		// AST 分類が ClassCall のもののみ caller
		if ref.Class == ast.ClassCall {
			callers = append(callers, ref)
		}
	}

	total := len(callers)
	if total > limit {
		return callers[:limit], total, true
	}
	return callers, total, false
}

// classifyRefs は参照一覧から一般的な参照（変数や型の使用箇所）を抽出する。
// AST 分類を使い、定義行・テスト・コール箇所を除外する。
func classifyRefs(refs []Reference, def SymbolCandidate, limit int) ([]Reference, int, bool) {
	var results []Reference

	for _, ref := range refs {
		// 定義行自体を除外
		if ref.File == def.File && ref.Line >= def.Line && ref.Line <= def.EndLine {
			continue
		}
		// テストファイルは tests で扱う
		if ref.IsTest {
			continue
		}
		// コール、定義宣言、コメント、文字列、インポートは除外
		switch ref.Class {
		case ast.ClassDef, ast.ClassCall, ast.ClassImport, ast.ClassComment, ast.ClassString:
			continue
		default:
			results = append(results, ref)
		}
	}

	total := len(results)
	if total > limit {
		return results[:limit], total, true
	}
	return results, total, false
}

// mustAbs はパスを絶対パスに変換する。エラー時はそのまま返す。
func mustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
