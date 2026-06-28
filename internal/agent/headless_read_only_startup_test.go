package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestHeadlessReadOnlyDoesNotLoadOrSavePersistentToolCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := testSubDir(t)
	cacheDir := filepath.Join(dir, ".xelyon", "cache")
	cacheFile := filepath.Join(cacheDir, "tool_cache.json")
	sentinel := []byte("not json{{{")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cacheFile, sentinel, 0o600); err != nil {
		t.Fatal(err)
	}

	provider := &sequenceMockProvider{
		name:      "openai",
		responses: []string{"final response without tools"},
	}
	result := RunHeadlessWithConfigOptions(context.Background(), "no tools", "gpt-5.4", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
		ReadOnly: true,
	})
	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}

	got, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("read-only headless should leave existing tool cache file untouched, read error = %v", err)
	}
	if string(got) != string(sentinel) {
		t.Fatalf("tool cache file = %q, want unchanged sentinel %q", string(got), string(sentinel))
	}
}

func TestHeadlessReadOnlyDoesNotCleanupDevArtifacts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := testSubDir(t)
	artifactDir := filepath.Join(dir, ".xelyon", "artifacts")
	artifactFile := filepath.Join(artifactDir, "output_old.txt")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactFile, []byte("old artifact\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(artifactFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	provider := &sequenceMockProvider{
		name:      "openai",
		responses: []string{"final response without tools"},
	}
	result := RunHeadlessWithConfigOptions(context.Background(), "no tools", "gpt-5.4", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
		ReadOnly: true,
	})
	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}

	if _, err := os.Stat(artifactFile); err != nil {
		t.Fatalf("read-only headless should not remove existing dev artifact, stat error = %v", err)
	}
}

func TestHeadlessReadOnlyDoesNotCreateStartupPersistenceDirs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XELYON_AUDIT_LOG", "1")

	provider := &headlessToolSetProbeProvider{name: "openai"}
	result := RunHeadlessWithConfigOptions(context.Background(), "probe", "gpt-5.4", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
		ReadOnly: true,
	})
	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	for _, dir := range []string{
		filepath.Join(home, ".xelyon", "history"),
		filepath.Join(home, ".xelyon", "changes"),
		filepath.Join(home, ".xelyon", "audit"),
	} {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatalf("read-only headless should not create startup persistence dir %s, stat error = %v", dir, err)
		}
	}
}

func TestHeadlessReadOnlyDisablesStartupProjectMapWriters(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := testSubDir(t)
	markProjectMapTestRoot(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	if !cfg.ProjectMap.Enabled {
		t.Fatal("default config should enable ProjectMap for this regression test")
	}
	runtimeCfg := headlessRuntimeConfigForOptions(cfg, HeadlessRunOptions{ReadOnly: true})
	if runtimeCfg == cfg {
		t.Fatal("read-only runtime config should be cloned, not mutate caller config")
	}
	if runtimeCfg.ProjectMap.Enabled {
		t.Fatal("read-only runtime config should disable ProjectMap")
	}

	provider := &headlessToolSetProbeProvider{name: "openai"}
	result := RunHeadlessWithConfigOptions(context.Background(), "main.go を見て", "gpt-5.4", provider, cfg, HeadlessRunOptions{
		ReadOnly: true,
	})
	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	if !cfg.ProjectMap.Enabled {
		t.Fatal("caller config ProjectMap.Enabled was mutated, want unchanged true")
	}
	if strings.Contains(provider.systemPrompt, "<project_map_data>") || strings.Contains(provider.systemPrompt, "Focus files for current task:") {
		t.Fatalf("read-only headless should not inject startup project map:\n%s", provider.systemPrompt)
	}
	if _, err := os.Stat(filepath.Join(home, ".xelyon", "cache", "projectmap")); !os.IsNotExist(err) {
		t.Fatalf("read-only headless should not create project map cache directory, stat error = %v", err)
	}
}

func TestHeadlessReadOnlyRuntimeConfigDisablesLSPWithoutMutatingCaller(t *testing.T) {
	cfg := config.DefaultConfig()
	if !cfg.LSP.Enabled {
		t.Fatal("default config should enable LSP for this regression test")
	}

	runtimeCfg := headlessRuntimeConfigForOptions(cfg, HeadlessRunOptions{ReadOnly: true})
	if runtimeCfg == cfg {
		t.Fatal("read-only runtime config should be cloned, not mutate caller config")
	}
	if runtimeCfg.LSP.Enabled {
		t.Fatal("read-only runtime config should disable LSP")
	}
	if !cfg.LSP.Enabled {
		t.Fatal("caller config LSP.Enabled was mutated, want unchanged true")
	}

	normalCfg := headlessRuntimeConfigForOptions(cfg, HeadlessRunOptions{})
	if normalCfg != cfg {
		t.Fatal("normal headless runtime config should keep caller config")
	}
	if !normalCfg.LSP.Enabled {
		t.Fatal("normal headless runtime config should keep LSP enabled")
	}
}

func TestHeadlessReadOnlySkipsSkillRouterGitStatusAndUsageLedger(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := testSubDir(t)
	markProjectMapTestRoot(t, dir)
	writeSkillRoutingTestSkillWithSidecar(t, dir, "strict-review", "Review diffs and report actionable findings.", strings.Join([]string{
		"version: 1",
		"intents:",
		"  - code-review",
		"role: primary",
		"read_only: true",
		"modes:",
		"  - review",
		"triggers:",
		"  - review",
		"activation: hint",
		"",
	}, "\n"))

	gitMarker := installFakeGitStatusMarker(t)
	cfg := newProjectMapDisabledConfig()
	cfg.Skills.Router.UsageLedger = true
	provider := &headlessToolSetProbeProvider{name: "openai"}

	result := RunHeadlessWithConfigOptions(context.Background(), "review this diff", "gpt-5.4", provider, cfg, HeadlessRunOptions{
		ReadOnly: true,
	})
	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	if !strings.Contains(provider.systemPrompt, "strict-review") {
		t.Fatalf("read-only headless should keep skill-router recommendations from task/catalog signals:\n%s", provider.systemPrompt)
	}
	if _, err := os.Stat(gitMarker); !os.IsNotExist(err) {
		t.Fatalf("read-only headless should not run git status signal, marker stat error = %v", err)
	}
	usageDir := filepath.Join(home, ".xelyon", "skills", "router", "usage")
	if _, err := os.Stat(usageDir); !os.IsNotExist(err) {
		t.Fatalf("read-only headless should not write skill router usage ledger, stat error = %v", err)
	}
}

func TestHeadlessReadOnlyActivateSkillDoesNotWriteUsageLedger(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := testSubDir(t)
	markProjectMapTestRoot(t, dir)
	writeSkillRoutingTestSkill(t, dir, "demo", "Demo skill description.")

	cfg := newProjectMapDisabledConfig()
	cfg.Skills.Router.UsageLedger = true
	provider := &sequenceMockProvider{
		name: "openai",
		responses: []string{
			headlessToolCallJSON(t, "activate_skill", map[string]string{"name": "demo"}),
			"final response after activated skill",
		},
	}
	result := RunHeadlessWithConfigOptions(context.Background(), "activate demo skill", "gpt-5.4", provider, cfg, HeadlessRunOptions{
		ReadOnly: true,
	})
	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Tool != "activate_skill" || !result.ToolCalls[0].Success {
		t.Fatalf("ToolCalls = %+v, want successful activate_skill call", result.ToolCalls)
	}
	usageDir := filepath.Join(home, ".xelyon", "skills", "router", "usage")
	if _, err := os.Stat(usageDir); !os.IsNotExist(err) {
		t.Fatalf("read-only headless should not write skill activation usage ledger, stat error = %v", err)
	}
}

func TestHeadlessReadOnlyDoesNotBootstrapMCP(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg := newProjectMapDisabledConfig()
	cfg.MCP.Enabled = true
	cfg.MCP.Headless = true

	provider := &headlessToolSetProbeProvider{name: "openai"}
	result := RunHeadlessWithConfigOptions(context.Background(), "probe", "gpt-5.4", provider, cfg, HeadlessRunOptions{
		ReadOnly: true,
	})
	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	if !cfg.MCP.Headless {
		t.Fatal("caller config MCP.Headless was mutated, want unchanged true")
	}
	if _, err := os.Stat(filepath.Join(home, ".xelyon", "mcp.json")); !os.IsNotExist(err) {
		t.Fatalf("read-only headless should not create/load MCP config, stat error = %v", err)
	}
	for _, name := range provider.toolNames {
		if strings.HasPrefix(name, "mcp_") {
			t.Fatalf("read-only headless should not expose MCP tool %s: %v", name, provider.toolNames)
		}
	}
}

func installFakeGitStatusMarker(t *testing.T) string {
	t.Helper()
	binDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "git-called")
	scriptPath := filepath.Join(binDir, "git")
	script := strings.Join([]string{
		"#!/bin/sh",
		"case \" $* \" in",
		"  *\" status \"*)",
		"    printf called > \"$FAKE_GIT_MARKER\"",
		"    printf ' M fake-status-path.go\\n'",
		"    exit 0",
		"    ;;",
		"esac",
		"exit 1",
		"",
	}, "\n")
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(fake git) error = %v", err)
	}
	t.Setenv("FAKE_GIT_MARKER", marker)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return marker
}
