package navigation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/ast"
)

func TestReadDefinitionBody_MaxLines(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.go")
	var lines []string
	lines = append(lines, "package test")
	lines = append(lines, "")
	lines = append(lines, "func Foo() {")
	for i := 0; i < 50; i++ {
		lines = append(lines, "    // line")
	}
	lines = append(lines, "}")
	if err := os.WriteFile(file, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}

	cand := SymbolCandidate{
		File:    file,
		Line:    3,
		EndLine: 54,
	}

	body := readDefinitionBody(cand, 10)
	if len(body) == 0 {
		t.Fatal("expected body lines")
	}
	// 10行 + truncation メッセージ = 11 行
	if len(body) != 11 {
		t.Errorf("expected 11 lines (10 + truncation), got %d", len(body))
	}
	lastLine := body[len(body)-1]
	if !strings.Contains(lastLine, "more lines") {
		t.Errorf("expected truncation message, got: %s", lastLine)
	}
}

func TestClassifyCallers_UsesASTClass(t *testing.T) {
	def := SymbolCandidate{
		Name: "Build", File: "agent.go", Line: 10, EndLine: 20,
	}
	refs := []Reference{
		{File: "agent.go", Line: 15, Snippet: "Build()", Scope: "func Build", Class: ast.ClassCall},
		{File: "main.go", Line: 5, Snippet: "Build()", Scope: "func main", Class: ast.ClassCall},
		{File: "ref.go", Line: 8, Snippet: "var x = Build", Scope: "package-level", Class: ast.ClassRef},
		{File: "test_test.go", Line: 3, Snippet: "Build()", IsTest: true, Class: ast.ClassCall},
	}

	callers, _, more := classifyCallers(refs, def, 10)
	if more {
		t.Error("expected no truncation")
	}

	if len(callers) != 1 {
		t.Errorf("expected 1 caller, got %d", len(callers))
	}
	if len(callers) > 0 && callers[0].File != "main.go" {
		t.Errorf("expected main.go, got %s", callers[0].File)
	}

	// limit超過テスト
	refs = []Reference{}
	for i := 0; i < 20; i++ {
		refs = append(refs, Reference{
			File:  "main.go",
			Line:  i + 10,
			Class: ast.ClassCall,
		})
	}

	callers, _, more = classifyCallers(refs, def, 3)
	if !more {
		t.Error("expected truncation")
	}
	if len(callers) != 3 {
		t.Errorf("expected 3 callers, got %d", len(callers))
	}
}

func TestClassifyRefs(t *testing.T) {
	def := SymbolCandidate{
		Name: "Build", Kind: "function", File: "agent.go", Line: 10, EndLine: 20,
	}
	refs := []Reference{
		// テストファイルの参照 → classifyRefs では除外（Related tests で扱う）
		{File: "agent_test.go", Line: 55, Snippet: "Build()", IsTest: true, Class: ast.ClassCall},
		{File: "agent_test.go", Line: 60, Snippet: "var b = Build", IsTest: true, Class: ast.ClassRef},
		// 非テストの参照 → refs に含まれる
		{File: "init.go", Line: 30, Snippet: "var builder = Build", Class: ast.ClassRef},
	}

	classified, _, _ := classifyRefs(refs, def, 10)
	for _, ref := range classified {
		if ref.IsTest {
			t.Errorf("test ref should not appear in References: %s:%d", ref.File, ref.Line)
		}
	}
	if len(classified) != 1 {
		t.Errorf("expected 1 non-test ref, got %d", len(classified))
	}
	if len(classified) > 0 && classified[0].File != "init.go" {
		t.Errorf("expected surviving reference from init.go, got %s", classified[0].File)
	}
}

func TestFilterRefsByCandidate_AmbiguousFileExclusion(t *testing.T) {
	cand := SymbolCandidate{
		Name: "Build", Kind: "function", File: "agent.go", Line: 10, EndLine: 20,
	}
	ambiguousFiles := map[string]bool{"config.go": true}

	refs := []Reference{
		{File: "agent.go", Line: 15, Snippet: "func Build()", Class: ast.ClassDef},
		{File: "config.go", Line: 88, Snippet: "func (c *Config) Build()", Class: ast.ClassDef},
		{File: "config.go", Line: 95, Snippet: "c.Build()", Class: ast.ClassCall},
		{File: "config.go", Line: 100, Snippet: "var b = Build", Class: ast.ClassRef},
		{File: "main.go", Line: 5, Snippet: "Build(\"test\")", Class: ast.ClassCall},
		{File: "init.go", Line: 30, Snippet: "var builder = Build", Class: ast.ClassRef},
	}

	filtered := filterRefsByCandidate(refs, cand, ambiguousFiles)
	if len(filtered) != 2 {
		t.Errorf("expected 2 refs after filtering, got %d", len(filtered))
	}

	gotFiles := map[string]bool{}
	for _, ref := range filtered {
		gotFiles[ref.File] = true
	}
	if gotFiles["config.go"] {
		t.Errorf("config.go refs must be filtered out, got: %+v", filtered)
	}
	if !gotFiles["main.go"] || !gotFiles["init.go"] {
		t.Errorf("expected survivors main.go and init.go, got: %+v", filtered)
	}
}

func TestFilterRefsByCandidate_ReverseDirection(t *testing.T) {
	cand := SymbolCandidate{
		Name: "Build", Kind: "method", File: "config.go", Line: 88, EndLine: 95,
	}
	// agent.go は同名シンボル Build を持つ曖昧ファイル
	ambiguousFiles := map[string]bool{"agent.go": true}

	refs := []Reference{
		// agent.go の call → 曖昧ファイルで除外
		{File: "agent.go", Line: 30, Snippet: "Build(\"test\")", Class: ast.ClassCall},
		// agent.go の ref → 曖昧ファイルで除外
		{File: "agent.go", Line: 40, Snippet: "var b = Build", Class: ast.ClassRef},
		// config.go 自身の定義行 → 除外
		{File: "config.go", Line: 90, Snippet: "c.Build()", Class: ast.ClassCall},
		// 安全な参照 → 残る
		{File: "handler.go", Line: 10, Snippet: "c.Build()", Class: ast.ClassCall},
	}

	filtered := filterRefsByCandidate(refs, cand, ambiguousFiles)
	if len(filtered) != 1 {
		t.Errorf("expected 1 ref (handler.go), got %d", len(filtered))
	}
	if len(filtered) > 0 && filtered[0].File != "handler.go" {
		t.Errorf("expected handler.go survivor, got %+v", filtered[0])
	}
}

func TestFilterRefsByCandidate_AmbiguousFileAllowsPrecisePackageSelectors(t *testing.T) {
	cand := SymbolCandidate{Name: "Build", Kind: string(ast.SymbolFunction), File: "pkg/build.go", Line: 3, EndLine: 5}
	ambiguousFiles := map[string]bool{"main.go": true}

	refs := []Reference{
		{File: "main.go", Line: 12, Snippet: "return pkg.Build()", Class: ast.ClassCall, NodeType: "field_identifier", SelectorKind: "package"},
		{File: "main.go", Line: 19, Snippet: "var pkgRef = pkg.Build", Class: ast.ClassRef, NodeType: "field_identifier", SelectorKind: "package"},
		{File: "main.go", Line: 16, Snippet: "return c.Build()", Class: ast.ClassCall, NodeType: "field_identifier", SelectorKind: "method", ReceiverType: "Config"},
		{File: "main.go", Line: 7, Snippet: "func (Config) Build() string {", Class: ast.ClassDef, NodeType: "identifier"},
		{File: "main.go", Line: 22, Snippet: "var local = Build", Class: ast.ClassRef, NodeType: "identifier"},
	}

	filtered := filterRefsByCandidate(refs, cand, ambiguousFiles)
	if len(filtered) != 2 {
		t.Fatalf("expected only precise package selectors to survive, got %+v", filtered)
	}
	for _, ref := range filtered {
		if ref.SelectorKind != "package" {
			t.Fatalf("unexpected non-package selector survived: %+v", filtered)
		}
	}
}

func TestFilterRefsByCandidate_FiltersFunctionAndMethodShapes(t *testing.T) {
	refs := []Reference{
		{File: "main.go", Line: 10, Snippet: "Build()", Class: ast.ClassCall, NodeType: "identifier"},
		{File: "main.go", Line: 11, Snippet: "pkg.Build()", Class: ast.ClassCall, NodeType: "field_identifier", SelectorKind: "package"},
		{File: "main.go", Line: 12, Snippet: "c.Build()", Class: ast.ClassCall, NodeType: "field_identifier", SelectorKind: "method", ReceiverType: "Config"},
	}

	functionFiltered := filterRefsByCandidate(refs, SymbolCandidate{Name: "Build", Kind: string(ast.SymbolFunction), File: "agent.go", Line: 1, EndLine: 3}, nil)
	if len(functionFiltered) != 2 {
		t.Fatalf("expected plain and package-qualified function refs, got %+v", functionFiltered)
	}
	for _, ref := range functionFiltered {
		if ref.SelectorKind == "method" {
			t.Fatalf("instance method selector must not be attributed to function candidate, got %+v", functionFiltered)
		}
	}

	methodFiltered := filterRefsByCandidate(refs, SymbolCandidate{Name: "Build", Kind: string(ast.SymbolMethod), File: "agent.go", Line: 1, EndLine: 3, Receiver: "Config"}, nil)
	if len(methodFiltered) != 1 || methodFiltered[0].SelectorKind != "method" || methodFiltered[0].ReceiverType != "Config" {
		t.Fatalf("expected only matching method selector refs for method candidate, got %+v", methodFiltered)
	}
}

func TestFilterRefsByCandidate_ReceiverQualifiedMethodSameFile(t *testing.T) {
	cand := SymbolCandidate{Name: "Build", Kind: string(ast.SymbolMethod), File: "example.go", Line: 10, EndLine: 12, Receiver: "A"}
	refs := []Reference{
		{File: "example.go", Line: 20, Snippet: "a.Build()", Class: ast.ClassCall, NodeType: "field_identifier", SelectorKind: "method", ReceiverType: "A"},
		{File: "example.go", Line: 21, Snippet: "var refA = A{}.Build", Class: ast.ClassRef, NodeType: "field_identifier", SelectorKind: "method", ReceiverType: "A"},
		{File: "example.go", Line: 30, Snippet: "b.Build()", Class: ast.ClassCall, NodeType: "field_identifier", SelectorKind: "method", ReceiverType: "B"},
		{File: "example.go", Line: 31, Snippet: "var refB = B{}.Build", Class: ast.ClassRef, NodeType: "field_identifier", SelectorKind: "method", ReceiverType: "B"},
	}

	filtered := filterRefsByCandidate(refs, cand, nil)
	if len(filtered) != 2 {
		t.Fatalf("expected only A receiver refs to survive, got %+v", filtered)
	}
	for _, ref := range filtered {
		if ref.ReceiverType != "A" {
			t.Fatalf("unexpected foreign receiver survived: %+v", filtered)
		}
	}
}

func TestClassifyCallers_InterfaceAndFuncTypeNotCaller(t *testing.T) {
	def := SymbolCandidate{
		Name: "Build", File: "agent.go", Line: 10, EndLine: 20,
	}
	refs := []Reference{
		{File: "iface.go", Line: 5, Snippet: "Build() error", Class: ast.ClassDef},
		{File: "type.go", Line: 10, Snippet: "type Builder func() Build", Class: ast.ClassDef},
		{File: "main.go", Line: 15, Snippet: "Build()", Class: ast.ClassCall},
	}

	callers, _, _ := classifyCallers(refs, def, 10)
	if len(callers) != 1 {
		t.Errorf("expected 1 caller, got %d", len(callers))
	}
	if len(callers) > 0 && callers[0].File != "main.go" {
		t.Errorf("expected only executable caller main.go, got %+v", callers[0])
	}
}
