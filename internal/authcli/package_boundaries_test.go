package authcli

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
		rule:       "internal/authcli must not import Cobra; cmd owns command binding",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/cmd",
		rule:       "internal/authcli must not import cmd",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/tui",
		rule:       "internal/authcli must not import TUI packages",
	},
}

func TestArchitectureBoundaries(t *testing.T) {
	repoRoot, err := architectureTestRepoRoot()
	if err != nil {
		t.Fatal(err)
	}

	packageRoot := filepath.Join(repoRoot, "internal", "authcli")
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
			for _, rule := range forbiddenImportRules {
				if importPath != rule.importRoot && !strings.HasPrefix(importPath, rule.importRoot+"/") {
					continue
				}
				position := fset.Position(imported.Path.Pos())
				relFile, err := filepath.Rel(repoRoot, position.Filename)
				if err != nil {
					relFile = position.Filename
				}
				t.Errorf("%s:%d imports %q; violates rule %q", filepath.ToSlash(relFile), position.Line, importPath, rule.rule)
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("walk internal/authcli imports: %v", err)
	}
}

func architectureTestRepoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fs.ErrInvalid
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..")), nil
}
