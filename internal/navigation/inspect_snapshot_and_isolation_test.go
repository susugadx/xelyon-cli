package navigation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/repomap"
)

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
