package search

import (
	goast "go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
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

type crossPackageHelperGraphKey struct {
	packageDir     string
	allowTestFiles bool
}

type crossPackageHelperGraphCache struct {
	graphs map[crossPackageHelperGraphKey]*crossPackageHelperGraph
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
	graphCache       *crossPackageHelperGraphCache
}

type goMethodCrossPackageTestMatcher struct {
	ctx        goMethodTestProbeContext
	testFiles  map[string]crossPackageMethodTestFile
	graphCache *crossPackageHelperGraphCache
}

type crossPackageMethodTestFile struct {
	src    []byte
	parsed *ast.ParsedFile
	valid  bool
}

func newGoMethodCrossPackageTestMatcher(ctx goMethodTestProbeContext) *goMethodCrossPackageTestMatcher {
	return &goMethodCrossPackageTestMatcher{
		ctx:        ctx,
		testFiles:  make(map[string]crossPackageMethodTestFile),
		graphCache: newCrossPackageHelperGraphCache(),
	}
}

func (m *goMethodCrossPackageTestMatcher) matches(absPath string, test navigation.TestRef) bool {
	src, parsed, ok := m.testFile(absPath)
	if !ok {
		return false
	}
	testSymbol, ok := findCrossPackageMethodProbeTestSymbol(absPath, src, test)
	if !ok {
		return false
	}
	if methodTestBodyMatchesSymbol(m.ctx.matchContext(absPath, src), parsed, testSymbol, false) {
		return true
	}
	graph := newCrossPackageHelperGraphWithCache(m.ctx, filepath.Dir(absPath), true, m.graphCache)
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

func (m *goMethodCrossPackageTestMatcher) testFile(absPath string) ([]byte, *ast.ParsedFile, bool) {
	absPath = filepath.Clean(strings.TrimSpace(absPath))
	if absPath == "" {
		return nil, nil, false
	}
	if cached, ok := m.testFiles[absPath]; ok {
		return cached.src, cached.parsed, cached.valid
	}

	m.ctx.dependencies.add(absPath)
	src, err := os.ReadFile(absPath)
	if err != nil {
		m.testFiles[absPath] = crossPackageMethodTestFile{}
		return nil, nil, false
	}
	parsed, err := ast.ParseBytesForReuse(absPath, src)
	if err != nil {
		m.testFiles[absPath] = crossPackageMethodTestFile{}
		return nil, nil, false
	}

	cached := crossPackageMethodTestFile{src: src, parsed: parsed, valid: true}
	m.testFiles[absPath] = cached
	return cached.src, cached.parsed, cached.valid
}

func newCrossPackageHelperGraphCache() *crossPackageHelperGraphCache {
	return &crossPackageHelperGraphCache{graphs: make(map[crossPackageHelperGraphKey]*crossPackageHelperGraph)}
}

func (c *crossPackageHelperGraphCache) graph(ctx goMethodTestProbeContext, packageDir string, allowTestFiles bool) *crossPackageHelperGraph {
	if c == nil {
		return buildCrossPackageHelperGraph(ctx, packageDir, allowTestFiles, nil)
	}
	key := crossPackageHelperGraphKey{
		packageDir:     cleanCrossPackageHelperGraphDir(packageDir),
		allowTestFiles: allowTestFiles,
	}
	if graph, ok := c.graphs[key]; ok {
		return graph
	}
	graph := buildCrossPackageHelperGraph(ctx, key.packageDir, allowTestFiles, c)
	c.graphs[key] = graph
	return graph
}

func newCrossPackageHelperGraphWithCache(ctx goMethodTestProbeContext, packageDir string, allowTestFiles bool, cache *crossPackageHelperGraphCache) *crossPackageHelperGraph {
	if cache == nil {
		cache = newCrossPackageHelperGraphCache()
	}
	return cache.graph(ctx, packageDir, allowTestFiles)
}

func buildCrossPackageHelperGraph(ctx goMethodTestProbeContext, packageDir string, allowTestFiles bool, cache *crossPackageHelperGraphCache) *crossPackageHelperGraph {
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
		graphCache:       cache,
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

func cleanCrossPackageHelperGraphDir(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return ""
	}
	return filepath.Clean(dir)
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
	graph := newCrossPackageHelperGraphWithCache(g.ctx, dir, false, g.graphCache)
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
