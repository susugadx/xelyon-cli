package search

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFindJSFamilyReferencesWithAST_CollectsMultilineNamedImportAliasJSXUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.tsx": "export function Button() { return <button /> }\n",
		"src/App.tsx": "import {\n" +
			"  Button as PrimaryButton,\n" +
			"} from './Button'\n" +
			"export function App() {\n" +
			"  return <PrimaryButton />\n" +
			"}\n",
	})

	opts := SearchOptions{
		Path:               dir,
		FileType:           "tsx",
		InvocationCWD:      dir,
		ProjectMapRootPath: dir,
	}
	refs := findJSFamilyReferencesWithAST("Button", genericSymbolDef{File: "src/Button.tsx"}, opts)
	classified := classifyJSFamilySymbolRefsFromAST(refs)

	if !genericRefsContainSnippet(classified.imports, "Button as PrimaryButton") {
		t.Fatalf("imports = %+v, want multiline named import alias", classified.imports)
	}
	if !genericRefsContainSnippet(classified.callers, "<PrimaryButton />") {
		t.Fatalf("callers = %+v, want JSX usage through named import alias", classified.callers)
	}
}

func TestFindJSFamilyReferencesWithAST_CollectsNamedImportAliasValueUsages(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id) { return id }\n",
		"src/App.ts": "import { buildUser as createUser } from './build'\n" +
			"const copy = createUser\n" +
			"export function run() { return createUser('1') }\n",
	})

	opts := SearchOptions{
		Path:               dir,
		FileType:           "ts",
		InvocationCWD:      dir,
		ProjectMapRootPath: dir,
	}
	refs := findJSFamilyReferencesWithAST("buildUser", genericSymbolDef{File: "src/build.ts"}, opts)
	classified := classifyJSFamilySymbolRefsFromAST(refs)

	if !genericRefsContainSnippet(classified.imports, "buildUser as createUser") {
		t.Fatalf("imports = %+v, want named import alias", classified.imports)
	}
	if !genericRefsContainSnippet(classified.callers, "createUser('1')") {
		t.Fatalf("callers = %+v, want call through named import alias", classified.callers)
	}
	if !genericRefsContainSnippet(classified.others, "const copy = createUser") {
		t.Fatalf("others = %+v, want value reference through named import alias", classified.others)
	}
}

func TestFindJSFamilyReferencesWithAST_CollectsTypeOnlyImportAliasTypeUsages(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/types.ts": "export interface BuildOptions { id: string }\n",
		"src/app.ts": "import type { BuildOptions as Options } from './types'\n" +
			"export type AppOptions = Options\n" +
			"export const input = {} as Options\n" +
			"export const raw = Options\n",
	})

	opts := SearchOptions{
		Path:               dir,
		FileType:           "ts",
		InvocationCWD:      dir,
		ProjectMapRootPath: dir,
	}
	refs := findJSFamilyReferencesWithAST("BuildOptions", genericSymbolDef{File: "src/types.ts", Kind: "interface"}, opts)
	classified := classifyJSFamilySymbolRefsFromAST(refs)

	if !genericRefsContainSnippet(classified.imports, "BuildOptions as Options") {
		t.Fatalf("imports = %+v, want type-only named import alias", classified.imports)
	}
	if !genericRefsContainSnippet(classified.typeRefs, "AppOptions = Options") ||
		!genericRefsContainSnippet(classified.typeRefs, "{} as Options") {
		t.Fatalf("typeRefs = %+v, want type-only alias type usages", classified.typeRefs)
	}
	if genericRefsContainSnippet(classified.others, "raw = Options") {
		t.Fatalf("others = %+v, did not want value-space usage through type-only alias", classified.others)
	}
}

func TestFindJSFamilyReferencesWithAST_CollectsDefaultImportAliasUsageFromSourceMatch(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.tsx": "export default function Button() { return <button /> }\n",
		"src/App.tsx":    "import PrimaryButton from './Button'\nexport function App() { return <PrimaryButton /> }\n",
	})

	opts := SearchOptions{
		Path:               dir,
		FileType:           "tsx",
		InvocationCWD:      dir,
		ProjectMapRootPath: dir,
	}
	def := genericSymbolDef{
		File:      "src/Button.tsx",
		Signature: "export default function Button() { return <button /> }",
	}
	refs := findJSFamilyReferencesWithAST("Button", def, opts)
	classified := classifyJSFamilySymbolRefsFromAST(refs)

	if !genericRefsContainSnippet(classified.imports, "import PrimaryButton from './Button'") {
		t.Fatalf("imports = %+v, want default import source match", classified.imports)
	}
	if !genericRefsContainSnippet(classified.callers, "<PrimaryButton />") {
		t.Fatalf("callers = %+v, want JSX usage through default import alias", classified.callers)
	}
}

func TestFindJSFamilyReferencesWithAST_CollectsDefaultImportAliasUsageWithoutSymbolText(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export default function buildUser(id) { return id }\n",
		"src/app.js":   "import makeUser from './build.js'\nexport const user = makeUser('1')\n",
	})

	opts := SearchOptions{
		Path:               dir,
		FileType:           "js",
		InvocationCWD:      dir,
		ProjectMapRootPath: dir,
	}
	def := genericSymbolDef{
		File:      "src/build.js",
		Signature: "export default function buildUser(id) { return id }",
	}
	refs := findJSFamilyReferencesWithAST("buildUser", def, opts)
	classified := classifyJSFamilySymbolRefsFromAST(refs)

	if !genericRefsContainSnippet(classified.imports, "import makeUser from './build.js'") {
		t.Fatalf("imports = %+v, want default import source match without symbol text", classified.imports)
	}
	if !genericRefsContainSnippet(classified.callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, want call through default import alias without symbol text", classified.callers)
	}
}

func TestJSFamilyASTReferenceCollector_AliasExpansionDoesNotStopRealMatchStream(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.tsx":       "export function Button() { return <button /> }\n",
		"src/AApp.tsx":         jsFamilyAliasBudgetSource(maxGenericRefs + 5),
		"src/ZButton.test.tsx": "import { Button } from './Button'\nit('renders', () => <Button />)\n",
	})
	collector := newJSFamilyASTReferenceCollector("Button", genericSymbolDef{File: "src/Button.tsx"}, SearchOptions{
		Path:               dir,
		FileType:           "tsx",
		InvocationCWD:      dir,
		ProjectMapRootPath: dir,
	}, maxGenericRefs)
	defer collector.Close()

	matches := []genericSymbolMatch{
		{File: "src/AApp.tsx", Line: 1, Content: "import { Button as PrimaryButton } from './Button'"},
		{File: "src/ZButton.test.tsx", Line: 1, Content: "import { Button } from './Button'"},
		{File: "src/ZButton.test.tsx", Line: 2, Content: "it('renders', () => <Button />)"},
	}
	for _, match := range matches {
		if !collector.AddNameMatch(match) {
			break
		}
	}

	classified := classifyJSFamilySymbolRefsFromAST(collector.Result())

	if !genericRefsContainSnippet(classified.callers, "<PrimaryButton />") {
		t.Fatalf("callers = %+v, want alias JSX caller", classified.callers)
	}
	if !genericRefsContainSnippet(classified.tests, "import { Button }") ||
		!genericRefsContainSnippet(classified.tests, "<Button />") {
		t.Fatalf("tests = %+v, want later real direct test refs after alias expansion", classified.tests)
	}
}

func TestJSFamilyASTReferenceCollector_AliasExpansionStopsAtAliasBudget(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.tsx": "export function Button() { return <button /> }\n",
		"src/AApp.tsx":   jsFamilyAliasBudgetSource(maxGenericRefs + 5),
	})
	collector := newJSFamilyASTReferenceCollector("Button", genericSymbolDef{File: "src/Button.tsx"}, SearchOptions{
		Path:               dir,
		FileType:           "tsx",
		InvocationCWD:      dir,
		ProjectMapRootPath: dir,
	}, 3)
	defer collector.Close()

	collector.AddNameMatch(genericSymbolMatch{
		File:    "src/AApp.tsx",
		Line:    1,
		Content: "import { Button as PrimaryButton } from './Button'",
	})

	result := collector.DetailedResult()
	classified := classifyJSFamilySymbolRefsFromAST(result.refs)

	if len(classified.callers) != 3 {
		t.Fatalf("callers len = %d, want alias caller budget 3; callers=%+v", len(classified.callers), classified.callers)
	}
	if !result.budgetLimitHit || !result.truncated {
		t.Fatalf("diagnostics = budgetLimitHit %v truncated %v, want true/true for alias budget", result.budgetLimitHit, result.truncated)
	}
	if result.rawMatchCount < len(result.refs) {
		t.Fatalf("rawMatchCount = %d, refs = %d, want raw count to include alias expansion refs", result.rawMatchCount, len(result.refs))
	}
}

func TestJSFamilyASTReferenceCollector_AliasExpansionSkipsUnmatchedImportSource(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.tsx": "export function Button() { return <button /> }\n",
		"src/App.tsx": "import { Button as PrimaryButton } from '@external/ui'\n" +
			"export function App() { return <PrimaryButton dataExternal /> }\n",
	})
	collector := newJSFamilyASTReferenceCollector("Button", genericSymbolDef{File: "src/Button.tsx"}, SearchOptions{
		Path:               dir,
		FileType:           "tsx",
		InvocationCWD:      dir,
		ProjectMapRootPath: dir,
	}, maxGenericRefs)
	defer collector.Close()

	collector.AddNameMatch(genericSymbolMatch{
		File:    "src/App.tsx",
		Line:    1,
		Content: "import { Button as PrimaryButton } from '@external/ui'",
	})
	classified := classifyJSFamilySymbolRefsFromAST(collector.Result())

	if genericRefsContainSnippet(classified.callers, "dataExternal") {
		t.Fatalf("callers = %+v, did not want external alias JSX caller", classified.callers)
	}
}

func TestJSFamilyImportSourceBaseMatchesFileEmittedJSSpecifiers(t *testing.T) {
	tests := []struct {
		name       string
		sourceBase string
		file       string
	}{
		{
			name:       "js specifier to tsx source",
			sourceBase: filepath.Join("src", "Button.js"),
			file:       filepath.Join("src", "Button.tsx"),
		},
		{
			name:       "js specifier to ts source",
			sourceBase: filepath.Join("src", "Button.js"),
			file:       filepath.Join("src", "Button.ts"),
		},
		{
			name:       "jsx specifier to tsx source",
			sourceBase: filepath.Join("src", "Button.jsx"),
			file:       filepath.Join("src", "Button.tsx"),
		},
		{
			name:       "mjs specifier to mts source",
			sourceBase: filepath.Join("src", "Button.mjs"),
			file:       filepath.Join("src", "Button.mts"),
		},
		{
			name:       "cjs specifier to cts source",
			sourceBase: filepath.Join("src", "Button.cjs"),
			file:       filepath.Join("src", "Button.cts"),
		},
		{
			name:       "index js specifier to index tsx source",
			sourceBase: filepath.Join("src", "Button", "index.js"),
			file:       filepath.Join("src", "Button", "index.tsx"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !jsFamilyImportSourceBaseMatchesFile(tt.sourceBase, tt.file) {
				t.Fatalf("jsFamilyImportSourceBaseMatchesFile(%q, %q) = false, want true", tt.sourceBase, tt.file)
			}
		})
	}
}

func TestResolveJSSymbol_TSXFindsNamedImportAliasJSXUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.tsx": "export function Button() { return <button /> }\n",
		"src/App.tsx": "import { Button as PrimaryButton } from './Button'\n" +
			"export function App() { return <PrimaryButton /> }\n",
	})

	result := resolveJSFamilySymbol("Button", SearchOptions{
		Path:          dir,
		FileType:      "tsx",
		InvocationCWD: dir,
	})

	if result.Status != genericSymbolSingle {
		t.Fatalf("status = %s, want %s; output:\n%s", result.Status, genericSymbolSingle, result.Output)
	}
	callers := symbolBundleSectionItems(result.Bundle, "callers")
	if !symbolBundleItemsContainFileSuffix(callers, "src/App.tsx") || !symbolBundleItemsContainSnippet(callers, "<PrimaryButton />") {
		t.Fatalf("callers = %+v, want named import alias JSX usage in normal symbol path", callers)
	}
}

func jsFamilyAliasBudgetSource(aliasUsageCount int) string {
	var source strings.Builder
	source.WriteString("import { Button as PrimaryButton } from './Button'\n")
	source.WriteString("export function AApp() {\n")
	source.WriteString("  return <>\n")
	for i := 0; i < aliasUsageCount; i++ {
		source.WriteString("    <PrimaryButton />\n")
	}
	source.WriteString("  </>\n")
	source.WriteString("}\n")
	return source.String()
}

func symbolBundleItemsContainFileSuffix(items []SymbolBundleItem, suffix string) bool {
	for _, item := range items {
		if strings.HasSuffix(item.File, suffix) {
			return true
		}
	}
	return false
}
