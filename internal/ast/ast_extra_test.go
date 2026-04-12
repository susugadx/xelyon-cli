package ast

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/odvcencio/gotreesitter"
)

func findNodesByType(node *gotreesitter.Node, lang *gotreesitter.Language, want string, out *[]*gotreesitter.Node) {
	if node == nil {
		return
	}
	if node.Type(lang) == want {
		*out = append(*out, node)
	}
	for i := 0; i < int(node.NamedChildCount()); i++ {
		findNodesByType(node.NamedChild(i), lang, want, out)
	}
}

func TestParseFileAndExtractSymbols_Wrappers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	src := []byte(`package main

type Config struct{}

func Build() error {
	return nil
}
`)
	if err := os.WriteFile(path, src, 0600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	tree, parsed, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	if string(parsed) != string(src) {
		t.Fatalf("ParseFile() src = %q, want %q", string(parsed), string(src))
	}
	if got := tree.RootNode().Type(tree.Language()); got != "source_file" {
		t.Fatalf("root node type = %q, want %q", got, "source_file")
	}

	symbols, err := ExtractSymbols(path)
	if err != nil {
		t.Fatalf("ExtractSymbols() error = %v", err)
	}
	names := make(map[string]SymbolKind, len(symbols))
	for _, symbol := range symbols {
		names[symbol.Name] = symbol.Kind
	}
	if names["Config"] != SymbolStruct {
		t.Fatalf("Config kind = %q, want %q", names["Config"], SymbolStruct)
	}
	if names["Build"] != SymbolFunction {
		t.Fatalf("Build kind = %q, want %q", names["Build"], SymbolFunction)
	}
}

func TestClassifyLineWithParsed_NilParsedFile(t *testing.T) {
	_, err := ClassifyLineWithParsed(nil, 1, "Build")
	if err == nil || !strings.Contains(err.Error(), "ParsedFile is nil") {
		t.Fatalf("ClassifyLineWithParsed(nil) error = %v, want ParsedFile is nil", err)
	}
}

func TestImportNameHelpers(t *testing.T) {
	tests := []struct {
		spec string
		want string
	}{
		{spec: `"fmt"`, want: "fmt"},
		{spec: `alias "example.com/pkg/service"`, want: "alias"},
		{spec: `. "fmt"`, want: ""},
		{spec: `_ "fmt"`, want: ""},
		{spec: "", want: ""},
	}

	for _, tt := range tests {
		if got := importNameFromSpec(tt.spec); got != tt.want {
			t.Fatalf("importNameFromSpec(%q) = %q, want %q", tt.spec, got, tt.want)
		}
	}

	if got := defaultImportName(`"example.com/team/service"`); got != "service" {
		t.Fatalf("defaultImportName() = %q, want %q", got, "service")
	}
	if got := defaultImportName(""); got != "" {
		t.Fatalf("defaultImportName(empty) = %q, want empty", got)
	}
}

func TestInferTypeFromGroupedVarDecl(t *testing.T) {
	prefix := `
var alpha, beta *Config
var gamma, target AnotherType
`
	if got := inferTypeFromGroupedVarDecl(prefix, "alpha"); got != "*Config" {
		t.Fatalf("inferTypeFromGroupedVarDecl(alpha) = %q, want %q", got, "*Config")
	}
	if got := inferTypeFromGroupedVarDecl(prefix, "target"); got != "AnotherType" {
		t.Fatalf("inferTypeFromGroupedVarDecl(target) = %q, want %q", got, "AnotherType")
	}
	if got := inferTypeFromGroupedVarDecl(prefix, "missing"); got != "" {
		t.Fatalf("inferTypeFromGroupedVarDecl(missing) = %q, want empty", got)
	}
}

func TestReceiverInferenceHelpersAndNamedChildSelection(t *testing.T) {
	src := []byte(`package main

type Config struct{}

func use(cfg Config) {
	_ = (&cfg).Build()
	_ = (cfg).Build()
	_ = Config{}.Build()
}
`)
	tree, parsed, err := ParseBytes("main.go", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	lang := tree.Language()

	var selectors []*gotreesitter.Node
	findNodesByType(tree.RootNode(), lang, "selector_expression", &selectors)
	if len(selectors) != 3 {
		t.Fatalf("selector_expression count = %d, want 3", len(selectors))
	}

	parenthesizedUnaryOperand := selectors[0].ChildByFieldName("operand", lang)
	if parenthesizedUnaryOperand == nil || parenthesizedUnaryOperand.Type(lang) != "parenthesized_expression" {
		t.Fatalf("selector[0] operand = %#v, want parenthesized_expression", parenthesizedUnaryOperand)
	}
	if child := firstNamedChild(parenthesizedUnaryOperand); child == nil || child.Type(lang) != "unary_expression" {
		t.Fatalf("firstNamedChild(parenthesized unary) = %#v, want unary_expression", child)
	}
	unary := firstNamedChild(parenthesizedUnaryOperand)
	if child := lastNamedChild(unary); child == nil || child.Type(lang) != "identifier" {
		t.Fatalf("lastNamedChild(unary) = %#v, want identifier", child)
	}

	if got := inferReceiverTypeFromOperand(nil, parenthesizedUnaryOperand, parsed, lang, uint32(len(parsed))); got != "Config" {
		t.Fatalf("inferReceiverTypeFromOperand(unary) = %q, want %q", got, "Config")
	}

	parenthesizedOperand := selectors[1].ChildByFieldName("operand", lang)
	if child := firstNamedChild(parenthesizedOperand); child == nil || child.Type(lang) != "identifier" {
		t.Fatalf("firstNamedChild(parenthesized) = %#v, want identifier", child)
	}
	if got := inferReceiverTypeFromOperand(nil, parenthesizedOperand, parsed, lang, uint32(len(parsed))); got != "Config" {
		t.Fatalf("inferReceiverTypeFromOperand(parenthesized) = %q, want %q", got, "Config")
	}

	compositeOperand := selectors[2].ChildByFieldName("operand", lang)
	if compositeOperand == nil || compositeOperand.Type(lang) != "composite_literal" {
		t.Fatalf("selector[2] operand = %#v, want composite_literal", compositeOperand)
	}
	if got := inferReceiverTypeFromOperand(nil, compositeOperand, parsed, lang, uint32(len(parsed))); got != "Config" {
		t.Fatalf("inferReceiverTypeFromOperand(composite) = %q, want %q", got, "Config")
	}
}

func findFirstNode(node *gotreesitter.Node, visit func(*gotreesitter.Node) bool) *gotreesitter.Node {
	if node == nil {
		return nil
	}
	if visit(node) {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		if found := findFirstNode(node.Child(i), visit); found != nil {
			return found
		}
	}
	return nil
}

func TestInferenceHelperFunctions(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		{expr: "Config{}", want: "Config"},
		{expr: "&[]Item{}", want: "[]Item"},
		{expr: "", want: ""},
		{expr: "value", want: ""},
	}
	for _, tt := range tests {
		if got := inferTypeFromCompositeLiteralText(tt.expr); got != tt.want {
			t.Fatalf("inferTypeFromCompositeLiteralText(%q) = %q, want %q", tt.expr, got, tt.want)
		}
	}

	signatureTests := []struct {
		signature string
		name      string
		want      string
	}{
		{signature: "func Build(cfg Config, count int)", name: "cfg", want: "Config"},
		{signature: "func (s *Server) Handle(ctx context.Context, req Request)", name: "s", want: "*Server"},
		{signature: "func Build(a, target AnotherType)", name: "target", want: "AnotherType"},
		{signature: "type Config struct{}", name: "cfg", want: ""},
	}
	for _, tt := range signatureTests {
		if got := inferIdentifierTypeFromSignature(tt.signature, tt.name); got != tt.want {
			t.Fatalf("inferIdentifierTypeFromSignature(%q, %q) = %q, want %q", tt.signature, tt.name, got, tt.want)
		}
	}

	prefix := `
value := Config{}
other := &Request{}
var alpha, target AnotherType
`
	if got := inferIdentifierTypeFromPrefix(prefix, "value"); got != "Config" {
		t.Fatalf("inferIdentifierTypeFromPrefix(value) = %q, want %q", got, "Config")
	}
	if got := inferIdentifierTypeFromPrefix(prefix, "other"); got != "Request" {
		t.Fatalf("inferIdentifierTypeFromPrefix(other) = %q, want %q", got, "Request")
	}
	if got := inferIdentifierTypeFromPrefix(prefix, "target"); got != "AnotherType" {
		t.Fatalf("inferIdentifierTypeFromPrefix(target) = %q, want %q", got, "AnotherType")
	}
}

func TestFindEnclosingScopeAndBuildSyntaxError(t *testing.T) {
	src := []byte(`package main

type Service struct{}

func (s *Service) Handle() {
	Build()
}

func Build() {}
`)
	tree, parsed, err := ParseBytes("main.go", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	lang := tree.Language()

	var identifiers []*gotreesitter.Node
	findNodesByType(tree.RootNode(), lang, "identifier", &identifiers)

	var buildCall *gotreesitter.Node
	for _, node := range identifiers {
		if strings.TrimSpace(node.Text(parsed)) == "Build" && findAncestorByType(node, lang, "call_expression") != nil {
			buildCall = node
			break
		}
	}
	if buildCall == nil {
		t.Fatal("failed to find Build call identifier")
	}
	if got := findEnclosingScope(buildCall, parsed, lang); got != "method Handle" {
		t.Fatalf("findEnclosingScope(Build call) = %q, want %q", got, "method Handle")
	}

	invalid := []byte("package main\n\nfunc Build( error {\n\treturn nil\n}\n")
	brokenTree, brokenSrc, err := ParseBytes("main.go", invalid)
	if err != nil {
		t.Fatalf("ParseBytes(invalid) error = %v", err)
	}

	errorNode := findFirstNode(brokenTree.RootNode(), func(node *gotreesitter.Node) bool {
		return node != nil && (node.IsError() || node.IsMissing())
	})
	if errorNode == nil {
		t.Fatal("expected syntax error node")
	}

	syntaxErr := buildSyntaxError(errorNode, brokenTree.Language(), brokenSrc)
	if syntaxErr.Line == 0 || syntaxErr.Column == 0 {
		t.Fatalf("buildSyntaxError() returned invalid location: %#v", syntaxErr)
	}
	if !strings.Contains(syntaxErr.Message, "L") {
		t.Fatalf("buildSyntaxError() message = %q, want location prefix", syntaxErr.Message)
	}
	if !strings.Contains(syntaxErr.Message, "near") {
		t.Fatalf("buildSyntaxError() message = %q, want snippet", syntaxErr.Message)
	}
}
