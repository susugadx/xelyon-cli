package ast

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseBytes_GoFile は Go ファイルを正常にパースできることを確認する。
func TestParseBytes_GoFile(t *testing.T) {
	src := []byte("package main\n\nfunc Build() error {\n\treturn nil\n}\n")
	tree, _, err := ParseBytes("main.go", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	root := tree.RootNode()
	if got := root.Type(tree.Language()); got != "source_file" {
		t.Fatalf("root type = %s, want source_file", got)
	}
}

// TestParseBytes_UnsupportedFile は非対応拡張子でエラーになることを確認する。
func TestParseBytes_UnsupportedFile(t *testing.T) {
	_, _, err := ParseBytes("data.csv", []byte("a,b,c"))
	if err == nil {
		t.Fatal("expected error for unsupported file")
	}
}

// TestIsSupportedFile は Phase 1 で Go のみ対応していることを確認する。
func TestIsSupportedFile(t *testing.T) {
	if !IsSupportedFile("main.go") {
		t.Fatal("main.go should be supported")
	}
	if IsSupportedFile("main.py") {
		t.Fatal("main.py should not be supported in phase 1")
	}
}

// TestValidateSyntax_Valid は正常な Go コードで構文エラーが返らないことを確認する。
func TestValidateSyntax_Valid(t *testing.T) {
	src := []byte("package main\n\nfunc Build() error {\n\treturn nil\n}\n")
	errors := ValidateSyntax("main.go", src)
	if len(errors) != 0 {
		t.Fatalf("expected no errors, got %d", len(errors))
	}
}

// TestValidateSyntax_SyntaxError は閉じ括弧不足で構文エラーが返ることを確認する。
func TestValidateSyntax_SyntaxError(t *testing.T) {
	src := []byte("package main\n\nfunc Build() error {\n\treturn nil\n")
	errors := ValidateSyntax("main.go", src)
	if len(errors) == 0 {
		t.Fatal("expected syntax errors")
	}
}

// TestValidateSyntax_NonGoFile は非 Go ファイルでは検証をスキップすることを確認する。
func TestValidateSyntax_NonGoFile(t *testing.T) {
	errors := ValidateSyntax("main.py", []byte("invalid go code"))
	if errors != nil {
		t.Fatalf("expected nil for non-Go file, got %d errors", len(errors))
	}
}

// TestValidateSyntax_MissingElement は欠落した構文要素を含むコードで構文エラーが返ることを確認する。
func TestValidateSyntax_MissingElement(t *testing.T) {
	src := []byte("package main\n\nfunc Build( error {\n\treturn nil\n}\n")
	errors := ValidateSyntax("main.go", src)
	if len(errors) == 0 {
		t.Fatal("expected syntax errors for malformed function signature")
	}
}

// TestExtractSymbols_GoFunction は Go シンボル抽出の基本動作を確認する。
func TestExtractSymbols_GoFunction(t *testing.T) {
	src := []byte(`package main

func Build() error {
    return nil
}

func (s *Server) HandleRequest(ctx context.Context) error {
    return nil
}

type Config struct {
    Name string
}

const MaxRetry = 3

var DefaultTimeout = 30
`)
	symbols, err := ExtractSymbolsFromBytes("main.go", src)
	if err != nil {
		t.Fatalf("ExtractSymbolsFromBytes() error = %v", err)
	}

	names := make(map[string]SymbolKind)
	for _, s := range symbols {
		names[s.Name] = s.Kind
	}

	for _, want := range []struct {
		name string
		kind SymbolKind
	}{
		{"Build", SymbolFunction},
		{"HandleRequest", SymbolMethod},
		{"Config", SymbolStruct},
		{"MaxRetry", SymbolConst},
		{"DefaultTimeout", SymbolVar},
	} {
		if got, ok := names[want.name]; !ok {
			t.Errorf("missing symbol %s", want.name)
		} else if got != want.kind {
			t.Errorf("symbol %s kind = %s, want %s", want.name, got, want.kind)
		}
	}
}

// TestExtractSymbols_TypeAlias は type alias を type シンボルとして抽出できることを確認する。
func TestExtractSymbols_TypeAlias(t *testing.T) {
	src := []byte("package main\n\ntype UserID = string\n")
	symbols, err := ExtractSymbolsFromBytes("main.go", src)
	if err != nil {
		t.Fatalf("ExtractSymbolsFromBytes() error = %v", err)
	}
	if len(symbols) != 1 {
		t.Fatalf("symbols length = %d, want 1", len(symbols))
	}
	if symbols[0].Name != "UserID" {
		t.Fatalf("name = %s, want UserID", symbols[0].Name)
	}
	if symbols[0].Kind != SymbolType {
		t.Fatalf("kind = %s, want %s", symbols[0].Kind, SymbolType)
	}
	if symbols[0].Line != 3 || symbols[0].EndLine != 3 {
		t.Fatalf("location = %d-%d, want 3-3", symbols[0].Line, symbols[0].EndLine)
	}
}

// TestExtractSymbols_GroupedTypeAlias は type block 内の複数 alias を抽出できることを確認する。
func TestExtractSymbols_GroupedTypeAlias(t *testing.T) {
	src := []byte("package main\n\ntype (\n\tUserID = string\n\tAccountID = int\n)\n")
	symbols, err := ExtractSymbolsFromBytes("main.go", src)
	if err != nil {
		t.Fatalf("ExtractSymbolsFromBytes() error = %v", err)
	}

	kinds := make(map[string]SymbolKind, len(symbols))
	for _, symbol := range symbols {
		kinds[symbol.Name] = symbol.Kind
	}
	for _, name := range []string{"UserID", "AccountID"} {
		if kinds[name] != SymbolType {
			t.Fatalf("%s kind = %s, want %s", name, kinds[name], SymbolType)
		}
	}
}

// TestExtractSymbols_Exported は Exported 判定を確認する。
func TestExtractSymbols_Exported(t *testing.T) {
	src := []byte("package main\n\nfunc Build() {}\nfunc helper() {}\n")
	symbols, err := ExtractSymbolsFromBytes("main.go", src)
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range symbols {
		switch s.Name {
		case "Build":
			if !s.Exported {
				t.Error("Build should be exported")
			}
		case "helper":
			if s.Exported {
				t.Error("helper should not be exported")
			}
		}
	}
}

// TestExtractSymbols_MultiLineSignature は複数行シグネチャを保持できることを確認する。
func TestExtractSymbols_MultiLineSignature(t *testing.T) {
	src := []byte(`package main

func HandleRequest(
    ctx context.Context,
    req *Request,
) error {
    return nil
}
`)
	symbols, err := ExtractSymbolsFromBytes("main.go", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(symbols) == 0 {
		t.Fatal("expected at least 1 symbol")
	}
	if symbols[0].Name != "HandleRequest" {
		t.Fatalf("name = %s, want HandleRequest", symbols[0].Name)
	}
	if got := symbols[0].Signature; got != "func HandleRequest(\n    ctx context.Context,\n    req *Request,\n) error" {
		t.Fatalf("signature = %q", got)
	}
}

// TestClassifyLine_FunctionDef は関数定義を def と判定できることを確認する。
func TestClassifyLine_FunctionDef(t *testing.T) {
	src := []byte("package main\n\nfunc Build() error {\n\treturn nil\n}\n")
	info, err := ClassifyLine("main.go", src, 3, "Build")
	if err != nil {
		t.Fatal(err)
	}
	if info.Class != ClassDef {
		t.Fatalf("class = %s, want def", info.Class)
	}
}

// TestClassifyLine_FunctionCall は関数呼び出しを call と判定できることを確認する。
func TestClassifyLine_FunctionCall(t *testing.T) {
	src := []byte("package main\n\nfunc main() {\n\tBuild()\n}\n\nfunc Build() {}\n")
	info, err := ClassifyLine("main.go", src, 4, "Build")
	if err != nil {
		t.Fatal(err)
	}
	if info.Class != ClassCall {
		t.Fatalf("class = %s, want call", info.Class)
	}
	if info.Scope != "func main" {
		t.Fatalf("scope = %s, want func main", info.Scope)
	}
}

// TestClassifyLine_SelectorFunctionCall は selector_expression の関数呼び出しを call と判定できることを確認する。
func TestClassifyLine_SelectorFunctionCall(t *testing.T) {
	src := []byte("package main\n\nimport \"pkg\"\n\nfunc main() {\n\tpkg.Build()\n}\n")
	info, err := ClassifyLine("main.go", src, 6, "Build")
	if err != nil {
		t.Fatal(err)
	}
	if info.Class != ClassCall {
		t.Fatalf("class = %s, want call", info.Class)
	}
	if info.NodeType != "field_identifier" {
		t.Fatalf("nodeType = %s, want field_identifier", info.NodeType)
	}
	if info.SelectorKind != "package" {
		t.Fatalf("selectorKind = %s, want package", info.SelectorKind)
	}
	if info.ReceiverType != "" {
		t.Fatalf("receiverType = %q, want empty for package selector", info.ReceiverType)
	}
}

// TestClassifyLine_MethodSelectorInfersReceiver は method selector から推定レシーバ型を返すことを確認する。
func TestClassifyLine_MethodSelectorInfersReceiver(t *testing.T) {
	src := []byte("package main\n\ntype Config struct{}\n\nfunc use(c Config) string {\n\treturn c.Build()\n}\n")
	info, err := ClassifyLine("main.go", src, 6, "Build")
	if err != nil {
		t.Fatal(err)
	}
	if info.Class != ClassCall {
		t.Fatalf("class = %s, want call", info.Class)
	}
	if info.SelectorKind != "method" {
		t.Fatalf("selectorKind = %s, want method", info.SelectorKind)
	}
	if info.ReceiverType != "Config" {
		t.Fatalf("receiverType = %q, want Config", info.ReceiverType)
	}
}

// TestClassifyLine_Comment はコメント内マッチを comment と判定できることを確認する。
func TestClassifyLine_Comment(t *testing.T) {
	src := []byte("package main\n\n// Build は廃止済み\nfunc Build() {}\n")
	info, err := ClassifyLine("main.go", src, 3, "Build")
	if err != nil {
		t.Fatal(err)
	}
	if info.Class != ClassComment {
		t.Fatalf("class = %s, want comment", info.Class)
	}
}

// TestClassifyLine_StringLiteral は文字列リテラル内マッチを string と判定できることを確認する。
func TestClassifyLine_StringLiteral(t *testing.T) {
	src := []byte(`package main

func main() {
    fmt.Println("Build failed")
}
`)
	info, err := ClassifyLine("main.go", src, 4, "Build")
	if err != nil {
		t.Fatal(err)
	}
	if info.Class != ClassString {
		t.Fatalf("class = %s, want string", info.Class)
	}
}

// TestClassifyLine_Reference は参照を ref と判定できることを確認する。
func TestClassifyLine_Reference(t *testing.T) {
	src := []byte("package main\n\nvar handler = Build\n\nfunc Build() {}\n")
	info, err := ClassifyLine("main.go", src, 3, "Build")
	if err != nil {
		t.Fatal(err)
	}
	if info.Class != ClassRef {
		t.Fatalf("class = %s, want ref", info.Class)
	}
}

// TestClassifyLine_Import は import を import と判定できることを確認する。
func TestClassifyLine_Import(t *testing.T) {
	src := []byte("package main\n\nimport \"fmt\"\n\nfunc main() {}\n")
	info, err := ClassifyLine("main.go", src, 3, "fmt")
	if err != nil {
		t.Fatal(err)
	}
	if info.Class != ClassImport {
		t.Fatalf("class = %s, want import", info.Class)
	}
}

// BenchmarkParseGoFile は実ファイルのパース速度を測定する。
func BenchmarkParseGoFile(b *testing.B) {
	src, err := os.ReadFile(filepath.Join("..", "..", "internal", "tools", "search", "search_code.go"))
	if err != nil {
		b.Skip("search_code.go not found")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = ParseBytes("search_code.go", src)
	}
}

// BenchmarkExtractSymbols は実ファイルのシンボル抽出速度を測定する。
func BenchmarkExtractSymbols(b *testing.B) {
	src, err := os.ReadFile(filepath.Join("..", "..", "internal", "tools", "search", "search_code.go"))
	if err != nil {
		b.Skip("search_code.go not found")
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ExtractSymbolsFromBytes("search_code.go", src)
	}
}
