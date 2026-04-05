package search

import (
	goast "go/ast"
	"path/filepath"
)

func (g *crossPackageHelperGraph) matchesForwardedHelperCalls(sym packageHelper, visited map[string]struct{}) bool {
	decl := g.funcDecl(sym)
	if decl == nil || decl.Body == nil {
		return false
	}
	return g.matchesForwardedStmtList(sym, decl.Body.List, make(map[string]helperValue), visited)
}

func (g *crossPackageHelperGraph) matchesForwardedStmtList(sym packageHelper, stmts []goast.Stmt, env map[string]helperValue, visited map[string]struct{}) bool {
	for _, stmt := range stmts {
		if g.matchesForwardedStmt(sym, stmt, env, visited) {
			return true
		}
	}
	return false
}

func (g *crossPackageHelperGraph) matchesForwardedStmt(sym packageHelper, stmt goast.Stmt, env map[string]helperValue, visited map[string]struct{}) bool {
	switch node := stmt.(type) {
	case *goast.BlockStmt:
		return g.matchesForwardedStmtList(sym, node.List, cloneHelperEnv(env), visited)
	case *goast.ExprStmt:
		return g.matchesForwardedExpr(sym, node.X, env, visited)
	case *goast.AssignStmt:
		for _, rhs := range node.Rhs {
			if g.matchesForwardedExpr(sym, rhs, env, visited) {
				return true
			}
		}
		if len(node.Rhs) == 1 && len(node.Lhs) > 1 {
			values := g.resolveExprToHelperValues(sym, node.Rhs[0], env)
			if len(values) > 0 {
				for i, lhs := range node.Lhs {
					ident, ok := lhs.(*goast.Ident)
					if !ok || ident.Name == "_" {
						continue
					}
					if i < len(values) {
						env[ident.Name] = values[i]
					}
				}
				return false
			}
		}
		for i, lhs := range node.Lhs {
			ident, ok := lhs.(*goast.Ident)
			if !ok || ident.Name == "_" {
				continue
			}
			if i < len(node.Rhs) {
				env[ident.Name] = g.resolveExprToHelperValue(sym, node.Rhs[i], env)
			}
		}
	case *goast.DeclStmt:
		gen, ok := node.Decl.(*goast.GenDecl)
		if !ok {
			return false
		}
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*goast.ValueSpec)
			if !ok {
				continue
			}
			for _, value := range valueSpec.Values {
				if g.matchesForwardedExpr(sym, value, env, visited) {
					return true
				}
			}
			if len(valueSpec.Values) == 1 && len(valueSpec.Names) > 1 {
				values := g.resolveExprToHelperValues(sym, valueSpec.Values[0], env)
				if len(values) > 0 {
					for idx, name := range valueSpec.Names {
						if name == nil || name.Name == "_" {
							continue
						}
						if idx < len(values) {
							env[name.Name] = values[idx]
						}
					}
					continue
				}
			}
			for idx, name := range valueSpec.Names {
				if name == nil || name.Name == "_" {
					continue
				}
				if idx < len(valueSpec.Values) {
					env[name.Name] = g.resolveExprToHelperValue(sym, valueSpec.Values[idx], env)
					continue
				}
				if valueSpec.Type != nil {
					env[name.Name] = helperValue{objects: g.resolveTypeExprToHelperObjects(sym, valueSpec.Type)}
				}
			}
		}
	case *goast.ReturnStmt:
		for _, result := range node.Results {
			if g.matchesForwardedExpr(sym, result, env, visited) {
				return true
			}
		}
	case *goast.IfStmt:
		if node.Init != nil && g.matchesForwardedStmt(sym, node.Init, cloneHelperEnv(env), visited) {
			return true
		}
		thenEnv := cloneHelperEnv(env)
		if g.matchesForwardedStmtList(sym, node.Body.List, thenEnv, visited) {
			return true
		}
		elseEnv := cloneHelperEnv(env)
		if node.Else != nil {
			if g.matchesForwardedStmt(sym, node.Else, elseEnv, visited) {
				return true
			}
		}
		mergeHelperEnvInto(env, thenEnv)
		mergeHelperEnvInto(env, elseEnv)
	case *goast.ForStmt:
		if node.Init != nil && g.matchesForwardedStmt(sym, node.Init, cloneHelperEnv(env), visited) {
			return true
		}
		if node.Body != nil {
			loopEnv := cloneHelperEnv(env)
			if g.matchesForwardedStmtList(sym, node.Body.List, loopEnv, visited) {
				return true
			}
			mergeHelperEnvInto(env, loopEnv)
		}
	case *goast.RangeStmt:
		if g.matchesForwardedExpr(sym, node.X, env, visited) {
			return true
		}
		if node.Body != nil {
			rangeEnv := cloneHelperEnv(env)
			if g.matchesForwardedStmtList(sym, node.Body.List, rangeEnv, visited) {
				return true
			}
			mergeHelperEnvInto(env, rangeEnv)
		}
	case *goast.SwitchStmt:
		if node.Init != nil && g.matchesForwardedStmt(sym, node.Init, cloneHelperEnv(env), visited) {
			return true
		}
		merged := cloneHelperEnv(env)
		for _, clause := range node.Body.List {
			cc, ok := clause.(*goast.CaseClause)
			if !ok {
				continue
			}
			caseEnv := cloneHelperEnv(env)
			if g.matchesForwardedStmtList(sym, cc.Body, caseEnv, visited) {
				return true
			}
			mergeHelperEnvInto(merged, caseEnv)
		}
		mergeHelperEnvInto(env, merged)
	case *goast.TypeSwitchStmt:
		if node.Init != nil && g.matchesForwardedStmt(sym, node.Init, cloneHelperEnv(env), visited) {
			return true
		}
		merged := cloneHelperEnv(env)
		for _, clause := range node.Body.List {
			cc, ok := clause.(*goast.CaseClause)
			if !ok {
				continue
			}
			caseEnv := cloneHelperEnv(env)
			if g.matchesForwardedStmtList(sym, cc.Body, caseEnv, visited) {
				return true
			}
			mergeHelperEnvInto(merged, caseEnv)
		}
		mergeHelperEnvInto(env, merged)
	}
	return false
}

func (g *crossPackageHelperGraph) matchesForwardedExpr(sym packageHelper, expr goast.Expr, env map[string]helperValue, visited map[string]struct{}) bool {
	call, ok := expr.(*goast.CallExpr)
	if !ok {
		return false
	}

	if selector, ok := call.Fun.(*goast.SelectorExpr); ok && selector.Sel != nil && selector.Sel.Name == "Call" {
		value := g.resolveExprToHelperValue(sym, selector.X, env)
		if value.uncertain && g.uncertainValueMatchesTarget(sym, value, nil, env, visited) {
			return true
		}
		for _, edge := range value.edges {
			if g.matchesHelperEdge(edge, visited) {
				return true
			}
		}
	}

	calleeValue := g.resolveExprToHelperValue(sym, call.Fun, env)
	if calleeValue.uncertain && g.uncertainValueMatchesTarget(sym, calleeValue, call.Args, env, visited) {
		return true
	}
	for _, edge := range calleeValue.edges {
		if g.matchesHelperEdge(edge, visited) {
			return true
		}
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
		for _, use := range targetGraph.helperSummary(helper).params {
			if use.index >= len(call.Args) {
				continue
			}
			argValue := g.resolveExprToHelperValue(sym, call.Args[use.index], env)
			if argValue.uncertain && g.uncertainValueMatchesTarget(sym, argValue, nil, env, visited) {
				return true
			}
			for _, forwarded := range argValue.edges {
				if g.matchesHelperEdge(forwarded, visited) {
					return true
				}
			}
			if use.methodName == "" {
				continue
			}
			for _, forwarded := range g.edgesForHelperObjects(argValue.objects, use.methodName) {
				if g.matchesHelperEdge(forwarded, visited) {
					return true
				}
			}
		}
	}

	for _, arg := range call.Args {
		if g.matchesForwardedExpr(sym, arg, env, visited) {
			return true
		}
	}
	return false
}

func cloneHelperEnv(env map[string]helperValue) map[string]helperValue {
	cloned := make(map[string]helperValue, len(env))
	for key, value := range env {
		cloned[key] = value
	}
	return cloned
}

func mergeHelperEnvInto(dst, src map[string]helperValue) {
	for key, value := range src {
		dst[key] = mergeHelperValue(dst[key], value)
	}
}

func mergeHelperValue(a, b helperValue) helperValue {
	return helperValue{
		edges:     dedupeHelperEdges(append(append([]helperEdge(nil), a.edges...), b.edges...)),
		objects:   dedupeHelperObjects(append(append([]helperObject(nil), a.objects...), b.objects...)),
		uncertain: a.uncertain || b.uncertain,
	}
}
