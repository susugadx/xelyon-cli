package search

import (
	goast "go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

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
