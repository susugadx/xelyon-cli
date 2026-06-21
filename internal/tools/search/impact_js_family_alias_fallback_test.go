package search

import "testing"

func TestExecuteSearchCodeArtifactWithConfig_TSXStructuredImpactFindsNamedImportAliasJSXUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.tsx": "export function Button() { return <button /> }\n",
		"src/App.tsx": "import { Button as PrimaryButton } from './Button'\n" +
			"export function App() { return <PrimaryButton /> }\n",
		"src/Button.test.tsx": "import { Button as PrimaryButton } from './Button'\n" +
			"it('renders', () => <PrimaryButton />)\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTSXImpactSearchOptions(dir, "Button"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "Button", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/App.tsx") || !symbolBundleItemsContainSnippet(callers, "<PrimaryButton />") {
		t.Fatalf("callers = %+v, want named import alias JSX usage", callers)
	}
	if symbolBundleItemsContainFile(callers, "src/Button.test.tsx") {
		t.Fatalf("direct TSX alias test should be separated from callers, got %+v", callers)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/Button.test.tsx") {
		t.Fatalf("expected direct TSX alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if !symbolBundleItemsContainSnippet(imports, "Button as PrimaryButton") {
		t.Fatalf("imports = %+v, want named import alias line", imports)
	}
}
func TestExecuteSearchCodeArtifactWithConfig_TSXStructuredImpactSkipsExternalSameNameImports(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.tsx": "export function Button() { return <button /> }\n",
		"src/App.tsx": "import { Button } from './Button'\n" +
			"export function App() { return <Button dataLocal /> }\n",
		"src/External.tsx": "import { Button } from '@external/ui'\n" +
			"export function External() { return <Button dataExternal /> }\n",
		"src/ReExport.tsx": "export { Button } from '@external/ui'\n" +
			"export { Button as LocalButton } from './Button'\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTSXImpactSearchOptions(dir, "Button"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "Button", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/App.tsx") ||
		!symbolBundleItemsContainSnippet(callers, "dataLocal") {
		t.Fatalf("callers = %+v, want local Button caller", callers)
	}
	if symbolBundleItemsContainFile(callers, "src/External.tsx") ||
		symbolBundleItemsContainSnippet(callers, "dataExternal") {
		t.Fatalf("callers = %+v, did not want external Button caller", callers)
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if !symbolBundleItemsContainSnippet(imports, "Button as LocalButton") {
		t.Fatalf("imports = %+v, want local re-export evidence", imports)
	}
	if symbolBundleItemsContainFile(imports, "src/External.tsx") ||
		symbolBundleItemsContainSnippet(imports, "@external/ui") {
		t.Fatalf("imports = %+v, did not want external same-name import or re-export", imports)
	}
}
func TestExecuteSearchCodeArtifactWithConfig_JSXStructuredImpactFindsNamedImportAliasJSXUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.jsx": "export function Button() { return <button /> }\n",
		"src/App.jsx": "import { Button as PrimaryButton } from './Button'\n" +
			"export function App() { return <PrimaryButton /> }\n",
		"src/Button.test.jsx": "import { Button as PrimaryButton } from './Button'\n" +
			"it('renders', () => <PrimaryButton />)\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJSXImpactSearchOptions(dir, "Button"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "Button", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/App.jsx") || !symbolBundleItemsContainSnippet(callers, "<PrimaryButton />") {
		t.Fatalf("callers = %+v, want named import alias JSX usage", callers)
	}
	if symbolBundleItemsContainFile(callers, "src/Button.test.jsx") {
		t.Fatalf("direct JSX alias test should be separated from callers, got %+v", callers)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/Button.test.jsx") {
		t.Fatalf("expected direct JSX alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}
func TestExecuteSearchCodeArtifactWithConfig_TSXStructuredImpactFindsDefaultImportAliasJSXUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.tsx":      "export default function Button() { return <button /> }\n",
		"src/App.tsx":         "import PrimaryButton from './Button'\nexport function App() { return <PrimaryButton /> }\n",
		"src/Button.test.tsx": "import PrimaryButton from './Button'\nit('renders', () => <PrimaryButton />)\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTSXImpactSearchOptions(dir, "Button"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "Button", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/App.tsx") || !symbolBundleItemsContainSnippet(callers, "<PrimaryButton />") {
		t.Fatalf("callers = %+v, want default import alias JSX usage", callers)
	}
	if symbolBundleItemsContainFile(callers, "src/Button.test.tsx") {
		t.Fatalf("direct TSX default alias test should be separated from callers, got %+v", callers)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/Button.test.tsx") {
		t.Fatalf("expected direct TSX default alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}
