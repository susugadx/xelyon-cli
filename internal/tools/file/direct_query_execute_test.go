package file

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestExecuteDirectReadTargetsWithDetail_PreservesDisplayPath(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	subTarget := filepath.Join(subdir, "target.go")
	if err := os.WriteFile(subTarget, []byte("package pkg\nconst selected = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	targetInput, ok := parseDirectQueryEntryInput("target.go")
	if !ok {
		t.Fatal("expected direct query input to parse")
	}
	target, errResult := resolveDirectQueryTarget(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}, targetInput)
	if errResult != "" {
		t.Fatalf("expected direct query target to resolve, got %q", errResult)
	}

	result := ExecuteDirectReadTargetsWithDetail(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}, []DirectQueryTarget{target}, "auto")
	if !strings.Contains(result, "📄 File: target.go") {
		t.Fatalf("expected display path to stay relative, got:\n%s", result)
	}
	if !strings.Contains(result, "const selected = true") {
		t.Fatalf("expected resolved file content, got:\n%s", result)
	}
}

func TestExecuteDirectListDirTarget_PreservesAllowedRoots(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "cmd")
	targetDir := filepath.Join(root, "internal", "agent")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "agent.go"), []byte("package agent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	chdirForListDirTest(t, subdir)

	execCtx := tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}
	targetInput, ok := parseDirectQueryEntryInput("internal/agent")
	if !ok {
		t.Fatal("expected direct directory target input to parse")
	}
	target, errResult := resolveDirectQueryTarget(execCtx, targetInput)
	if errResult != "" {
		t.Fatalf("expected direct directory target to resolve from project root, got %q", errResult)
	}

	result := ExecuteDirectListDirTarget(execCtx, target, 1)
	if strings.Contains(result, "path escape attempt") {
		t.Fatalf("expected allowed-roots directory listing, got:\n%s", result)
	}
	if !strings.Contains(result, "summary: depth=1") || !strings.Contains(result, "agent.go") {
		t.Fatalf("expected directory listing output, got:\n%s", result)
	}
}

func TestExecuteDirectListDirTarget_HonorsFileFilter(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "pkg", "nested")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "impl.go"), []byte("package nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("nested docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteDirectListDirTarget(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, DirectQueryTarget{
		FilePath:     filepath.Join("pkg", "nested"),
		ResolvedPath: targetDir,
		AllowedRoots: []string{root},
		FileFilter:   "go",
		Kind:         DirectQueryTargetDirectory,
	}, 1)
	if !strings.Contains(result, "impl.go") {
		t.Fatalf("expected filtered directory listing to include go file, got:\n%s", result)
	}
	if strings.Contains(result, "README.md") {
		t.Fatalf("expected filtered directory listing to exclude non-go file, got:\n%s", result)
	}
}

func TestExecuteDirectListDirTarget_HonorsExtensionTokenFileFilters(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		filepath.Join(targetDir, "main.go"):                "package pkg\n",
		filepath.Join(targetDir, "models.py"):              "def build() -> None:\n    pass\n",
		filepath.Join(targetDir, "types.pyi"):              "def build() -> None: ...\n",
		filepath.Join(targetDir, "main.rs"):                "fn main() {}\n",
		filepath.Join(targetDir, "config.json"):            "{\"name\":\"pkg\"}\n",
		filepath.Join(targetDir, "App.ts"):                 "export const app = 1;\n",
		filepath.Join(targetDir, "App.tsx"):                "export function App() { return <div /> }\n",
		filepath.Join(targetDir, "App.js"):                 "export const app = 1;\n",
		filepath.Join(targetDir, "native.c"):               "int main(void) { return 0; }\n",
		filepath.Join(targetDir, "native.h"):               "#define TARGET_MACRO 1\n",
		filepath.Join(targetDir, "application.properties"): "app.name=pkg\n",
		filepath.Join(targetDir, "view.jsp"):               "<%-- jsp --%>\n",
		filepath.Join(targetDir, "shell.zsh"):              "echo zsh\n",
		filepath.Join(targetDir, "shell.sh"):               "echo sh\n",
	}
	for path, body := range files {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name         string
		fileFilter   string
		wantContains []string
		wantExcludes []string
	}{
		{name: "rs", fileFilter: "rs", wantContains: []string{"main.rs"}, wantExcludes: []string{"main.go", "config.json"}},
		{name: "json", fileFilter: "json", wantContains: []string{"config.json"}, wantExcludes: []string{"main.go", "main.rs"}},
		{name: "python", fileFilter: "py", wantContains: []string{"models.py", "types.pyi"}, wantExcludes: []string{"main.go", "main.rs", "config.json", "App.ts", "App.tsx", "App.js"}},
		{name: "c", fileFilter: "c", wantContains: []string{"native.c", "native.h"}, wantExcludes: []string{"main.go", "application.properties"}},
		{name: "java", fileFilter: "java", wantContains: []string{"application.properties", "view.jsp"}, wantExcludes: []string{"main.go", "native.c"}},
		{name: "sh", fileFilter: "sh", wantContains: []string{"shell.sh", "shell.zsh"}, wantExcludes: []string{"main.go", "application.properties"}},
		{name: "typescript", fileFilter: "typescript", wantContains: []string{"App.ts", "App.tsx"}, wantExcludes: []string{"App.js", "main.go", "main.rs", "config.json"}},
	}

	execCtx := tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExecuteDirectListDirTarget(execCtx, DirectQueryTarget{
				FilePath:      filepath.Join("pkg"),
				ResolvedPath:  targetDir,
				AllowedRoots:  []string{root},
				WorkspaceRoot: root,
				FileFilter:    tt.fileFilter,
				Kind:          DirectQueryTargetDirectory,
			}, 1)
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Fatalf("expected %q in result, got:\n%s", want, result)
				}
			}
			for _, exclude := range tt.wantExcludes {
				if strings.Contains(result, exclude) {
					t.Fatalf("did not expect %q in result, got:\n%s", exclude, result)
				}
			}
		})
	}
}

func TestExecuteDirectListDirTarget_HonorsWorkspaceRelativeGlobFilter(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(targetDir, "main.js"):  "export const main = 1;\n",
		filepath.Join(targetDir, "other.ts"): "export const other = 1;\n",
	}
	for path, body := range files {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	result := ExecuteDirectListDirTarget(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, DirectQueryTarget{
		FilePath:      filepath.Join("pkg"),
		ResolvedPath:  targetDir,
		AllowedRoots:  []string{root},
		WorkspaceRoot: root,
		FileFilter:    "pkg/*.js",
		Kind:          DirectQueryTargetDirectory,
	}, 1)
	if !strings.Contains(result, "main.js") {
		t.Fatalf("expected workspace-relative glob listing to include main.js, got:\n%s", result)
	}
	if strings.Contains(result, "other.ts") {
		t.Fatalf("expected workspace-relative glob listing to exclude other.ts, got:\n%s", result)
	}
}

func TestExecuteDirectListDirTarget_FilteredSummarySkipsNonMatchingDirs(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxRootDirsShown; i++ {
		dir := filepath.Join(targetDir, "js-only-"+string(rune('a'+i)))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "index.js"), []byte("export const jsOnly = true\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	goDir := filepath.Join(targetDir, "zzz-go", "nested")
	if err := os.MkdirAll(goDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goDir, "main.go"), []byte("package nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteDirectListDirTarget(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, DirectQueryTarget{
		FilePath:      filepath.Join("pkg"),
		ResolvedPath:  targetDir,
		AllowedRoots:  []string{root},
		WorkspaceRoot: root,
		FileFilter:    "go",
		Kind:          DirectQueryTargetDirectory,
	}, 1)
	if !strings.Contains(result, "zzz-go/") {
		t.Fatalf("expected filtered directory listing to keep matching subtree, got:\n%s", result)
	}
	if strings.Contains(result, "js-only-") {
		t.Fatalf("expected filtered directory listing to drop non-matching dirs, got:\n%s", result)
	}
}

func TestExecuteDirectListDirTarget_BypassesIgnoreRulesForDirectTargets(t *testing.T) {
	root := t.TempDir()
	chdirForListDirTest(t, root)
	targetDir := filepath.Join(root, "node_modules", "dep")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "xelyon.yaml"), []byte("ignore:\n  patterns:\n    - node_modules\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "package.json"), []byte("{\"name\":\"dep\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("# dep\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteDirectListDirTarget(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, DirectQueryTarget{
		FilePath:      filepath.Join("node_modules", "dep"),
		ResolvedPath:  targetDir,
		AllowedRoots:  []string{root},
		WorkspaceRoot: root,
		FileFilter:    "json",
		BypassIgnores: true,
		Kind:          DirectQueryTargetDirectory,
	}, 1)
	if !strings.Contains(result, "package.json") {
		t.Fatalf("expected direct directory listing to include ignored-tree package.json, got:\n%s", result)
	}
	if strings.Contains(result, "README.md") {
		t.Fatalf("expected direct directory listing to preserve file filter inside ignored tree, got:\n%s", result)
	}
}
