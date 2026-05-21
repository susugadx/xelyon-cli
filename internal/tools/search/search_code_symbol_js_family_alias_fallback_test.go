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

	classified := classifyJSFamilySymbolRefsFromAST(collector.Result())

	if len(classified.callers) != 3 {
		t.Fatalf("callers len = %d, want alias caller budget 3; callers=%+v", len(classified.callers), classified.callers)
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

	result := resolveJSSymbol("Button", SearchOptions{
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
