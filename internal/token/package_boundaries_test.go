package token

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

var tokenForbiddenImportRules = []architectureImportRule{
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/agent",
		rule:       "internal/token must not import internal/agent; keep shared token helpers outside turn orchestration",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/tui",
		rule:       "internal/token must not import internal/tui; keep terminal concerns outside token helpers",
	},
	{
		importRoot: "github.com/charmbracelet/bubbletea",
		rule:       "internal/token must not import Bubble Tea directly",
	},
	{
		importRoot: "github.com/charmbracelet/lipgloss",
		rule:       "internal/token must not import Lip Gloss directly",
	},
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

	packageRoot := filepath.Join(repoRoot, "internal", "token")
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
			rule, ok := violatedArchitectureImportRule(importPath, tokenForbiddenImportRules)
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
		t.Fatalf("walk internal/token imports: %v", err)
	}
}

func architectureTestRepoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fs.ErrInvalid
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..")), nil
}

func violatedArchitectureImportRule(importPath string, rules []architectureImportRule) (architectureImportRule, bool) {
	for _, rule := range rules {
		if importPath == rule.importRoot || strings.HasPrefix(importPath, rule.importRoot+"/") {
			return rule, true
		}
	}
	return architectureImportRule{}, false
}
