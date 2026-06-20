package importguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// Rule は禁止する import root と失敗時に表示する説明を表す。
type Rule struct {
	ImportRoot string
	Message    string
}

// PackageBoundaryFileSet は package boundary guard が対象にする Go file 範囲を表す。
type PackageBoundaryFileSet string

const (
	// PackageBoundaryAllGoFiles は packageRoot 配下の test を含む全 Go file を対象にする。
	PackageBoundaryAllGoFiles PackageBoundaryFileSet = ""
	// PackageBoundaryProductionGoFiles は packageRoot 配下の non-test Go file を対象にする。
	PackageBoundaryProductionGoFiles PackageBoundaryFileSet = "production"
	// PackageBoundaryRootProductionGoFiles は packageRoot 直下の non-test Go file だけを対象にする。
	PackageBoundaryRootProductionGoFiles PackageBoundaryFileSet = "root-production"
)

// PackageBoundaryOptions は package architecture test の import / package name / facade symbol rule を表す。
type PackageBoundaryOptions struct {
	PackageRoot string

	ImportRules      []Rule
	ExactImportRules []Rule
	ImportFiles      PackageBoundaryFileSet

	ForbidGenericPackageNames   bool
	ForbiddenPackageNameMessage string
	PackageNameFiles            PackageBoundaryFileSet
	ForbiddenFacadeSymbols      map[string]string
	ForbidTypeAliases           bool
	ExportedTypeAliasesOnly     bool
	TypeAliasMessage            string
	FacadeSymbolFiles           PackageBoundaryFileSet
}

// PackageRootFromCaller は呼び出し元 test file の package directory を返す。
func PackageRootFromCaller(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}

// DefaultUIBoundaryRules は UI owner package 共通の禁止 import rule を返す。
func DefaultUIBoundaryRules(owner string) []Rule {
	return []Rule{
		{
			ImportRoot: "github.com/susugadx/xelyon-cli/internal/agent",
			Message:    owner + " must not import internal/agent; agent owns turn orchestration",
		},
		{
			ImportRoot: "github.com/susugadx/xelyon-cli/internal/tui",
			Message:    owner + " must not import internal/tui; Bubble Tea rendering stays in internal/tui",
		},
		{
			ImportRoot: "github.com/susugadx/xelyon-cli/internal/api",
			Message:    owner + " must not import internal/api; provider runtime consumes UI contracts, not the reverse",
		},
		{
			ImportRoot: "github.com/charmbracelet/bubbletea",
			Message:    owner + " must not import Bubble Tea directly",
		},
		{
			ImportRoot: "github.com/charmbracelet/lipgloss",
			Message:    owner + " must not import Lip Gloss directly",
		},
	}
}

// AssertNoImports は packageRoot 配下の Go import が rules に違反しないことを確認する。
func AssertNoImports(t testing.TB, packageRoot string, rules []Rule) {
	t.Helper()
	repoRoot := findRepoRoot(packageRoot)
	fset := token.NewFileSet()
	if err := filepath.WalkDir(packageRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			rule, ok := violatedRule(importPath, rules)
			if !ok {
				continue
			}
			position := fset.Position(imported.Path.Pos())
			relFile, err := filepath.Rel(repoRoot, position.Filename)
			if err != nil {
				relFile = position.Filename
			}
			t.Errorf("%s:%d imports %q; violates rule %q", filepath.ToSlash(relFile), position.Line, importPath, rule.Message)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s imports: %v", packageRoot, err)
	}
}

// AssertPackageBoundaries は packageRoot 配下の architecture boundary rule 違反を検出する。
func AssertPackageBoundaries(t testing.TB, opts PackageBoundaryOptions) {
	t.Helper()
	if opts.PackageRoot == "" {
		t.Fatal("PackageBoundaryOptions.PackageRoot is required")
	}
	repoRoot := findRepoRoot(opts.PackageRoot)
	fset := token.NewFileSet()

	if len(opts.ImportRules) > 0 || len(opts.ExactImportRules) > 0 {
		assertPackageBoundaryImports(t, repoRoot, fset, opts)
	}
	if opts.ForbidGenericPackageNames {
		assertPackageBoundaryPackageNames(t, repoRoot, fset, opts)
	}
	if opts.ForbidTypeAliases || len(opts.ForbiddenFacadeSymbols) > 0 {
		assertPackageBoundaryFacadeSymbols(t, repoRoot, fset, opts)
	}
}

func assertPackageBoundaryImports(t testing.TB, repoRoot string, fset *token.FileSet, opts PackageBoundaryOptions) {
	t.Helper()
	if err := walkPackageBoundaryFiles(opts.PackageRoot, opts.ImportFiles, func(path string) error {
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			if rule, ok := violatedExactRule(importPath, opts.ExactImportRules); ok {
				reportPackageBoundaryViolation(t, repoRoot, fset, imported.Path.Pos(), "imports %q; violates rule %q", importPath, rule.Message)
				continue
			}
			if rule, ok := violatedRule(importPath, opts.ImportRules); ok {
				reportPackageBoundaryViolation(t, repoRoot, fset, imported.Path.Pos(), "imports %q; violates rule %q", importPath, rule.Message)
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s imports: %v", opts.PackageRoot, err)
	}
}

func assertPackageBoundaryPackageNames(t testing.TB, repoRoot string, fset *token.FileSet, opts PackageBoundaryOptions) {
	t.Helper()
	if err := walkPackageBoundaryFiles(opts.PackageRoot, opts.PackageNameFiles, func(path string) error {
		file, err := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly)
		if err != nil {
			return err
		}
		if forbiddenGenericPackageName(file.Name.Name) {
			message := opts.ForbiddenPackageNameMessage
			if message == "" {
				message = "domain policy must not move into generic buckets"
			}
			reportPackageBoundaryViolation(t, repoRoot, fset, file.Name.Pos(), "uses package name %q; %s", file.Name.Name, message)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s package names: %v", opts.PackageRoot, err)
	}
}

func assertPackageBoundaryFacadeSymbols(t testing.TB, repoRoot string, fset *token.FileSet, opts PackageBoundaryOptions) {
	t.Helper()
	if err := walkPackageBoundaryFiles(opts.PackageRoot, opts.FacadeSymbolFiles, func(path string) error {
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				checkPackageBoundaryGenDecl(t, repoRoot, fset, opts, decl)
			case *ast.FuncDecl:
				if decl.Recv != nil || !decl.Name.IsExported() {
					continue
				}
				if owner, ok := opts.ForbiddenFacadeSymbols[decl.Name.Name]; ok {
					reportPackageBoundaryViolation(t, repoRoot, fset, decl.Name.Pos(), "reintroduces facade function %s; %s", decl.Name.Name, owner)
				}
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("walk %s facade symbols: %v", opts.PackageRoot, err)
	}
}

func checkPackageBoundaryGenDecl(t testing.TB, repoRoot string, fset *token.FileSet, opts PackageBoundaryOptions, decl *ast.GenDecl) {
	t.Helper()
	switch decl.Tok {
	case token.TYPE:
		if !opts.ForbidTypeAliases {
			return
		}
		for _, spec := range decl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || !typeSpec.Assign.IsValid() {
				continue
			}
			if opts.ExportedTypeAliasesOnly && !typeSpec.Name.IsExported() {
				continue
			}
			message := opts.TypeAliasMessage
			if message == "" {
				message = "type alias reintroduces an owner-package facade"
			}
			reportPackageBoundaryViolation(t, repoRoot, fset, typeSpec.Name.Pos(), "reintroduces type alias %s; %s", typeSpec.Name.Name, message)
		}
	case token.CONST:
		for _, spec := range decl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range valueSpec.Names {
				if owner, ok := opts.ForbiddenFacadeSymbols[name.Name]; ok {
					reportPackageBoundaryViolation(t, repoRoot, fset, name.Pos(), "reintroduces facade symbol %s; %s", name.Name, owner)
				}
			}
		}
	}
}

func walkPackageBoundaryFiles(packageRoot string, fileSet PackageBoundaryFileSet, visit func(path string) error) error {
	root := filepath.Clean(packageRoot)
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			if fileSet == PackageBoundaryRootProductionGoFiles && filepath.Clean(path) != root {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		if (fileSet == PackageBoundaryProductionGoFiles || fileSet == PackageBoundaryRootProductionGoFiles) && strings.HasSuffix(path, "_test.go") {
			return nil
		}
		return visit(path)
	})
}

func reportPackageBoundaryViolation(t testing.TB, repoRoot string, fset *token.FileSet, pos token.Pos, format string, args ...any) {
	t.Helper()
	position := fset.Position(pos)
	t.Errorf("%s:%d "+format, append([]any{relPackageBoundaryFile(repoRoot, position.Filename), position.Line}, args...)...)
}

func relPackageBoundaryFile(repoRoot, filename string) string {
	relFile, err := filepath.Rel(repoRoot, filename)
	if err != nil {
		return filepath.ToSlash(filename)
	}
	return filepath.ToSlash(relFile)
}

func violatedExactRule(importPath string, rules []Rule) (Rule, bool) {
	for _, rule := range rules {
		if importPath == rule.ImportRoot {
			return rule, true
		}
	}
	return Rule{}, false
}

func violatedRule(importPath string, rules []Rule) (Rule, bool) {
	for _, rule := range rules {
		if importPath == rule.ImportRoot || strings.HasPrefix(importPath, rule.ImportRoot+"/") {
			return rule, true
		}
	}
	return Rule{}, false
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", "generated", "node_modules", "testdata", "vendor":
		return true
	default:
		return false
	}
}

func forbiddenGenericPackageName(name string) bool {
	return name == "common" || name == "helpers" || name == "utils" ||
		strings.HasSuffix(name, "helpers") || strings.HasSuffix(name, "utils")
}

func findRepoRoot(start string) string {
	dir := filepath.Clean(start)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return start
		}
		dir = parent
	}
}
