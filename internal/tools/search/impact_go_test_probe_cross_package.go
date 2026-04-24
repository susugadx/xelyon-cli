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
