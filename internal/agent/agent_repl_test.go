package agent

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func TestCheckRipgrepAvailability_NoRg(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	t.Setenv("PATH", t.TempDir())

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(config.DefaultConfig())
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{Runtime: runtime}

	checkRipgrepAvailability(agent)

	if !strings.Contains(out.String(), "ripgrep (rg) not found") {
		t.Fatalf("expected ripgrep warning, got: %s", out.String())
	}
}

func TestCheckRipgrepAvailability_WithRg(t *testing.T) {
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	rgPath, err := exec.LookPath("rg")
	if err != nil {
		t.Skip("ripgrep (rg) not available")
	}
	t.Setenv("PATH", filepath.Dir(rgPath))

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(config.DefaultConfig())
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{Runtime: runtime}

	checkRipgrepAvailability(agent)

	if out.Len() != 0 {
		t.Fatalf("expected no output when rg exists, got: %s", out.String())
	}
}

func TestInjectProjectMap_AppendsProjectMap(t *testing.T) {
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

	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc Build() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	cfg := config.DefaultConfig()
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{
		Runtime:      runtime,
		SystemPrompt: "base prompt",
	}

	injectProjectMap(agent, "main.go を見て")

	if !strings.Contains(agent.SystemPrompt, "## Project Map") {
		t.Fatalf("expected Project Map in system prompt, got: %s", agent.SystemPrompt)
	}
	if agent.projectMapFileCount == 0 {
		t.Fatal("expected projectMapFileCount to be populated")
	}
	if agent.projectMapSymbolCount == 0 {
		t.Fatal("expected full project map with symbols")
	}
	if !strings.Contains(out.String(), "Project map loaded") {
		t.Fatalf("expected load message, got: %s", out.String())
	}
	if !strings.Contains(agent.SystemPrompt, "func Build()") {
		t.Fatalf("expected project map symbols, got: %s", agent.SystemPrompt)
	}
}

func TestRefreshProjectPrompt_ReusesCachedProjectMapWithoutRelogging(t *testing.T) {
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

	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc Build() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	cfg := config.DefaultConfig()
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{
		Runtime:      runtime,
		SystemPrompt: "base prompt",
		CurrentModel: "deepseek-chat",
	}

	injectProjectMap(agent, "")
	agent.refreshProjectPrompt("main.go を見て")

	if count := strings.Count(out.String(), "Project map loaded"); count != 1 {
		t.Fatalf("project map load message count = %d, want 1; output=%q", count, out.String())
	}
	if !strings.Contains(agent.SystemPrompt, "func Build()") {
		t.Fatalf("expected refreshed project map to include symbols:\n%s", agent.SystemPrompt)
	}
}

func TestRefreshProjectPrompt_ProjectMapStaysInDynamicSystemBlock(t *testing.T) {
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

	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc Build() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	cfg.PromptCache.Enabled = true

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{
		Runtime:      runtime,
		SystemPrompt: "base prompt",
		CurrentModel: "claude-opus-4-6",
	}

	injectProjectMap(agent, "")
	firstPrompt := agent.SystemPrompt

	if !strings.Contains(firstPrompt, api.SystemPromptCacheBoundary) {
		t.Fatalf("expected project map cache boundary in system prompt:\n%s", firstPrompt)
	}

	firstField := api.BuildSystemFieldWithConfig(firstPrompt, cfg)
	firstBlocks, ok := firstField.([]api.SystemBlock)
	if !ok {
		t.Fatalf("expected []api.SystemBlock, got %T", firstField)
	}
	if len(firstBlocks) != 2 {
		t.Fatalf("expected 2 system blocks after project map injection, got %d", len(firstBlocks))
	}

	if err := os.WriteFile(filepath.Join(root, "extra.go"), []byte("package main\n\nfunc Extra() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	agent.refreshProjectPrompt("extra.go を見て")
	secondField := api.BuildSystemFieldWithConfig(agent.SystemPrompt, cfg)
	secondBlocks, ok := secondField.([]api.SystemBlock)
	if !ok {
		t.Fatalf("expected []api.SystemBlock, got %T", secondField)
	}
	if len(secondBlocks) != 2 {
		t.Fatalf("expected 2 system blocks after project map refresh, got %d", len(secondBlocks))
	}

	if firstBlocks[0].Text != secondBlocks[0].Text {
		t.Fatalf("expected static cache block to remain stable after project map change")
	}
	if firstBlocks[1].Text == secondBlocks[1].Text {
		t.Fatalf("expected dynamic project map block to change after repo update")
	}
	if secondBlocks[1].CacheControl == nil {
		t.Fatal("expected cache_control on dynamic project map block")
	}
}

func TestRefreshProjectPrompt_RebuildsProjectMapWhenRepoStateChanges(t *testing.T) {
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

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(config.DefaultConfig())
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{
		Runtime:      runtime,
		SystemPrompt: "base prompt",
		CurrentModel: "deepseek-chat",
	}

	injectProjectMap(agent, "")

	if err := os.WriteFile(filepath.Join(root, "extra.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	agent.refreshProjectPrompt("extra.go を見て")

	if count := strings.Count(out.String(), "Project map loaded"); count != 2 {
		t.Fatalf("project map should rebuild after repo state change, got count=%d output=%q", count, out.String())
	}
	if !strings.Contains(agent.SystemPrompt, "extra.go") {
		t.Fatalf("expected rebuilt project map to include extra.go:\n%s", agent.SystemPrompt)
	}
}

func TestRefreshProjectPrompt_RebuildsProjectMapWhenNonGitNestedFileAdded(t *testing.T) {
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

	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "main.go"), []byte("package pkg\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(config.DefaultConfig())
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{
		Runtime:      runtime,
		SystemPrompt: "base prompt",
		CurrentModel: "deepseek-chat",
	}

	injectProjectMap(agent, "")

	if err := os.WriteFile(filepath.Join(root, "pkg", "extra.go"), []byte("package pkg\n"), 0644); err != nil {
		t.Fatal(err)
	}

	agent.refreshProjectPrompt("pkg/extra.go を見て")

	if count := strings.Count(out.String(), "Project map loaded"); count != 2 {
		t.Fatalf("project map should rebuild after nested non-git file add, got count=%d output=%q", count, out.String())
	}
	if !strings.Contains(agent.SystemPrompt, "extra.go") {
		t.Fatalf("expected rebuilt project map to include extra.go:\n%s", agent.SystemPrompt)
	}
}

func TestCurrentProjectMapStateKey_NonGitWatchesEmptyDirectories(t *testing.T) {
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "pkg", "empty"), 0755); err != nil {
		t.Fatal(err)
	}

	watchDirs := collectProjectMapWatchDirs(root, nil)
	if !slices.Contains(watchDirs, "pkg") || !slices.Contains(watchDirs, "pkg/empty") {
		t.Fatalf("collectProjectMapWatchDirs() = %v, want empty directories included", watchDirs)
	}

	agent := &Agent{projectMapWatchDirs: watchDirs}
	before := currentProjectMapStateKey(agent, root)
	if before == "" {
		t.Fatal("expected non-empty state key before change")
	}

	if err := os.WriteFile(filepath.Join(root, "pkg", "empty", "new.go"), []byte("package empty\n"), 0644); err != nil {
		t.Fatal(err)
	}

	after := currentProjectMapStateKey(agent, root)
	if after == "" {
		t.Fatal("expected non-empty state key after change")
	}
	if before == after {
		t.Fatalf("expected state key to change after nested file add in empty dir, before=%q after=%q", before, after)
	}
}

func TestCurrentProjectMapStateKey_NonGitIgnoresIgnoredEntries(t *testing.T) {
	root := t.TempDir()
	ignorePatterns := []string{"dist", "*.gen.go"}

	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0755); err != nil {
		t.Fatal(err)
	}

	watchDirs := collectProjectMapWatchDirs(root, ignorePatterns)
	agent := &Agent{
		projectMapWatchDirs: watchDirs,
		projectMapIgnoreKey: strings.Join(ignorePatterns, "\x00"),
	}

	before := currentProjectMapStateKey(agent, root)
	if before == "" {
		t.Fatal("expected non-empty state key before ignored changes")
	}

	if err := os.MkdirAll(filepath.Join(root, "dist"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "types.gen.go"), []byte("package pkg\n"), 0644); err != nil {
		t.Fatal(err)
	}

	after := currentProjectMapStateKey(agent, root)
	if after == "" {
		t.Fatal("expected non-empty state key after ignored changes")
	}
	if before != after {
		t.Fatalf("expected ignored entries to keep state key stable, before=%q after=%q", before, after)
	}
}

func TestRefreshProjectPromptIfDirty_RebuildsProjectMapAfterToolMutation(t *testing.T) {
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

	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(config.DefaultConfig())
	runtime.UI = ui.NewRuntime(strings.NewReader(""), &out, io.Discard)
	agent := &Agent{
		Runtime:      runtime,
		SystemPrompt: "base prompt",
		CurrentModel: "deepseek-chat",
	}

	injectProjectMap(agent, "")

	if err := os.WriteFile(filepath.Join(root, "generated.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	agent.noteProjectMapMutation(&tools.ToolCall{Tool: "bash", Args: map[string]string{"command": "go build ./..."}}, nil)
	if !agent.projectMapDirty {
		t.Fatal("expected projectMapDirty after bash mutation")
	}

	agent.refreshProjectPromptIfDirty("generated.go を見て")

	if agent.projectMapDirty {
		t.Fatal("expected refreshProjectPromptIfDirty to clear dirty flag")
	}
	if count := strings.Count(out.String(), "Project map loaded"); count != 2 {
		t.Fatalf("project map should rebuild after dirty refresh, got count=%d output=%q", count, out.String())
	}
	if !strings.Contains(agent.SystemPrompt, "generated.go") {
		t.Fatalf("expected rebuilt project map to include generated.go:\n%s", agent.SystemPrompt)
	}
}

func TestNoteProjectMapMutation_DoesNotInvalidateReadOnlyBash(t *testing.T) {
	agent := &Agent{projectMapDirty: false}

	agent.noteProjectMapMutation(&tools.ToolCall{
		Tool: "bash",
		Args: map[string]string{"command": "git status"},
	}, nil)

	if agent.projectMapDirty {
		t.Fatal("expected read-only bash to keep project map cache valid")
	}
}

func TestExtractProjectMapSection_WithTrailingSection(t *testing.T) {
	prompt := "base\n\n## Project Map\n📂 src/\n└── 📄 main.go (10 lines)\n\n## Project Context:\nSome context"
	section := extractProjectMapSection(prompt)
	if strings.Contains(section, "Project Context") {
		t.Fatalf("section should not include trailing content:\n%s", section)
	}
	if !strings.Contains(section, "main.go") {
		t.Fatalf("section should include Project Map content:\n%s", section)
	}
}

func TestExtractProjectMapSection_AtEnd(t *testing.T) {
	prompt := "base\n\n## Project Map\n📂 src/\n└── 📄 main.go (10 lines)"
	section := extractProjectMapSection(prompt)
	if !strings.Contains(section, "main.go") {
		t.Fatalf("section should include Project Map content:\n%s", section)
	}
}

func TestExtractProjectMapSection_NotPresent(t *testing.T) {
	prompt := "base prompt without project map"
	section := extractProjectMapSection(prompt)
	if section != "" {
		t.Fatalf("expected empty string, got: %s", section)
	}
}

func TestStripProjectMapSection(t *testing.T) {
	prompt := "base prompt\n\n## Project Map\nTop-level files:\n- main.go"
	stripped := stripProjectMapSection(prompt)
	if stripped != "base prompt" {
		t.Fatalf("stripProjectMapSection() = %q, want %q", stripped, "base prompt")
	}
}
