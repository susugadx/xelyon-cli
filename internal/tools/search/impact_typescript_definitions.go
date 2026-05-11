package search

import "strings"

type structuredTypeScriptPreferredDefs struct {
	defs                      []genericSymbolDef
	suppressedDeclarationDefs []genericSymbolDef
}

func preferStructuredTypeScriptImplementationDefs(defs []genericSymbolDef) structuredTypeScriptPreferredDefs {
	if len(defs) <= 1 {
		return structuredTypeScriptPreferredDefs{defs: defs}
	}

	implementationPaths := make(map[string]struct{}, len(defs))
	for _, def := range defs {
		if isTypeScriptImplementationFilePath(def.File) {
			if key := structuredTypeScriptDefPathKey(def.File); key != "" {
				implementationPaths[key] = struct{}{}
			}
		}
	}
	if len(implementationPaths) == 0 {
		return structuredTypeScriptPreferredDefs{defs: defs}
	}

	filtered := make([]genericSymbolDef, 0, len(defs))
	suppressed := make([]genericSymbolDef, 0)
	for _, def := range defs {
		if isPairedTypeScriptDeclarationDef(def, implementationPaths) {
			suppressed = append(suppressed, def)
			continue
		}
		filtered = append(filtered, def)
	}
	return structuredTypeScriptPreferredDefs{
		defs:                      filtered,
		suppressedDeclarationDefs: suppressed,
	}
}

func filterStructuredTypeScriptSuppressedDeclarationRefs(refs []genericSymbolRef, suppressed []genericSymbolDef) []genericSymbolRef {
	if len(refs) == 0 || len(suppressed) == 0 {
		return refs
	}

	suppressedFiles := make(map[string]struct{}, len(suppressed))
	for _, def := range suppressed {
		if file := structuredTypeScriptDefPathKey(def.File); file != "" {
			suppressedFiles[file] = struct{}{}
		}
	}
	if len(suppressedFiles) == 0 {
		return refs
	}

	filtered := make([]genericSymbolRef, 0, len(refs))
	for _, ref := range refs {
		if _, ok := suppressedFiles[structuredTypeScriptDefPathKey(ref.File)]; ok {
			continue
		}
		filtered = append(filtered, ref)
	}
	return filtered
}

func isPairedTypeScriptDeclarationDef(def genericSymbolDef, implementationPaths map[string]struct{}) bool {
	implementationPath, ok := structuredTypeScriptDeclarationImplementationPath(def.File)
	if !ok {
		return false
	}
	_, ok = implementationPaths[implementationPath]
	return ok
}

func structuredTypeScriptDeclarationImplementationPath(path string) (string, bool) {
	key := structuredTypeScriptDefPathKey(path)
	if key == "" {
		return "", false
	}
	lowerKey := strings.ToLower(key)
	if !strings.HasSuffix(lowerKey, ".d.ts") {
		return "", false
	}
	return key[:len(key)-len(".d.ts")] + ".ts", true
}

func collectStructuredTypeScriptDefAffectedFiles(defs []genericSymbolDef, opts SearchOptions) []string {
	rootPath := structuredTypeScriptImpactFileRoot(opts)
	paths := make([]string, 0, len(defs))
	for _, def := range defs {
		if absPath := absoluteAffectedFilePathWithBase(def.File, rootPath); absPath != "" {
			paths = append(paths, absPath)
		}
	}
	return dedupePaths(paths)
}
