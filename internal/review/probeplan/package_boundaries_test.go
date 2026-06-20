package probeplan

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

type architectureImportRule struct {
	importRoot string
	rule       string
}

const parentReviewPackageImportPath = "github.com/susugadx/xelyon-cli/internal/review"

var probePlanForbiddenImportRules = []architectureImportRule{
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/agent",
		rule:       "internal/review/probeplan must not import internal/agent; keep runner orchestration outside probe plan schema",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/tui",
		rule:       "internal/review/probeplan must not import internal/tui; keep terminal concerns outside probe plan schema",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/tuiagent",
		rule:       "internal/review/probeplan must not import internal/tuiagent; keep TUI adapters outside probe plan schema",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/api",
		rule:       "internal/review/probeplan must not import internal/api; keep provider payload concerns outside probe plan schema",
	},
	{
		importRoot: "github.com/charmbracelet/bubbletea",
		rule:       "internal/review/probeplan must not import Bubble Tea directly",
	},
	{
		importRoot: "github.com/charmbracelet/lipgloss",
		rule:       "internal/review/probeplan must not import Lip Gloss directly",
	},
}

var probePlanForbiddenFacadeSymbols = map[string]string{
	"ReviewCommandIndex":             "internal/review/report owns evidence command indexes",
	"ReviewEvidenceKindDiff":         "internal/review/report owns evidence kinds",
	"ReviewEvidenceKindExternalDoc":  "internal/review/report owns evidence kinds",
	"ReviewEvidenceKindFile":         "internal/review/report owns evidence kinds",
	"ReviewEvidenceKindGitStatus":    "internal/review/report owns evidence kinds",
	"ReviewEvidenceKindProbe":        "internal/review/report owns evidence kinds",
	"ReviewEvidenceKindProbeCommand": "internal/review/report owns evidence kinds",
	"ReviewEvidenceKindRuleFile":     "internal/review/report owns evidence kinds",
	"ReviewEvidenceRef":              "internal/review/report owns evidence refs",
	"ReviewGroupSeverity":            "internal/review/report owns group severity",
	"ReviewGroupSeverityCritical":    "internal/review/report owns group severity values",
	"ReviewGroupSeverityHigh":        "internal/review/report owns group severity values",
	"ReviewGroupSeverityInfo":        "internal/review/report owns group severity values",
	"ReviewGroupSeverityLow":         "internal/review/report owns group severity values",
	"ReviewGroupSeverityMedium":      "internal/review/report owns group severity values",
	"ReviewProbeHostReadOnly":        "internal/review/domain owns probe modes",
	"ReviewProbeMode":                "internal/review/domain owns probe modes",
	"ReviewProbeRepoSandbox":         "internal/review/domain owns probe modes",
	"ReviewProbeScratchOnly":         "internal/review/domain owns probe modes",
	"TargetCurrentChanges":           "internal/review/domain owns target kinds",
	"TargetKind":                     "internal/review/domain owns target kinds",
}

var architectureTestSkippedDirs = map[string]struct{}{
	".git":         {},
	"generated":    {},
	"node_modules": {},
	"testdata":     {},
	"vendor":       {},
}

func TestArchitectureBoundaries(t *testing.T) {
	repoRoot, err := architectureTestRepoRoot()
	if err != nil {
		t.Fatal(err)
	}

	packageRoot := filepath.Join(repoRoot, "internal", "review", "probeplan")
	fset := token.NewFileSet()
	if err := filepath.WalkDir(packageRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if _, ok := architectureTestSkippedDirs[d.Name()]; ok {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		checkPackageArchitecture(t, repoRoot, fset, file, probePlanForbiddenImportRules, probePlanForbiddenFacadeSymbols)
		return nil
	}); err != nil {
		t.Fatalf("walk internal/review/probeplan: %v", err)
	}
}

func checkPackageArchitecture(t *testing.T, repoRoot string, fset *token.FileSet, file *ast.File, forbiddenImports []architectureImportRule, forbiddenSymbols map[string]string) {
	t.Helper()
	if forbiddenReviewPackageName(file.Name.Name) {
		position := fset.Position(file.Name.Pos())
		relFile := architectureRelFile(repoRoot, position.Filename)
		t.Errorf("%s:%d uses package name %q; review probe plan policy must not move into generic buckets", relFile, position.Line, file.Name.Name)
	}
	for _, imported := range file.Imports {
		importPath, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		if importPath == parentReviewPackageImportPath {
			position := fset.Position(imported.Path.Pos())
			relFile := architectureRelFile(repoRoot, position.Filename)
			t.Errorf("%s:%d imports %q; internal/review/probeplan must not import parent internal/review", relFile, position.Line, importPath)
			continue
		}
		rule, ok := violatedArchitectureImportRule(importPath, forbiddenImports)
		if !ok {
			continue
		}
		position := fset.Position(imported.Path.Pos())
		relFile := architectureRelFile(repoRoot, position.Filename)
		t.Errorf("%s:%d imports %q; violates rule %q", relFile, position.Line, importPath, rule.rule)
	}
	checkForbiddenFacadeSymbols(t, repoRoot, fset, file, forbiddenSymbols)
}

func checkForbiddenFacadeSymbols(t *testing.T, repoRoot string, fset *token.FileSet, file *ast.File, forbiddenSymbols map[string]string) {
	t.Helper()
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.GenDecl:
			switch decl.Tok {
			case token.TYPE:
				for _, spec := range decl.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || !typeSpec.Assign.IsValid() || !typeSpec.Name.IsExported() {
						continue
					}
					reportForbiddenFacadeSymbol(t, repoRoot, fset, typeSpec.Name.Pos(), typeSpec.Name.Name, "exported type alias reintroduces an owner-package facade")
				}
			case token.CONST:
				for _, spec := range decl.Specs {
					valueSpec, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					for _, name := range valueSpec.Names {
						if owner, ok := forbiddenSymbols[name.Name]; ok {
							reportForbiddenFacadeSymbol(t, repoRoot, fset, name.Pos(), name.Name, owner)
						}
					}
				}
			}
		case *ast.FuncDecl:
			if decl.Recv != nil || !decl.Name.IsExported() {
				continue
			}
			if owner, ok := forbiddenSymbols[decl.Name.Name]; ok {
				reportForbiddenFacadeSymbol(t, repoRoot, fset, decl.Name.Pos(), decl.Name.Name, owner)
			}
		}
	}
}

func reportForbiddenFacadeSymbol(t *testing.T, repoRoot string, fset *token.FileSet, pos token.Pos, name, reason string) {
	t.Helper()
	position := fset.Position(pos)
	relFile := architectureRelFile(repoRoot, position.Filename)
	t.Errorf("%s:%d reintroduces facade symbol %s; %s", relFile, position.Line, name, reason)
}

func forbiddenReviewPackageName(name string) bool {
	return name == "common" || name == "helpers" || name == "utils" ||
		strings.HasSuffix(name, "helpers") || strings.HasSuffix(name, "utils")
}

func architectureTestRepoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fs.ErrInvalid
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..")), nil
}

func architectureRelFile(repoRoot, filename string) string {
	relFile, err := filepath.Rel(repoRoot, filename)
	if err != nil {
		return filepath.ToSlash(filename)
	}
	return filepath.ToSlash(relFile)
}

func violatedArchitectureImportRule(importPath string, rules []architectureImportRule) (architectureImportRule, bool) {
	for _, rule := range rules {
		if importPath == rule.importRoot || strings.HasPrefix(importPath, rule.importRoot+"/") {
			return rule, true
		}
	}
	return architectureImportRule{}, false
}
