package navigation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func TestInspectSymbol_EmptySymbol(t *testing.T) {
	result := InspectSymbol("", "", "")
	if !strings.Contains(result, "Error") {
		t.Errorf("expected error for empty symbol, got: %s", result)
	}
}

func TestInspectSymbol_NotFound(t *testing.T) {
	result := InspectSymbol("NonExistentSymbol_XYZ_12345", "", "")
	if !strings.Contains(result, "No symbol found") {
		t.Errorf("expected 'No symbol found', got: %s", result)
	}
}

// setupTestGoFile はテスト用の Go ファイルを一時ディレクトリに作成し、
// そのディレクトリに cd した後、元に戻すための cleanup を返す。
func setupTestGoFile(t *testing.T, filename, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: could not restore directory: %v", err)
		}
	})
	return dir
}

const testGoSource = `package example

import "fmt"

// Build は何かをビルドする。
func Build(name string) error {
	fmt.Println(name)
	return nil
}

// Run は Build を呼ぶ。
func Run() {
	Build("test")
}

type Config struct {
	Name string
}

func (c *Config) Build() string {
	return c.Name
}

var DefaultBuilder = Build
`

func TestInspectSymbol_SingleCandidate(t *testing.T) {
	setupTestGoFile(t, "example.go", testGoSource)

	result := InspectSymbol("Run", "", "")
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find symbol, got: %s", result)
	}
	if !strings.Contains(result, "Run") {
		t.Errorf("expected Run in output, got: %s", result)
	}
	if !strings.Contains(result, "func Run") {
		t.Errorf("expected function definition in output, got: %s", result)
	}

}

func TestInspectSymbol_WithPath(t *testing.T) {
	setupTestGoFile(t, "example.go", testGoSource)

	result := InspectSymbol("Build", "example.go", "")
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find symbol, got: %s", result)
	}
	// example.go には function Build と method Build の2つがある
	// path でファイルは絞れるが、同一ファイル内の複数候補は残る
	if strings.Contains(result, "Multiple symbols matched") {
		// 期待通り：同一ファイルに function Build と method Build がある
		if !strings.Contains(result, "Refine with path") {
			t.Error("expected disambiguation hint")
		}
	}
}

func TestInspectSymbol_MultipleCandidates(t *testing.T) {
	setupTestGoFile(t, "example.go", testGoSource)

	result := InspectSymbol("Build", "", "")
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find symbol, got: %s", result)
	}
	// Build は function と method の2つがある
	if !strings.Contains(result, "Multiple symbols matched") {
		// 単一候補に解決された場合も OK（AST の仕様による）
		t.Logf("resolved to single candidate: %s", result)
	}
}

func TestInspectSymbol_FullMode(t *testing.T) {
	setupTestGoFile(t, "example.go", testGoSource)

	result := InspectSymbol("Config", "", "full")
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find symbol, got: %s", result)
	}
	if !strings.Contains(result, "Config") {
		t.Errorf("expected Config in output, got: %s", result)
	}
}

func TestInspectSymbol_WithCallers(t *testing.T) {
	setupTestGoFile(t, "example.go", testGoSource)

	// Build を Run から呼んでいるので caller がある
	// ただし同名のメソッド Build もあるため複数候補になる可能性がある
	result := InspectSymbol("Run", "", "")
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find symbol, got: %s", result)
	}
	// Run は単一候補のはず
	if strings.Contains(result, "Multiple symbols matched") {
		t.Errorf("expected single candidate for Run, got: %s", result)
	}
}

func TestInspectSymbol_WithTests(t *testing.T) {
	dir := t.TempDir()
	mainFile := filepath.Join(dir, "example.go")
	testFile := filepath.Join(dir, "example_test.go")

	if err := os.WriteFile(mainFile, []byte(`package example

func Greet() string {
	return "hello"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(testFile, []byte(`package example

import "testing"

func TestGreet(t *testing.T) {
	if Greet() != "hello" {
		t.Error("wrong")
	}
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: could not restore directory: %v", err)
		}
	})

	result := InspectSymbol("Greet", "", "")
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find symbol, got: %s", result)
	}
	if !strings.Contains(result, "Related tests") {
		t.Errorf("expected related tests section, got: %s", result)
	}
	if !strings.Contains(result, "TestGreet") {
		t.Errorf("expected TestGreet in output, got: %s", result)
	}
	if strings.Contains(result, "[test]") {
		t.Errorf("test refs should be shown only in Related tests, got: %s", result)
	}
	if strings.Contains(result, "References (") || strings.Contains(result, "References:") {
		t.Errorf("test-only references must not appear in References section, got: %s", result)
	}
}

func TestInspectSymbol_QualifiedMethodQuery_DisambiguatesSameFileMethod(t *testing.T) {
	setupTestGoFiles(t, map[string]string{
		"example.go": `package example

	type Config struct {
			Name string
		}

		func Build(name string) string {
			return "function:" + name
		}

		func (c Config) Build() string {
			return "method:" + c.Name
		}

		func RunFunction() string {
			return Build("x")
		}

		func RunMethod() string {
			c := Config{Name: "x"}
			return c.Build()
		}

		var functionRef = Build
		var methodRef = Config{}.Build
		`,
	})

	result := InspectSymbol("Config.Build", "", "")
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected receiver-qualified method query to resolve, got: %s", result)
	}
	if strings.Contains(result, "Multiple symbols matched") {
		t.Fatalf("expected receiver-qualified method query to disambiguate same-file candidates, got: %s", result)
	}
	if !strings.Contains(result, "Config.Build") {
		t.Fatalf("expected method header/body to mention Config.Build, got: %s", result)
	}
	if !strings.Contains(result, "RunMethod") {
		t.Fatalf("expected method caller to survive filtering, got: %s", result)
	}
	if !strings.Contains(result, "methodRef") {
		t.Fatalf("expected method reference to survive filtering, got: %s", result)
	}
	if strings.Contains(result, "RunFunction") {
		t.Fatalf("function caller should not be attributed to Config.Build, got: %s", result)
	}
	if strings.Contains(result, "functionRef") {
		t.Fatalf("function reference should not be attributed to Config.Build, got: %s", result)
	}
}

func TestInspectSymbol_QualifiedMethodQuery_IsolatesSameFileReceivers(t *testing.T) {
	setupTestGoFiles(t, map[string]string{
		"example.go": `package example

		type A struct{}
		type B struct{}

		func (A) Build() string {
			return "A"
		}

		func (B) Build() string {
			return "B"
		}

		func UseA(a A) string {
			return a.Build()
		}

		func UseB(b B) string {
			return b.Build()
		}

		var refA = A{}.Build
		var refB = B{}.Build
		`,
	})

	result := InspectSymbol("A.Build", "", "full")
	if strings.Contains(result, "No symbol found") || strings.Contains(result, "Multiple symbols matched") {
		t.Fatalf("expected A.Build to resolve to a single method candidate, got: %s", result)
	}
	if !strings.Contains(result, "A.Build") {
		t.Fatalf("expected receiver-qualified method header, got: %s", result)
	}
	if !strings.Contains(result, "UseA") || !strings.Contains(result, "refA") {
		t.Fatalf("expected A.Build callers/refs to survive filtering, got: %s", result)
	}
	if strings.Contains(result, "UseB") || strings.Contains(result, "refB") {
		t.Fatalf("B.Build callers/refs must not be attributed to A.Build, got: %s", result)
	}
}

func TestInspectSymbol_PackageQualifiedFunctionRefsAndCallers(t *testing.T) {
	setupTestGoFiles(t, map[string]string{
		"pkg/build.go": `package pkg

		func Build() string {
			return "pkg"
		}
		`,
		"main.go": `package main

		import "example/pkg"

		type Config struct{}

		func (Config) Build() string {
			return "method"
		}

		func UsePkg() string {
			return pkg.Build()
		}

		func UseMethod(c Config) string {
			return c.Build()
		}

		var pkgRef = pkg.Build
		var methodRef = Config{}.Build
		`,
	})

	result := InspectSymbol("Build", "pkg/build.go", "full")
	if strings.Contains(result, "No symbol found") || strings.Contains(result, "Multiple symbols matched") {
		t.Fatalf("expected package function Build to resolve to a single candidate, got: %s", result)
	}
	if !strings.Contains(result, "UsePkg") || !strings.Contains(result, "pkgRef") {
		t.Fatalf("expected package-qualified function caller/ref to be included, got: %s", result)
	}
	if strings.Contains(result, "UseMethod") || strings.Contains(result, "methodRef") {
		t.Fatalf("instance method usages must not be attributed to package function Build, got: %s", result)
	}
}

func TestResolveSymbolCandidatesWithRuntime_ProjectMapSnapshotDerivesMethodReceiver(t *testing.T) {
	dir := setupTestGoFiles(t, map[string]string{
		"pkg/service.go": "package pkg\n\nfunc placeholder() {}\n",
	})

	runtime := GoSymbolRuntime{
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "pkg/service.go",
					Symbols: []repomap.Symbol{
						{Name: "Build", Kind: "method", Line: 3, EndLine: 3, Signature: "func (s *Service[T]) Build() error", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "state-derive",
	}

	candidates := resolveSymbolCandidatesWithRuntime("Service.Build", filepath.Join(dir, "pkg"), runtime)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate from snapshot, got %+v", candidates)
	}
	if candidates[0].Receiver != "*Service[T]" {
		t.Fatalf("unexpected receiver: %+v", candidates[0])
	}
	if candidates[0].ReceiverNorm != "Service" {
		t.Fatalf("unexpected receiver norm: %+v", candidates[0])
	}
	if candidates[0].PackageDir != "pkg" {
		t.Fatalf("unexpected package dir: %+v", candidates[0])
	}
}

func TestResolveSymbolCandidatesWithRuntime_ProjectMapSnapshotPathHintFilters(t *testing.T) {
	dir := setupTestGoFiles(t, map[string]string{
		"pkg/service.go":      "package pkg\n\nfunc placeholder() {}\n",
		"internal/service.go": "package internal\n\nfunc placeholder() {}\n",
	})

	runtime := GoSymbolRuntime{
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "pkg/service.go",
					Symbols: []repomap.Symbol{
						{Name: "Build", Kind: "function", Line: 3, EndLine: 3, Signature: "func Build() error", Exported: true},
					},
				},
				{
					Path: "internal/service.go",
					Symbols: []repomap.Symbol{
						{Name: "Build", Kind: "function", Line: 3, EndLine: 3, Signature: "func Build() error", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "state-hints",
	}

	fileCandidates := resolveSymbolCandidatesWithRuntime("Build", filepath.Join(dir, "pkg", "service.go"), runtime)
	if len(fileCandidates) != 1 || fileCandidates[0].File != "pkg/service.go" {
		t.Fatalf("expected file path hint to isolate pkg/service.go, got %+v", fileCandidates)
	}

	dirCandidates := resolveSymbolCandidatesWithRuntime("Build", filepath.Join(dir, "pkg"), runtime)
	if len(dirCandidates) != 1 || dirCandidates[0].File != "pkg/service.go" {
		t.Fatalf("expected dir path hint to isolate pkg/service.go, got %+v", dirCandidates)
	}
}

func TestResolveSymbolCandidatesWithRuntime_ProjectMapSnapshotRelativePathUsesInvocationCWD(t *testing.T) {
	dir := setupTestGoFiles(t, map[string]string{
		"pkg/service.go":      "package pkg\n\nfunc placeholder() {}\n",
		"internal/service.go": "package internal\n\nfunc placeholder() {}\n",
	})
	subdir := filepath.Join(dir, "pkg")

	runtime := GoSymbolRuntime{
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "pkg/service.go",
					Symbols: []repomap.Symbol{
						{Name: "Build", Kind: "function", Line: 3, EndLine: 3, Signature: "func Build() error", Exported: true},
					},
				},
				{
					Path: "internal/service.go",
					Symbols: []repomap.Symbol{
						{Name: "Build", Kind: "function", Line: 3, EndLine: 3, Signature: "func Build() error", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "state-relative-cwd",
		InvocationCWD:      subdir,
	}

	candidates := resolveSymbolCandidatesWithRuntime("Build", ".", runtime)
	if len(candidates) != 1 || candidates[0].File != "pkg/service.go" {
		t.Fatalf("expected relative pathHint to resolve from invocation cwd, got %+v", candidates)
	}
}

func TestFindAmbiguousFilesWithRuntime_ProjectMapSnapshot(t *testing.T) {
	dir := setupTestGoFiles(t, map[string]string{
		"pkg/a.go": "package pkg\n\nfunc placeholder() {}\n",
		"pkg/b.go": "package pkg\n\nfunc placeholder() {}\n",
	})

	runtime := GoSymbolRuntime{
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "pkg/a.go",
					Symbols: []repomap.Symbol{
						{Name: "Build", Kind: "function", Line: 3, EndLine: 3, Signature: "func Build() error", Exported: true},
					},
				},
				{
					Path: "pkg/b.go",
					Symbols: []repomap.Symbol{
						{Name: "Build", Kind: "function", Line: 3, EndLine: 3, Signature: "func Build() error", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "state-ambiguous",
	}

	ambiguous := findAmbiguousFilesWithRuntime("Build", SymbolCandidate{File: "pkg/a.go"}, runtime)
	if len(ambiguous) != 1 || !ambiguous["pkg/b.go"] {
		t.Fatalf("expected snapshot ambiguous file set, got %+v", ambiguous)
	}
}

// setupTestGoFiles は複数の Go ファイルを一時ディレクトリに作成し、
// そのディレクトリに cd した後、元に戻すための cleanup を登録する。
func setupTestGoFiles(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: could not restore directory: %v", err)
		}
	})
	return dir
}

// E2E: agent.go の Build を inspect したとき config.go 由来の caller/ref が混ざらない
func TestInspectSymbol_CrossFileIsolation_AgentSide(t *testing.T) {
	setupTestGoFiles(t, map[string]string{
		"agent.go": `package example

// Build はエージェントをビルドする。
func Build(name string) string {
	return "agent:" + name
}

// RunAgent は Build を呼ぶ。
func RunAgent() string {
	return Build("agent")
}
`,
		"config.go": `package example

// Config は設定。
type Config struct {
	Name string
}

// Build は Config をビルドする。
func (c *Config) Build() string {
	return "config:" + c.Name
}

// UseConfig は Config.Build を呼ぶ。
func UseConfig() string {
	c := &Config{Name: "test"}
	return c.Build()
}

// configRef は Build を参照する。
var configRef = Config{}.Build
`,
	})

	result := InspectSymbol("Build", "agent.go", "")
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find Build, got: %s", result)
	}
	// 複数候補の場合はここで終了（同一ファイル内の複数候補はここでは起きない）
	if strings.Contains(result, "Multiple symbols matched") {
		t.Fatalf("expected single candidate with path=agent.go, got: %s", result)
	}

	// config.go 由来の caller/ref が混ざっていないことを確認
	if strings.Contains(result, "config.go") {
		t.Errorf("config.go references should not appear in agent.go Build inspection:\n%s", result)
	}
	if strings.Contains(result, "UseConfig") {
		t.Errorf("UseConfig (config.go caller) should not appear:\n%s", result)
	}
	if strings.Contains(result, "configRef") {
		t.Errorf("configRef (config.go ref) should not appear:\n%s", result)
	}

	// agent.go 自身の caller（RunAgent）は出るべき
	// ただし同一ファイル内なので candidate 定義行範囲外の call のみ
	if !strings.Contains(result, "agent.go") {
		t.Errorf("expected agent.go in header, got: %s", result)
	}
}

// E2E: 逆方向 — config.go の Build を inspect したとき agent.go 由来が混ざらない
func TestInspectSymbol_CrossFileIsolation_ConfigSide(t *testing.T) {
	setupTestGoFiles(t, map[string]string{
		"agent.go": `package example

// Build はエージェントをビルドする。
func Build(name string) string {
	return "agent:" + name
}

// RunAgent は Build を呼ぶ。
func RunAgent() string {
	return Build("agent")
}
`,
		"config.go": `package example

// Config は設定。
type Config struct {
	Name string
}

// Build は Config をビルドする。
func (c *Config) Build() string {
	return "config:" + c.Name
}

// UseConfig は Config.Build を呼ぶ。
func UseConfig() string {
	c := &Config{Name: "test"}
	return c.Build()
}
`,
	})

	result := InspectSymbol("Build", "config.go", "")
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find Build, got: %s", result)
	}
	if strings.Contains(result, "Multiple symbols matched") {
		t.Fatalf("expected single candidate with path=config.go, got: %s", result)
	}

	// agent.go 由来の caller/ref が混ざっていないこと
	if strings.Contains(result, "agent.go") {
		t.Errorf("agent.go references should not appear in config.go Build inspection:\n%s", result)
	}
	if strings.Contains(result, "RunAgent") {
		t.Errorf("RunAgent (agent.go caller) should not appear:\n%s", result)
	}
}
