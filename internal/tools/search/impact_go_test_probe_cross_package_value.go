package search

import (
	goast "go/ast"
	"path/filepath"
	"sort"
	"strings"
)

func (g *crossPackageHelperGraph) resolveExprToHelperValue(sym packageHelper, expr goast.Expr, env map[string]helperValue) helperValue {
	switch node := expr.(type) {
	case *goast.Ident:
		if value, ok := env[node.Name]; ok {
			return value
		}
		return helperValue{edges: g.helperEdgesByName("", node.Name)}
	case *goast.SelectorExpr:
		if pkgIdent, ok := node.X.(*goast.Ident); ok {
			if importDir, ok := localHelperImportDirs(sym.abs, sym.src, g.ctx.opts)[pkgIdent.Name]; ok {
				return helperValue{
					edges:     g.helperEdgesByName(importDir, node.Sel.Name),
					objects:   []helperObject{{packageDir: importDir, receiver: node.Sel.Name}},
					uncertain: false,
				}
			}
		}
		base := g.resolveExprToHelperValue(sym, node.X, env)
		value := helperValue{
			edges:     g.edgesForHelperObjects(base.objects, node.Sel.Name),
			objects:   dedupeHelperObjects(append([]helperObject(nil), base.objects...)),
			uncertain: base.uncertain,
		}
		if isDynamicSelectorName(node.Sel.Name) {
			value.uncertain = true
		}
		return value
	case *goast.IndexExpr:
		base := g.resolveExprToHelperValue(sym, node.X, env)
		if len(base.edges) > 0 || len(base.objects) > 0 || base.uncertain {
			base.uncertain = true
			return base
		}
		return helperValue{uncertain: true}
	case *goast.IndexListExpr:
		base := g.resolveExprToHelperValue(sym, node.X, env)
		if len(base.edges) > 0 || len(base.objects) > 0 || base.uncertain {
			base.uncertain = true
			return base
		}
		return helperValue{uncertain: true}
	case *goast.SliceExpr:
		base := g.resolveExprToHelperValue(sym, node.X, env)
		base.uncertain = true
		return base
	case *goast.TypeAssertExpr:
		base := g.resolveExprToHelperValue(sym, node.X, env)
		base.uncertain = true
		return base
	case *goast.CallExpr:
		if isReflectValueOfCall(node) && len(node.Args) == 1 {
			return g.resolveExprToHelperValue(sym, node.Args[0], env)
		}
		if isDynamicCallFactory(node) {
			value := helperValue{uncertain: true}
			value = mergeHelperValue(value, g.resolveExprToHelperValue(sym, node.Fun, env))
			for _, arg := range node.Args {
				value = mergeHelperValue(value, g.resolveExprToHelperValue(sym, arg, env))
			}
			return value
		}
		callee := g.resolveExprToHelperValue(sym, node.Fun, env)
		value := helperValue{
			edges:     g.returnEdgesForValues(callee),
			objects:   g.returnObjectsForValues(callee),
			uncertain: callee.uncertain,
		}
		if len(node.Args) == 1 {
			arg := g.resolveExprToHelperValue(sym, node.Args[0], env)
			if len(value.edges) == 0 && len(value.objects) == 0 && arg.uncertain {
				value.uncertain = true
			}
		}
		return value
	case *goast.CompositeLit:
		value := helperValue{objects: g.resolveTypeExprToHelperObjects(sym, node.Type)}
		for _, elt := range node.Elts {
			expr := elt
			if kv, ok := elt.(*goast.KeyValueExpr); ok {
				expr = kv.Value
			}
			value = mergeHelperValue(value, g.resolveExprToHelperValue(sym, expr, env))
		}
		return value
	case *goast.StarExpr:
		return g.resolveExprToHelperValue(sym, node.X, env)
	case *goast.UnaryExpr:
		base := g.resolveExprToHelperValue(sym, node.X, env)
		if node.Op.String() == "<-" {
			base.uncertain = true
		}
		return base
	case *goast.ParenExpr:
		return g.resolveExprToHelperValue(sym, node.X, env)
	}
	return helperValue{}
}

func (g *crossPackageHelperGraph) resolveExprToHelperValues(sym packageHelper, expr goast.Expr, env map[string]helperValue) []helperValue {
	call, ok := expr.(*goast.CallExpr)
	if !ok {
		return nil
	}
	if isDynamicCallFactory(call) {
		return nil
	}
	callee := g.resolveExprToHelperValue(sym, call.Fun, env)
	return g.returnValuesForHelperValue(callee)
}

func (g *crossPackageHelperGraph) returnEdgesForValues(value helperValue) []helperEdge {
	var edges []helperEdge
	for _, edge := range value.edges {
		targetGraph := g
		if edge.packageDir != "" && filepath.Clean(edge.packageDir) != filepath.Clean(g.packageDir) {
			targetGraph = g.importedGraph(edge.packageDir)
		}
		if targetGraph == nil {
			continue
		}
		helper, ok := targetGraph.helpers[edge.key]
		if !ok {
			continue
		}
		edges = append(edges, targetGraph.helperSummary(helper).returns...)
	}
	return dedupeHelperEdges(edges)
}

func (g *crossPackageHelperGraph) returnValuesForHelperValue(value helperValue) []helperValue {
	if len(value.edges) == 0 {
		return nil
	}

	var merged []helperValue
	for _, edge := range value.edges {
		targetGraph := g
		if edge.packageDir != "" && filepath.Clean(edge.packageDir) != filepath.Clean(g.packageDir) {
			targetGraph = g.importedGraph(edge.packageDir)
		}
		if targetGraph == nil {
			continue
		}
		helper, ok := targetGraph.helpers[edge.key]
		if !ok {
			continue
		}
		values := targetGraph.helperSummary(helper).resultValues
		if len(values) == 0 {
			continue
		}
		if len(merged) < len(values) {
			extra := make([]helperValue, len(values)-len(merged))
			merged = append(merged, extra...)
		}
		for i, result := range values {
			merged[i] = mergeHelperValue(merged[i], result)
		}
	}

	if len(merged) == 0 {
		return nil
	}
	return merged
}

func (g *crossPackageHelperGraph) returnObjectsForValues(value helperValue) []helperObject {
	var objects []helperObject
	for _, edge := range value.edges {
		targetGraph := g
		if edge.packageDir != "" && filepath.Clean(edge.packageDir) != filepath.Clean(g.packageDir) {
			targetGraph = g.importedGraph(edge.packageDir)
		}
		if targetGraph == nil {
			continue
		}
		helper, ok := targetGraph.helpers[edge.key]
		if !ok {
			continue
		}
		objects = append(objects, targetGraph.helperSummary(helper).returnObjects...)
	}
	return dedupeHelperObjects(objects)
}

func (g *crossPackageHelperGraph) uncertainValueMatchesTarget(sym packageHelper, value helperValue, args []goast.Expr, env map[string]helperValue, visited map[string]struct{}) bool {
	if g.helperValueMayReachTarget(value, visited) {
		return true
	}
	for _, arg := range args {
		if g.helperValueMayReachTarget(g.resolveExprToHelperValue(sym, arg, env), visited) {
			return true
		}
	}
	return false
}

func (g *crossPackageHelperGraph) helperValueMayReachTarget(value helperValue, visited map[string]struct{}) bool {
	for _, object := range value.objects {
		if g.helperObjectMatchesTarget(object) {
			return true
		}
	}
	for _, edge := range value.edges {
		if g.matchesHelperEdge(edge, visited) {
			return true
		}
	}
	return false
}

func (g *crossPackageHelperGraph) helperObjectMatchesTarget(object helperObject) bool {
	targetReceiver := strings.TrimSpace(g.ctx.receiver)
	if targetReceiver == "" || strings.TrimSpace(object.receiver) != targetReceiver {
		return false
	}

	targetPackageDir := filepath.Clean(strings.TrimSpace(g.ctx.targetPackageDir))
	objectPackageDir := filepath.Clean(strings.TrimSpace(object.packageDir))
	switch {
	case targetPackageDir == "" || targetPackageDir == ".":
		return true
	case objectPackageDir == "" || objectPackageDir == ".":
		return true
	default:
		return objectPackageDir == targetPackageDir
	}
}

func (g *crossPackageHelperGraph) resolveTypeExprToHelperObjects(sym packageHelper, expr goast.Expr) []helperObject {
	switch node := expr.(type) {
	case *goast.Ident:
		if hasHelperReceiver(g, node.Name) {
			return []helperObject{{packageDir: g.packageDir, receiver: node.Name}}
		}
	case *goast.SelectorExpr:
		if pkgIdent, ok := node.X.(*goast.Ident); ok {
			if importDir, ok := localHelperImportDirs(sym.abs, sym.src, g.ctx.opts)[pkgIdent.Name]; ok {
				if imported := g.importedGraph(importDir); imported != nil && hasHelperReceiver(imported, node.Sel.Name) {
					return []helperObject{{packageDir: importDir, receiver: node.Sel.Name}}
				}
			}
		}
	case *goast.StarExpr:
		return g.resolveTypeExprToHelperObjects(sym, node.X)
	case *goast.IndexExpr:
		return g.resolveTypeExprToHelperObjects(sym, node.X)
	case *goast.IndexListExpr:
		return g.resolveTypeExprToHelperObjects(sym, node.X)
	case *goast.ParenExpr:
		return g.resolveTypeExprToHelperObjects(sym, node.X)
	}
	return nil
}

func (g *crossPackageHelperGraph) edgesForHelperObjects(objects []helperObject, methodName string) []helperEdge {
	var edges []helperEdge
	for _, object := range objects {
		targetGraph := g
		if object.packageDir != "" && filepath.Clean(object.packageDir) != filepath.Clean(g.packageDir) {
			targetGraph = g.importedGraph(object.packageDir)
		}
		if targetGraph == nil {
			continue
		}
		edges = append(edges, targetGraph.helperEdgesByReceiver(object.receiver, methodName)...)
	}
	return dedupeHelperEdges(edges)
}

func (g *crossPackageHelperGraph) helperEdgesByName(packageDir, name string) []helperEdge {
	targetGraph := g
	if packageDir != "" && filepath.Clean(packageDir) != filepath.Clean(g.packageDir) {
		targetGraph = g.importedGraph(packageDir)
	}
	if targetGraph == nil {
		return nil
	}
	keys := targetGraph.helperKeysByName[name]
	if len(keys) == 0 {
		return nil
	}
	edges := make([]helperEdge, 0, len(keys))
	for _, key := range keys {
		helper := targetGraph.helpers[key]
		if helper.receiver != "" {
			continue
		}
		edges = append(edges, helperEdge{packageDir: targetGraph.packageDir, key: key})
	}
	return edges
}

func (g *crossPackageHelperGraph) helperEdgesByReceiver(receiver, methodName string) []helperEdge {
	receiver = strings.TrimSpace(receiver)
	methodName = strings.TrimSpace(methodName)
	if receiver == "" || methodName == "" {
		return nil
	}
	var edges []helperEdge
	for _, key := range g.helperKeysByName[methodName] {
		helper := g.helpers[key]
		if helper.receiver != receiver {
			continue
		}
		edges = append(edges, helperEdge{packageDir: g.packageDir, key: key})
	}
	return edges
}

func hasHelperReceiver(graph *crossPackageHelperGraph, receiver string) bool {
	receiver = strings.TrimSpace(receiver)
	if graph == nil || receiver == "" {
		return false
	}
	for _, helper := range graph.helpers {
		if helper.receiver == receiver {
			return true
		}
	}
	return false
}

func dedupeHelperEdges(edges []helperEdge) []helperEdge {
	if len(edges) == 0 {
		return nil
	}
	seen := make(map[string]helperEdge, len(edges))
	for _, edge := range edges {
		seen[filepath.Clean(edge.packageDir)+"|"+edge.key] = edge
	}
	out := make([]helperEdge, 0, len(seen))
	for _, edge := range seen {
		out = append(out, edge)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].packageDir != out[j].packageDir {
			return out[i].packageDir < out[j].packageDir
		}
		return out[i].key < out[j].key
	})
	return out
}

func dedupeHelperObjects(objects []helperObject) []helperObject {
	if len(objects) == 0 {
		return nil
	}
	seen := make(map[string]helperObject, len(objects))
	for _, object := range objects {
		seen[filepath.Clean(object.packageDir)+"|"+object.receiver] = object
	}
	out := make([]helperObject, 0, len(seen))
	for _, object := range seen {
		out = append(out, object)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].packageDir != out[j].packageDir {
			return out[i].packageDir < out[j].packageDir
		}
		return out[i].receiver < out[j].receiver
	})
	return out
}

func isReflectValueOfCall(call *goast.CallExpr) bool {
	selector, ok := call.Fun.(*goast.SelectorExpr)
	if !ok || selector.Sel == nil || selector.Sel.Name != "ValueOf" {
		return false
	}
	ident, ok := selector.X.(*goast.Ident)
	return ok && ident.Name == "reflect"
}

func isDynamicSelectorName(name string) bool {
	switch strings.TrimSpace(name) {
	case "MethodByName", "Lookup", "Load", "Open", "Call":
		return true
	default:
		return false
	}
}

func isDynamicCallFactory(call *goast.CallExpr) bool {
	selector, ok := call.Fun.(*goast.SelectorExpr)
	if !ok || selector.Sel == nil {
		return false
	}
	name := strings.TrimSpace(selector.Sel.Name)
	switch name {
	case "MethodByName", "Lookup":
		return true
	case "Open":
		if ident, ok := selector.X.(*goast.Ident); ok && ident.Name == "plugin" {
			return true
		}
	case "Pointer":
		if ident, ok := selector.X.(*goast.Ident); ok && ident.Name == "unsafe" {
			return true
		}
	}
	return false
}
