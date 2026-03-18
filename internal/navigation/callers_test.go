package navigation

import (
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

type stagedErrorReader struct {
	chunks   [][]byte
	finalErr error
}

func (r *stagedErrorReader) Read(p []byte) (int, error) {
	if len(r.chunks) > 0 {
		chunk := r.chunks[0]
		n := copy(p, chunk)
		if n == len(chunk) {
			r.chunks = r.chunks[1:]
		} else {
			r.chunks[0] = r.chunks[0][n:]
		}
		return n, nil
	}
	if r.finalErr != nil {
		err := r.finalErr
		r.finalErr = nil
		return 0, err
	}
	return 0, io.EOF
}

func makeRipgrepOutput(symbol string, count int) string {
	var sb strings.Builder
	for i := range count {
		sb.WriteString("file.go:")
		sb.WriteString(strconv.Itoa(i + 1))
		sb.WriteString(":")
		sb.WriteString(symbol)
		sb.WriteString("()\n")
	}
	return sb.String()
}

func TestCollectReferenceSearchResult_LongLineDoesNotSilentlyDrop(t *testing.T) {
	symbol := "Target"
	longLine := "file.go:1:" + strings.Repeat("x", 70*1024) + symbol + "()\n"

	result := collectReferenceSearchResult(strings.NewReader(longLine), symbol)
	if result.Incomplete {
		t.Fatal("expected complete result for line larger than default scanner buffer")
	}
	if result.Truncated {
		t.Fatal("expected no truncation for single long line")
	}
	if len(result.Refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(result.Refs))
	}
	if !strings.Contains(result.Refs[0].Snippet, symbol+"()") {
		t.Fatalf("expected snippet to contain symbol call, got: %q", result.Refs[0].Snippet)
	}
}

func TestCollectReferenceSearchResult_ReaderErrorMarksIncomplete(t *testing.T) {
	symbol := "Target"
	reader := &stagedErrorReader{
		chunks:   [][]byte{[]byte("file.go:1:Target()\n")},
		finalErr: errors.New("read failed"),
	}

	result := collectReferenceSearchResult(reader, symbol)
	if !result.Incomplete {
		t.Fatal("expected incomplete result when reader returns an error")
	}
	if result.Truncated {
		t.Fatal("expected no truncation on reader error")
	}
	if len(result.Refs) != 1 {
		t.Fatalf("expected first parsed ref to be preserved, got %d", len(result.Refs))
	}
}

func TestRunReferenceSearch_WaitErrorMarksIncomplete(t *testing.T) {
	refs, truncated, incomplete := runReferenceSearch(
		strings.NewReader("file.go:1:Target()\n"),
		"Target",
		nil,
		func() error { return errors.New("rg exit status 2") },
	)

	if truncated {
		t.Fatal("expected no truncation")
	}
	if !incomplete {
		t.Fatal("expected incomplete result when wait returns an error")
	}
	if len(refs) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(refs))
	}
}

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

func TestFindReferences_Exactly200NotTruncated(t *testing.T) {
	refs, truncated, incomplete := runReferenceSearch(
		strings.NewReader(makeRipgrepOutput("Target", maxRipgrepResults)),
		"Target",
		nil,
		func() error { return nil },
	)

	if truncated {
		t.Fatalf("expected truncated=false for exactly %d refs", maxRipgrepResults)
	}
	if incomplete {
		t.Fatal("expected complete result for exactly-on-limit refs")
	}
	if len(refs) != maxRipgrepResults {
		t.Fatalf("expected %d refs, got %d", maxRipgrepResults, len(refs))
	}
}

func TestFindReferences_Over200Truncated(t *testing.T) {
	canceled := false
	refs, truncated, incomplete := runReferenceSearch(
		strings.NewReader(makeRipgrepOutput("Target", maxRipgrepResults+1)),
		"Target",
		func() { canceled = true },
		func() error { return errors.New("signal: killed") },
	)

	if !truncated {
		t.Fatalf("expected truncated=true for %d refs", maxRipgrepResults+1)
	}
	if incomplete {
		t.Fatal("expected intentional cancellation to remain complete")
	}
	if len(refs) != maxRipgrepResults {
		t.Fatalf("expected %d refs after truncation, got %d", maxRipgrepResults, len(refs))
	}
	if !canceled {
		t.Fatal("expected cancel to be invoked after detecting the 201st result")
	}
}

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
