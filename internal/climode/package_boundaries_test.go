package climode

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

var forbiddenImportRules = []architectureImportRule{
	{
		importRoot: "github.com/spf13/cobra",
		rule:       "internal/climode must not import Cobra; keep CLI flag binding in cmd",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/cmd",
		rule:       "internal/climode must not import cmd; keep mode policy below Cobra wiring",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/config",
		rule:       "internal/climode must not import config; mode resolution uses already-resolved flag values",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/app",
		rule:       "internal/climode must not import app; startup orchestration stays outside pure mode policy",
	},
}

func TestArchitectureBoundaries(t *testing.T) {
	assertNoForbiddenImports(t, "climode", forbiddenImportRules)
}

func assertNoForbiddenImports(t *testing.T, packageDir string, rules []architectureImportRule) {
	t.Helper()

	repoRoot, err := architectureTestRepoRoot()
	if err != nil {
		t.Fatal(err)
	}

	packageRoot := filepath.Join(repoRoot, "internal", packageDir)
	fset := token.NewFileSet()
	if err := filepath.WalkDir(packageRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
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
			rule, ok := violatedArchitectureImportRule(importPath, rules)
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
		t.Fatalf("walk internal/%s imports: %v", packageDir, err)
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
