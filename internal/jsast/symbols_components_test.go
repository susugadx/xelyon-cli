package jsast

import (
	"slices"
	"testing"
)

func TestExtractSymbolsDefaultWrappedComponents(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want struct {
			name     string
			kind     string
			exported bool
		}
	}{
		{
			name: "memo function",
			src:  "import { memo } from 'react'\nexport default memo(function DefaultButton() { return <button /> })\n",
			want: struct {
				name     string
				kind     string
				exported bool
			}{name: "DefaultButton", kind: "function", exported: true},
		},
		{
			name: "react memo class",
			src:  "import React from 'react'\nexport default React.memo(class DefaultButton extends React.Component { render() { return <button /> } })\n",
			want: struct {
				name     string
				kind     string
				exported bool
			}{name: "DefaultButton", kind: "class", exported: true},
		},
		{
			name: "forwardRef function with nested type arguments",
			src:  "import { forwardRef } from 'react'\nexport default forwardRef<HTMLButtonElement, Props<Foo>>(function ForwardButton() { return <button /> })\n",
			want: struct {
				name     string
				kind     string
				exported bool
			}{name: "ForwardButton", kind: "function", exported: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbols, err := ExtractSymbols("src/Button.tsx", []byte(tt.src))
			if err != nil {
				t.Fatalf("ExtractSymbols() error = %v", err)
			}
			if !slices.ContainsFunc(symbols, func(symbol Symbol) bool {
				return symbol.Name == tt.want.name &&
					symbol.Kind == tt.want.kind &&
					symbol.Exported == tt.want.exported
			}) {
				t.Fatalf("symbols = %+v, want %+v", symbols, tt.want)
			}
		})
	}
}

func TestExtractSymbolsDefaultWrappedComponentUsesDefinitionNameCharacter(t *testing.T) {
	line := "export default forwardRef<HTMLButtonElement, ForwardButtonProps<Foo>>(function ForwardButton() { return <button /> })"
	symbols, err := ExtractSymbols("src/Button.tsx", []byte(line+"\n"))
	if err != nil {
		t.Fatalf("ExtractSymbols() error = %v", err)
	}

	wantStart, _ := testLSPRangeForToken(line, "(function ForwardButton")
	wantStart += len("(function ")
	for _, symbol := range symbols {
		if symbol.Name == "ForwardButton" {
			if symbol.Character != wantStart {
				t.Fatalf("ForwardButton character = %d, want %d; symbols = %+v", symbol.Character, wantStart, symbols)
			}
			return
		}
	}
	t.Fatalf("symbols = %+v, want ForwardButton", symbols)
}
