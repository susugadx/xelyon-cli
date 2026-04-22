package navigation

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

func TestApplySnippetReferenceHints_PromotesSelectorCallFromRef(t *testing.T) {
	class, nodeType, selectorKind, receiverType := applySnippetReferenceHints(
		"return pkg.Build()",
		"Build",
		ast.ClassRef,
		"field_identifier",
		"package",
		"",
	)

	if class != ast.ClassCall {
		t.Fatalf("class = %s, want %s", class, ast.ClassCall)
	}
	if nodeType != "field_identifier" {
		t.Fatalf("nodeType = %q, want field_identifier", nodeType)
	}
	if selectorKind != "package" {
		t.Fatalf("selectorKind = %q, want package", selectorKind)
	}
	if receiverType != "" {
		t.Fatalf("receiverType = %q, want empty", receiverType)
	}
}

func TestParseRipgrepLine_UsesGoParserFallbackForScopeAndSelectors(t *testing.T) {
	setupTestGoFiles(t, map[string]string{
		"pkg/build.go": `package pkg

		func Build() string {
			return "pkg"
		}
		`,
		"main.go": `package main

		import "example/pkg"

		type Config struct{}

		func (Config) Build() string {
			return "method"
		}

		func UsePkg() string {
			return pkg.Build()
		}

		func UseMethod(c Config) string {
			return c.Build()
		}
		`,
	})

	cache := newReferenceParseCache()

	defRef := parseRipgrepLine("main.go:7:func (Config) Build() string {", "Build", cache)
	if defRef == nil {
		t.Fatal("expected method definition line to parse")
	}
	if defRef.Class != ast.ClassDef {
		t.Fatalf("definition class = %s, want %s", defRef.Class, ast.ClassDef)
	}

	pkgCall := parseRipgrepLine("main.go:12:return pkg.Build()", "Build", cache)
	if pkgCall == nil {
		t.Fatal("expected package selector call to parse")
	}
	if pkgCall.Scope != "func UsePkg" {
		t.Fatalf("pkg call scope = %q, want func UsePkg", pkgCall.Scope)
	}
	if pkgCall.Class != ast.ClassCall {
		t.Fatalf("pkg call class = %s, want %s", pkgCall.Class, ast.ClassCall)
	}
	if pkgCall.SelectorKind != "package" {
		t.Fatalf("pkg call selectorKind = %q, want package", pkgCall.SelectorKind)
	}

	methodCall := parseRipgrepLine("main.go:16:return c.Build()", "Build", cache)
	if methodCall == nil {
		t.Fatal("expected method selector call to parse")
	}
	if methodCall.Scope != "func UseMethod" {
		t.Fatalf("method call scope = %q, want func UseMethod", methodCall.Scope)
	}
	if methodCall.SelectorKind != "method" {
		t.Fatalf("method call selectorKind = %q, want method", methodCall.SelectorKind)
	}
}

func TestCacheReusesFileData(t *testing.T) {
	setupTestGoFiles(t, map[string]string{
		"main.go": `package main

func Build() string {
	return ""
}

func A() string {
	return Build()
}

func B() string {
	return Build()
}
`,
	})

	cache := newReferenceParseCache()
	ref1 := parseRipgrepLine("main.go:8:return Build()", "Build", cache)
	ref2 := parseRipgrepLine("main.go:12:return Build()", "Build", cache)

	if ref1 == nil || ref2 == nil {
		t.Fatal("expected both references to parse")
	}
	if ref1.Scope != "func A" {
		t.Fatalf("ref1 scope = %q, want func A", ref1.Scope)
	}
	if ref2.Scope != "func B" {
		t.Fatalf("ref2 scope = %q, want func B", ref2.Scope)
	}

	// キャッシュが1ファイル分のみ保持していることを確認
	if len(cache.files) != 1 {
		t.Fatalf("cache should have 1 file entry, got %d", len(cache.files))
	}
}
