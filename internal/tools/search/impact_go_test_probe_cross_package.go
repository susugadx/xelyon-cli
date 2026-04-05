package search

import (
	goast "go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

type packageHelper struct {
	key      string
	name     string
	receiver string
	abs      string
	src      []byte
	parsed   *ast.ParsedFile
	sym      ast.Symbol
}

type helperEdge struct {
	packageDir string
	key        string
}

type helperObject struct {
	packageDir string
	receiver   string
}

type helperValue struct {
	edges     []helperEdge
	objects   []helperObject
	uncertain bool
}

type helperParamUse struct {
	index      int
	methodName string
}

type helperSummary struct {
	returns       []helperEdge
	returnObjects []helperObject
	resultValues  []helperValue
	params        []helperParamUse
}

type parsedGoHelperFile struct {
	fset *token.FileSet
	file *goast.File
}

type crossPackageHelperGraph struct {
	ctx              goMethodTestProbeContext
	packageDir       string
	allowTestFiles   bool
	helpers          map[string]packageHelper
	helperKeysByName map[string][]string
	importedGraphs   map[string]*crossPackageHelperGraph
	callCache        map[string][]string
	fileASTCache     map[string]parsedGoHelperFile
	declCache        map[string]*goast.FuncDecl
	summaryCache     map[string]helperSummary
}

func crossPackageMethodTestMatchesSymbol(ctx goMethodTestProbeContext, absPath string, test navigation.TestRef) bool {
	ctx.dependencies.add(absPath)
	src, err := os.ReadFile(absPath)
	if err != nil {
		return false
	}
	parsed, err := ast.ParseBytesForReuse(absPath, src)
	if err != nil {
		return false
	}
	testSymbol, ok := findCrossPackageMethodProbeTestSymbol(absPath, src, test)
	if !ok {
		return false
	}
	if methodTestBodyMatchesSymbol(ctx.matchContext(absPath, src), parsed, testSymbol, false) {
		return true
	}
	graph := newCrossPackageHelperGraph(ctx, filepath.Dir(absPath), true)
	helper := packageHelper{
		key:    helperCacheKeyFromFields(absPath, testSymbol.Name, testSymbol.Line, testSymbol.EndLine),
		name:   testSymbol.Name,
		abs:    absPath,
		src:    src,
		parsed: parsed,
		sym:    testSymbol,
	}
	return graph.matchesSymbol(helper, make(map[string]struct{}))
}

func newCrossPackageHelperGraph(ctx goMethodTestProbeContext, packageDir string, allowTestFiles bool) *crossPackageHelperGraph {
	graph := &crossPackageHelperGraph{
		ctx:              ctx,
		packageDir:       packageDir,
		allowTestFiles:   allowTestFiles,
		helpers:          make(map[string]packageHelper),
		helperKeysByName: make(map[string][]string),
		importedGraphs:   make(map[string]*crossPackageHelperGraph),
		callCache:        make(map[string][]string),
		fileASTCache:     make(map[string]parsedGoHelperFile),
		declCache:        make(map[string]*goast.FuncDecl),
		summaryCache:     make(map[string]helperSummary),
	}

	entries, err := os.ReadDir(packageDir)
	if err != nil {
		return graph
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		filePath := filepath.Join(packageDir, entry.Name())
		if !shouldIncludeStructuredGoImpactProbeFile(filePath, structuredGoImpactProbeRootPath(ctx.opts, packageDir), ctx.opts) {
			continue
		}
		ctx.dependencies.add(filePath)
		fileSrc, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		if !isImportableCrossPackageHelperFile(filePath, fileSrc, allowTestFiles) {
			continue
		}
		parsed, err := ast.ParseBytesForReuse(filePath, fileSrc)
		if err == nil {
			symbols, symErr := ast.ExtractSymbolsFromBytes(filePath, fileSrc)
			if symErr == nil {
				added := false
				for _, candidate := range symbols {
					helper, ok := newPackageHelper(filePath, fileSrc, parsed, candidate)
					if !ok {
						continue
					}
					graph.addHelper(helper)
					added = true
				}
				if added {
					continue
				}
			}
		}
		for _, helper := range fallbackCrossPackageHelpersFromBrokenFile(filePath, fileSrc) {
			graph.addHelper(helper)
		}
	}

	return graph
}

func isImportableCrossPackageHelperFile(filePath string, src []byte, allowTestFiles bool) bool {
	if allowTestFiles {
		return true
	}
	if strings.HasSuffix(strings.TrimSpace(filePath), "_test.go") {
		return false
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, src, parser.PackageClauseOnly)
	if err != nil || file == nil || file.Name == nil {
		return false
	}
	return !strings.HasSuffix(strings.TrimSpace(file.Name.Name), "_test")
}

func newPackageHelper(absPath string, src []byte, parsed *ast.ParsedFile, candidate ast.Symbol) (packageHelper, bool) {
	switch candidate.Kind {
	case ast.SymbolFunction:
		if strings.HasPrefix(candidate.Name, "Test") {
			return packageHelper{}, false
		}
	case ast.SymbolMethod:
	default:
		return packageHelper{}, false
	}

	return packageHelper{
		key:      helperCacheKeyFromFields(absPath, candidate.Name, candidate.Line, candidate.EndLine),
		name:     candidate.Name,
		receiver: canonicalProbeReceiver(extractProbeMethodReceiver(candidate.Signature)),
		abs:      absPath,
		src:      src,
		parsed:   parsed,
		sym:      candidate,
	}, true
}

func fallbackCrossPackageHelpersFromBrokenFile(absPath string, src []byte) []packageHelper {
	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, absPath, src, parser.AllErrors)

	helpers := make([]packageHelper, 0, 4)
	seen := make(map[string]struct{})
	add := func(name, receiver string, line, endLine int) {
		helper, ok := newFallbackPackageHelper(absPath, src, name, receiver, line, endLine)
		if !ok {
			return
		}
		if _, exists := seen[helper.key]; exists {
			return
		}
		seen[helper.key] = struct{}{}
		helpers = append(helpers, helper)
	}

	if file != nil {
		for _, decl := range file.Decls {
			fn, ok := decl.(*goast.FuncDecl)
			if !ok || fn.Name == nil {
				continue
			}
			line := fset.Position(fn.Pos()).Line
			endLine := fset.Position(fn.End()).Line
			add(fn.Name.Name, fallbackReceiverFromFuncDecl(src, fset, fn), line, endLine)
		}
		if len(helpers) > 0 {
			return helpers
		}
	}

	srcText := string(src)
	re := regexp.MustCompile(`(?m)^func\s*(\(([^)]*)\)\s*)?([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	for _, match := range re.FindAllStringSubmatchIndex(srcText, -1) {
		if len(match) < 8 {
			continue
		}
		receiver := ""
		if match[4] >= 0 && match[5] >= 0 {
			receiver = fallbackReceiverFromDeclText(srcText[match[4]:match[5]])
		}
		name := strings.TrimSpace(srcText[match[6]:match[7]])
		line := 1 + strings.Count(srcText[:match[0]], "\n")
		add(name, receiver, line, countLines(srcText))
	}
	return helpers
}

func newFallbackPackageHelper(absPath string, src []byte, name, receiver string, line, endLine int) (packageHelper, bool) {
	name = strings.TrimSpace(name)
	if name == "" || strings.HasPrefix(name, "Test") {
		return packageHelper{}, false
	}
	if line <= 0 {
		line = 1
	}
	totalLines := countLines(string(src))
	if endLine < line {
		endLine = totalLines
	}

	kind := ast.SymbolFunction
	if receiver != "" {
		kind = ast.SymbolMethod
	}

	return packageHelper{
		key:      helperCacheKeyFromFields(absPath, name, line, endLine),
		name:     name,
		receiver: canonicalProbeReceiver(receiver),
		abs:      absPath,
		src:      src,
		parsed:   nil,
		sym: ast.Symbol{
			Name:     name,
			Kind:     kind,
			Line:     line,
			EndLine:  endLine,
			Exported: isProbeExportedName(name),
		},
	}, true
}

func fallbackReceiverFromFuncDecl(src []byte, fset *token.FileSet, fn *goast.FuncDecl) string {
	if fn == nil || fn.Recv == nil || len(fn.Recv.List) == 0 || fset == nil {
		return ""
	}
	start := fset.Position(fn.Recv.List[0].Type.Pos()).Offset
	end := fset.Position(fn.Recv.List[0].Type.End()).Offset
	if start < 0 || end <= start || end > len(src) {
		return ""
	}
	return string(src[start:end])
}

func fallbackReceiverFromDeclText(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func countLines(src string) int {
	if src == "" {
		return 1
	}
	return strings.Count(src, "\n") + 1
}

func isProbeExportedName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	r := rune(name[0])
	return r >= 'A' && r <= 'Z'
}

func (g *crossPackageHelperGraph) addHelper(helper packageHelper) {
	g.helpers[helper.key] = helper
	g.helperKeysByName[helper.name] = append(g.helperKeysByName[helper.name], helper.key)
}

func (g *crossPackageHelperGraph) matchesSymbol(sym packageHelper, visited map[string]struct{}) bool {
	for _, helperKey := range g.calledHelpers(sym) {
		if g.helperMatchesTarget(helperKey, visited) {
			return true
		}
	}
	if g.matchesForwardedHelperCalls(sym, visited) {
		return true
	}
	return g.matchesImportedPackageHelpers(sym, visited)
}

func (g *crossPackageHelperGraph) helperMatchesTarget(key string, visited map[string]struct{}) bool {
	if key == "" {
		return false
	}
	if _, ok := visited[key]; ok {
		return false
	}
	visited[key] = struct{}{}

	helper, ok := g.helpers[key]
	if !ok {
		return false
	}
	if methodTestBodyMatchesSymbol(g.ctx.matchContext(helper.abs, helper.src), helper.parsed, helper.sym, false) {
		return true
	}
	for _, edge := range g.helperSummary(helper).returns {
		if g.matchesHelperEdge(edge, visited) {
			return true
		}
	}
	return g.matchesSymbol(helper, visited)
}

func (g *crossPackageHelperGraph) matchesHelperEdge(edge helperEdge, visited map[string]struct{}) bool {
	targetGraph := g
	if edge.packageDir != "" && filepath.Clean(edge.packageDir) != filepath.Clean(g.packageDir) {
		targetGraph = g.importedGraph(edge.packageDir)
	}
	if targetGraph == nil {
		return false
	}
	return targetGraph.helperMatchesTarget(edge.key, visited)
}

func (g *crossPackageHelperGraph) matchesImportedPackageHelpers(sym packageHelper, visited map[string]struct{}) bool {
	for qualifier, importDir := range localHelperImportDirs(sym.abs, sym.src, g.ctx.opts) {
		imported := g.importedGraph(importDir)
		if imported == nil {
			continue
		}
		for _, helper := range imported.functionHelpers() {
			if !symbolCallsImportedPackageHelper(sym, qualifier, helper) {
				continue
			}
			if methodTestBodyMatchesSymbol(g.ctx.matchContext(helper.abs, helper.src), helper.parsed, helper.sym, false) {
				return true
			}
			if imported.matchesSymbol(helper, visited) {
				return true
			}
		}
	}
	return false
}

func (g *crossPackageHelperGraph) importedGraph(dir string) *crossPackageHelperGraph {
	dir = filepath.Clean(strings.TrimSpace(dir))
	if dir == "" || dir == filepath.Clean(g.packageDir) {
		return nil
	}
	if graph, ok := g.importedGraphs[dir]; ok {
		return graph
	}
	graph := newCrossPackageHelperGraph(g.ctx, dir, false)
	g.importedGraphs[dir] = graph
	return graph
}

func (g *crossPackageHelperGraph) functionHelpers() []packageHelper {
	helpers := make([]packageHelper, 0, len(g.helpers))
	for _, helper := range g.helpers {
		if helper.receiver != "" {
			continue
		}
		helpers = append(helpers, helper)
	}
	sort.Slice(helpers, func(i, j int) bool { return helpers[i].key < helpers[j].key })
	return helpers
}

func (g *crossPackageHelperGraph) helperSummary(helper packageHelper) helperSummary {
	if summary, ok := g.summaryCache[helper.key]; ok {
		return summary
	}

	summary := helperSummary{}
	decl := g.funcDecl(helper)
	if decl == nil || decl.Body == nil {
		g.summaryCache[helper.key] = summary
		return summary
	}

	paramIndex := make(map[string]int)
	if decl.Type != nil && decl.Type.Params != nil {
		idx := 0
		for _, field := range decl.Type.Params.List {
			for _, name := range field.Names {
				if name == nil {
					idx++
					continue
				}
				paramIndex[name.Name] = idx
				idx++
			}
		}
	}

	returns := make(map[string]helperEdge)
	returnObjects := make(map[string]helperObject)
	resultValues := make(map[int]helperValue)
	paramUses := make(map[string]helperParamUse)
	goast.Inspect(decl.Body, func(n goast.Node) bool {
		switch node := n.(type) {
		case *goast.ReturnStmt:
			for idx, result := range node.Results {
				value := g.resolveExprToHelperValue(helper, result, nil)
				resultValues[idx] = mergeHelperValue(resultValues[idx], value)
				for _, edge := range value.edges {
					returns[edge.packageDir+"|"+edge.key] = edge
				}
				for _, object := range value.objects {
					returnObjects[filepath.Clean(object.packageDir)+"|"+object.receiver] = object
				}
			}
		case *goast.CallExpr:
			switch fun := node.Fun.(type) {
			case *goast.Ident:
				if idx, ok := paramIndex[fun.Name]; ok {
					paramUses[strconv.Itoa(idx)+":"] = helperParamUse{index: idx}
				}
			case *goast.SelectorExpr:
				if ident, ok := fun.X.(*goast.Ident); ok {
					if idx, ok := paramIndex[ident.Name]; ok {
						key := strconv.Itoa(idx) + ":" + fun.Sel.Name
						paramUses[key] = helperParamUse{index: idx, methodName: fun.Sel.Name}
					}
				}
			}
		}
		return true
	})

	if len(returns) > 0 {
		summary.returns = make([]helperEdge, 0, len(returns))
		for _, edge := range returns {
			summary.returns = append(summary.returns, edge)
		}
		sort.Slice(summary.returns, func(i, j int) bool {
			if summary.returns[i].packageDir != summary.returns[j].packageDir {
				return summary.returns[i].packageDir < summary.returns[j].packageDir
			}
			return summary.returns[i].key < summary.returns[j].key
		})
	}
	if len(returnObjects) > 0 {
		summary.returnObjects = make([]helperObject, 0, len(returnObjects))
		for _, object := range returnObjects {
			summary.returnObjects = append(summary.returnObjects, object)
		}
		sort.Slice(summary.returnObjects, func(i, j int) bool {
			if summary.returnObjects[i].packageDir != summary.returnObjects[j].packageDir {
				return summary.returnObjects[i].packageDir < summary.returnObjects[j].packageDir
			}
			return summary.returnObjects[i].receiver < summary.returnObjects[j].receiver
		})
	}
	if len(resultValues) > 0 {
		maxIdx := -1
		for idx := range resultValues {
			if idx > maxIdx {
				maxIdx = idx
			}
		}
		summary.resultValues = make([]helperValue, maxIdx+1)
		for idx, value := range resultValues {
			summary.resultValues[idx] = value
		}
	}
	if len(paramUses) > 0 {
		summary.params = make([]helperParamUse, 0, len(paramUses))
		for _, use := range paramUses {
			summary.params = append(summary.params, use)
		}
		sort.Slice(summary.params, func(i, j int) bool {
			if summary.params[i].index != summary.params[j].index {
				return summary.params[i].index < summary.params[j].index
			}
			return summary.params[i].methodName < summary.params[j].methodName
		})
	}

	g.summaryCache[helper.key] = summary
	return summary
}

func (g *crossPackageHelperGraph) funcDecl(helper packageHelper) *goast.FuncDecl {
	if decl, ok := g.declCache[helper.key]; ok {
		return decl
	}
	parsed := g.goASTFile(helper.abs, helper.src)
	if parsed.file == nil || parsed.fset == nil {
		return nil
	}
	for _, decl := range parsed.file.Decls {
		fn, ok := decl.(*goast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != helper.name {
			continue
		}
		line := parsed.fset.Position(fn.Pos()).Line
		if line != helper.sym.Line {
			continue
		}
		g.declCache[helper.key] = fn
		return fn
	}
	return nil
}

func (g *crossPackageHelperGraph) goASTFile(absPath string, src []byte) parsedGoHelperFile {
	absPath = filepath.Clean(absPath)
	if parsed, ok := g.fileASTCache[absPath]; ok {
		return parsed
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absPath, src, 0)
	if err != nil || file == nil {
		return parsedGoHelperFile{}
	}
	parsed := parsedGoHelperFile{fset: fset, file: file}
	g.fileASTCache[absPath] = parsed
	return parsed
}

func (g *crossPackageHelperGraph) calledHelpers(sym packageHelper) []string {
	key := helperCacheKey(sym)
	if cached, ok := g.callCache[key]; ok {
		return cached
	}

	keys := make([]string, 0, len(g.helpers))
	for _, helper := range g.helpers {
		if helper.key == sym.key {
			continue
		}
		if symbolCallsPackageHelper(sym, helper) {
			keys = append(keys, helper.key)
		}
	}
	sort.Strings(keys)
	g.callCache[key] = keys
	return keys
}

func helperCacheKey(sym packageHelper) string {
	if sym.key != "" {
		return sym.key
	}
	return helperCacheKeyFromFields(sym.abs, sym.name, sym.sym.Line, sym.sym.EndLine)
}

func helperCacheKeyFromFields(absPath, name string, line, endLine int) string {
	return strings.Join([]string{
		filepath.Clean(absPath),
		name,
		strconv.Itoa(line),
		strconv.Itoa(endLine),
	}, ":")
}

func structuredGoImpactProbeRootPath(opts SearchOptions, fallback string) string {
	if root := strings.TrimSpace(opts.ProjectMapRootPath); root != "" {
		return root
	}
	path := strings.TrimSpace(opts.Path)
	if path == "" {
		return fallback
	}
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return filepath.Dir(path)
	}
	return path
}

func symbolBodyFromLines(src []byte, sym ast.Symbol) string {
	lines := strings.Split(string(src), "\n")
	start := sym.Line - 1
	if start < 0 {
		start = 0
	}
	end := min(len(lines), sym.EndLine)
	if start >= end {
		return ""
	}
	return strings.Join(lines[start:end], "\n")
}

func findCrossPackageMethodProbeTestSymbol(absPath string, src []byte, test navigation.TestRef) (ast.Symbol, bool) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, absPath, src, 0)
	if err != nil || file == nil {
		return ast.Symbol{}, false
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*goast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != test.Name {
			continue
		}

		line := fset.Position(fn.Pos()).Line
		if test.Line > 0 && line != test.Line {
			continue
		}

		return ast.Symbol{
			Name:     test.Name,
			Kind:     ast.SymbolFunction,
			Line:     line,
			EndLine:  fset.Position(fn.End()).Line,
			Exported: true,
		}, true
	}

	return ast.Symbol{}, false
}
