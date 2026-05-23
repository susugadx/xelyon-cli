package search

type typeScriptImpactRefs struct {
	imports          []genericSymbolRef
	callers          []genericSymbolRef
	typeRefs         []genericSymbolRef
	others           []genericSymbolRef
	directTests      []genericSymbolRef
	nearbyTests      []genericSymbolRef
	hasTotalRefs     bool
	totalImports     []genericSymbolRef
	totalCallers     []genericSymbolRef
	totalTypeRefs    []genericSymbolRef
	totalOthers      []genericSymbolRef
	totalDirectTests []genericSymbolRef
}

func typeScriptImpactRefsForDisplayAndTotalRefs(def genericSymbolDef, refs []genericSymbolRef, totalRefs []genericSymbolRef, opts SearchOptions) typeScriptImpactRefs {
	classified := classifyJSFamilySymbolRefsFromAST(refs)
	totalClassified := classifyJSFamilySymbolRefsFromAST(totalRefs)
	result := typeScriptImpactRefs{
		imports:          classified.imports,
		callers:          classified.callers,
		typeRefs:         classified.typeRefs,
		others:           classified.others,
		directTests:      classified.tests,
		totalImports:     totalClassified.imports,
		totalCallers:     totalClassified.callers,
		totalTypeRefs:    totalClassified.typeRefs,
		totalOthers:      totalClassified.others,
		totalDirectTests: totalClassified.tests,
		hasTotalRefs:     true,
	}
	result.nearbyTests = findNearbyTypeScriptTests(def, opts, result.directTests)
	return result
}

func (refs typeScriptImpactRefs) allTests() []genericSymbolRef {
	return allJSFamilyTests(refs.directTests, refs.nearbyTests)
}

func (refs typeScriptImpactRefs) allTotalTests() []genericSymbolRef {
	if !refs.hasTotalRefs {
		return refs.allTests()
	}
	return allJSFamilyTests(refs.totalDirectTests, refs.nearbyTests)
}

func (refs typeScriptImpactRefs) totalImportsForRisk() []genericSymbolRef {
	if refs.hasTotalRefs {
		return refs.totalImports
	}
	return refs.imports
}

func (refs typeScriptImpactRefs) totalCallersForRisk() []genericSymbolRef {
	if refs.hasTotalRefs {
		return refs.totalCallers
	}
	return refs.callers
}

func (refs typeScriptImpactRefs) totalTypeRefsForRisk() []genericSymbolRef {
	if refs.hasTotalRefs {
		return refs.totalTypeRefs
	}
	return refs.typeRefs
}

func (refs typeScriptImpactRefs) totalOthersForRisk() []genericSymbolRef {
	if refs.hasTotalRefs {
		return refs.totalOthers
	}
	return refs.others
}

func findNearbyTypeScriptTests(def genericSymbolDef, opts SearchOptions, directTests []genericSymbolRef) []genericSymbolRef {
	rootPath := structuredTypeScriptImpactFileRoot(opts)
	target, ok := structuredTypeScriptImplementationTargetForPath(def.File)
	if rootPath == "" || !ok {
		return nil
	}

	return findNearbyJSFamilyTests(rootPath, typeScriptNearbyTestCandidatePaths(def.File), directTests, func(absPath string, displayPath string) bool {
		return isUsableNearbyTypeScriptTest(absPath, displayPath, opts, target)
	})
}

func typeScriptNearbyTestCandidatePaths(defFile string) []string {
	target, ok := structuredTypeScriptImplementationTargetForPath(defFile)
	if !ok {
		return nil
	}
	return target.nearbyTestCandidatePaths(defFile)
}

func isUsableNearbyTypeScriptTest(absPath, displayPath string, opts SearchOptions, target structuredTypeScriptImpactTarget) bool {
	if !target.matchesNearbyTestPath(displayPath) {
		return false
	}
	return isUsableNearbyJSFamilyTest(absPath, displayPath, opts)
}
