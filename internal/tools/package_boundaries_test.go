package tools

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

type toolsArchitectureImportRule struct {
	importRoot string
	rule       string
}

var toolsForbiddenImportRules = []toolsArchitectureImportRule{
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/agent",
		rule:       "internal/tools must not import internal/agent; agent owns orchestration and tool runtime context assembly",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/tui",
		rule:       "internal/tools must not import internal/tui; tools may use uiruntime/uiprompt contracts only",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/tuiagent",
		rule:       "internal/tools must not import internal/tuiagent; keep TUI adapter concerns outside tools",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/toolruntime",
		rule:       "internal/tools must not import internal/toolruntime; toolruntime consumes tool results",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/taskstate",
		rule:       "internal/tools must not import internal/taskstate; taskstate consumes RuntimeObservation",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/providerhistory",
		rule:       "internal/tools must not import internal/providerhistory; provider projection consumes tool history",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/mcp",
		rule:       "internal/tools must not import internal/mcp; MCP manager owns external server lifecycle",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/mcptool",
		rule:       "internal/tools must not import internal/mcptool; MCP wrappers implement tools.Tool from outside",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/review",
		rule:       "internal/tools must not import internal/review; review may call tools, not the reverse",
	},
	{
		importRoot: "github.com/susugadx/xelyon-cli/internal/reviewadapter",
		rule:       "internal/tools must not import internal/reviewadapter; review adapters assemble tool surfaces externally",
	},
	{
		importRoot: "github.com/charmbracelet/bubbletea",
		rule:       "internal/tools must not import Bubble Tea directly",
	},
	{
		importRoot: "github.com/charmbracelet/lipgloss",
		rule:       "internal/tools must not import Lip Gloss directly",
	},
}

var toolsArchitectureSkippedDirs = map[string]struct{}{
	".git":         {},
	"generated":    {},
	"node_modules": {},
	"testdata":     {},
	"vendor":       {},
}

func TestToolsPackageBoundaries(t *testing.T) {
	repoRoot, toolsRoot, err := toolsArchitectureRoots()
	if err != nil {
		t.Fatal(err)
	}

	fset := token.NewFileSet()
	if err := filepath.WalkDir(toolsRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			if _, ok := toolsArchitectureSkippedDirs[d.Name()]; ok {
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
			rule, ok := violatedToolsArchitectureRule(importPath)
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
		t.Fatalf("walk internal/tools imports: %v", err)
	}
}

func toolsArchitectureRoots() (repoRoot, toolsRoot string, err error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", "", fs.ErrInvalid
	}
	toolsRoot = filepath.Dir(file)
	repoRoot = filepath.Clean(filepath.Join(toolsRoot, "..", ".."))
	return repoRoot, toolsRoot, nil
}

func violatedToolsArchitectureRule(importPath string) (toolsArchitectureImportRule, bool) {
	for _, rule := range toolsForbiddenImportRules {
		if importPath == rule.importRoot || strings.HasPrefix(importPath, rule.importRoot+"/") {
			return rule, true
		}
	}
	return toolsArchitectureImportRule{}, false
}
