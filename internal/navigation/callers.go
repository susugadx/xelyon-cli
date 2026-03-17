package navigation

import (
	"bufio"
	"context"
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
const maxRipgrepResults = 200

const (
	ripgrepScannerInitialBufferSize = 64 * 1024
	ripgrepScannerMaxBufferSize     = 1024 * 1024
)

// referenceSearchResult は ripgrep 参照検索の内部状態を保持する。
type referenceSearchResult struct {
	Refs          []Reference
	Truncated     bool
	Incomplete    bool
	StopRequested bool
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

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, ripgrepScannerInitialBufferSize), ripgrepScannerMaxBufferSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		ref := parseRipgrepLine(line, symbol)
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

// parseRipgrepLine は "file:line:content" 形式の行をパースする。
func parseRipgrepLine(line, symbol string) *Reference {
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

	// AST 分類を試みる
	scope := ""
	class := ast.ClassUnknown
	nodeType := ""
	selectorKind := ""
	receiverType := ""
	absPath := mustAbs(filePath)
	if ast.IsSupportedFile(absPath) {
		src, err := os.ReadFile(absPath)
		if err == nil {
			if info, err := ast.ClassifyLine(absPath, src, lineNum, symbol); err == nil && info != nil {
				scope = info.Scope
				class = info.Class
				nodeType = info.NodeType
				selectorKind = info.SelectorKind
				receiverType = info.ReceiverType
			}
			scope, class, nodeType, selectorKind, receiverType = applyGoParserReferenceHints(
				absPath,
				src,
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

func applyGoParserReferenceHints(path string, src []byte, line int, symbol, scope string, class ast.MatchClass, nodeType, selectorKind, receiverType string) (string, ast.MatchClass, string, string, string) {
	fallback, ok := classifyLineWithGoParser(path, src, line, symbol)
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

func classifyLineWithGoParser(path string, src []byte, line int, symbol string) (Reference, bool) {
	if len(src) == 0 || line <= 0 || symbol == "" {
		return Reference{}, false
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
	if err != nil {
		return Reference{}, false
	}

	result := Reference{}
	if scope := enclosingScopeFromGoAST(file, fset, line); scope != "" {
		result.Scope = scope
	}

	imports := importedPackageNames(file)
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
					result.SelectorKind = selectorKindFromGoExpr(fun.X, imports)
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
					result.SelectorKind = selectorKindFromGoExpr(node.X, imports)
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

func selectorKindFromGoExpr(expr goast.Expr, imports map[string]bool) string {
	ident, ok := expr.(*goast.Ident)
	if ok && imports[ident.Name] {
		return "package"
	}
	return "method"
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
		if node.Sel != nil {
			return canonicalReceiver(node.Sel.Name)
		}
	}
	return ""
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
	if strings.ContainsAny(operand, "{}[]") {
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
