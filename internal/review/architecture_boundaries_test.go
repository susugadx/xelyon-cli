package review

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

var reviewRootForbiddenImportRules = []architectureImportRule{
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/agent",
		rule:       "internal/review root must not import internal/agent; keep agent adapters outside review orchestration",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/tui",
		rule:       "internal/review root must not import internal/tui; keep terminal concerns outside review orchestration",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/tuiagent",
		rule:       "internal/review root must not import internal/tuiagent; keep TUI adapters outside review orchestration",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/api",
		rule:       "internal/review root must not import internal/api; keep provider payloads behind ReviewModel",
	},
}

var reviewRootForbiddenForwarders = map[string]string{
	"BuildReviewProbeRequestsFromPlan":       "internal/review/probe owns probe runtime request conversion",
	"BuildReviewProbeSummaries":              "internal/review/probe owns probe summary construction",
	"ComputeReviewReportComputedSummary":     "internal/review/report owns report computed summaries",
	"DecodeReviewProbePlanJSON":              "internal/review/probeplan owns probe plan decoding",
	"DecodeReviewReportJSON":                 "internal/review/report owns report decoding",
	"DecodeReviewSaturationCheckJSON":        "internal/review/report owns saturation check decoding",
	"KnownReviewProbeKind":                   "internal/review/probeplan owns probe plan enums",
	"NewBufferedReviewRunArtifactWriter":     "internal/review/artifact owns artifact writers",
	"NewProbeRunner":                         "internal/review/probe owns probe execution",
	"NewReviewEvidenceBuilder":               "internal/review/evidence owns evidence collection",
	"NewReviewRunDirectoryArtifactWriter":    "internal/review/artifact owns artifact writers",
	"NewReviewWebSearchEvidenceCollector":    "internal/review/evidence owns web-search evidence collection",
	"ValidateReviewProbePlan":                "internal/review/probeplan owns probe plan validation",
	"ValidateReviewReport":                   "internal/review/report owns report validation",
	"ValidateReviewReportExternalDocSupport": "internal/review/report owns report validation",
	"ValidateReviewSaturationCheck":          "internal/review/report owns saturation validation",
	"WithReviewWebSearchEvidenceProvider":    "internal/review/evidence owns evidence builder options",
}

func TestArchitectureBoundaries(t *testing.T) {
	repoRoot, err := architectureTestRepoRoot()
	if err != nil {
		t.Fatal(err)
	}
	reviewRoot := filepath.Join(repoRoot, "internal", "review")
	fset := token.NewFileSet()

	if err := filepath.WalkDir(reviewRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if path != reviewRoot {
				return nil
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		checkReviewRootImports(t, repoRoot, fset, file)
		return nil
	}); err != nil {
		t.Fatalf("walk internal/review root imports: %v", err)
	}

	if err := filepath.WalkDir(reviewRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly)
		if err != nil {
			return err
		}
		if forbiddenReviewPackageName(file.Name.Name) {
			position := fset.Position(file.Name.Pos())
			relFile := architectureRelFile(repoRoot, position.Filename)
			t.Errorf("%s:%d uses package name %q; review domain policy must not move into generic buckets", relFile, position.Line, file.Name.Name)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk internal/review package names: %v", err)
	}

	if err := filepath.WalkDir(reviewRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if path != reviewRoot {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		checkReviewRootFacadeSymbols(t, repoRoot, fset, file)
		return nil
	}); err != nil {
		t.Fatalf("walk internal/review root facade symbols: %v", err)
	}
}

func checkReviewRootImports(t *testing.T, repoRoot string, fset *token.FileSet, file *ast.File) {
	t.Helper()
	for _, imported := range file.Imports {
		importPath, err := strconv.Unquote(imported.Path.Value)
		if err != nil {
			t.Fatal(err)
		}
		rule, ok := violatedArchitectureImportRule(importPath, reviewRootForbiddenImportRules)
		if !ok {
			continue
		}
		position := fset.Position(imported.Path.Pos())
		relFile := architectureRelFile(repoRoot, position.Filename)
		t.Errorf("%s:%d imports %q; violates rule %q", relFile, position.Line, importPath, rule.rule)
	}
}

func checkReviewRootFacadeSymbols(t *testing.T, repoRoot string, fset *token.FileSet, file *ast.File) {
	t.Helper()
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.GenDecl:
			if decl.Tok != token.TYPE {
				continue
			}
			for _, spec := range decl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || !typeSpec.Assign.IsValid() {
					continue
				}
				position := fset.Position(typeSpec.Name.Pos())
				relFile := architectureRelFile(repoRoot, position.Filename)
				t.Errorf("%s:%d reintroduces type alias %s; root internal/review must not be a facade over owner packages", relFile, position.Line, typeSpec.Name.Name)
			}
		case *ast.FuncDecl:
			if decl.Recv != nil || !decl.Name.IsExported() {
				continue
			}
			owner, ok := reviewRootForbiddenForwarders[decl.Name.Name]
			if !ok {
				continue
			}
			position := fset.Position(decl.Name.Pos())
			relFile := architectureRelFile(repoRoot, position.Filename)
			t.Errorf("%s:%d reintroduces facade function %s; %s", relFile, position.Line, decl.Name.Name, owner)
		}
	}
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
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..")), nil
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
