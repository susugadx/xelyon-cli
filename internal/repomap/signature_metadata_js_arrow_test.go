package repomap

import "testing"

func TestExtractJSArrowFunctionMetadata(t *testing.T) {
	tests := []struct {
		name     string
		sig      string
		wantName string
		wantKind string
		wantOK   bool
	}{
		{
			name:     "const arrow",
			sig:      "const buildMap = () => {",
			wantName: "buildMap",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:     "const async arrow with return type",
			sig:      "const buildMap = async (): Promise<Map<string, string>> => {",
			wantName: "buildMap",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:     "export const async arrow",
			sig:      "export const buildMap = async () => {",
			wantName: "buildMap",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:     "export const arrow with repeated spaces",
			sig:      "export  const buildMap = () => {",
			wantName: "buildMap",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:     "const arrow with tab spacing",
			sig:      "const\tbuildMap = () => {",
			wantName: "buildMap",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:     "const async arrow with tab spacing",
			sig:      "const buildMap = async\t() => {",
			wantName: "buildMap",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:     "export const typed arrow",
			sig:      "export const buildMap: BuilderFactory = (input: string) => ({ input })",
			wantName: "buildMap",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:     "const typed arrow",
			sig:      "const buildMap: BuilderFactory = (input: string) => ({ input })",
			wantName: "buildMap",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:     "export const inline function type arrow",
			sig:      "export const buildMap: (input: string) => string = (input) => input",
			wantName: "buildMap",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:     "const generic inline function type arrow",
			sig:      "const buildMap: <T = string>(input: T) => T = (input) => input",
			wantName: "buildMap",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:     "export const generic arrow",
			sig:      "export const identity = <T>(value: T): T => value",
			wantName: "identity",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:     "const constrained generic arrow",
			sig:      "const identity = <T extends string>(value: T) => value",
			wantName: "identity",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:     "const async generic arrow",
			sig:      "const identity = async <T>(value: T) => value",
			wantName: "identity",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:     "generic arrow with function type constraint",
			sig:      "export const memoize = <T extends (...args: any[]) => any>(fn: T) => fn",
			wantName: "memoize",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:     "generic arrow with function type default",
			sig:      "export const memoize = <T = (...args: any[]) => any>(fn: T) => fn",
			wantName: "memoize",
			wantKind: "function",
			wantOK:   true,
		},
		{
			name:   "plain const value",
			sig:    "const buildMap = new Map()",
			wantOK: false,
		},
		{
			name:   "typed const value",
			sig:    "const buildMap: BuilderFactory = createBuilder()",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotKind, ok := extractJSArrowFunctionMetadata(tt.sig)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if gotName != tt.wantName {
				t.Fatalf("name = %q, want %q", gotName, tt.wantName)
			}
			if gotKind != tt.wantKind {
				t.Fatalf("kind = %q, want %q", gotKind, tt.wantKind)
			}
		})
	}
}
