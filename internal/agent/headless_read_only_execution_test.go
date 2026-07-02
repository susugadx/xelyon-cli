package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
