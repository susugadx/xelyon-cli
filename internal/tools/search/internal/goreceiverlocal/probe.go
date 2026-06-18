package goreceiverlocal

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type Role string

const (
	RoleUnknown   Role = "unknown"
	RoleConcrete  Role = "concrete"
	RoleInterface Role = "interface"
)

func RoleFromDir(baseName, dir string) (Role, bool) {
	files, complete := parseGoPackageFilesInDir(dir)
	if len(files) == 0 {
		return RoleUnknown, complete
	}

	role := RoleUnknown
	for _, file := range files {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name == nil || typeSpec.Name.Name != baseName {
					continue
				}
				if _, ok := typeSpec.Type.(*ast.InterfaceType); ok {
					return RoleInterface, complete
				}
				role = RoleConcrete
			}
		}
	}
	return role, complete
}

func HasDirectMethod(baseName, methodName, dir string) (bool, bool) {
	files, complete := parseGoPackageFilesInDir(dir)
	if len(files) == 0 {
		return false, complete
	}

	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil || fn.Name.Name != methodName || fn.Recv == nil {
				continue
			}
			for _, recv := range fn.Recv.List {
				if receiverTypeName(recv.Type) == baseName {
					return true, complete
				}
			}
		}
	}
	return false, complete
}

func parseGoPackageFilesInDir(dir string) ([]*ast.File, bool) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, false
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, false
	}

	complete := true
	files := make([]*ast.File, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil || file == nil {
			complete = false
			continue
		}
		files = append(files, file)
	}
	return files, complete
}

func receiverTypeName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return receiverTypeName(e.X)
	case *ast.IndexExpr:
		return receiverTypeName(e.X)
	case *ast.IndexListExpr:
		return receiverTypeName(e.X)
	case *ast.SelectorExpr:
		if e.Sel != nil {
			return e.Sel.Name
		}
	}
	return ""
}
