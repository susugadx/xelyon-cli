package importguard

import (
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
