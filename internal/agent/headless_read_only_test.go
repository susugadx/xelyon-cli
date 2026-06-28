package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestHeadlessReadOnlyHidesWriteToolsFromProviderDefinitions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	provider := &headlessToolSetProbeProvider{name: "openai"}
	result := RunHeadlessWithConfigOptions(context.Background(), "probe", "gpt-5.4", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
		ReadOnly: true,
	})
	if result.Status != HeadlessStatusSuccess {
		t.Fatalf("Status = %q, want success: %+v", result.Status, result.Error)
	}
	for _, name := range []string{"apply_patch", "write_file", "str_replace", "delete_file"} {
		if toolNameInList(provider.toolNames, name) {
			t.Fatalf("read-only headless mode should hide write tool %s from provider definitions: %v", name, provider.toolNames)
		}
	}
	for _, name := range []string{"bash", "run_skill_script"} {
		if toolNameInList(provider.toolNames, name) {
			t.Fatalf("read-only headless mode should hide shell execution tool %s from provider definitions: %v", name, provider.toolNames)
		}
	}
	for _, name := range []string{"spawn_agent", "wait_agent"} {
		if toolNameInList(provider.toolNames, name) {
			t.Fatalf("read-only headless mode should hide sub-agent tool %s from provider definitions: %v", name, provider.toolNames)
		}
	}
	for _, name := range []string{"gather_context", "read_file", "search_code"} {
		if !toolNameInList(provider.toolNames, name) {
			t.Fatalf("read-only headless mode should keep read/search tool %s visible: %v", name, provider.toolNames)
		}
	}
}

func TestHeadlessReadOnlyDeniesSkillScriptToolWithoutMutation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := testSubDir(t)
	skillDir := filepath.Join(dir, ".agents", "skills", "demo-script")
	scriptDir := filepath.Join(skillDir, "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillMD := strings.Join([]string{
		"---",
		"name: demo-script",
		"description: test skill script",
		"---",
		"# demo script",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMD), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptDir, "write.sh"), []byte("printf changed > target.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	provider := &sequenceMockProvider{
		name: "openai",
		responses: []string{
			headlessToolCallJSON(t, "run_skill_script", map[string]string{"skill": "demo-script", "script": "write.sh"}),
			"final response after denied skill script",
		},
	}
	result := RunHeadlessWithConfigOptions(context.Background(), "attempt skill script", "gpt-5.4", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
		FailOnToolError: true,
		ReadOnly:        true,
	})

	if result.Status != HeadlessStatusError {
		t.Fatalf("Status = %q, want error", result.Status)
	}
	if result.Error == nil || result.Error.Type != HeadlessErrorTypeReadOnlyViolation {
		t.Fatalf("Error = %+v, want %s", result.Error, HeadlessErrorTypeReadOnlyViolation)
	}
	if result.FailureReason != HeadlessFailureReasonReadOnlyViolation {
		t.Fatalf("FailureReason = %q, want %q", result.FailureReason, HeadlessFailureReasonReadOnlyViolation)
	}
	if result.Response != "final response after denied skill script" {
		t.Fatalf("Response = %q, want preserved final response", result.Response)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("ToolCalls length = %d, want 1", len(result.ToolCalls))
	}
	call := result.ToolCalls[0]
	if call.Tool != "run_skill_script" || call.Success {
		t.Fatalf("ToolCalls[0] = %+v, want failed run_skill_script call", call)
	}
	if !strings.Contains(call.Output, "read-only mode denied skill script execution tool") {
		t.Fatalf("ToolCalls[0].Output = %q, want skill script read-only denial", call.Output)
	}
	if _, err := os.Stat(filepath.Join(dir, "target.txt")); !os.IsNotExist(err) {
		t.Fatalf("target.txt stat error = %v, want absent file", err)
	}
}

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

func TestHeadlessReadOnlyDeniesMCPToolCall(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	testSubDir(t)

	provider := &sequenceMockProvider{
		name: "openai",
		responses: []string{
			headlessToolCallJSON(t, "mcp_github_create_issue", map[string]string{"title": "should not run"}),
			"final response after denied MCP",
		},
	}
	result := RunHeadlessWithConfigOptions(context.Background(), "attempt mcp write", "gpt-5.4", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
		FailOnToolError: true,
		ReadOnly:        true,
	})

	if result.Status != HeadlessStatusError {
		t.Fatalf("Status = %q, want error", result.Status)
	}
	if result.Error == nil || result.Error.Type != HeadlessErrorTypeReadOnlyViolation {
		t.Fatalf("Error = %+v, want %s", result.Error, HeadlessErrorTypeReadOnlyViolation)
	}
	if result.FailureReason != HeadlessFailureReasonReadOnlyViolation {
		t.Fatalf("FailureReason = %q, want %q", result.FailureReason, HeadlessFailureReasonReadOnlyViolation)
	}
	if result.Response != "final response after denied MCP" {
		t.Fatalf("Response = %q, want preserved final response", result.Response)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("ToolCalls length = %d, want 1", len(result.ToolCalls))
	}
	call := result.ToolCalls[0]
	if call.Tool != "mcp_github_create_issue" || call.Success {
		t.Fatalf("ToolCalls[0] = %+v, want failed MCP call", call)
	}
	if !strings.Contains(call.Output, "read-only mode denied MCP tool") {
		t.Fatalf("ToolCalls[0].Output = %q, want MCP read-only denial", call.Output)
	}
}

func TestHeadlessReadOnlyDeniesSubAgentToolCall(t *testing.T) {
	tests := []struct {
		name string
		tool string
		args map[string]string
	}{
		{
			name: "spawn_agent",
			tool: "spawn_agent",
			args: map[string]string{"message": "write something", "task_type": "edit"},
		},
		{
			name: "wait_agent",
			tool: "wait_agent",
			args: map[string]string{"ids": `["agent-1"]`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			testSubDir(t)

			provider := &sequenceMockProvider{
				name: "openai",
				responses: []string{
					headlessToolCallJSON(t, tt.tool, tt.args),
					"final response after denied sub-agent",
				},
			}
			result := RunHeadlessWithConfigOptions(context.Background(), "attempt sub-agent", "gpt-5.4", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
				FailOnToolError: true,
				ReadOnly:        true,
			})

			if result.Status != HeadlessStatusError {
				t.Fatalf("Status = %q, want error", result.Status)
			}
			if result.Error == nil || result.Error.Type != HeadlessErrorTypeReadOnlyViolation {
				t.Fatalf("Error = %+v, want %s", result.Error, HeadlessErrorTypeReadOnlyViolation)
			}
			if result.FailureReason != HeadlessFailureReasonReadOnlyViolation {
				t.Fatalf("FailureReason = %q, want %q", result.FailureReason, HeadlessFailureReasonReadOnlyViolation)
			}
			if result.Response != "final response after denied sub-agent" {
				t.Fatalf("Response = %q, want preserved final response", result.Response)
			}
			if len(result.ToolCalls) != 1 {
				t.Fatalf("ToolCalls length = %d, want 1", len(result.ToolCalls))
			}
			call := result.ToolCalls[0]
			if call.Tool != tt.tool || call.Success {
				t.Fatalf("ToolCalls[0] = %+v, want failed %s call", call, tt.tool)
			}
			if !strings.Contains(call.Output, "read-only mode denied sub-agent tool") {
				t.Fatalf("ToolCalls[0].Output = %q, want sub-agent read-only denial", call.Output)
			}
		})
	}
}

func TestHeadlessReadOnlyDeniesWriteToolsWithoutMutation(t *testing.T) {
	tests := []struct {
		name        string
		tool        string
		args        map[string]string
		initial     string
		wantContent string
		wantExists  bool
	}{
		{
			name:       "write_file",
			tool:       "write_file",
			args:       map[string]string{"path": "target.txt", "content": "changed\n"},
			wantExists: false,
		},
		{
			name:        "str_replace",
			tool:        "str_replace",
			args:        map[string]string{"path": "target.txt", "old_str": "original", "new_str": "changed"},
			initial:     "original\n",
			wantContent: "original\n",
			wantExists:  true,
		},
		{
			name:        "delete_file",
			tool:        "delete_file",
			args:        map[string]string{"path": "target.txt"},
			initial:     "original\n",
			wantContent: "original\n",
			wantExists:  true,
		},
		{
			name:       "apply_patch",
			tool:       "apply_patch",
			args:       map[string]string{"patch": "*** Begin Patch\n*** Add File: target.txt\n+changed\n*** End Patch"},
			wantExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			dir := testSubDir(t)
			targetPath := filepath.Join(dir, "target.txt")
			if tt.initial != "" {
				if err := os.WriteFile(targetPath, []byte(tt.initial), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			provider := &sequenceMockProvider{
				name: "openai",
				responses: []string{
					headlessToolCallJSON(t, tt.tool, tt.args),
					"final response after denied write",
				},
			}
			result := RunHeadlessWithConfigOptions(context.Background(), "attempt write", "gpt-5.4", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
				ReadOnly: true,
			})

			if result.Status != HeadlessStatusSuccess {
				t.Fatalf("Status = %q, want success without FailOnToolError: %+v", result.Status, result.Error)
			}
			if result.FailureReason != "" {
				t.Fatalf("FailureReason = %q, want empty", result.FailureReason)
			}
			if result.Response != "final response after denied write" {
				t.Fatalf("Response = %q, want final response", result.Response)
			}
			if len(result.ToolCalls) != 1 {
				t.Fatalf("ToolCalls length = %d, want 1", len(result.ToolCalls))
			}
			call := result.ToolCalls[0]
			if call.Tool != tt.tool || call.Success {
				t.Fatalf("ToolCalls[0] = %+v, want failed %s", call, tt.tool)
			}
			if !strings.Contains(call.Output, "read-only mode denied") {
				t.Fatalf("ToolCalls[0].Output = %q, want read-only denial", call.Output)
			}
			content, err := os.ReadFile(targetPath)
			if !tt.wantExists {
				if !os.IsNotExist(err) {
					t.Fatalf("target file exists or read error = %v, want absent", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			if string(content) != tt.wantContent {
				t.Fatalf("target content = %q, want %q", string(content), tt.wantContent)
			}
		})
	}
}

func TestHeadlessReadOnlyDeniesAllBashAndSummarizesCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		setup   func(t *testing.T, dir string)
		assert  func(t *testing.T, dir string)
	}{
		{
			name:    "read_only_like_command",
			command: "pwd",
		},
		{
			name:    "find_delete",
			command: "find . -delete",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("keep\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			assert: func(t *testing.T, dir string) {
				t.Helper()
				if _, err := os.Stat(filepath.Join(dir, "keep.txt")); err != nil {
					t.Fatalf("keep.txt stat error = %v, want file preserved", err)
				}
			},
		},
		{
			name:    "command_substitution_touch",
			command: "echo $(touch target.txt)",
			assert: func(t *testing.T, dir string) {
				t.Helper()
				if _, err := os.Stat(filepath.Join(dir, "target.txt")); !os.IsNotExist(err) {
					t.Fatalf("target.txt stat error = %v, want absent file", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			dir := testSubDir(t)
			if tt.setup != nil {
				tt.setup(t, dir)
			}

			provider := &sequenceMockProvider{
				name: "openai",
				responses: []string{
					headlessToolCallJSON(t, "bash", map[string]string{"command": tt.command}),
					"final response after denied bash",
				},
			}
			result := RunHeadlessWithConfigOptions(context.Background(), "attempt bash", "gpt-5.4", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
				ReadOnly: true,
			})

			if result.Status != HeadlessStatusSuccess {
				t.Fatalf("Status = %q, want success without FailOnToolError: %+v", result.Status, result.Error)
			}
			if len(result.ToolCalls) != 1 || result.ToolCalls[0].Success {
				t.Fatalf("ToolCalls = %+v, want one failed bash call", result.ToolCalls)
			}
			if !strings.Contains(result.ToolCalls[0].Output, "read-only mode denied bash tool") {
				t.Fatalf("ToolCalls[0].Output = %q, want bash read-only denial", result.ToolCalls[0].Output)
			}
			if tt.assert != nil {
				tt.assert(t, dir)
			}
			if result.Summary == nil || len(result.Summary.Commands) != 1 {
				t.Fatalf("Summary = %+v, want one denied command", result.Summary)
			}
			summary := result.Summary.Commands[0]
			if summary.Command != tt.command || summary.Status != headlessSummaryStatusFailed || summary.ExitCode != -1 || summary.Source != headlessCommandSourceTool {
				t.Fatalf("command summary = %+v, want failed/-1 tool summary", summary)
			}
		})
	}
}

func TestHeadlessReadOnlyStrictPromotesReadOnlyViolation(t *testing.T) {
	provider := &headlessToolErrorUsageProvider{
		responses: []string{
			headlessToolCallJSON(t, "write_file", map[string]string{"path": "target.txt", "content": "changed\n"}),
			"final response after denied write",
		},
	}
	t.Setenv("HOME", t.TempDir())
	testSubDir(t)

	result := RunHeadlessWithConfigOptions(context.Background(), "attempt write", "gpt-5.4-nano", provider, newProjectMapDisabledConfig(), HeadlessRunOptions{
		FailOnToolError: true,
		ReadOnly:        true,
	})

	if result.Status != HeadlessStatusError {
		t.Fatalf("Status = %q, want error", result.Status)
	}
	if result.Error == nil || result.Error.Type != HeadlessErrorTypeReadOnlyViolation {
		t.Fatalf("Error = %+v, want %s", result.Error, HeadlessErrorTypeReadOnlyViolation)
	}
	if result.FailureReason != HeadlessFailureReasonReadOnlyViolation {
		t.Fatalf("FailureReason = %q, want %q", result.FailureReason, HeadlessFailureReasonReadOnlyViolation)
	}
	if result.Response != "final response after denied write" {
		t.Fatalf("Response = %q, want preserved final response", result.Response)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Success {
		t.Fatalf("ToolCalls = %+v, want one failed denied tool call", result.ToolCalls)
	}
	if result.Tokens == nil {
		t.Fatal("Tokens = nil, want preserved token usage")
	}
	if result.Tokens.Input != 100 || result.Tokens.Cached != 20 || result.Tokens.Output != 30 || result.Tokens.Thinking != 5 || result.Tokens.Total != 135 {
		t.Fatalf("Tokens = %+v, want input=100 cached=20 output=30 thinking=5 total=135", result.Tokens)
	}
	if result.Cost <= 0 {
		t.Fatalf("Cost = %f, want preserved positive cost", result.Cost)
	}
}

func headlessToolCallJSON(t *testing.T, tool string, args map[string]string) string {
	t.Helper()
	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf(`{"tool":%q,"args":%s}`, tool, argsJSON)
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
