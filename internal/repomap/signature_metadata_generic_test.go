package repomap

import "testing"

func TestExtractSignatureMetadata(t *testing.T) {
	tests := []struct {
		name     string
		sig      string
		wantName string
		wantKind string
	}{
		{
			name:     "Go method",
			sig:      "func (a *Agent) maybeAutoCompress() bool",
			wantName: "maybeAutoCompress",
			wantKind: "method",
		},
		{
			name:     "Python async def",
			sig:      "async def build_map():",
			wantName: "build_map",
			wantKind: "function",
		},
		{
			name:     "TypeScript interface",
			sig:      "export interface Config",
			wantName: "Config",
			wantKind: "interface",
		},
		{
			name:     "TypeScript async arrow function",
			sig:      "const buildMap = async (): Promise<Map<string, string>> => {",
			wantName: "buildMap",
			wantKind: "function",
		},
		{
			name:     "TSX function expression stays const in generic extractor",
			sig:      "export const Button = function Button() { return <button /> }",
			wantName: "Button",
			wantKind: "const",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotKind, ok := extractSignatureMetadata(tt.sig)
			if !ok {
				t.Fatalf("extractSignatureMetadata(%q) returned ok=false", tt.sig)
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
