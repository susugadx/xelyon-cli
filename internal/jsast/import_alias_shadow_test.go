package jsast

import "testing"

func TestNamedImportAliasJSXUsageSkipsValueDeclarationShadows(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantLines []int
	}{
		{
			name: "function declaration",
			body: "export function ShadowedFunction() {\n" +
				"  function PrimaryButton() {}\n" +
				"  return <PrimaryButton />\n" +
				"}\n",
			wantLines: []int{2},
		},
		{
			name: "class declaration",
			body: "export function ShadowedClass() {\n" +
				"  class PrimaryButton {}\n" +
				"  return <PrimaryButton />\n" +
				"}\n",
			wantLines: []int{2},
		},
		{
			name: "enum declaration",
			body: "export function ShadowedEnum() {\n" +
				"  enum PrimaryButton { One }\n" +
				"  return <PrimaryButton />\n" +
				"}\n",
			wantLines: []int{2},
		},
		{
			name: "abstract class declaration",
			body: "export function ShadowedAbstractClass() {\n" +
				"  abstract class PrimaryButton {}\n" +
				"  return <PrimaryButton />\n" +
				"}\n",
			wantLines: []int{2},
		},
		{
			name: "namespace declaration",
			body: "namespace PrimaryButton { export const One = 1 }\n" +
				"export function ShadowedNamespace() { return <PrimaryButton /> }\n",
			wantLines: nil,
		},
		{
			name: "nested namespace declaration",
			body: "namespace PrimaryButton.Inner { export const One = 1 }\n" +
				"export function ShadowedNestedNamespace() { return <PrimaryButton /> }\n",
			wantLines: nil,
		},
	}
	assertNamedImportAliasUsageLines(t, tests)
}

func TestNamedImportAliasJSXUsageKeepsTypeOnlyDeclarations(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantLines []int
	}{
		{
			name: "type alias",
			body: "export function TypeOnlyAlias() {\n" +
				"  type PrimaryButton = { label: string }\n" +
				"  return <PrimaryButton dataType />\n" +
				"}\n",
			wantLines: []int{2, 5},
		},
		{
			name: "interface",
			body: "export function TypeOnlyInterface() {\n" +
				"  interface PrimaryButton { label: string }\n" +
				"  return <PrimaryButton dataInterface />\n" +
				"}\n",
			wantLines: []int{2, 5},
		},
	}
	assertNamedImportAliasUsageLines(t, tests)
}

func TestNamedImportAliasJSXUsageUsesLexicalScopeBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantLines []int
	}{
		{
			name: "block scoped const",
			body: "export function Blocked() {\n" +
				"  if (ready) {\n" +
				"    const PrimaryButton = Other\n" +
				"    return <PrimaryButton />\n" +
				"  }\n" +
				"  return <PrimaryButton label=\"Save\" />\n" +
				"}\n",
			wantLines: []int{2, 8},
		},
		{
			name: "block scoped destructured shorthand",
			body: "export function BlockedDestructure() {\n" +
				"  if (ready) {\n" +
				"    const { PrimaryButton } = props\n" +
				"    return <PrimaryButton />\n" +
				"  }\n" +
				"  return <PrimaryButton dataLate />\n" +
				"}\n",
			wantLines: []int{2, 8},
		},
		{
			name: "switch scoped const",
			body: "export function SwitchScoped(kind) {\n" +
				"  switch (kind) {\n" +
				"  case 'x':\n" +
				"    const PrimaryButton = Other\n" +
				"    render(<PrimaryButton dataSwitch />)\n" +
				"    break\n" +
				"  }\n" +
				"  return <PrimaryButton dataAfterSwitch />\n" +
				"}\n",
			wantLines: []int{2, 10},
		},
		{
			name: "switch scoped destructured shorthand",
			body: "export function SwitchDestructure(kind) {\n" +
				"  switch (kind) {\n" +
				"  case 'x':\n" +
				"    const { PrimaryButton } = props\n" +
				"    render(<PrimaryButton dataSwitch />)\n" +
				"    break\n" +
				"  }\n" +
				"  return <PrimaryButton dataAfterSwitch />\n" +
				"}\n",
			wantLines: []int{2, 10},
		},
		{
			name: "block scoped enum declaration",
			body: "export function BlockedEnum() {\n" +
				"  if (ready) {\n" +
				"    enum PrimaryButton { One }\n" +
				"    return <PrimaryButton dataEnum />\n" +
				"  }\n" +
				"  return <PrimaryButton dataAfter />\n" +
				"}\n",
			wantLines: []int{2, 8},
		},
	}
	assertNamedImportAliasUsageLines(t, tests)
}

func TestNamedImportAliasJSXUsageSkipsFunctionScopedVarShadows(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantLines []int
	}{
		{
			name: "var declaration",
			body: "export function VarShadowed() {\n" +
				"  var PrimaryButton = Other\n" +
				"  return <PrimaryButton />\n" +
				"}\n",
			wantLines: []int{2},
		},
		{
			name: "switch var binding",
			body: "export function SwitchVar(kind) {\n" +
				"  switch (kind) {\n" +
				"  case 'x':\n" +
				"    var PrimaryButton = Other\n" +
				"    render(<PrimaryButton dataSwitch />)\n" +
				"    break\n" +
				"  }\n" +
				"  return <PrimaryButton dataAfterSwitch />\n" +
				"}\n",
			wantLines: []int{2},
		},
		{
			name: "for-of var binding",
			body: "export function LoopVar(items) {\n" +
				"  for (var PrimaryButton of items) {\n" +
				"    render(<PrimaryButton dataInside />)\n" +
				"  }\n" +
				"  return <PrimaryButton dataAfter />\n" +
				"}\n",
			wantLines: []int{2},
		},
	}
	assertNamedImportAliasUsageLines(t, tests)
}

func TestNamedImportAliasJSXUsageSkipsParameterAndCatchShadows(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantLines []int
	}{
		{
			name: "parameter",
			body: "export function ShadowedParam(PrimaryButton) {\n" +
				"  return <PrimaryButton />\n" +
				"}\n",
			wantLines: []int{2},
		},
		{
			name: "destructured parameter shorthand",
			body: "export function ShadowedDestructuredParam({ PrimaryButton }) {\n" +
				"  return <PrimaryButton />\n" +
				"}\n",
			wantLines: []int{2},
		},
		{
			name: "destructured parameter alias",
			body: "export function ShadowedDestructuredAliasParam({ Button: PrimaryButton }) {\n" +
				"  return <PrimaryButton />\n" +
				"}\n",
			wantLines: []int{2},
		},
		{
			name: "destructured array parameter",
			body: "export function ShadowedArrayParam([PrimaryButton]) {\n" +
				"  return <PrimaryButton />\n" +
				"}\n",
			wantLines: []int{2},
		},
		{
			name: "catch parameter",
			body: "export function Caught() {\n" +
				"  try { risky() } catch (PrimaryButton) {\n" +
				"    return <PrimaryButton />\n" +
				"  }\n" +
				"}\n",
			wantLines: []int{2},
		},
		{
			name: "destructured catch parameter",
			body: "export function CaughtDestructured() {\n" +
				"  try { risky() } catch ({ PrimaryButton }) {\n" +
				"    return <PrimaryButton />\n" +
				"  }\n" +
				"}\n",
			wantLines: []int{2},
		},
		{
			name:      "arrow parameter",
			body:      "export const ArrowShadowed = PrimaryButton => <PrimaryButton />\n",
			wantLines: []int{2},
		},
	}
	assertNamedImportAliasUsageLines(t, tests)
}

func TestNamedImportAliasJSXUsageUsesLoopBindingScopes(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantLines []int
	}{
		{
			name: "for-of let binding",
			body: "export function LoopDirect(items) {\n" +
				"  for (let PrimaryButton of items) {\n" +
				"    render(<PrimaryButton />)\n" +
				"  }\n" +
				"  return <PrimaryButton dataAfter />\n" +
				"}\n",
			wantLines: []int{2, 7},
		},
		{
			name: "for-of source expression is not a binding",
			body: "export function LoopSource(items) {\n" +
				"  for (let item of PrimaryButton) {\n" +
				"    render(<PrimaryButton dataInside />)\n" +
				"  }\n" +
				"}\n",
			wantLines: []int{2, 5},
		},
		{
			name: "classic for let binding",
			body: "export function LoopClassic(items) {\n" +
				"  render(<PrimaryButton dataBefore />)\n" +
				"  for (let PrimaryButton = 0; PrimaryButton < items.length; PrimaryButton++) {\n" +
				"    render(<PrimaryButton />)\n" +
				"  }\n" +
				"  return <PrimaryButton dataAfter />\n" +
				"}\n",
			wantLines: []int{2, 4, 8},
		},
		{
			name: "classic for condition is not a binding",
			body: "export function LoopCondition(items) {\n" +
				"  for (let i = 0; PrimaryButton && i < items.length; i++) {\n" +
				"    render(<PrimaryButton dataCondition />)\n" +
				"  }\n" +
				"}\n",
			wantLines: []int{2, 5},
		},
		{
			name: "for-of destructured binding",
			body: "export function LoopDestructured(items) {\n" +
				"  for (const { PrimaryButton } of items) {\n" +
				"    render(<PrimaryButton />)\n" +
				"  }\n" +
				"  return <PrimaryButton dataAfter />\n" +
				"}\n",
			wantLines: []int{2, 7},
		},
	}
	assertNamedImportAliasUsageLines(t, tests)
}

func TestNamedImportAliasJSXUsageSkipsSelfScopedExpressionNames(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantLines []int
	}{
		{
			name: "named function expression",
			body: "export const Wrapped = function PrimaryButton() {\n" +
				"  return <PrimaryButton />\n" +
				"}\n",
			wantLines: []int{2},
		},
		{
			name: "named class expression",
			body: "export const Wrapped = class PrimaryButton {\n" +
				"  render() { return <PrimaryButton /> }\n" +
				"}\n",
			wantLines: []int{2},
		},
	}
	assertNamedImportAliasUsageLines(t, tests)
}

func assertNamedImportAliasUsageLines(t *testing.T, tests []struct {
	name      string
	body      string
	wantLines []int
}) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()
			assertNamedImportAliasUsageLinesForBody(t, tt.body, tt.wantLines)
		})
	}
}

func assertNamedImportAliasUsageLinesForBody(t *testing.T, body string, wantLines []int) {
	t.Helper()
	src := []byte("import { Button as PrimaryButton } from './Button'\n" +
		"export function App() { return <PrimaryButton /> }\n" +
		body)
	parsed, err := ParseBytes("src/App.tsx", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	defer parsed.Close()

	aliases := NamedImportAliasesWithParsed(parsed, "Button")
	if len(aliases) != 1 {
		t.Fatalf("aliases = %+v, want one value alias", aliases)
	}
	usages := JSXLocalNameUsagesForNamedImportAliasWithParsed(parsed, aliases[0])

	if len(usages) != len(wantLines) {
		t.Fatalf("usages = %+v, want lines %v", usages, wantLines)
	}
	for i, wantLine := range wantLines {
		if usages[i].Line != wantLine {
			t.Fatalf("usage %d line = %d, want %d; usages=%+v", i, usages[i].Line, wantLine, usages)
		}
	}
}
