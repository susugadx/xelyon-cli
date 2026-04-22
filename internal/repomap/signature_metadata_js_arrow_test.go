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
			name:   "plain const value",
			sig:    "const buildMap = new Map()",
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
