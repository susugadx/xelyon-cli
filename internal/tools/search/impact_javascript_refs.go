package search

type javaScriptImpactRefs struct {
	imports          []genericSymbolRef
	callers          []genericSymbolRef
	others           []genericSymbolRef
	directTests      []genericSymbolRef
	nearbyTests      []genericSymbolRef
	hasTotalRefs     bool
	totalImports     []genericSymbolRef
	totalCallers     []genericSymbolRef
	totalOthers      []genericSymbolRef
	totalDirectTests []genericSymbolRef
}

func javaScriptImpactRefsForDisplayAndTotalRefs(def genericSymbolDef, refs []genericSymbolRef, totalRefs []genericSymbolRef, opts SearchOptions) javaScriptImpactRefs {
	classified := classifyJSFamilySymbolRefsFromAST(refs)
	totalClassified := classifyJSFamilySymbolRefsFromAST(totalRefs)
	result := javaScriptImpactRefs{
		imports:          classified.imports,
		callers:          classified.callers,
		others:           classified.others,
		directTests:      classified.tests,
		totalImports:     totalClassified.imports,
		totalCallers:     totalClassified.callers,
		totalOthers:      totalClassified.others,
		totalDirectTests: totalClassified.tests,
		hasTotalRefs:     true,
	}
	result.nearbyTests = findNearbyJavaScriptTests(def, opts, result.directTests)
	return result
}

func (refs javaScriptImpactRefs) allTests() []genericSymbolRef {
	return allJSFamilyTests(refs.directTests, refs.nearbyTests)
}

func (refs javaScriptImpactRefs) allTotalTests() []genericSymbolRef {
	if !refs.hasTotalRefs {
		return refs.allTests()
	}
	return allJSFamilyTests(refs.totalDirectTests, refs.nearbyTests)
}

func (refs javaScriptImpactRefs) totalImportsForRisk() []genericSymbolRef {
	if refs.hasTotalRefs {
		return refs.totalImports
	}
	return refs.imports
}

func (refs javaScriptImpactRefs) totalCallersForRisk() []genericSymbolRef {
	if refs.hasTotalRefs {
		return refs.totalCallers
	}
	return refs.callers
}

func (refs javaScriptImpactRefs) totalOthersForRisk() []genericSymbolRef {
	if refs.hasTotalRefs {
		return refs.totalOthers
	}
	return refs.others
}

func findNearbyJavaScriptTests(def genericSymbolDef, opts SearchOptions, directTests []genericSymbolRef) []genericSymbolRef {
	rootPath := structuredJavaScriptImpactFileRoot(opts)
	target, ok := structuredJavaScriptImpactTargetForPath(def.File)
	if rootPath == "" || !ok {
		return nil
	}

	return findNearbyJSFamilyTests(rootPath, javaScriptNearbyTestCandidatePaths(def.File), directTests, func(absPath string, displayPath string) bool {
		return isUsableNearbyJavaScriptTest(absPath, displayPath, opts, target)
	})
}

func javaScriptNearbyTestCandidatePaths(defFile string) []string {
	target, ok := structuredJavaScriptImpactTargetForPath(defFile)
	if !ok {
		return nil
	}
	return target.nearbyTestCandidatePaths(defFile)
}

func isUsableNearbyJavaScriptTest(absPath, displayPath string, opts SearchOptions, target structuredJavaScriptImpactTarget) bool {
	if !target.matchesNearbyTestPath(displayPath) {
		return false
	}
	return isUsableNearbyJSFamilyTest(absPath, displayPath, opts)
}
