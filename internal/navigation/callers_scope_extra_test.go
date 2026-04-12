package navigation

import (
	goast "go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func parseNavigationStmt(t *testing.T, body string) (goast.Stmt, *token.FileSet) {
	t.Helper()

	src := "package sample\nfunc f() {\n" + body + "\n}\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "sample.go", src, 0)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	fn := file.Decls[0].(*goast.FuncDecl)
	if len(fn.Body.List) == 0 {
		t.Fatal("parsed function body is empty")
	}
	return fn.Body.List[0], fset
}

func TestCheckNestedDeclInStmt_CoversCompoundStatements(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		useLine int
		nameRef string
	}{
		{
			name: "if init declaration",
			body: "if target := 1; target > 0 {\n\t\tprintln(target)\n\t}",
			useLine: 4,
			nameRef: "target",
		},
		{
			name: "else-if recursive declaration",
			body: "if false {\n\t\tprintln(\"no\")\n\t} else if nested := 1; nested > 0 {\n\t\tprintln(nested)\n\t}",
			useLine: 5,
			nameRef: "nested",
		},
		{
			name: "for init declaration",
			body: "for index := 0; index < 1; index++ {\n\t\tprintln(index)\n\t}",
			useLine: 4,
			nameRef: "index",
		},
		{
			name: "range key and value declaration",
			body: "for key, value := range []int{1} {\n\t\tprintln(key, value)\n\t}",
			useLine: 4,
			nameRef: "value",
		},
		{
			name: "switch init declaration",
			body: "switch target := 1; target {\n\tcase 1:\n\t\tprintln(target)\n\t}",
			useLine: 5,
			nameRef: "target",
		},
		{
			name: "type switch assign declaration",
			body: "switch typed := any(1).(type) {\n\tcase int:\n\t\tprintln(typed)\n\t}",
			useLine: 5,
			nameRef: "typed",
		},
		{
			name: "select comm declaration",
			body: "select {\n\tcase msg := <-make(chan string):\n\t\tprintln(msg)\n\tdefault:\n\t}",
			useLine: 5,
			nameRef: "msg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, fset := parseNavigationStmt(t, tt.body)
			if !checkNestedDeclInStmt(stmt, fset, tt.useLine, tt.nameRef) {
				t.Fatalf("checkNestedDeclInStmt() = false, want true for %s", tt.name)
			}
		})
	}
}

func TestCheckNestedDeclInStmt_BlockAndNegativeCases(t *testing.T) {
	stmt, fset := parseNavigationStmt(t, "{\n\t\tshadowed := 1\n\t\tprintln(shadowed)\n\t}")
	if !checkNestedDeclInStmt(stmt, fset, 5, "shadowed") {
		t.Fatal("checkNestedDeclInStmt() should detect declaration inside nested block")
	}

	negativeStmt, negativeFset := parseNavigationStmt(t, "if true {\n\t\tprintln(\"no decl\")\n\t}")
	if checkNestedDeclInStmt(negativeStmt, negativeFset, 4, "missing") {
		t.Fatal("checkNestedDeclInStmt() should return false when declaration is absent")
	}
}

func TestMatchesDeclName_AssignAndVarDeclarations(t *testing.T) {
	assignStmt := &goast.AssignStmt{
		Tok: token.DEFINE,
		Lhs: []goast.Expr{goast.NewIdent("target")},
	}
	if !matchesDeclName(assignStmt, "target") {
		t.Fatal("matchesDeclName() should detect := declaration")
	}

	reassignStmt := &goast.AssignStmt{
		Tok: token.ASSIGN,
		Lhs: []goast.Expr{goast.NewIdent("target")},
	}
	if matchesDeclName(reassignStmt, "target") {
		t.Fatal("matchesDeclName() should ignore plain assignment")
	}

	declStmt := &goast.DeclStmt{
		Decl: &goast.GenDecl{
			Specs: []goast.Spec{&goast.ValueSpec{Names: []*goast.Ident{goast.NewIdent("declared")}}},
		},
	}
	if !matchesDeclName(declStmt, "declared") {
		t.Fatal("matchesDeclName() should detect var declaration")
	}
}
