package agent

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/token"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func TestInjectProjectMap_AppendsProjectManifestAndKeepsFullRuntimeMap(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	if _, err := exec.LookPath("rg"); err != nil {
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
	markProjectMapTestRoot(t, root)

	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc Build() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	cfg := config.DefaultConfig()
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{
		Runtime:      runtime,
		SystemPrompt: "base prompt",
	}

	injectProjectMap(agent, "main.go を見て")

	if !strings.Contains(agent.SystemPrompt, "## Project Map") {
		t.Fatalf("expected Project Map in system prompt, got: %s", agent.SystemPrompt)
	}
	if !strings.Contains(agent.SystemPrompt, "<project_map_data>") {
		t.Fatalf("expected Project Map data wrapper in system prompt, got: %s", agent.SystemPrompt)
	}
	if !strings.Contains(agent.SystemPrompt, "Top-level files:") {
		t.Fatalf("expected manifest-style project map in system prompt, got: %s", agent.SystemPrompt)
	}
	if agent.projectMapFileCount == 0 {
		t.Fatal("expected projectMapFileCount to be populated")
	}
	if agent.projectMapSymbolCount == 0 {
		t.Fatal("expected full project map with symbols")
	}
	if agent.projectMap == nil {
		t.Fatal("expected runtime projectMap to be retained")
	}
	if agent.projectMap.GetSymbolCount() == 0 {
		t.Fatal("expected runtime projectMap to keep full symbols")
	}
	if !strings.Contains(out.String(), "Project map loaded") {
		t.Fatalf("expected load message, got: %s", out.String())
	}
	if strings.Contains(agent.SystemPrompt, "func Build()") {
		t.Fatalf("expected prompt project map to use manifest instead of full symbols, got: %s", agent.SystemPrompt)
	}
	if strings.Contains(agent.projectMapSection, "func Build()") {
		t.Fatalf("expected cached projectMapSection to be manifest, got: %s", agent.projectMapSection)
	}
	if !strings.Contains(agent.projectMapSection, "main.go") {
		t.Fatalf("expected cached manifest to mention main.go, got: %s", agent.projectMapSection)
	}
}

func TestInjectProjectMap_AddsFocusOverlayForNestedInputPath(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	if _, err := exec.LookPath("rg"); err != nil {
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
	markProjectMapTestRoot(t, root)

	nested := filepath.Join(root, "internal", "agent", "compress.go")
	if err := os.MkdirAll(filepath.Dir(nested), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nested, []byte("package agent\n\nfunc Compress() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(config.DefaultConfig())
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{
		Runtime:      runtime,
		SystemPrompt: "base prompt",
	}

	injectProjectMap(agent, "internal/agent/compress.go を見て")

	if !strings.Contains(agent.SystemPrompt, "Focus files for current task:") {
		t.Fatalf("expected focus overlay section:\n%s", agent.SystemPrompt)
	}
	if !strings.Contains(agent.SystemPrompt, "internal/agent/compress.go") {
		t.Fatalf("expected focus overlay to keep nested input path:\n%s", agent.SystemPrompt)
	}
}

func TestInjectProjectMap_ExcludesImportPathFromFocusOverlay(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	if _, err := exec.LookPath("rg"); err != nil {
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
	markProjectMapTestRoot(t, root)

	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(config.DefaultConfig())
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{
		Runtime:      runtime,
		SystemPrompt: "base prompt",
	}

	injectProjectMap(agent, "github.com/acme/lib の import を直して")

	if strings.Contains(agent.SystemPrompt, "Focus files for current task:") {
		t.Fatalf("expected import path to be excluded from focus overlay:\n%s", agent.SystemPrompt)
	}
	if strings.Contains(agent.SystemPrompt, "github.com/acme/lib") {
		t.Fatalf("expected import path not to appear in project map prompt:\n%s", agent.SystemPrompt)
	}
}

func TestInjectProjectMap_DoesNotUseRecentToolPathsForFocusOverlay(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	if _, err := exec.LookPath("rg"); err != nil {
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
	markProjectMapTestRoot(t, root)

	recentRead := filepath.Join(root, "pkg", "recent_read.go")
	recentEdit := filepath.Join(root, "pkg", "recent_edit.go")
	for _, path := range []string{recentRead, recentEdit} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("package pkg\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cache := NewToolCache()
	cache.SetFile(recentRead, "package pkg\n")

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(config.DefaultConfig())
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), &out, io.Discard)
	runtime.ToolCache = cache
	agent := &Agent{
		Runtime:      runtime,
		SystemPrompt: "base prompt",
		ToolCache:    cache,
		agentWorkspaceState: agentWorkspaceState{
			changeStack: []tools.FileChange{
				{FilePath: recentEdit},
			},
		},
	}

	injectProjectMap(agent, "")

	if strings.Contains(agent.SystemPrompt, "Focus files for current task:") {
		t.Fatalf("expected recent tool paths to be excluded from focus overlay:\n%s", agent.SystemPrompt)
	}
	if strings.Contains(agent.SystemPrompt, "pkg/recent_read.go") || strings.Contains(agent.SystemPrompt, "pkg/recent_edit.go") {
		t.Fatalf("expected recent tool paths to stay out of prompt focus:\n%s", agent.SystemPrompt)
	}
}

func TestInjectProjectMap_RebuildsBaseManifestWhenBudgetShrinks(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep (rg) not available")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
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

	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	runGit("init")
	runGit("config", "user.name", "Test User")
	runGit("config", "user.email", "test@example.com")

	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit("add", "main.go")
	runGit("commit", "-m", "init")

	const changedFileCount = 500
	for i := 0; i < changedFileCount; i++ {
		path := filepath.Join(root, "pkg", fmt.Sprintf("file%03d.go", i))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("package pkg\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	runGit("add", "pkg")
	runGit("commit", "-m", "add-pkg")
	for i := 0; i < changedFileCount; i++ {
		path := filepath.Join(root, "pkg", fmt.Sprintf("file%03d.go", i))
		if err := os.WriteFile(path, []byte(fmt.Sprintf("package pkg\n// change %03d\n", i)), 0644); err != nil {
			t.Fatal(err)
		}
	}

	var out bytes.Buffer
	cfg := config.DefaultConfig()
	cfg.ProjectMap.ContextRatio = 0.05
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = uiruntime.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{
		Runtime:      runtime,
		SystemPrompt: "base prompt",
		CurrentModel: "deepseek-chat",
	}

	injectProjectMap(agent, "")

	firstMapPtr := agent.projectMap
	firstBaseKey := agent.projectMapBaseKey
	firstBaseSection := agent.projectMapBaseSection
	firstBaseTokens := token.EstimateTokenCount(firstBaseSection)
	if firstBaseTokens <= 1280 {
		t.Fatalf("expected initial base manifest to exceed shrunk budget, got %d tokens", firstBaseTokens)
	}

	runtime.Config.ProjectMap.ContextRatio = 0.01
	agent.refreshProjectPrompt("")

	secondBaseKey := agent.projectMapBaseKey
	secondBaseSection := agent.projectMapBaseSection
	secondBudget := calcProjectMapBudget(agent, agent.cfg(), agent.projectMapFileCount, agent.projectMapSymbolCount)

	if agent.projectMap != firstMapPtr {
		t.Fatal("expected full project map to be reused when only budget changes")
	}
	if firstBaseKey == secondBaseKey {
		t.Fatalf("expected base manifest cache key to change after budget shrink: %q", firstBaseKey)
	}
	if got := token.EstimateTokenCount(secondBaseSection); got > secondBudget {
		t.Fatalf("expected regenerated base manifest to fit new budget, got %d tokens > budget %d", got, secondBudget)
	}
}
