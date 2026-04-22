package navigation

import "testing"

func TestReceiverTypeFromGoExpr_SelectorChainReturnsEmpty(t *testing.T) {
	// foo.Bar.Method() のような selector chain では ReceiverType を推定しない
	setupTestGoFiles(t, map[string]string{
		"main.go": `package main

type Foo struct {
	Bar BarType
}

type BarType struct{}

func (BarType) Build() string {
	return ""
}

func Use(f Foo) string {
	return f.Bar.Build()
}
`,
	})

	cache := newReferenceParseCache()
	ref := parseRipgrepLine("main.go:14:return f.Bar.Build()", "Build", cache)
	if ref == nil {
		t.Fatal("expected reference to parse")
	}
	if ref.SelectorKind != "method" {
		t.Fatalf("selectorKind = %q, want method", ref.SelectorKind)
	}
	// selector chain ではフィールド名を型名として誤帰属しないことを確認
	if ref.ReceiverType == "Bar" {
		t.Fatal("ReceiverType should not be 'Bar' (field name, not type)")
	}
}

func TestSelectorKindWithShadowedImport_ShortVarDecl(t *testing.T) {
	// ローカル変数がインポート名をシャドーイングする場合、method として判定する
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

func UseShadowed() string {
	pkg, _ := newConfig()
	return pkg.Build()
}

func newConfig() (*Config, error) {
	return &Config{}, nil
}
`,
	})

	cache := newReferenceParseCache()
	ref := parseRipgrepLine("main.go:13:return pkg.Build()", "Build", cache)
	if ref == nil {
		t.Fatal("expected reference to parse")
	}
	if ref.SelectorKind == "package" {
		t.Fatal("selectorKind should not be 'package' when import is shadowed by local variable")
	}
}

func TestSelectorKindWithShadowedImport_GroupedParam(t *testing.T) {
	// グループ化パラメータでインポート名がシャドーイングされる場合
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

func UseGrouped(pkg, other Config) string {
	return pkg.Build()
}
`,
	})

	cache := newReferenceParseCache()
	ref := parseRipgrepLine("main.go:12:return pkg.Build()", "Build", cache)
	if ref == nil {
		t.Fatal("expected reference to parse")
	}
	if ref.SelectorKind == "package" {
		t.Fatal("selectorKind should not be 'package' when import is shadowed by grouped parameter")
	}
}

func TestSelectorKindWithShadowedImport_NestedBlock(t *testing.T) {
	// if ブロック内の短変数宣言でインポート名がシャドーイングされる場合
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

func UseNestedShadow() string {
	if true {
		pkg := &Config{}
		return pkg.Build()
	}
	return ""
}
`,
	})

	cache := newReferenceParseCache()
	ref := parseRipgrepLine("main.go:14:return pkg.Build()", "Build", cache)
	if ref == nil {
		t.Fatal("expected reference to parse")
	}
	if ref.SelectorKind == "package" {
		t.Fatal("selectorKind should not be 'package' when import is shadowed inside nested block")
	}
}

func TestSelectorKindWithShadowedImport_SiblingBlock(t *testing.T) {
	// 兄弟ブロック内の宣言はスコープ外 → package として扱う
	setupTestGoFiles(t, map[string]string{
		"pkg/build.go": `package pkg

func Build() string {
	return "pkg"
}
`,
		"main.go": `package main

import "example/pkg"

func UseSiblingBlock() string {
	if true {
		pkg := 1
		_ = pkg
	}
	return pkg.Build()
}
`,
	})

	cache := newReferenceParseCache()
	ref := parseRipgrepLine("main.go:10:return pkg.Build()", "Build", cache)
	if ref == nil {
		t.Fatal("expected reference to parse")
	}
	if ref.SelectorKind != "package" {
		t.Fatalf("selectorKind = %q, want package (sibling block var is out of scope)", ref.SelectorKind)
	}
}
