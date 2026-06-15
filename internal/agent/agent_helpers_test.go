package agent

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestCalcProjectMapBudget_Auto_LargeContext(t *testing.T) {
	// gpt-5.4: 1M context × 2% = 20000
	agent := &Agent{CurrentModel: "gpt-5.4"}
	cfg := config.DefaultConfig()
	cfg.ProjectMap.ContextRatio = 0.02

	got := calcProjectMapBudget(agent, cfg, 50, 500)
	if got != 20000 {
		t.Errorf("calcProjectMapBudget() = %d, want 20000", got)
	}
}

func TestCalcProjectMapBudget_Auto_SmallContext(t *testing.T) {
	// unknown model → GetModelTokenLimit default 100K × 2% = 2000
	agent := &Agent{CurrentModel: "unknown-small-model"}
	cfg := config.DefaultConfig()
	cfg.ProjectMap.ContextRatio = 0.02

	got := calcProjectMapBudget(agent, cfg, 50, 500)
	if got != 2000 {
		t.Errorf("calcProjectMapBudget() = %d, want 2000 (100K default × 2%%)", got)
	}
}

func TestCalcProjectMapBudget_Auto_MediumContext(t *testing.T) {
	// claude-sonnet-4-5: 200K context × 2% = 4000
	agent := &Agent{CurrentModel: "claude-sonnet-4-5"}
	cfg := config.DefaultConfig()
	cfg.ProjectMap.ContextRatio = 0.02

	got := calcProjectMapBudget(agent, cfg, 50, 500)
	if got != 4000 {
		t.Errorf("calcProjectMapBudget() = %d, want 4000", got)
	}
}

func TestCalcProjectMapBudget_InvalidRatio(t *testing.T) {
	agent := &Agent{CurrentModel: "claude-sonnet-4-5"}
	cfg := config.DefaultConfig()

	tests := []struct {
		name  string
		ratio float64
	}{
		{"zero", 0},
		{"negative", -0.5},
		{"over_max", 0.25},
		{"nan", math.NaN()},
		{"inf", math.Inf(1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg.ProjectMap.ContextRatio = tt.ratio
			got := calcProjectMapBudget(agent, cfg, 50, 500)
			// デフォルト 0.05 にフォールバック → 200K × 5% = 10000
			if got != 10000 {
				t.Errorf("calcProjectMapBudget() = %d, want 10000 (fallback to 0.05)", got)
			}
		})
	}
}

func TestCalcProjectMapBudget_RatioOverride(t *testing.T) {
	// claude-sonnet-4-5: 200K × 5% = 10000
	agent := &Agent{CurrentModel: "claude-sonnet-4-5"}
	cfg := config.DefaultConfig()
	cfg.ProjectMap.ContextRatio = 0.05

	got := calcProjectMapBudget(agent, cfg, 50, 500)
	if got != 10000 {
		t.Errorf("calcProjectMapBudget() = %d, want 10000", got)
	}
}

func TestCalcProjectMapBudget_SmallModelHasNoFixedFloor(t *testing.T) {
	// deepseek-chat: V4 Flash alias 1M context × 1% = 10000
	agent := &Agent{CurrentModel: "deepseek-chat"}
	cfg := config.DefaultConfig()
	cfg.ProjectMap.ContextRatio = 0.01

	got := calcProjectMapBudget(agent, cfg, 50, 500)
	if got != 10000 {
		t.Errorf("calcProjectMapBudget() = %d, want 10000", got)
	}
}

func TestEffectiveProjectMapContextRatio_AutoBoost(t *testing.T) {
	tests := []struct {
		name      string
		baseRatio float64
		fileCount int
		symbols   int
		wantRatio float64
	}{
		{name: "small repo keeps default", baseRatio: 0.02, fileCount: 80, symbols: 900, wantRatio: 0.02},
		{name: "medium repo boosts to three percent", baseRatio: 0.02, fileCount: 220, symbols: 1800, wantRatio: 0.03},
		{name: "large repo boosts to four percent", baseRatio: 0.02, fileCount: 420, symbols: 3500, wantRatio: 0.04},
		{name: "symbol-heavy repo boosts to four percent", baseRatio: 0.02, fileCount: 120, symbols: 4500, wantRatio: 0.04},
		{name: "user higher ratio is preserved", baseRatio: 0.05, fileCount: 420, symbols: 4500, wantRatio: 0.05},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveProjectMapContextRatio(tt.baseRatio, tt.fileCount, tt.symbols)
			if got != tt.wantRatio {
				t.Errorf("effectiveProjectMapContextRatio() = %v, want %v", got, tt.wantRatio)
			}
		})
	}
}

func TestCalcProjectMapBudget_AutoBoostForLargeRepo(t *testing.T) {
	// gpt-5.1: 400K context, large repo boosts 2% -> 4%
	agent := &Agent{CurrentModel: "gpt-5.1"}
	cfg := config.DefaultConfig()
	cfg.ProjectMap.ContextRatio = 0.02

	got := calcProjectMapBudget(agent, cfg, 430, 3200)
	if got != 16000 {
		t.Errorf("calcProjectMapBudget() = %d, want 16000", got)
	}
}

func TestExtractProjectMapFocusPaths_RepoRelativePathSurvivesNestedRootSession(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "work")
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cwd, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "x.go"), []byte("package pkg\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got := extractProjectMapFocusPaths(cwd, root, "pkg/x.go を見て", projectMapFocusMaxPaths)

	if len(got) != 1 || got[0] != "pkg/x.go" {
		t.Fatalf("extractProjectMapFocusPaths() = %v, want [pkg/x.go]", got)
	}
}

func TestExtractProjectMapFocusPaths_CwdRelativePathSurvivesNestedRootSession(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "sub")
	if err := os.MkdirAll(cwd, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, "edited.go"), []byte("package sub\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got := extractProjectMapFocusPaths(cwd, root, "edited.go を見て", projectMapFocusMaxPaths)

	if len(got) != 1 || got[0] != "sub/edited.go" {
		t.Fatalf("extractProjectMapFocusPaths() = %v, want [sub/edited.go]", got)
	}
}

func TestExtractProjectMapFocusPaths_WindowsRelativePathIsIncluded(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "internal", "agent", "compress.go")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package agent\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got := extractProjectMapFocusPaths(root, root, `internal\agent\compress.go を見て`, projectMapFocusMaxPaths)
	if len(got) != 1 || got[0] != "internal/agent/compress.go" {
		t.Fatalf("extractProjectMapFocusPaths() = %v, want [internal/agent/compress.go]", got)
	}
}

func TestExtractProjectMapFocusPaths_WindowsAbsolutePathIsIncluded(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "internal", "agent", "compress.go")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package agent\n"), 0644); err != nil {
		t.Fatal(err)
	}

	windowsRoot := "C:" + filepath.ToSlash(root)
	windowsQueryPath := strings.ReplaceAll(windowsRoot+"/internal/agent/compress.go", "/", `\`)

	got := extractProjectMapFocusPaths(root, root, windowsQueryPath+` を見て`, projectMapFocusMaxPaths)
	if len(got) != 1 || got[0] != "internal/agent/compress.go" {
		t.Fatalf("extractProjectMapFocusPaths() = %v, want [internal/agent/compress.go]", got)
	}
}

func TestExtractProjectMapFocusPaths_WindowsAbsolutePathDoesNotMisprioritizeBasename(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "internal", "agent", "compress.go")
	other := filepath.Join(root, "pkg", "compress.go")
	for _, path := range []string{target, other} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("package main\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	windowsRoot := "C:" + filepath.ToSlash(root)
	windowsQueryPath := strings.ReplaceAll(windowsRoot+"/internal/agent/compress.go", "/", `\`)

	got := extractProjectMapFocusPaths(root, root, windowsQueryPath+` を見て`, projectMapFocusMaxPaths)
	if len(got) != 1 || got[0] != "internal/agent/compress.go" {
		t.Fatalf("extractProjectMapFocusPaths() = %v, want [internal/agent/compress.go]", got)
	}
}

func TestExtractProjectMapFocusPaths_ImportPathIsExcluded(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got := extractProjectMapFocusPaths(root, root, "github.com/acme/lib の import を直して", projectMapFocusMaxPaths)
	if got != nil {
		t.Fatalf("extractProjectMapFocusPaths() = %v, want nil for import path", got)
	}
}

func TestExtractProjectMapFocusPaths_DirectoryPathIsIncluded(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "internal", "agent")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	got := extractProjectMapFocusPaths(root, root, "internal/agent を見て", projectMapFocusMaxPaths)
	if len(got) != 1 || got[0] != "internal/agent" {
		t.Fatalf("extractProjectMapFocusPaths() = %v, want [internal/agent]", got)
	}
}

func TestExtractProjectMapFocusPaths_NonexistentPathIsExcluded(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got := extractProjectMapFocusPaths(root, root, "pkg/errors を見て", projectMapFocusMaxPaths)
	if got != nil {
		t.Fatalf("extractProjectMapFocusPaths() = %v, want nil for nonexistent path", got)
	}
}

func TestExtractProjectMapFocusPaths_EmptyWhenQueryHasNoPaths(t *testing.T) {
	root := t.TempDir()
	got := extractProjectMapFocusPaths(root, root, "実装方針を整理して", projectMapFocusMaxPaths)
	if got != nil {
		t.Fatalf("extractProjectMapFocusPaths() = %v, want nil", got)
	}
}

func TestBuildProjectMapFocusKey_PreservesOrder(t *testing.T) {
	left := buildProjectMapFocusKey([]string{"pkg/a.go", "pkg/b.go"})
	right := buildProjectMapFocusKey([]string{"pkg/b.go", "pkg/a.go"})

	if left == right {
		t.Fatalf("buildProjectMapFocusKey() should differ when order changes: left=%q right=%q", left, right)
	}
}

func TestBuildProjectMapFocusKey_DedupesWithoutReordering(t *testing.T) {
	got := buildProjectMapFocusKey([]string{"pkg/a.go", "pkg/b.go", "pkg/a.go"})
	want := "pkg/a.go\x00pkg/b.go"
	if got != want {
		t.Fatalf("buildProjectMapFocusKey() = %q, want %q", got, want)
	}
}

func TestBuildProjectMapBaseKey_ChangesWhenBudgetChanges(t *testing.T) {
	cfg := config.DefaultConfig()
	agent := &Agent{
		CurrentModel: "deepseek-chat",
		agentProjectPromptState: agentProjectPromptState{
			projectMapStateKey: "state",
		},
	}

	first := buildProjectMapBaseKey(agent, cfg, 6400, 120, 1200)
	second := buildProjectMapBaseKey(agent, cfg, 1280, 120, 1200)

	if first == second {
		t.Fatalf("buildProjectMapBaseKey() should change when budget changes: first=%q second=%q", first, second)
	}
}

func setupProjectPromptRefreshWorkspace(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.MkdirAll(filepath.Join(root, "internal", "agent"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "internal", "agent", "compress.go")
	if err := os.WriteFile(target, []byte("package agent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, target
}

func newProjectPromptRefreshTestAgent(stateKey, focusKey, root string) *Agent {
	fileCount := 1
	symbolCount := 1
	baseAgent := &Agent{
		CurrentModel: "deepseek-chat",
		agentProjectPromptState: agentProjectPromptState{
			projectMapStateKey: stateKey,
		},
	}
	baseKey := buildProjectMapBaseKey(baseAgent, config.DefaultConfig(), calcProjectMapBudget(baseAgent, config.DefaultConfig(), fileCount, symbolCount), fileCount, symbolCount)

	return &Agent{
		Runtime:      NewAgentRuntimeWithConfig(config.DefaultConfig()),
		CurrentModel: "deepseek-chat",
		agentProjectPromptState: agentProjectPromptState{
			projectMap:            &repomap.ProjectMap{},
			projectMapStateKey:    stateKey,
			projectMapBaseKey:     baseKey,
			projectMapFocusKey:    focusKey,
			projectMapBaseSection: "cached-base",
			projectMapSection:     "cached",
			projectMapDirty:       false,
			projectMapRootPath:    root,
			projectMapIgnoreKey:   "",
			projectMapWatchDirs:   []string{"."},
			projectMapFileCount:   fileCount,
			projectMapSymbolCount: symbolCount,
		},
	}
}

func TestShouldRefreshProjectPrompt_FocusKeyChangeTriggersRefresh(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep (rg) not available")
	}

	root, _ := setupProjectPromptRefreshWorkspace(t)
	stateKey := currentProjectMapStateKey(&Agent{}, root)

	agent := newProjectPromptRefreshTestAgent(stateKey, "", root)

	if !agent.shouldRefreshProjectPrompt("internal/agent/compress.go を見て") {
		t.Fatal("expected focus key change to trigger prompt refresh")
	}
}

func TestShouldRefreshProjectPrompt_SameQueryReusesPrompt(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep (rg) not available")
	}

	root, _ := setupProjectPromptRefreshWorkspace(t)
	stateKey := currentProjectMapStateKey(&Agent{}, root)

	focusPaths := []string{"internal/agent/compress.go"}
	agent := newProjectPromptRefreshTestAgent(stateKey, buildProjectMapFocusKey(focusPaths), root)

	if agent.shouldRefreshProjectPrompt("internal/agent/compress.go を見て") {
		t.Fatal("expected same focus key to reuse cached prompt")
	}
}

func TestShouldRefreshProjectPrompt_IgnoresRecentToolCacheChurn(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep (rg) not available")
	}

	root := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	filePath := filepath.Join(root, "recent.go")
	mainPath := filepath.Join(root, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	stateKey := currentProjectMapStateKey(&Agent{}, root)

	cache := NewToolCache()
	agent := newProjectPromptRefreshTestAgent(stateKey, buildProjectMapFocusKey([]string{"main.go"}), root)
	agent.ToolCache = cache

	cache.SetFile(filePath, "package main\n")
	cache.SetSearch("recent", root, "result", []string{filePath})

	if agent.shouldRefreshProjectPrompt("main.go を見て") {
		t.Fatal("expected recent read/search cache churn to be ignored")
	}
}

func TestProjectPromptRefreshDecision_FocusKeyChangeReason(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep (rg) not available")
	}

	root, _ := setupProjectPromptRefreshWorkspace(t)
	stateKey := currentProjectMapStateKey(&Agent{}, root)

	agent := newProjectPromptRefreshTestAgent(stateKey, "", root)

	decision := agent.promptManager().ProjectPromptRefreshDecision("internal/agent/compress.go を見て")
	if !decision.NeedRefresh {
		t.Fatal("expected decision.NeedRefresh=true")
	}
	if decision.Reason != refreshReasonFocusKeyChanged {
		t.Fatalf("decision.Reason = %q, want %q", decision.Reason, refreshReasonFocusKeyChanged)
	}
}

func TestProjectPromptRefreshDecision_NoChangeReason(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep (rg) not available")
	}

	root, _ := setupProjectPromptRefreshWorkspace(t)
	stateKey := currentProjectMapStateKey(&Agent{}, root)
	focusPaths := []string{"internal/agent/compress.go"}

	agent := newProjectPromptRefreshTestAgent(stateKey, buildProjectMapFocusKey(focusPaths), root)

	decision := agent.promptManager().ProjectPromptRefreshDecision("internal/agent/compress.go を見て")
	if decision.NeedRefresh {
		t.Fatal("expected decision.NeedRefresh=false")
	}
	if decision.Reason != refreshReasonNoChange {
		t.Fatalf("decision.Reason = %q, want %q", decision.Reason, refreshReasonNoChange)
	}
}
