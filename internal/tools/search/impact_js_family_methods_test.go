package search

import "testing"

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactClassMethod(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/view.ts": "export class View {\n  render() { return 'ok' }\n}\n",
		"src/app.ts":  "import { View } from './view'\nconst view = new View()\nview.render()\nview.render.call(view)\nview.render.apply(view, [])\nconst bound = view.render.bind(view)\nview.render.extra()\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "render"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "render", "method")
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/view.ts" {
		t.Fatalf("definition file = %q, want src/view.ts", got)
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "view.render()") {
		t.Fatalf("callers = %+v, want view.render() caller", callers)
	}
	for _, snippet := range []string{"view.render.call(view)", "view.render.apply(view, [])"} {
		if !symbolBundleItemsContainSnippet(callers, snippet) {
			t.Fatalf("callers = %+v, want %q as method caller", callers, snippet)
		}
	}
	for _, snippet := range []string{"view.render.bind(view)", "view.render.extra()"} {
		if symbolBundleItemsContainSnippet(callers, snippet) {
			t.Fatalf("callers = %+v, want %q as non-caller method read", callers, snippet)
		}
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactCallNamedMethod(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/view.ts": "export class View {\n  call() { return 'ok' }\n}\n",
		"src/app.ts":  "import { View } from './view'\nconst view = new View()\nview.call()\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "call"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "call", "method")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "view.call()") {
		t.Fatalf("callers = %+v, want view.call() caller", callers)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactInterfaceMethod(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/contracts.ts": "export interface Store {\n  save(id: string): void\n}\n",
		"src/app.ts":       "import type { Store } from './contracts'\ndeclare const store: Store\nstore.save('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "save"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "save", "method")
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/contracts.ts" {
		t.Fatalf("definition file = %q, want src/contracts.ts", got)
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "store.save('1')") {
		t.Fatalf("callers = %+v, want store.save('1') caller", callers)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactIntersectionTypeAliasMethod(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/contracts.ts": "type BaseStore = { id: string }\nexport type Store = BaseStore & {\n  save(id: string): void\n}\n",
		"src/app.ts":       "import type { Store } from './contracts'\ndeclare const store: Store\nstore.save('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "save"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "save", "method")
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/contracts.ts" {
		t.Fatalf("definition file = %q, want src/contracts.ts", got)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactClassMethod(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/service.js": "class UserService {\n  buildUser(id) { return id }\n}\nmodule.exports = { UserService }\n",
		"src/app.js":     "const { UserService } = require('./service')\nnew UserService().buildUser('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "method")
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/service.js" {
		t.Fatalf("definition file = %q, want src/service.js", got)
	}
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainSnippet(callers, "new UserService().buildUser('1')") {
		t.Fatalf("callers = %+v, want buildUser caller", callers)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactIgnoresObjectLiteralMethodDefinition(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n" +
			"export const handlers = {\n" +
			"  buildUser(id: string) { return id }\n" +
			"}\n",
		"src/app.ts": "import { buildUser } from './build'\nbuildUser('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "buildUser"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if got := artifact.Metadata.Bundle.Definition.Line; got != 1 {
		t.Fatalf("definition line = %d, want exported function line 1", got)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactIgnoresInlineObjectTypeMethodDefinition(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export function buildUser(id: string) { return id }\n" +
			"export function useStore(store: {\n" +
			"  buildUser(id: string): void\n" +
			"}) { return store }\n",
		"src/app.ts": "import { buildUser } from './build'\nbuildUser('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "buildUser"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if got := artifact.Metadata.Bundle.Definition.Line; got != 1 {
		t.Fatalf("definition line = %d, want exported function line 1", got)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_JavaScriptStructuredImpactIgnoresObjectLiteralMethodDefinition(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.js": "function buildUser(id) { return id }\n" +
			"const handlers = {\n" +
			"  buildUser(id) { return id }\n" +
			"}\n" +
			"module.exports = { buildUser, handlers }\n",
		"src/app.js": "const { buildUser } = require('./build')\nbuildUser('1')\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newJavaScriptImpactSearchOptions(dir, "buildUser"))

	assertJavaScriptStructuredImpactArtifact(t, artifact, "buildUser", "function")
	if got := artifact.Metadata.Bundle.Definition.Line; got != 1 {
		t.Fatalf("definition line = %d, want function line 1", got)
	}
}
