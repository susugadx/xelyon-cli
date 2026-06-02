package modeloutput

import (
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

var reviewModelOutputForbiddenImportRules = []architectureImportRule{
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/review/evidence",
		rule:       "internal/review/modeloutput must not import review evidence; receive fetched external docs as input",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/review/artifact",
		rule:       "internal/review/modeloutput must not import review artifacts; artifact save timing belongs to internal/review runner",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/review/modelinput",
		rule:       "internal/review/modeloutput must not import modelinput; prompt assembly and output finalization are separate boundaries",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/agent",
		rule:       "internal/review/modeloutput must not import internal/agent; keep runner/model orchestration outside deterministic output finalization",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/tui",
		rule:       "internal/review/modeloutput must not import internal/tui; keep terminal concerns outside model output finalization",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/tuiagent",
		rule:       "internal/review/modeloutput must not import internal/tuiagent; keep TUI adapters outside model output finalization",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/api",
		rule:       "internal/review/modeloutput must not import internal/api; keep provider payload concerns outside review model output finalization",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/commandruntime",
		rule:       "internal/review/modeloutput must not import command runtime packages",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/toolruntime",
		rule:       "internal/review/modeloutput must not import tool runtime packages",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/providerdiag",
		rule:       "internal/review/modeloutput must not import provider runtime packages",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/providerhistory",
		rule:       "internal/review/modeloutput must not import provider runtime packages",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/providerpicker",
		rule:       "internal/review/modeloutput must not import provider runtime packages",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/llmcatalog",
		rule:       "internal/review/modeloutput must not import provider/model catalog packages",
	},
	{
		importRoot: "github.com/charmbracelet/bubbletea",
		rule:       "internal/review/modeloutput must not import Bubble Tea directly",
	},
	{
		importRoot: "github.com/charmbracelet/lipgloss",
		rule:       "internal/review/modeloutput must not import Lip Gloss directly",
	},
}

const parentReviewPackageImportPath = "github.com/susugadx/xelyon-cli/internal/review"

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

	packageRoot := filepath.Join(repoRoot, "internal", "review", "modeloutput")
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

		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range file.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return err
			}
			if importPath == parentReviewPackageImportPath {
				position := fset.Position(imported.Path.Pos())
				relFile, err := filepath.Rel(repoRoot, position.Filename)
				if err != nil {
					relFile = position.Filename
				}
				t.Errorf("%s:%d imports %q; violates rule %q", filepath.ToSlash(relFile), position.Line, importPath, "internal/review/modeloutput must not import parent internal/review")
				continue
			}
			rule, ok := violatedArchitectureImportRule(importPath, reviewModelOutputForbiddenImportRules)
			if !ok {
				continue
			}

			position := fset.Position(imported.Path.Pos())
			relFile, err := filepath.Rel(repoRoot, position.Filename)
			if err != nil {
				relFile = position.Filename
			}
			t.Errorf("%s:%d imports %q; violates rule %q", filepath.ToSlash(relFile), position.Line, importPath, rule.rule)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk internal/review/modeloutput imports: %v", err)
	}
}

func architectureTestRepoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fs.ErrInvalid
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..")), nil
}

func violatedArchitectureImportRule(importPath string, rules []architectureImportRule) (architectureImportRule, bool) {
	for _, rule := range rules {
		if importPath == rule.importRoot || strings.HasPrefix(importPath, rule.importRoot+"/") {
			return rule, true
		}
	}
	return architectureImportRule{}, false
}
