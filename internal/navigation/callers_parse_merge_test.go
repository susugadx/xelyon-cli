package navigation

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

func TestMergeFallbackReferenceClassification(t *testing.T) {
	tests := []struct {
		name     string
		current  referenceClassification
		fallback referenceClassification
		check    func(t *testing.T, got referenceClassification)
	}{
		{
			name: "definition always wins",
			current: referenceClassification{
				Class: ast.ClassCall,
			},
			fallback: referenceClassification{
				Class: ast.ClassDef,
			},
			check: func(t *testing.T, got referenceClassification) {
				if got.Class != ast.ClassDef {
					t.Fatalf("Class = %v, want %v", got.Class, ast.ClassDef)
				}
			},
		},
		{
			name: "unknown class adopts fallback",
			current: referenceClassification{
				Class: ast.ClassUnknown,
			},
			fallback: referenceClassification{
				Class: ast.ClassRef,
			},
			check: func(t *testing.T, got referenceClassification) {
				if got.Class != ast.ClassRef {
					t.Fatalf("Class = %v, want %v", got.Class, ast.ClassRef)
				}
			},
		},
		{
			name: "existing concrete class is preserved",
			current: referenceClassification{
				Class: ast.ClassCall,
			},
			fallback: referenceClassification{
				Class: ast.ClassRef,
			},
			check: func(t *testing.T, got referenceClassification) {
				if got.Class != ast.ClassCall {
					t.Fatalf("Class = %v, want %v", got.Class, ast.ClassCall)
				}
			},
		},
		{
			name: "scope and selector hints are filled only when missing",
			current: referenceClassification{
				Scope:        "package-level",
				Class:        ast.ClassUnknown,
				SelectorKind: "unknown",
			},
			fallback: referenceClassification{
				Scope:        "func Build",
				Class:        ast.ClassRef,
				NodeType:     "field_identifier",
				SelectorKind: "method",
				ReceiverType: "Config",
			},
			check: func(t *testing.T, got referenceClassification) {
				if got.Scope != "func Build" {
					t.Fatalf("Scope = %q, want func Build", got.Scope)
				}
				if got.NodeType != "field_identifier" {
					t.Fatalf("NodeType = %q, want field_identifier", got.NodeType)
				}
				if got.SelectorKind != "method" {
					t.Fatalf("SelectorKind = %q, want method", got.SelectorKind)
				}
				if got.ReceiverType != "Config" {
					t.Fatalf("ReceiverType = %q, want Config", got.ReceiverType)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeFallbackReferenceClassification(tt.current, tt.fallback)
			tt.check(t, got)
		})
	}
}

func TestReferenceClassificationFromReference(t *testing.T) {
	got := referenceClassificationFromReference(Reference{
		Scope:        "func Build",
		Class:        ast.ClassCall,
		NodeType:     "identifier",
		SelectorKind: "method",
		ReceiverType: "Config",
	})

	if got.Scope != "func Build" {
		t.Fatalf("Scope = %q, want func Build", got.Scope)
	}
	if got.Class != ast.ClassCall {
		t.Fatalf("Class = %v, want %v", got.Class, ast.ClassCall)
	}
	if got.NodeType != "identifier" {
		t.Fatalf("NodeType = %q, want identifier", got.NodeType)
	}
	if got.SelectorKind != "method" {
		t.Fatalf("SelectorKind = %q, want method", got.SelectorKind)
	}
	if got.ReceiverType != "Config" {
		t.Fatalf("ReceiverType = %q, want Config", got.ReceiverType)
	}
}
