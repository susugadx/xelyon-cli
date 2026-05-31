package agent

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

var agentForbiddenImportRules = []architectureImportRule{
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/tui",
		rule:       "internal/agent must not import internal/tui; route TUI integration through internal/tuiagent or internal/app",
	},
	{
		importRoot: "github.com/charmbracelet/bubbletea",
		rule:       "internal/agent must not import Bubble Tea directly; keep terminal lifecycle in internal/tui or internal/app",
	},
	{
		importRoot: "github.com/charmbracelet/lipgloss",
		rule:       "internal/agent must not import Lip Gloss directly; keep rendering style concerns in internal/tui",
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

	agentRoot := filepath.Join(repoRoot, "internal", "agent")
	fset := token.NewFileSet()

	if err := filepath.WalkDir(agentRoot, func(path string, d fs.DirEntry, walkErr error) error {
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
			rule, ok := violatedArchitectureImportRule(importPath, agentForbiddenImportRules)
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
		t.Fatalf("walk internal/agent imports: %v", err)
	}
}

func TestViolatedArchitectureImportRule(t *testing.T) {
	tests := []struct {
		name       string
		importPath string
		want       bool
	}{
		{
			name:       "exact internal tui import",
			importPath: "github.com/susugadx/xelyon-cli/internal/tui",
			want:       true,
		},
		{
			name:       "internal tui subpackage import",
			importPath: "github.com/susugadx/xelyon-cli/internal/tui/lifecycle",
			want:       true,
		},
		{
			name:       "similarly prefixed internal package is allowed",
			importPath: "github.com/susugadx/xelyon-cli/internal/tuiagent",
			want:       false,
		},
		{
			name:       "exact bubbletea import",
			importPath: "github.com/charmbracelet/bubbletea",
			want:       true,
		},
		{
			name:       "bubbletea subpackage import",
			importPath: "github.com/charmbracelet/bubbletea/v2",
			want:       true,
		},
		{
			name:       "exact lipgloss import",
			importPath: "github.com/charmbracelet/lipgloss",
			want:       true,
		},
		{
			name:       "lipgloss subpackage import",
			importPath: "github.com/charmbracelet/lipgloss/tree",
			want:       true,
		},
		{
			name:       "similarly prefixed charmbracelet package is allowed",
			importPath: "github.com/charmbracelet/lipglossion",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := violatedArchitectureImportRule(tt.importPath, agentForbiddenImportRules)
			if got != tt.want {
				t.Fatalf("violatedArchitectureImportRule(%q) = %v, want %v", tt.importPath, got, tt.want)
			}
		})
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
