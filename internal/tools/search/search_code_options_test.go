package search

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/filefilter"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestSearchCode_FileTypePreferred(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	goFile := filepath.Join(dir, "typed.go")
	jsFile := filepath.Join(dir, "typed.js")
	if err := os.WriteFile(goFile, []byte("func typedTarget() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsFile, []byte("function typedTarget() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "typedTarget", Path: dir, FilePattern: "*.js", FileType: "go", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})
	if !strings.Contains(result, "typed.go") {
		t.Fatalf("expected go file in result, got: %s", result)
	}
	if strings.Contains(result, "typed.js") {
		t.Fatalf("file_type should take precedence over file_pattern, got: %s", result)
	}
}

func TestSearchCode_FixedStrings(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "fixed.go")
	if err := os.WriteFile(file1, []byte("var name = \"a+b\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "a+b", Mode: string(SearchModeLiteral), Path: dir, FilePattern: "*.go", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: false, Multiline: false})
	if strings.Contains(result, "No matches found") || !strings.Contains(result, "a+b") {
		t.Fatalf("expected literal match with is_regex=false, got: %s", result)
	}
}

func TestSearchCode_IncludeHidden(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=hidden_value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{
		Pattern:     "hidden_value",
		Path:        dir,
		IsRegex:     true,
		CtxLines:    -1,
		TokenBudget: -1,
	})
	if strings.Contains(result, ".env") {
		t.Fatalf("hidden files should be excluded by default, got: %s", result)
	}

	result = ExecuteSearchCode(SearchOptions{
		Pattern:       "hidden_value",
		Path:          dir,
		IsRegex:       true,
		IncludeHidden: true,
		CtxLines:      -1,
		TokenBudget:   -1,
	})
	if !strings.Contains(result, ".env") {
		t.Fatalf("hidden files should be included with IncludeHidden, got: %s", result)
	}
}

func TestSearchCode_GrepFallback_DoesNotExcludeRootDot(t *testing.T) {
	setupSearchTestMocks(t)

	if runtime.GOOS == "windows" {
		t.Skip("grep fallback regression test is linux/mac specific")
	}

	grepPath, err := exec.LookPath("grep")
	if err != nil {
		t.Skip("grep not available")
	}

	binDir := t.TempDir()
	if err := os.Symlink(grepPath, filepath.Join(binDir, "grep")); err != nil {
		t.Skipf("failed to prepare isolated grep PATH: %v", err)
	}

	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldwd)
	})

	t.Setenv("PATH", binDir)

	file1 := filepath.Join(dir, "search_target.go")
	if err := os.WriteFile(file1, []byte("package main\n\nfunc maybeAutoCompress() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{
		Pattern:     "maybeAutoCompress",
		Path:        ".",
		FilePattern: "*.go",
		CtxLines:    0,
		TokenBudget: 3000,
		IsRegex:     true,
		Multiline:   false,
	})

	if strings.Contains(result, "No matches found") {
		t.Fatalf("expected grep fallback to find match from root dot, got: %s", result)
	}
	if strings.Contains(result, "Warning: ripgrep (rg) not found; using grep fallback mode.") {
		t.Fatalf("unexpected per-call grep fallback warning, got: %s", result)
	}
	if !strings.Contains(result, "search_target.go") {
		t.Fatalf("expected file name in result, got: %s", result)
	}
}

func TestSearchCode_TypeToGlobMapping(t *testing.T) {
	tests := []struct {
		fileType string
		wantGlob string
		wantOK   bool
	}{
		{fileType: "go", wantGlob: "*.go", wantOK: true},
		{fileType: "py", wantGlob: "*.py", wantOK: true},
		{fileType: "rust", wantGlob: "*.rs", wantOK: true},
		{fileType: "json", wantGlob: "*.json", wantOK: true},
		{fileType: "ex", wantGlob: "*.ex", wantOK: true},
		{fileType: "unknown", wantGlob: "", wantOK: false},
	}

	for _, tt := range tests {
		got, ok := filefilter.FileTypeGlob(tt.fileType)
		if ok != tt.wantOK || got != tt.wantGlob {
			t.Fatalf("filefilter.FileTypeGlob(%q) = (%q, %v), want (%q, %v)", tt.fileType, got, ok, tt.wantGlob, tt.wantOK)
		}
	}
}

func TestSearchCode_FileTypeToGlobsMapping(t *testing.T) {
	tests := []struct {
		fileType string
		want     []string
		wantOK   bool
	}{
		{fileType: "c", want: []string{"*.c", "*.h", "*.H", "*.[chH].in", "*.cats"}, wantOK: true},
		{fileType: "java", want: []string{"*.java", "*.jsp", "*.jspx", "*.properties"}, wantOK: true},
		{fileType: "sh", want: []string{"*.sh", "*.bash", "*.bashrc", "*.csh", "*.cshrc", "*.env", "*.ksh", "*.kshrc", "*.tcsh", "*.tcshrc", "*.zsh", ".bash_login", ".bash_logout", ".bash_profile", ".bashrc", ".cshrc", ".env", ".kshrc", ".login", ".logout", ".profile", ".tcshrc", ".zlogin", ".zlogout", ".zprofile", ".zshenv", ".zshrc", "bash_login", "bash_logout", "bash_profile", "bashrc", "profile", "zlogin", "zlogout", "zprofile", "zshenv", "zshrc"}, wantOK: true},
		{fileType: "py", want: []string{"*.py", "*.pyi"}, wantOK: true},
		{fileType: "python", want: []string{"*.py", "*.pyi"}, wantOK: true},
		{fileType: "typescript", want: []string{"*.ts", "*.tsx"}, wantOK: true},
		{fileType: "javascript", want: []string{"*.js", "*.jsx", "*.mjs", "*.cjs"}, wantOK: true},
		{fileType: "unknown", want: nil, wantOK: false},
	}

	for _, tt := range tests {
		got, ok := filefilter.FileTypeGlobs(tt.fileType)
		if ok != tt.wantOK {
			t.Fatalf("filefilter.FileTypeGlobs(%q) ok = %v, want %v", tt.fileType, ok, tt.wantOK)
		}
		if len(got) != len(tt.want) {
			t.Fatalf("filefilter.FileTypeGlobs(%q) len = %d, want %d", tt.fileType, len(got), len(tt.want))
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("filefilter.FileTypeGlobs(%q)[%d] = %q, want %q", tt.fileType, i, got[i], tt.want[i])
			}
		}
	}
}

func TestSearchCode_RawFileFilterToRipgrepArgs(t *testing.T) {
	tests := []struct {
		name        string
		fileType    string
		filePattern string
		want        []string
	}{
		{
			name:     "c type uses broadened contract globs",
			fileType: "c",
			want:     []string{"--glob", "*.c", "--glob", "*.h", "--glob", "*.H", "--glob", "*.[chH].in", "--glob", "*.cats"},
		},
		{
			name:     "java type uses broadened contract globs",
			fileType: "java",
			want:     []string{"--glob", "*.java", "--glob", "*.jsp", "--glob", "*.jspx", "--glob", "*.properties"},
		},
		{
			name:     "sh type uses broadened contract globs",
			fileType: "sh",
			want:     []string{"--glob", "*.sh", "--glob", "*.bash", "--glob", "*.bashrc", "--glob", "*.csh", "--glob", "*.cshrc", "--glob", "*.env", "--glob", "*.ksh", "--glob", "*.kshrc", "--glob", "*.tcsh", "--glob", "*.tcshrc", "--glob", "*.zsh", "--glob", ".bash_login", "--glob", ".bash_logout", "--glob", ".bash_profile", "--glob", ".bashrc", "--glob", ".cshrc", "--glob", ".env", "--glob", ".kshrc", "--glob", ".login", "--glob", ".logout", "--glob", ".profile", "--glob", ".tcshrc", "--glob", ".zlogin", "--glob", ".zlogout", "--glob", ".zprofile", "--glob", ".zshenv", "--glob", ".zshrc", "--glob", "bash_login", "--glob", "bash_logout", "--glob", "bash_profile", "--glob", "bashrc", "--glob", "profile", "--glob", "zlogin", "--glob", "zlogout", "--glob", "zprofile", "--glob", "zshenv", "--glob", "zshrc"},
		},
		{
			name:     "cjs token uses glob args",
			fileType: "cjs",
			want:     []string{"--glob", "*.cjs"},
		},
		{
			name:     "javascript alias uses contract globs",
			fileType: "javascript",
			want:     []string{"--glob", "*.js", "--glob", "*.jsx", "--glob", "*.mjs", "--glob", "*.cjs"},
		},
		{
			name:     "python alias includes stub globs",
			fileType: "python",
			want:     []string{"--glob", "*.py", "--glob", "*.pyi"},
		},
		{
			name:     "typescript alias uses contract globs",
			fileType: "typescript",
			want:     []string{"--glob", "*.ts", "--glob", "*.tsx"},
		},
		{
			name:        "file type still overrides file pattern",
			fileType:    "go",
			filePattern: "*.js",
			want:        []string{"--glob", "*.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filefilter.RipgrepArgs(tt.fileType, tt.filePattern)
			if len(got) != len(tt.want) {
				t.Fatalf("filefilter.RipgrepArgs(%q, %q) len = %d, want %d", tt.fileType, tt.filePattern, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("filefilter.RipgrepArgs(%q, %q)[%d] = %q, want %q", tt.fileType, tt.filePattern, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSearchFileFilterMatchPath_UsesSharedBasis(t *testing.T) {
	root := t.TempDir()
	absFile := filepath.Join(root, "pkg", "service.js")

	tests := []struct {
		name string
		path string
		file string
		want string
	}{
		{
			name: "absolute path becomes search-root relative",
			path: root,
			file: absFile,
			want: filepath.ToSlash(filepath.Join("pkg", "service.js")),
		},
		{
			name: "relative file path stays relative",
			path: "pkg",
			file: filepath.ToSlash(filepath.Join("pkg", "service.js")),
			want: filepath.ToSlash(filepath.Join("pkg", "service.js")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filefilter.MatchPath(tt.file, tt.path); got != tt.want {
				t.Fatalf("filefilter.MatchPath(%q, %q) = %q, want %q", tt.file, tt.path, got, tt.want)
			}
		})
	}
}

func TestSearchFileFilterMatchPathWithWorkspace_PreservesWorkspaceRelativeGlobBasis(t *testing.T) {
	root := t.TempDir()
	absFile := filepath.Join(root, "pkg", "service.js")

	got := filefilter.MatchPathWithWorkspace(absFile, filepath.Join(root, "pkg"), root)
	want := filepath.ToSlash(filepath.Join("pkg", "service.js"))
	if got != want {
		t.Fatalf("filefilter.MatchPathWithWorkspace(%q, %q, %q) = %q, want %q", absFile, filepath.Join(root, "pkg"), root, got, want)
	}
}

func TestResolveSearchPathBasis_AlignsExecutionAndMatchRoot(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "pkg", "service.js")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("class UserService {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		path          string
		wantWorkdir   string
		wantTarget    string
		wantMatchRoot string
	}{
		{
			name:          "absolute directory uses root-relative execution basis",
			path:          root,
			wantWorkdir:   root,
			wantTarget:    ".",
			wantMatchRoot: root,
		},
		{
			name:          "absolute file uses parent directory as basis",
			path:          file,
			wantWorkdir:   filepath.Dir(file),
			wantTarget:    filepath.Base(file),
			wantMatchRoot: filepath.Dir(file),
		},
		{
			name:          "relative path stays relative",
			path:          filepath.ToSlash(filepath.Join("pkg", "service.js")),
			wantWorkdir:   "",
			wantTarget:    filepath.ToSlash(filepath.Join("pkg", "service.js")),
			wantMatchRoot: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filefilter.ResolveSearchPathBasis(tt.path)
			if got.Workdir != tt.wantWorkdir || got.Target != tt.wantTarget || got.MatchRoot != tt.wantMatchRoot {
				t.Fatalf("filefilter.ResolveSearchPathBasis(%q) = %+v, want workdir=%q target=%q matchRoot=%q", tt.path, got, tt.wantWorkdir, tt.wantTarget, tt.wantMatchRoot)
			}
		})
	}
}

func TestResolveSearchPathBasisForOptions_UsesWorkspaceRootForAbsoluteScopedPath(t *testing.T) {
	root := t.TempDir()
	scopeDir := filepath.Join(root, "pkg")

	got := resolveSearchPathBasisForOptions(SearchOptions{
		Path:               scopeDir,
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	})
	if got.Workdir != root || got.Target != "pkg" || got.MatchRoot != root {
		t.Fatalf("resolveSearchPathBasisForOptions(%q) = %+v, want workdir=%q target=%q matchRoot=%q", scopeDir, got, root, "pkg", root)
	}
}

func TestSearchCode_CjsFileFilterUsesRipgrepGlobs(t *testing.T) {
	setupSearchTestMocks(t)
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.cjs"), []byte("const marker = 'cjs-only-marker'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.js"), []byte("const marker = 'cjs-only-marker'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{
		Pattern:     "cjs-only-marker",
		Mode:        string(SearchModeLiteral),
		Path:        dir,
		FileType:    "cjs",
		CtxLines:    0,
		TokenBudget: 3000,
	})
	if strings.Contains(result, "unrecognized file type") {
		t.Fatalf("expected cjs file_filter to avoid rg type error, got: %s", result)
	}
	if !strings.Contains(result, "main.cjs") {
		t.Fatalf("expected cjs file_filter to include main.cjs, got: %s", result)
	}
	if strings.Contains(result, "main.js") {
		t.Fatalf("expected cjs file_filter to exclude main.js, got: %s", result)
	}
}

func TestExecuteSearchCodeWithConfig_ProjectIgnorePatterns(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "xelyon.yaml"), []byte("ignore:\n  patterns:\n    - generated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "generated"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "generated", "skip.go"), []byte("package generated\n\nfunc target() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep.go"), []byte("package main\n\nfunc target() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	result := ExecuteSearchCodeWithConfig(config.DefaultConfig(), nil, SearchOptions{
		Pattern:     "target",
		Path:        dir,
		FilePattern: "*.go",
		CtxLines:    0,
		TokenBudget: 3000,
		IsRegex:     true,
	})

	if strings.Contains(result, "generated/skip.go") {
		t.Fatalf("generated/skip.go should be ignored by xelyon.yaml ignore.patterns, got %q", result)
	}
	if !strings.Contains(result, "keep.go") {
		t.Fatalf("keep.go should be included, got %q", result)
	}
}
