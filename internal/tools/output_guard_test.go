package tools

import (
	"go/ast"
	"go/parser"
	"go/token"
	stdfs "io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestToolPackages_AvoidDirectProcessOutput(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller() failed")
	}
	root := filepath.Dir(currentFile)

	allowedFiles := map[string]bool{
		"common/output.go":   true,
		"context.go":         true,
		"execute_preview.go": true,
		"execute_publish.go": true,
	}

	fset := token.NewFileSet()
	var violations []string

	err := filepath.WalkDir(root, func(path string, d stdfs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if allowedFiles[rel] {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}

		ast.Inspect(file, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			pkgIdent, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}

			switch pkgIdent.Name + "." + sel.Sel.Name {
			case "fmt.Print", "fmt.Printf", "fmt.Println", "os.Stdin", "os.Stdout", "color.Output":
				pos := fset.Position(sel.Pos())
				violations = append(violations, rel+":"+itoa(pos.Line)+": "+pkgIdent.Name+"."+sel.Sel.Name)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir() error = %v", err)
	}

	if len(violations) > 0 {
		t.Fatalf("direct process output is not allowed in internal/tools:\n%s", strings.Join(violations, "\n"))
	}
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
