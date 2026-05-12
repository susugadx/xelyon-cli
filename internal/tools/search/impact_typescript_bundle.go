package search

import (
	"fmt"
	"strings"
)

const (
	typeScriptImpactRecommendedReadPerGroupLimit  = 1
	typeScriptImpactHighNonTestReferenceThreshold = 8
	typeScriptImpactMediumReferenceThreshold      = 4
)

func buildTypeScriptImpactBundle(symbol string, def genericSymbolDef, opts SearchOptions, refs typeScriptImpactRefs) *SymbolBundle {
	rootPath := structuredTypeScriptImpactFileRoot(opts)
	impact := buildTypeScriptImpactMetadata(def, refs, rootPath)
	if impact == nil || len(impact.RecommendedReads) == 0 {
		return nil
	}

	displayName := def.Name
	if displayName == "" {
		displayName = symbol
	}
	bundle := &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Language:    "typescript",
			Query:       symbol,
			Canonical:   canonicalSymbolBundleKey("typescript", def.File, def.Line, displayName),
			DisplayName: displayName,
			Kind:        def.Kind,
			File:        def.File,
			Line:        def.Line,
			EndLine:     def.Line,
		},
		Definition: SymbolBundleDefinition{
			File:      def.File,
			Line:      def.Line,
			EndLine:   def.Line,
			Signature: def.Signature,
			Body:      []string{fmt.Sprintf("%d: %s", def.Line, def.Signature)},
		},
		Impact: impact,
		Debug: SymbolBundleDebug{
			Source:       typeScriptImpactDebugSource(def),
			FileRootPath: rootPath,
		},
	}

	appendJSFamilyImpactSection(bundle, def, "imports", "Imports", refs.imports, jsImportLimit, false, rootPath, symbol)
	appendJSFamilyImpactSection(bundle, def, "callers", "Callers", refs.callers, jsCallerLimit, false, rootPath, symbol)
	appendJSFamilyImpactSection(bundle, def, "type_refs", "Type References", refs.typeRefs, jsTypeRefLimit, false, rootPath, symbol)
	appendJSFamilyImpactSection(bundle, def, "references", "References", refs.others, genericRefLimit, false, rootPath, symbol)
	appendJSFamilyImpactSection(bundle, def, "tests", "Related Tests", refs.allTests(), genericTestLimit, true, rootPath, symbol)

	return bundle
}

func typeScriptImpactDebugSource(def genericSymbolDef) string {
	if isTypeScriptDeclarationFilePath(def.File) {
		return "typescript-impact-structured-declaration"
	}
	return "typescript-impact-structured"
}

func buildTypeScriptImpactMetadata(def genericSymbolDef, refs typeScriptImpactRefs, rootPath string) *SymbolBundleImpact {
	return buildJSFamilyImpactMetadata(def, rootPath, classifyTypeScriptImpactRisk(def, refs), typeScriptImpactRecommendedReadGroups(def, refs))
}

func typeScriptImpactRecommendedReadGroups(def genericSymbolDef, refs typeScriptImpactRefs) []jsFamilyImpactReadGroup {
	if isTypeScriptDeclarationFilePath(def.File) {
		return typeScriptDeclarationImpactRecommendedReadGroups(refs)
	}

	importReferenceGroups := []jsFamilyImpactReadGroup{
		{kind: "imports", refs: refs.imports, limit: typeScriptImpactRecommendedReadPerGroupLimit},
		{kind: "references", refs: refs.others, limit: typeScriptImpactRecommendedReadPerGroupLimit},
	}
	callerGroup := jsFamilyImpactReadGroup{kind: "callers", refs: refs.callers, limit: typeScriptImpactRecommendedReadPerGroupLimit}
	typeRefGroup := jsFamilyImpactReadGroup{kind: "type_refs", refs: refs.typeRefs, limit: typeScriptImpactRecommendedReadPerGroupLimit}
	directTestGroup := jsFamilyImpactReadGroup{kind: "tests", refs: refs.directTests, limit: typeScriptImpactRecommendedReadPerGroupLimit, isTest: true}
	nearbyTestGroup := jsFamilyImpactReadGroup{kind: "tests", refs: refs.nearbyTests, limit: typeScriptImpactRecommendedReadPerGroupLimit, isTest: true}

	if typeScriptImpactPrefersTypeRefs(def.Kind) {
		groups := []jsFamilyImpactReadGroup{typeRefGroup, directTestGroup}
		groups = append(groups, importReferenceGroups...)
		groups = append(groups, nearbyTestGroup)
		return groups
	}

	groups := []jsFamilyImpactReadGroup{callerGroup, directTestGroup}
	groups = append(groups, importReferenceGroups...)
	groups = append(groups, nearbyTestGroup)
	return groups
}

func typeScriptDeclarationImpactRecommendedReadGroups(refs typeScriptImpactRefs) []jsFamilyImpactReadGroup {
	return []jsFamilyImpactReadGroup{
		{kind: "type_refs", refs: refs.typeRefs, limit: typeScriptImpactRecommendedReadPerGroupLimit},
		{kind: "imports", refs: refs.imports, limit: typeScriptImpactRecommendedReadPerGroupLimit},
		{kind: "references", refs: refs.others, limit: typeScriptImpactRecommendedReadPerGroupLimit},
		{kind: "callers", refs: refs.callers, limit: typeScriptImpactRecommendedReadPerGroupLimit},
		{kind: "tests", refs: refs.directTests, limit: typeScriptImpactRecommendedReadPerGroupLimit, isTest: true},
	}
}

func typeScriptImpactPrefersTypeRefs(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "interface", "type":
		return true
	default:
		return false
	}
}

func classifyTypeScriptImpactRisk(def genericSymbolDef, refs typeScriptImpactRefs) string {
	nonTestRefCount := len(dedupeGenericRefs(append(append(append(append([]genericSymbolRef(nil), refs.imports...), refs.callers...), refs.typeRefs...), refs.others...)))
	hasTests := len(dedupeGenericRefs(refs.allTests())) > 0
	exported := typeScriptDefinitionIsExported(def)
	primaryRefCount := nonTestRefCount
	if typeScriptImpactPrefersTypeRefs(def.Kind) {
		primaryRefCount = len(dedupeGenericRefs(refs.typeRefs))
	}

	switch {
	case !hasTests && nonTestRefCount >= typeScriptImpactHighNonTestReferenceThreshold:
		return goImpactRiskHigh
	case exported && !hasTests && primaryRefCount >= typeScriptImpactMediumReferenceThreshold:
		return goImpactRiskHigh
	case exported || !hasTests || nonTestRefCount >= typeScriptImpactMediumReferenceThreshold || primaryRefCount >= typeScriptImpactMediumReferenceThreshold:
		return goImpactRiskMedium
	default:
		return goImpactRiskLow
	}
}

func typeScriptDefinitionIsExported(def genericSymbolDef) bool {
	fields := strings.Fields(def.Signature)
	return len(fields) > 0 && fields[0] == "export"
}
