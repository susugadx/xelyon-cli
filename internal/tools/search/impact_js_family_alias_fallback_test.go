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

func TestExecuteSearchCodeArtifactWithConfig_TSXStructuredImpactSkipsShadowedNamedImportAliasJSXUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.tsx": "export function Button() { return <button /> }\n",
		"src/App.tsx": "import { Button as PrimaryButton } from './Button'\n" +
			"export function App() { return <PrimaryButton /> }\n",
		"src/Shadowed.tsx": "import { Button as PrimaryButton } from './Button'\n" +
			"export function Shadowed() {\n" +
			"  const PrimaryButton = Other\n" +
			"  return <PrimaryButton />\n" +
			"}\n",
		"src/ShadowedProps.tsx": "import { Button as PrimaryButton } from './Button'\n" +
			"export function ShadowedProps({ PrimaryButton }) {\n" +
			"  return <PrimaryButton />\n" +
			"}\n",
		"src/ShadowedEnum.tsx": "import { Button as PrimaryButton } from './Button'\n" +
			"export function ShadowedEnum() {\n" +
			"  enum PrimaryButton { One }\n" +
			"  return <PrimaryButton dataEnum />\n" +
			"}\n",
		"src/ShadowedAbstract.tsx": "import { Button as PrimaryButton } from './Button'\n" +
			"export function ShadowedAbstract() {\n" +
			"  abstract class PrimaryButton {}\n" +
			"  return <PrimaryButton dataAbstract />\n" +
			"}\n",
		"src/EnumScoped.tsx": "import { Button as PrimaryButton } from './Button'\n" +
			"export function EnumScoped() {\n" +
			"  if (ready) {\n" +
			"    enum PrimaryButton { One }\n" +
			"    return <PrimaryButton dataEnumBlock />\n" +
			"  }\n" +
			"  return <PrimaryButton dataAfterEnum />\n" +
			"}\n",
		"src/SwitchScoped.tsx": "import { Button as PrimaryButton } from './Button'\n" +
			"export function SwitchScoped(kind) {\n" +
			"  switch (kind) {\n" +
			"  case 'x':\n" +
			"    const PrimaryButton = Other\n" +
			"    render(<PrimaryButton dataSwitch />)\n" +
			"    break\n" +
			"  }\n" +
			"  return <PrimaryButton dataAfterSwitch />\n" +
			"}\n",
		"src/ShadowedLoop.tsx": "import { Button as PrimaryButton } from './Button'\n" +
			"export function ShadowedLoop(items) {\n" +
			"  for (let PrimaryButton of items) {\n" +
			"    render(<PrimaryButton dataLoop />)\n" +
			"  }\n" +
			"}\n",
		"src/ShadowedDestructuredLoop.tsx": "import { Button as PrimaryButton } from './Button'\n" +
			"export function ShadowedDestructuredLoop(items) {\n" +
			"  for (const { PrimaryButton } of items) {\n" +
			"    render(<PrimaryButton dataDestructuredLoop />)\n" +
			"  }\n" +
			"}\n",
		"src/LoopScoped.tsx": "import { Button as PrimaryButton } from './Button'\n" +
			"export function LoopScoped(items) {\n" +
			"  for (let PrimaryButton = 0; PrimaryButton < items.length; PrimaryButton++) {\n" +
			"    render(<PrimaryButton dataClassicLoop />)\n" +
			"  }\n" +
			"  return <PrimaryButton dataAfter />\n" +
			"}\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTSXImpactSearchOptions(dir, "Button"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "Button", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/App.tsx") || !symbolBundleItemsContainSnippet(callers, "<PrimaryButton />") {
		t.Fatalf("callers = %+v, want unshadowed named import alias JSX usage", callers)
	}
	if symbolBundleItemsContainFile(callers, "src/Shadowed.tsx") {
		t.Fatalf("callers = %+v, did not want locally shadowed alias JSX usage", callers)
	}
	if symbolBundleItemsContainFile(callers, "src/ShadowedProps.tsx") {
		t.Fatalf("callers = %+v, did not want destructured prop shadowed alias JSX usage", callers)
	}
	if symbolBundleItemsContainFile(callers, "src/ShadowedEnum.tsx") {
		t.Fatalf("callers = %+v, did not want enum shadowed alias JSX usage", callers)
	}
	if symbolBundleItemsContainFile(callers, "src/ShadowedAbstract.tsx") {
		t.Fatalf("callers = %+v, did not want abstract class shadowed alias JSX usage", callers)
	}
	if !symbolBundleItemsContainFile(callers, "src/EnumScoped.tsx") ||
		!symbolBundleItemsContainSnippet(callers, "dataAfterEnum") {
		t.Fatalf("callers = %+v, want alias JSX usage after enum block scope", callers)
	}
	if symbolBundleItemsContainSnippet(callers, "dataEnumBlock") {
		t.Fatalf("callers = %+v, did not want enum block shadowed alias JSX usage", callers)
	}
	if !symbolBundleItemsContainFile(callers, "src/SwitchScoped.tsx") ||
		!symbolBundleItemsContainSnippet(callers, "dataAfterSwitch") {
		t.Fatalf("callers = %+v, want alias JSX usage after switch scope", callers)
	}
	if symbolBundleItemsContainSnippet(callers, "dataSwitch") {
		t.Fatalf("callers = %+v, did not want switch shadowed alias JSX usage", callers)
	}
	if symbolBundleItemsContainFile(callers, "src/ShadowedLoop.tsx") {
		t.Fatalf("callers = %+v, did not want for-of shadowed alias JSX usage", callers)
	}
	if symbolBundleItemsContainFile(callers, "src/ShadowedDestructuredLoop.tsx") {
		t.Fatalf("callers = %+v, did not want destructured for-of shadowed alias JSX usage", callers)
	}
	if !symbolBundleItemsContainFile(callers, "src/LoopScoped.tsx") ||
		!symbolBundleItemsContainSnippet(callers, "dataAfter") {
		t.Fatalf("callers = %+v, want alias JSX usage after classic for loop scope", callers)
	}
	if symbolBundleItemsContainSnippet(callers, "dataClassicLoop") {
		t.Fatalf("callers = %+v, did not want classic for loop shadowed alias JSX usage", callers)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TSXStructuredImpactSkipsTypeOnlyNamedImportAliasJSXUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.tsx": "export function Button() { return <button /> }\n",
		"src/ImportTypeComment.tsx": "import /*c*/ type { Button as PrimaryButton } from './Button'\n" +
			"export function App() { return <PrimaryButton dataComment /> }\n",
		"src/ImportTypeNoSpace.tsx": "import type{ Button as PrimaryButton } from './Button'\n" +
			"export function App() { return <PrimaryButton dataNoSpace /> }\n",
		"src/SpecifierType.tsx": "import { type Button as PrimaryButton } from './Button'\n" +
			"export function App() { return <PrimaryButton dataSpecifier /> }\n",
		"src/ValueAlias.tsx": "import { Button as PrimaryButton } from './Button'\n" +
			"export function App() { return <PrimaryButton dataValue /> }\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTSXImpactSearchOptions(dir, "Button"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "Button", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "dataValue") {
		t.Fatalf("callers = %+v, want value alias JSX usage", callers)
	}
	for _, snippet := range []string{"dataComment", "dataNoSpace", "dataSpecifier"} {
		if symbolBundleItemsContainSnippet(callers, snippet) {
			t.Fatalf("callers = %+v, did not want type-only alias JSX usage %q", callers, snippet)
		}
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TSXStructuredImpactSkipsUnmatchedNamedImportAliasSource(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.tsx": "export function Button() { return <button /> }\n",
		"src/External.tsx": "import { Button as PrimaryButton } from '@external/ui'\n" +
			"export function App() { return <PrimaryButton dataExternal /> }\n",
		"src/Local.tsx": "import { Button as PrimaryButton } from './Button'\n" +
			"export function App() { return <PrimaryButton dataLocal /> }\n",
		"src/EmittedSpecifier.tsx": "import { Button as PrimaryButton } from './Button.js'\n" +
			"export function App() { return <PrimaryButton dataEmitted /> }\n",
		"src/SameLine.tsx": "import { Button } from './Button'; import { Button as ExternalButton } from '@external/ui'\n" +
			"export function App() { return <ExternalButton dataSameLineExternal /> }\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTSXImpactSearchOptions(dir, "Button"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "Button", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "dataLocal") {
		t.Fatalf("callers = %+v, want local relative alias JSX usage", callers)
	}
	if !symbolBundleItemsContainSnippet(callers, "dataEmitted") {
		t.Fatalf("callers = %+v, want emitted .js alias JSX usage for local TSX definition", callers)
	}
	if symbolBundleItemsContainSnippet(callers, "dataExternal") {
		t.Fatalf("callers = %+v, did not want external package alias JSX usage", callers)
	}
	if symbolBundleItemsContainSnippet(callers, "dataSameLineExternal") {
		t.Fatalf("callers = %+v, did not want same-line external package alias JSX usage", callers)
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if !symbolBundleItemsContainSnippet(imports, "import { Button } from './Button';") {
		t.Fatalf("imports = %+v, want same-line direct local import to remain", imports)
	}
}
