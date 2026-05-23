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

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFindsDefaultImportAliasWithoutSymbolText(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js":      "export default function buildUser(id) { return id }\n",
		"src/app.js":        "import makeUser from './build.js'\nexport const user = makeUser('1')\n",
		"src/build.test.js": "import makeUser from './build.js'\nit('builds', () => makeUser('test'))\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") || !symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, want default import alias call without symbol text", callers)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/build.test.js") {
		t.Fatalf("expected direct JS default alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFindsNamedDefaultImportAlias(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export default function buildUser(id) { return id }\n",
		"src/app.js": "import { default as makeUser } from './build.js'\n" +
			"export const user = makeUser('1')\n",
		"src/build.spec.js": "import { default as makeUser } from './build.js'\n" +
			"it('builds', () => makeUser('test'))\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") || !symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, want named default import alias call", callers)
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if !symbolBundleItemsContainSnippet(imports, "default as makeUser") {
		t.Fatalf("imports = %+v, want named default import alias evidence", imports)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/build.spec.js") {
		t.Fatalf("expected direct JS named default alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFindsCommonJSRequireAliasUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export function buildUser(id) { return id }\n",
		"src/app.js": "const { buildUser: makeUser } = require('./build')\n" +
			"export const user = makeUser('1')\n",
		"src/build.test.js": "const { buildUser: makeUser } = require('./build')\n" +
			"it('builds', () => makeUser('test'))\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") || !symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, want CommonJS require alias call", callers)
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if !symbolBundleItemsContainSnippet(imports, "buildUser: makeUser") {
		t.Fatalf("imports = %+v, want CommonJS require alias evidence", imports)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/build.test.js") {
		t.Fatalf("expected direct JS require alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFindsCommonJSDefaultRequireAliasUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "module.exports = function buildUser(id) { return id }\n",
		"src/app.js": "const makeUser = require('./build')\n" +
			"export const user = makeUser('1')\n",
		"src/build.test.js": "const makeUser = require('./build')\n" +
			"it('builds', () => makeUser('test'))\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") || !symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, want CommonJS default require alias call", callers)
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if !symbolBundleItemsContainSnippet(imports, "const makeUser = require") {
		t.Fatalf("imports = %+v, want CommonJS default require evidence", imports)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/build.test.js") {
		t.Fatalf("expected direct JS default require alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFindsCommonJSSeparateDefaultRequireAliasUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "function buildUser(id) { return id }\nmodule.exports = buildUser\n",
		"src/app.js": "const makeUser = require('./build')\n" +
			"export const user = makeUser('1')\n",
		"src/build.test.js": "const makeUser = require('./build')\n" +
			"it('builds', () => makeUser('test'))\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") || !symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, want CommonJS separate default require alias call", callers)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/build.test.js") {
		t.Fatalf("expected direct JS separate default require alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFindsCommonJSMemberRequireAliasUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export function buildUser(id) { return id }\n",
		"src/app.js": "const makeUser = require('./build').buildUser\n" +
			"export const user = makeUser('1')\n",
		"src/build.test.js": "const makeUser = require('./build').buildUser\n" +
			"it('builds', () => makeUser('test'))\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") || !symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, want CommonJS member require alias call", callers)
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if !symbolBundleItemsContainSnippet(imports, "require('./build').buildUser") {
		t.Fatalf("imports = %+v, want CommonJS member require evidence", imports)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/build.test.js") {
		t.Fatalf("expected direct JS member require alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFindsMultilineNamedDefaultImportAlias(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export default function buildUser(id) { return id }\n",
		"src/app.js": "import {\n" +
			"  default as makeUser,\n" +
			"} from './build.js'\n" +
			"export const user = makeUser('1')\n",
		"src/build.spec.js": "import {\n" +
			"  default as makeUser,\n" +
			"} from './build.js'\n" +
			"it('builds', () => makeUser('test'))\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.js") || !symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, want multiline named default import alias call", callers)
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if !symbolBundleItemsContainSnippet(imports, "import {") {
		t.Fatalf("imports = %+v, want multiline named default import evidence", imports)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/build.spec.js") {
		t.Fatalf("expected direct JS multiline named default alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactSkipsNestedRequireDestructuringAlias(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "export function buildUser(id) { return id }\n",
		"src/app.js": "const { api: { buildUser: makeUser } } = require('./build')\n" +
			"export const user = makeUser('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if symbolBundleItemsContainFile(callers, "src/app.js") || symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
		t.Fatalf("callers = %+v, did not want nested CommonJS destructuring alias call", callers)
	}
	imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
	if symbolBundleItemsContainFile(imports, "src/app.js") || symbolBundleItemsContainSnippet(imports, "api:") {
		t.Fatalf("imports = %+v, did not want nested CommonJS destructuring evidence", imports)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactFindsSeparateDefaultExportAlias(t *testing.T) {
	tests := []struct {
		name   string
		export string
	}{
		{name: "export specifier default alias", export: "export { buildUser as default }"},
		{name: "export default identifier", export: "export default buildUser"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, map[string]string{
				"src/build.js": "function buildUser(id) { return id }\n" + tt.export + "\n",
				"src/app.js":   "import makeUser from './build.js'\nexport const user = makeUser('1')\n",
			})

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

			assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
			callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
			if !symbolBundleItemsContainFile(callers, "src/app.js") || !symbolBundleItemsContainSnippet(callers, "makeUser('1')") {
				t.Fatalf("callers = %+v, want default import alias call for %s", callers, tt.export)
			}
			imports := symbolBundleSectionItems(artifact.Metadata.Bundle, "imports")
			if !symbolBundleItemsContainSnippet(imports, "import makeUser") {
				t.Fatalf("imports = %+v, want default import evidence for %s", imports, tt.export)
			}
		})
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactFindsNamedImportAliasCallUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id) { return id }\n",
		"src/app.ts":   "import { buildUser as createUser } from './build'\nexport const user = createUser('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "buildUser"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/app.ts") || !symbolBundleItemsContainSnippet(callers, "createUser('1')") {
		t.Fatalf("callers = %+v, want named import alias call usage", callers)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactFindsTypeOnlyImportAliasTypeUsage(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/types.ts": "export interface BuildOptions { id: string }\n",
		"src/app.ts": "import type { BuildOptions as Options } from './types'\n" +
			"export type AppOptions = Options\n" +
			"export const input = {} as Options\n",
		"src/types.test.ts": "import type { BuildOptions as Options } from './types'\n" +
			"const input = {} as Options\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "BuildOptions"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "BuildOptions", "interface")
	typeRefs := symbolBundleSectionItems(artifact.Metadata.Bundle, "type_refs")
	if !symbolBundleItemsContainFile(typeRefs, "src/app.ts") || !symbolBundleItemsContainSnippet(typeRefs, "AppOptions = Options") {
		t.Fatalf("typeRefs = %+v, want type-only import alias type usage", typeRefs)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/types.test.ts") {
		t.Fatalf("expected type-only alias test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
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
