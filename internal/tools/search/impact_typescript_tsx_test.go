package search

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestExecuteSearchCodeArtifactWithConfig_TSXStructuredImpactShapes(t *testing.T) {
	tests := []struct {
		name       string
		symbol     string
		definition string
		app        string
		wantKind   string
	}{
		{
			name:       "export function component",
			symbol:     "Button",
			definition: "export function Button(props: ButtonProps) { return <button>{props.label}</button> }\n",
			wantKind:   "function",
		},
		{
			name:       "non-export function component",
			symbol:     "Button",
			definition: "function Button(props: ButtonProps) { return <button>{props.label}</button> }\n",
			wantKind:   "function",
		},
		{
			name:       "export class component",
			symbol:     "Button",
			definition: "export class Button extends React.Component<ButtonProps> { render() { return <button /> } }\n",
			wantKind:   "class",
		},
		{
			name:       "non-export class component",
			symbol:     "Button",
			definition: "class Button extends React.Component<ButtonProps> { render() { return <button /> } }\n",
			wantKind:   "class",
		},
		{
			name:       "default function component",
			symbol:     "Button",
			definition: "export default function Button(props: ButtonProps) { return <button>{props.label}</button> }\n",
			wantKind:   "function",
		},
		{
			name:       "default class component",
			symbol:     "Button",
			definition: "export default class Button extends React.Component<ButtonProps> { render() { return <button /> } }\n",
			wantKind:   "class",
		},
		{
			name:       "arrow component",
			symbol:     "Button",
			definition: "export const Button = (props: ButtonProps) => <button>{props.label}</button>\n",
			wantKind:   "function",
		},
		{
			name:       "typed React FC component",
			symbol:     "Button",
			definition: "export const Button: React.FC<ButtonProps> = (props) => <button>{props.label}</button>\n",
			wantKind:   "function",
		},
		{
			name:       "generic arrow component",
			symbol:     "Button",
			definition: "export const Button = <T,>(props: { value: T }) => <button>{String(props.value)}</button>\n",
			wantKind:   "function",
		},
		{
			name:       "function expression component",
			symbol:     "Button",
			definition: "export const Button = function Button(props: ButtonProps) { return <button>{props.label}</button> }\n",
			wantKind:   "function",
		},
		{
			name:       "interface props",
			symbol:     "ButtonProps",
			definition: "export interface ButtonProps { label: string }\n",
			app:        "import type { ButtonProps } from './Button'\nconst props: ButtonProps = { label: 'Save' }\n",
			wantKind:   "interface",
		},
		{
			name:       "type props",
			symbol:     "ButtonProps",
			definition: "type ButtonProps = { label: string }\n",
			app:        "const props: ButtonProps = { label: 'Save' }\n",
			wantKind:   "type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := tt.app
			if app == "" {
				app = "import { Button } from './Button'\nexport function App() { return <Button label=\"Save\" /> }\n"
			}
			dir := setupMultiLangDir(t, map[string]string{
				"src/Button.tsx": tt.definition,
				"src/App.tsx":    app,
			})

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTSXImpactSearchOptions(dir, tt.symbol))

			assertTypeScriptStructuredImpactArtifact(t, artifact, tt.symbol, tt.wantKind)
			if got := artifact.Metadata.Bundle.Impact.RecommendedReads[0].File; got != "src/Button.tsx" {
				t.Fatalf("definition recommended file = %q, want src/Button.tsx", got)
			}
		})
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TSXStructuredImpactJSXUsageCallers(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.tsx": "export function Button() { return <button /> }\n",
		"src/App.tsx": "import { Button } from './Button'\n" +
			"type ButtonProps = { label?: string }\n" +
			"export function App() { return <Button /> }\n" +
			"export function AppWithProps() { return <Button label=\"Save\" /> }\n" +
			"export function AppWithChildren() { return <Button>Save</Button> }\n" +
			"export function AppMultiline() {\n" +
			"  return <Button\n" +
			"    label=\"Save\"\n" +
			"  />\n" +
			"}\n" +
			"export function AppGeneric() { return <Button<ButtonProps> /> }\n",
		"src/Namespaced.tsx":  "export function View() { return <UI.Button /> }\nexport function Sub() { return <Button.Sub /> }\n",
		"src/StringOnly.tsx":  "const html = \"<Button />\"\n",
		"src/CommentOnly.tsx": "// <Button />\n",
		"src/Button.test.tsx": "import { Button } from './Button'\nit('renders', () => <Button />)\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTSXImpactSearchOptions(dir, "Button"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "Button", "function")
	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	for _, want := range []string{"<Button />", `<Button label="Save" />`, "<Button>Save</Button>"} {
		if !symbolBundleItemsContainSnippet(callers, want) {
			t.Fatalf("callers = %+v, want snippet %q", callers, want)
		}
	}
	for _, wantLine := range []int{3, 4, 5, 7, 11} {
		if !slices.ContainsFunc(callers, func(item SymbolBundleItem) bool {
			return item.File == "src/App.tsx" && item.Line == wantLine
		}) {
			t.Fatalf("callers = %+v, want src/App.tsx:%d", callers, wantLine)
		}
	}
	for _, notCaller := range []string{"<UI.Button />", "<Button.Sub />", `"<Button />"`, "// <Button />"} {
		if symbolBundleItemsContainSnippet(callers, notCaller) {
			t.Fatalf("callers = %+v, did not want JSX non-caller %q", callers, notCaller)
		}
	}
	if symbolBundleItemsContainFile(callers, "src/Button.test.tsx") {
		t.Fatalf("direct TSX test should be separated from callers, got %+v", callers)
	}
	if !recommendedReadsContainFile(artifact.Metadata.Bundle, "src/Button.test.tsx") {
		t.Fatalf("expected direct TSX test in recommended reads, got %v", recommendedReadFiles(artifact.Metadata.Bundle))
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TSXStructuredImpactPrefersImplementationOverPairedDeclaration(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.tsx":  "export function Button() { return <button /> }\n",
		"src/Button.d.ts": "export declare function Button(): JSX.Element\n",
		"src/App.tsx":     "import { Button } from './Button'\nexport function App() { return <Button /> }\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTSXImpactSearchOptions(dir, "Button"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "Button", "function")
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/Button.tsx" {
		t.Fatalf("definition file = %q, want src/Button.tsx", got)
	}
	if strings.Contains(artifact.Rendered, `Multiple definitions found for "Button"`) {
		t.Fatalf("paired declaration should not force ambiguity when TSX implementation exists, got:\n%s", artifact.Rendered)
	}

	reads := artifact.Metadata.Bundle.Impact.RecommendedReads
	assertRecommendedReadAt(t, reads, 0, "definition", "src/Button.tsx")
	if !slices.ContainsFunc(reads, func(item SymbolBundleItem) bool {
		return item.Kind == "callers" && item.File == "src/App.tsx"
	}) {
		t.Fatalf("RecommendedReads = %+v, want src/App.tsx JSX usage caller", reads)
	}
	assertTypeScriptImpactBundleExcludesEvidenceFile(t, artifact.Metadata.Bundle, "src/Button.d.ts")

	callers := symbolBundleSectionItems(artifact.Metadata.Bundle, "callers")
	if !symbolBundleItemsContainFile(callers, "src/App.tsx") {
		t.Fatalf("callers = %+v, want src/App.tsx JSX usage", callers)
	}
	for _, want := range []string{filepath.Join(dir, "src/Button.tsx"), filepath.Join(dir, "src/App.tsx")} {
		if !slices.Contains(artifact.Metadata.AffectedFiles, want) {
			t.Fatalf("AffectedFiles = %v, want %s", artifact.Metadata.AffectedFiles, want)
		}
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TSXStructuredImpactScopesOutUnrelatedDeclaration(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.tsx":    "export function Button() { return <button /> }\n",
		"src/App.tsx":       "import { Button } from './Button'\nexport function App() { return <Button /> }\n",
		"types/Button.d.ts": "export interface Button { id: string }\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTSXImpactSearchOptions(dir, "Button"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "Button", "function")
	if got := artifact.Metadata.Bundle.Definition.File; got != "src/Button.tsx" {
		t.Fatalf("definition file = %q, want src/Button.tsx", got)
	}
	if strings.Contains(artifact.Rendered, `Multiple definitions found for "Button"`) {
		t.Fatalf("unrelated declaration should remain outside file_filter=tsx impact scope, got:\n%s", artifact.Rendered)
	}
	assertTypeScriptImpactBundleExcludesEvidenceFile(t, artifact.Metadata.Bundle, "types/Button.d.ts")
}

func TestExecuteSearchCodeArtifactWithConfig_TSXStructuredImpactRelatedTestsOrder(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/Button.tsx":                "export function Button() { return <button /> }\n",
		"src/App.tsx":                   "import { Button } from './Button'\nexport function App() { return <Button /> }\n",
		"src/Button.test.tsx":           "import { Button } from './Button'\nit('renders', () => <Button />)\n",
		"src/Button.spec.tsx":           "import { describe } from 'vitest'\ndescribe('component', () => {})\n",
		"src/__tests__/Button.test.tsx": "import { describe } from 'vitest'\ndescribe('nested component', () => {})\n",
		"tests/Button.test.tsx":         "import { describe } from 'vitest'\ndescribe('workspace component', () => {})\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTSXImpactSearchOptions(dir, "Button"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "Button", "function")
	reads := artifact.Metadata.Bundle.Impact.RecommendedReads
	assertRecommendedReadAt(t, reads, 0, "definition", "src/Button.tsx")
	assertRecommendedReadAt(t, reads, 1, "callers", "src/App.tsx")
	assertRecommendedReadAt(t, reads, 2, "tests", "src/Button.test.tsx")
	assertRecommendedReadAt(t, reads, 3, "imports", "src/App.tsx")
	assertRecommendedReadAt(t, reads, 4, "tests", "src/Button.spec.tsx")

	tests := symbolBundleSectionItems(artifact.Metadata.Bundle, "tests")
	for _, want := range []string{"src/Button.test.tsx", "src/Button.spec.tsx", "src/__tests__/Button.test.tsx", "tests/Button.test.tsx"} {
		if !symbolBundleItemsContainFile(tests, want) {
			t.Fatalf("related test section = %+v, want %s", tests, want)
		}
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TSXStructuredImpactFiltersPatternsAndPath(t *testing.T) {
	tests := []struct {
		name string
		opts func(dir string) SearchOptions
	}{
		{
			name: "tsx file filter",
			opts: func(dir string) SearchOptions {
				return newTSXImpactSearchOptions(dir, "Button")
			},
		},
		{
			name: "basename tsx glob",
			opts: func(dir string) SearchOptions {
				return newTypeScriptImpactFilePatternSearchOptions(dir, "Button", "*.tsx")
			},
		},
		{
			name: "double star tsx glob",
			opts: func(dir string) SearchOptions {
				return newTypeScriptImpactFilePatternSearchOptions(dir, "Button", "**/*.tsx")
			},
		},
		{
			name: "src double star tsx glob",
			opts: func(dir string) SearchOptions {
				return newTypeScriptImpactFilePatternSearchOptions(dir, "Button", "src/**/*.tsx")
			},
		},
		{
			name: "direct tsx path",
			opts: func(dir string) SearchOptions {
				return SearchOptions{
					Pattern:       "Button",
					Intent:        "impact",
					Path:          filepath.Join(dir, "src", "Button.tsx"),
					InvocationCWD: dir,
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, map[string]string{
				"src/Button.tsx": "export function Button() { return <button /> }\n",
				"src/App.tsx":    "export function App() { return <Button /> }\n",
				"src/Button.ts":  "export function Button() { return 'ts' }\n",
			})

			artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, tt.opts(dir))

			assertTypeScriptStructuredImpactArtifact(t, artifact, "Button", "function")
			if !strings.Contains(artifact.Metadata.Bundle.Definition.File, "src/Button.tsx") {
				t.Fatalf("definition file = %q, want TSX implementation", artifact.Metadata.Bundle.Definition.File)
			}
			if strings.Contains(artifact.Rendered, "src/Button.ts ") {
				t.Fatalf("TSX structured impact should not mix .ts implementation, got:\n%s", artifact.Rendered)
			}
		})
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TSXStructuredImpactCacheKeepsRecommendedReadsAndScope(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	dir := setupMultiLangDir(t, map[string]string{
		"src/shared.ts":  "export function buildUser() { return 'ts' }\n",
		"src/shared.tsx": "export function buildUser() { return <button /> }\n",
		"src/app.ts":     "buildUser()\n",
		"src/App.tsx":    "export function App() { return <buildUser /> }\n",
	})
	cache := &testSearchCache{data: make(map[string]string)}

	tsFirst := ExecuteSearchCodeArtifactWithConfig(nil, cache, newTypeScriptImpactSearchOptions(dir, "buildUser"))
	assertTypeScriptStructuredImpactArtifact(t, tsFirst, "buildUser", "function")
	if got := tsFirst.Metadata.Bundle.Definition.File; got != "src/shared.ts" {
		t.Fatalf("ts definition file = %q, want src/shared.ts", got)
	}

	tsxFirst := ExecuteSearchCodeArtifactWithConfig(nil, cache, newTSXImpactSearchOptions(dir, "buildUser"))
	assertTypeScriptStructuredImpactArtifact(t, tsxFirst, "buildUser", "function")
	if got := tsxFirst.Metadata.Bundle.Definition.File; got != "src/shared.tsx" {
		t.Fatalf("tsx definition file = %q, want src/shared.tsx", got)
	}
	if len(tsxFirst.Metadata.Bundle.Impact.RecommendedReads) == 0 {
		t.Fatal("TSX structured impact lost recommended reads")
	}

	tsxSecond := ExecuteSearchCodeArtifactWithConfig(nil, cache, newTSXImpactSearchOptions(dir, "buildUser"))
	assertTypeScriptStructuredImpactArtifact(t, tsxSecond, "buildUser", "function")
	if got := tsxSecond.Metadata.Bundle.Definition.File; got != "src/shared.tsx" {
		t.Fatalf("cached tsx definition file = %q, want src/shared.tsx", got)
	}
	if len(tsxSecond.Metadata.Bundle.Impact.RecommendedReads) == 0 {
		t.Fatal("cached TSX structured impact lost recommended reads")
	}
}

func TestExecuteSearchCodeArtifactWithConfig_TypeScriptStructuredImpactKeepsFunctionExpressionKindInTS(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"src/build.ts": "export const buildUser = function buildUser() { return 'ts' }\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, newTypeScriptImpactSearchOptions(dir, "buildUser"))

	assertTypeScriptStructuredImpactArtifact(t, artifact, "buildUser", "const")
}

func TestClassifyJSRefs_JSXUsage(t *testing.T) {
	refs := []genericSymbolRef{
		{File: "app.tsx", Line: 1, Snippet: "return <Button />"},
		{File: "app.tsx", Line: 2, Snippet: `return <Button label="Save" />`},
		{File: "app.tsx", Line: 3, Snippet: "return <Button>Save</Button>"},
		{File: "app.tsx", Line: 4, Snippet: "return <Button"},
		{File: "app.tsx", Line: 5, Snippet: "return <Button<ButtonProps> />"},
		{File: "app.tsx", Line: 6, Snippet: "return <button />"},
		{File: "app.tsx", Line: 7, Snippet: "return <UI.Button />"},
		{File: "app.tsx", Line: 8, Snippet: "return <Button.Sub />"},
		{File: "app.tsx", Line: 9, Snippet: `const html = "<Button />"`},
		{File: "app.tsx", Line: 10, Snippet: "// <Button />"},
		{File: "app.tsx", Line: 11, Snippet: "render<Button>()"},
		{File: "app.ts", Line: 12, Snippet: "const asserted = <Button>raw"},
	}

	_, callers, _, _ := classifyJSRefs(refs, "Button")

	if len(callers) != 5 {
		t.Fatalf("callers = %+v, want exactly five TSX component usages", callers)
	}
	for _, wantLine := range []int{1, 2, 3, 4, 5} {
		if !slices.ContainsFunc(callers, func(ref genericSymbolRef) bool { return ref.Line == wantLine }) {
			t.Fatalf("callers = %+v, want line %d", callers, wantLine)
		}
	}
	for _, notCallerLine := range []int{6, 7, 8, 9, 10, 11, 12} {
		if slices.ContainsFunc(callers, func(ref genericSymbolRef) bool { return ref.Line == notCallerLine }) {
			t.Fatalf("callers = %+v, did not want line %d", callers, notCallerLine)
		}
	}
}

func symbolBundleSectionItems(bundle *SymbolBundle, kind string) []SymbolBundleItem {
	if bundle == nil {
		return nil
	}
	for _, section := range bundle.Sections {
		if section.Kind == kind {
			return section.Items
		}
	}
	return nil
}

func symbolBundleItemsContainSnippet(items []SymbolBundleItem, snippet string) bool {
	for _, item := range items {
		if strings.Contains(item.Snippet, snippet) {
			return true
		}
	}
	return false
}

func symbolBundleItemsContainFile(items []SymbolBundleItem, file string) bool {
	for _, item := range items {
		if item.File == file {
			return true
		}
	}
	return false
}
