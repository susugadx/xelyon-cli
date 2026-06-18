package tools

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
		rule:       "internal/tools must not import internal/tui; tools may use internal/ui runtime contracts only",
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

func TestFileToolPackageBoundaries(t *testing.T) {
	repoRoot, toolsRoot, err := toolsArchitectureRoots()
	if err != nil {
		t.Fatal(err)
	}

	fileRoot := filepath.Join(toolsRoot, "file")
	fileRootImport := "github.com/susugadx/xelyon-cli/internal/tools/file"
	rootImports, rootFiles, err := importsForPackageDir(fileRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rootFiles) != 1 || rootFiles[0] != "register.go" {
		t.Fatalf("internal/tools/file root files = %v, want registration facade only", rootFiles)
	}
	allowedRootImports := map[string]struct{}{
		"github.com/susugadx/xelyon-cli/internal/tools":               {},
		"github.com/susugadx/xelyon-cli/internal/tools/file/listtool": {},
		"github.com/susugadx/xelyon-cli/internal/tools/file/mutation": {},
		"github.com/susugadx/xelyon-cli/internal/tools/file/readtool": {},
	}
	for _, importPath := range rootImports {
		if _, ok := allowedRootImports[importPath]; ok {
			continue
		}
		t.Fatalf("internal/tools/file root imports %q; root must stay a registration facade", importPath)
	}

	rules := []struct {
		dir            string
		forbidden      []string
		exactForbidden []string
	}{
		{
			dir: "pathpolicy",
			forbidden: []string{
				"github.com/susugadx/xelyon-cli/internal/tools/file/directquery",
				"github.com/susugadx/xelyon-cli/internal/tools/file/listtool",
				"github.com/susugadx/xelyon-cli/internal/tools/file/mutation",
				"github.com/susugadx/xelyon-cli/internal/tools/file/mutation/replaceengine",
				"github.com/susugadx/xelyon-cli/internal/tools/file/readtool",
				"github.com/susugadx/xelyon-cli/internal/tools/file/schema",
			},
		},
		{
			dir: "schema",
			forbidden: []string{
				"github.com/susugadx/xelyon-cli/internal/tools/file/directquery",
				"github.com/susugadx/xelyon-cli/internal/tools/file/listtool",
				"github.com/susugadx/xelyon-cli/internal/tools/file/mutation",
				"github.com/susugadx/xelyon-cli/internal/tools/file/mutation/replaceengine",
				"github.com/susugadx/xelyon-cli/internal/tools/file/pathpolicy",
				"github.com/susugadx/xelyon-cli/internal/tools/file/readtool",
			},
		},
		{
			dir: "readtool",
			forbidden: []string{
				fileRootImport,
				"github.com/susugadx/xelyon-cli/internal/tools/file/directquery",
				"github.com/susugadx/xelyon-cli/internal/tools/file/listtool",
				"github.com/susugadx/xelyon-cli/internal/tools/file/mutation",
				"github.com/susugadx/xelyon-cli/internal/tools/file/mutation/replaceengine",
			},
		},
		{
			dir: "listtool",
			forbidden: []string{
				fileRootImport,
				"github.com/susugadx/xelyon-cli/internal/tools/file/directquery",
				"github.com/susugadx/xelyon-cli/internal/tools/file/mutation",
				"github.com/susugadx/xelyon-cli/internal/tools/file/mutation/replaceengine",
				"github.com/susugadx/xelyon-cli/internal/tools/file/readtool",
			},
		},
		{
			dir: "mutation",
			forbidden: []string{
				fileRootImport,
				"github.com/susugadx/xelyon-cli/internal/tools/file/directquery",
				"github.com/susugadx/xelyon-cli/internal/tools/file/listtool",
				"github.com/susugadx/xelyon-cli/internal/tools/file/readtool",
			},
		},
		{
			dir: "mutation/replaceengine",
			forbidden: []string{
				fileRootImport,
				"github.com/susugadx/xelyon-cli/internal/ast",
				"github.com/susugadx/xelyon-cli/internal/config",
				"github.com/susugadx/xelyon-cli/internal/lsp",
				"github.com/susugadx/xelyon-cli/internal/tools/file/directquery",
				"github.com/susugadx/xelyon-cli/internal/tools/file/listtool",
				"github.com/susugadx/xelyon-cli/internal/tools/file/mutation",
				"github.com/susugadx/xelyon-cli/internal/tools/file/pathpolicy",
				"github.com/susugadx/xelyon-cli/internal/tools/file/readtool",
				"github.com/susugadx/xelyon-cli/internal/tools/file/schema",
				"github.com/susugadx/xelyon-cli/internal/tools/lsp",
				"github.com/susugadx/xelyon-cli/internal/ui",
			},
			exactForbidden: []string{
				"github.com/susugadx/xelyon-cli/internal/tools",
			},
		},
		{
			dir: "directquery",
			forbidden: []string{
				fileRootImport,
				"github.com/susugadx/xelyon-cli/internal/tools/file/mutation",
				"github.com/susugadx/xelyon-cli/internal/tools/file/mutation/replaceengine",
				"github.com/susugadx/xelyon-cli/internal/tools/file/schema",
			},
		},
	}

	for _, rule := range rules {
		t.Run(rule.dir, func(t *testing.T) {
			imports, _, err := importsForPackageDir(filepath.Join(fileRoot, rule.dir), true)
			if err != nil {
				t.Fatal(err)
			}
			for _, importPath := range imports {
				for _, forbidden := range rule.exactForbidden {
					if importPath == forbidden {
						relDir, _ := filepath.Rel(repoRoot, filepath.Join(fileRoot, rule.dir))
						t.Fatalf("%s imports %q; violates file tool owner dependency direction", filepath.ToSlash(relDir), importPath)
					}
				}
			}
			for _, importPath := range imports {
				for _, forbidden := range rule.forbidden {
					if violatesFileToolImportRule(importPath, forbidden, fileRootImport) {
						relDir, _ := filepath.Rel(repoRoot, filepath.Join(fileRoot, rule.dir))
						t.Fatalf("%s imports %q; violates file tool owner dependency direction", filepath.ToSlash(relDir), importPath)
					}
				}
			}
		})
	}
}

func violatesFileToolImportRule(importPath, forbidden, fileRootImport string) bool {
	if forbidden == fileRootImport {
		return importPath == forbidden
	}
	return importPath == forbidden || strings.HasPrefix(importPath, forbidden+"/")
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

func importsForPackageDir(dir string, includeTests bool) ([]string, []string, error) {
	fset := token.NewFileSet()
	imports := make([]string, 0)
	files := make([]string, 0)
	entries, err := fs.ReadDir(os.DirFS(dir), ".")
	if err != nil {
		return nil, nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		if !includeTests && strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		files = append(files, entry.Name())
		parsed, err := parser.ParseFile(fset, filepath.Join(dir, entry.Name()), nil, parser.ImportsOnly)
		if err != nil {
			return nil, nil, err
		}
		for _, imported := range parsed.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return nil, nil, err
			}
			imports = append(imports, importPath)
		}
	}
	return imports, files, nil
}

func violatedToolsArchitectureRule(importPath string) (toolsArchitectureImportRule, bool) {
	for _, rule := range toolsForbiddenImportRules {
		if importPath == rule.importRoot || strings.HasPrefix(importPath, rule.importRoot+"/") {
			return rule, true
		}
	}
	return toolsArchitectureImportRule{}, false
}
