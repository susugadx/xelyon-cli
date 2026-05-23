package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactClassField(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/view.ts": "export class View {\n  render = () => 'ok'\n}\n",
		"src/app.ts":  "import { View } from './view'\nconst view = new View()\nview.render()\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "render"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "render", "field")
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/view.ts" {
		t.Fatalf("definition file = %q, want src/view.ts", got)
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "view.render()") {
		t.Fatalf("callers = %+v, want view.render() caller", callers)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactClassFieldUsesLSPReferences(t *testing.T) {
	appLine := "view.render()"
	dir := setupMultiLangDir(t, map[string]string{
		"src/view.ts":  "export class View {\n  render = () => 'ok'\n}\n",
		"src/app.ts":   "import { View } from './view'\nconst view = new View()\n" + appLine + "\n",
		"src/noise.ts": "declare const view: { render: () => string }\nview.render()\n",
	})
	start, end := testLSPRangeForSearchToken(appLine, "render")
	lspClient := &mockJSFamilyLSPClient{
		refs: []navigation.LSPLocation{{File: "src/app.ts", Line: 3, Character: start, EndLine: 3, EndChar: end}},
	}
	opts := newTypeScriptImpactSearchOptions(dir, "render")
	opts.LSPClient = lspClient

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertTypeScriptStructuredImpactArtifact(t, artifact, "render", "field")
	if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = false, want true")
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "view.render()") {
		t.Fatalf("callers = %+v, want LSP field caller", callers)
	}
	if strings.Contains(artifact.Rendered, "src/noise.ts") {
		t.Fatalf("rendered output includes fallback-only field ref, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactInterfaceProperty(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/contracts.ts": "export interface Store {\n  save: (id: string) => void\n}\n",
		"src/app.ts":       "import type { Store } from './contracts'\ndeclare const store: Store\nstore.save('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "save"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "save", "property")
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/contracts.ts" {
		t.Fatalf("definition file = %q, want src/contracts.ts", got)
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "store.save('1')") {
		t.Fatalf("callers = %+v, want store.save('1') caller", callers)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactInterfacePropertyUsesLSPReferences(t *testing.T) {
	appLine := "store.save('semantic')"
	dir := setupMultiLangDir(t, map[string]string{
		"src/contracts.ts": "export interface Store {\n  save: (id: string) => void\n}\n",
		"src/app.ts":       "import type { Store } from './contracts'\ndeclare const store: Store\n" + appLine + "\n",
		"src/noise.ts":     "declare const store: { save: (id: string) => void }\nstore.save('fallback-only')\n",
	})
	start, end := testLSPRangeForSearchToken(appLine, "save")
	lspClient := &mockJSFamilyLSPClient{
		refs: []navigation.LSPLocation{{File: "src/app.ts", Line: 3, Character: start, EndLine: 3, EndChar: end}},
	}
	opts := newTypeScriptImpactSearchOptions(dir, "save")
	opts.LSPClient = lspClient

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertTypeScriptStructuredImpactArtifact(t, artifact, "save", "property")
	if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = false, want true")
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "store.save('semantic')") {
		t.Fatalf("callers = %+v, want LSP property caller", callers)
	}
	if strings.Contains(artifact.Rendered, "fallback-only") {
		t.Fatalf("rendered output includes fallback-only property ref, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactIntersectionTypeAliasProperty(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/contracts.ts": "type BaseStore = { id: string }\nexport type Store = BaseStore & {\n  save: (id: string) => void\n}\n",
		"src/app.ts":       "import type { Store } from './contracts'\ndeclare const store: Store\nstore.save('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "save"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "save", "property")
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/contracts.ts" {
		t.Fatalf("definition file = %q, want src/contracts.ts", got)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactClassField(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/service.js": "class UserService {\n  buildUser = (id) => id\n}\nmodule.exports = { UserService }\n",
		"src/app.js":     "const { UserService } = require('./service')\nnew UserService().buildUser('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "field")
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/service.js" {
		t.Fatalf("definition file = %q, want src/service.js", got)
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "new UserService().buildUser('1')") {
		t.Fatalf("callers = %+v, want buildUser caller", callers)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactClassFieldUsesLSPReferences(t *testing.T) {
	appLine := "new UserService().buildUser('semantic')"
	dir := setupMultiLangDir(t, map[string]string{
		"src/service.js": "class UserService {\n  buildUser = (id) => id\n}\nmodule.exports = { UserService }\n",
		"src/app.js":     "const { UserService } = require('./service')\n" + appLine + "\n",
		"src/noise.js":   "const service = { buildUser: (id) => id }\nservice.buildUser('fallback-only')\n",
	})
	start, end := testLSPRangeForSearchToken(appLine, "buildUser")
	lspClient := &mockJSFamilyLSPClient{
		refs: []navigation.LSPLocation{{File: "src/app.js", Line: 2, Character: start, EndLine: 2, EndChar: end}},
	}
	opts := newJavaScriptImpactSearchOptions(dir, "buildUser")
	opts.LSPClient = lspClient

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, opts)

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "field")
	if !artifact.Metadata.Bundle.Diagnostics.ResolvedViaLSP {
		t.Fatal("ResolvedViaLSP = false, want true")
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "new UserService().buildUser('semantic')") {
		t.Fatalf("callers = %+v, want LSP field caller", callers)
	}
	if strings.Contains(artifact.Rendered, "fallback-only") {
		t.Fatalf("rendered output includes fallback-only JS field ref, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactIgnoresObjectLiteralPropertyDefinition(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n" +
			"export const handlers = {\n" +
			"  buildUser: (id: string) => id\n" +
			"}\n",
		"src/app.ts": "import { buildUser } from './build'\nbuildUser('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "buildUser"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if got := artifact.Metadata.Bundle.Definition.Line; got != 1 {
		t.Fatalf("definition line = %d, want exported function line 1", got)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactIgnoresInlineObjectTypePropertyDefinition(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n" +
			"export function useStore(store: {\n" +
			"  buildUser: (id: string) => void\n" +
			"}) { return store }\n",
		"src/app.ts": "import { buildUser } from './build'\nbuildUser('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "buildUser"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if got := artifact.Metadata.Bundle.Definition.Line; got != 1 {
		t.Fatalf("definition line = %d, want exported function line 1", got)
	}
}
